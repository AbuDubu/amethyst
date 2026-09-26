// Package httpserver assembles the application's HTTP routes: the JSON API
// under /api/ and the single-page frontend for every other path.
package httpserver

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// New returns the root handler. api serves every /api/ path, including
// unknown ones, so an API miss never falls through to the frontend. web must
// contain the built frontend, including index.html; a missing build is
// reported now rather than as blank pages later.
func New(web fs.FS, api http.Handler) (http.Handler, error) {
	if _, err := fs.Stat(web, "index.html"); err != nil {
		return nil, fmt.Errorf("frontend build not found (index.html missing); run `make web-build`: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", api)
	// "/" rather than "GET /": ServeMux rejects "GET /" alongside "/api/" because
	// neither pattern is more specific than the other (one narrows the method, the
	// other the path). spa enforces the method itself.
	mux.Handle("/", spa(web))
	return mux, nil
}

// spa serves files from the frontend build. Paths that are not files are
// client-side routes, so they receive index.html and React Router takes over.
// A missing path that looks like a file (has an extension) gets a real 404, so
// a stale script or image reference fails visibly instead of receiving HTML.
func spa(web fs.FS) http.Handler {
	files := http.FileServerFS(web)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name == "" {
			files.ServeHTTP(w, r)
			return
		}

		info, err := fs.Stat(web, name)
		switch {
		case err == nil && !info.IsDir():
			files.ServeHTTP(w, r)
		case errors.Is(err, fs.ErrNotExist) && path.Ext(name) != "":
			http.NotFound(w, r)
		default:
			http.ServeFileFS(w, r, web, "index.html")
		}
	})
}
