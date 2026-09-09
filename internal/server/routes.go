package server

import (
	"net/http"

	static "github.com/aztechian/heirloom"
	"github.com/aztechian/heirloom/internal/api/types"
)

func applyRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", healthz)
	mux.HandleFunc("GET /openapi.json", openapiSpec)
	mux.Handle("GET /docs/", http.StripPrefix("/docs", swaggerUI()))
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
