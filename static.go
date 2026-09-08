package static

import (
	"embed"
	"io/fs"
	"net/http"
)

// Embed the static directory
//
//go:embed static
var staticFiles embed.FS

func FileServer() http.Handler {
	// Create a sub-filesystem that strips the "static" prefix
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}

	return http.FileServer(http.FS(staticFS))
}
