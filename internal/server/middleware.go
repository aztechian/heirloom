package server

import (
	"net/http"
	"time"

	"github.com/gorilla/handlers"
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
