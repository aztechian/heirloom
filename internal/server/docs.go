package server

import (
	"net/http"
	"strings"

	swaggerFiles "github.com/swaggo/files"
)

// swaggerInitializer replaces the asset's default one, which points at the
// public Swagger Petstore demo, so the UI instead loads this instance's own
// embedded spec.
const swaggerInitializer = `window.onload = function() {
  window.ui = SwaggerUIBundle({
    url: "/openapi.json",
    dom_id: "#swagger-ui",
    presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
    layout: "StandaloneLayout",
  });
};`

// swaggerUI serves the vendored swagger-ui-dist assets, so the browsable
// docs work without a request ever leaving the instance.
func swaggerUI() http.Handler {
	assets := http.FileServer(&swaggerFiles.HTTPFS{})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/swagger-initializer.js") {
			w.Header().Set("Content-Type", "application/javascript")
			_, _ = w.Write([]byte(swaggerInitializer))
			return
		}
		assets.ServeHTTP(w, r)
	})
}
