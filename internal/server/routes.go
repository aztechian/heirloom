package server

import (
	"fmt"
	"net/http"

	static "github.com/aztechian/heirloom"
	"github.com/aztechian/heirloom/internal/api/types"
)

func applyRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", healthz)
	mux.HandleFunc("GET /openapi.json", openapiSpec)
	mux.Handle("GET /docs/", http.StripPrefix("/docs", swaggerUI()))

	spec, err := types.GetSpec()
	if err != nil {
		// The spec is embedded at build time, so a load failure here means
		// the binary itself is broken, not a condition to recover from.
		panic(fmt.Sprintf("failed to load embedded OpenAPI spec: %v", err))
	}

	// Request validation runs only in front of the generated API routes, so
	// it never gets a chance to reject requests for /healthz, /docs, or the
	// static frontend, none of which the spec describes.
	apiMux := http.NewServeMux()
	types.HandlerFromMux(types.NewStrictHandler(apiHandlers{}, nil), apiMux)
	mux.Handle("/api/v1/", requestValidation(spec)(apiMux))

	mux.Handle("/", static.FileServer())
}

func healthz(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("OK"))
}

// openapiSpec serves the embedded spec verbatim, so an instance always
// reports the contract it actually implements rather than a copy that can
// drift from the running binary.
func openapiSpec(w http.ResponseWriter, r *http.Request) {
	spec, err := types.GetSpecJSON()
	if err != nil {
		http.Error(w, "failed to load embedded OpenAPI spec", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(spec)
}
