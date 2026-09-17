package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/aztechian/heirloom/internal/api/types"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/gorilla/handlers"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
)

const (
	etagField       = "etag"
	requestIDField  = "requestid"
	remoteAddrField = "remoteaddr"
	requestIdHeader = "X-Request-Id"
)

func Chain(mws ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(finalHandler http.Handler) http.Handler {
		for i := len(mws) - 1; i >= 0; i-- {
			finalHandler = mws[i](finalHandler)
		}
		return finalHandler
	}
}

// applyGlobalMiddleware is the single place to add/remove middleware applied to every request.
// The stack is built here (not a package var) so hlog.NewHandler captures log.Logger
// as configured in main(), not the zerolog default set at package init.
func applyGlobalMiddleware(handler http.Handler) http.Handler {
	return Chain(
		hlog.NewHandler(log.Logger),
		handlers.ProxyHeaders,
		hlog.RequestIDHandler(requestIDField, requestIdHeader),
		hlog.EtagHandler(etagField),
		hlog.RemoteAddrHandler(remoteAddrField),
		serverHeader,
		logging,
		// TimeoutHandler runs the next handler in a separate goroutine, so
		// RecoveryHandler must wrap it (not the other way around) to catch panics.
		func(h http.Handler) http.Handler {
			return http.TimeoutHandler(h, requestTimeout, "server timeout")
		},
		handlers.RecoveryHandler(handlers.PrintRecoveryStack(true)),
	)(handler)
}

func logging(next http.Handler) http.Handler {
	return hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		hlog.FromRequest(r).
			Info().
			Int("status", status).
			Str("path", r.URL.Path).
			Int("size", size).
			Dur("duration", duration).
			Str("method", r.Method).
			Str("scheme", r.URL.Scheme).
			Str(remoteAddrField, r.RemoteAddr).
			Msg("http request")
	})(next)
}

func serverHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "heirloom")
		next.ServeHTTP(w, r)
	})
}

// requestValidation enforces the `pattern`/`minLength`/`maxLength`/etc.
// constraints declared in openapi.yaml before a request reaches a handler, so
// handlers only need to implement the constraints the schema can't express
// (e.g. invariants over a derived value). Authentication isn't implemented
// yet, so NoopAuthenticationFunc is used rather than rejecting every request
// for lacking credentials the server doesn't yet check.
func requestValidation(spec *openapi3.T) func(http.Handler) http.Handler {
	return nethttpmiddleware.OapiRequestValidatorWithOptions(spec, &nethttpmiddleware.Options{
		Options: openapi3filter.Options{
			MultiError:         true,
			AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
		},
		// The spec declares a `localhost` dev server; validating the Host
		// header against it would reject every request against a real deployment.
		SilenceServersWarning: true,
		DoNotValidateServers:  true,
		ErrorHandlerWithOpts:  validationErrorHandler,
	})
}

func validationErrorHandler(_ context.Context, err error, w http.ResponseWriter, _ *http.Request, _ nethttpmiddleware.ErrorHandlerOpts) {
	p := requestErrorToProblem(err)
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(int(p.Status))
	_ = json.NewEncoder(w).Encode(p)
}

// requestErrorToProblem classifies a validation error as either 400
// (structurally malformed: missing/unparsable body, wrong content type) or
// 422 (well-formed but violates a schema constraint), matching the status
// convention documented in openapi.yaml.
func requestErrorToProblem(err error) types.Problem {
	var multi openapi3.MultiError
	if errors.As(err, &multi) {
		fieldErrors := make([]types.FieldError, 0, len(multi))
		status := http.StatusUnprocessableEntity
		for _, e := range multi {
			fe, s := requestErrorDetail(e)
			fieldErrors = append(fieldErrors, fe)
			if s == http.StatusBadRequest {
				status = http.StatusBadRequest
			}
		}

		return problemForStatus(status, fieldErrors)
	}

	var secErr *openapi3filter.SecurityRequirementsError
	if errors.As(err, &secErr) {
		return problem(http.StatusUnauthorized, "Unauthorized", secErr.Error())
	}

	fe, status := requestErrorDetail(err)

	return problemForStatus(status, []types.FieldError{fe})
}

func problemForStatus(status int, fieldErrors []types.FieldError) types.Problem {
	if status == http.StatusBadRequest {
		return problem(http.StatusBadRequest, "Malformed request", "The request could not be parsed against the API contract.")
	}
	p := problem(http.StatusUnprocessableEntity, "Validation failed", "The request does not satisfy the API contract.")
	p.Errors = &fieldErrors

	return p
}

// requestErrorDetail reports the field a validation error applies to and
// whether it represents a schema constraint violation (422) or a
// structurally malformed request (400).
func requestErrorDetail(err error) (types.FieldError, int) {
	var reqErr *openapi3filter.RequestError
	if !errors.As(err, &reqErr) {
		return types.FieldError{Field: "", Message: err.Error()}, http.StatusBadRequest
	}

	var schemaErr *openapi3.SchemaError
	if errors.As(reqErr.Err, &schemaErr) {
		return types.FieldError{Field: requestErrorField(reqErr, schemaErr), Message: schemaErr.Reason}, http.StatusUnprocessableEntity
	}

	return types.FieldError{Field: requestErrorField(reqErr, nil), Message: reqErr.Reason}, http.StatusBadRequest
}

func requestErrorField(reqErr *openapi3filter.RequestError, schemaErr *openapi3.SchemaError) string {
	if reqErr.Parameter != nil {
		return reqErr.Parameter.Name
	}
	if schemaErr != nil {
		return strings.Join(schemaErr.JSONPointer(), "/")
	}

	return "body"
}
