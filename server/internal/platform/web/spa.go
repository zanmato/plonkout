// Package web serves the built single page app.
package web

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// serverPrefixes are paths the SPA never answers, so a typo in an API path is
// a 404 rather than index.html.
var serverPrefixes = []string{"/api/", "/oauth/", "/.well-known/", "/mcp"}

// SPA serves files from dir, falling back to index.html for any other GET.
//
// Hashed assets are cached forever. The entry point, the service worker and
// the manifest are always revalidated, because a stale one of those pins every
// user to an old build.
func SPA(dir string) http.Handler {
	files := http.FileServer(http.Dir(dir))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		for _, prefix := range serverPrefixes {
			if strings.HasPrefix(r.URL.Path, prefix) {
				http.NotFound(w, r)
				return
			}
		}

		clean := path.Clean("/" + r.URL.Path)
		if info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(clean))); err != nil || info.IsDir() {
			w.Header().Set("Cache-Control", "no-cache")
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}

		switch {
		case strings.HasPrefix(clean, "/assets/"):
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		default:
			w.Header().Set("Cache-Control", "no-cache")
		}
		files.ServeHTTP(w, r)
	})
}
