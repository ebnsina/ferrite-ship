package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// SPAHandler serves the built SvelteKit app from dir.
//
// Unknown paths fall back to 200.html — the adapter-static fallback page —
// so client-side routes such as /dashboard/servers resolve on a hard refresh
// instead of 404ing.
func SPAHandler(dir string) (http.Handler, error) {
	fallback := filepath.Join(dir, "200.html")
	if _, err := os.Stat(fallback); err != nil {
		return nil, err
	}

	files := http.FileServer(http.Dir(dir))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := filepath.Clean(r.URL.Path)

		// Hashed build assets are immutable; everything else must revalidate.
		if strings.HasPrefix(clean, "/_app/immutable/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}

		if path := filepath.Join(dir, clean); hasFile(path) {
			files.ServeHTTP(w, r)
			return
		}

		http.ServeFile(w, r, fallback)
	}), nil
}

func hasFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
