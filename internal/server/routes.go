package server

import (
	"net/http"

	static "github.com/aztechian/heirloom"
)

func applyRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", healthz)
	mux.Handle("/", static.FileServer())
}

func healthz(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("OK"))
}
