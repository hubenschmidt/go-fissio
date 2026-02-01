package editor

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed dist/*
var assets embed.FS

// Handler returns an http.Handler that serves the embedded editor UI.
// It serves static files from the embedded dist/ directory and falls back
// to index.html for SPA routing.
func Handler() http.Handler {
	distFS, _ := fs.Sub(assets, "dist")
	fileServer := http.FileServer(http.FS(distFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqPath := strings.TrimPrefix(r.URL.Path, "/")

		if reqPath == "" {
			reqPath = "index.html"
		}

		if _, err := fs.Stat(distFS, reqPath); err != nil {
			r.URL.Path = "/"
			reqPath = "index.html"
		}

		ext := path.Ext(reqPath)
		mimeTypes := map[string]string{
			".html": "text/html; charset=utf-8",
			".css":  "text/css; charset=utf-8",
			".js":   "application/javascript",
			".json": "application/json",
			".svg":  "image/svg+xml",
			".png":  "image/png",
			".ico":  "image/x-icon",
		}

		if mime, ok := mimeTypes[ext]; ok {
			w.Header().Set("Content-Type", mime)
		}

		fileServer.ServeHTTP(w, r)
	})
}
