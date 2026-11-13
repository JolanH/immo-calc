// Package server holds the HTTP routing shared by every module: the API
// helpers, the health endpoint and the embedded frontend.
package server

import (
	"io/fs"
	"net/http"
	"strings"
)

// AppName identifies immo-calc in the health response. It is also how a new
// instance recognises one that is already running.
const AppName = "immo-calc"

// Config holds what the router needs from the rest of the app.
type Config struct {
	Version string
	// Static is the built frontend. When nil, only the API is served.
	Static fs.FS
}

// New returns the root HTTP handler.
func New(cfg Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, struct {
			Status  string `json:"status"`
			App     string `json:"app"`
			Version string `json:"version"`
		}{"ok", AppName, cfg.Version})
	})
	apiFallback := func(w http.ResponseWriter, r *http.Request) { apiNotMatched(mux, w, r) }
	mux.HandleFunc("/api/", apiFallback)
	mux.HandleFunc("/api", apiFallback)
	if cfg.Static != nil {
		mux.Handle("/", staticHandler(cfg.Static))
	}
	return mux
}

// apiNotMatched answers API requests no route matched: 405 when the path
// exists for another method, otherwise the JSON 404.
func apiNotMatched(mux *http.ServeMux, w http.ResponseWriter, r *http.Request) {
	var allowed []string
	for _, m := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete} {
		if m == r.Method {
			continue
		}
		probe := r.Clone(r.Context())
		probe.Method = m
		if _, pattern := mux.Handler(probe); pattern != "/api/" && pattern != "/api" {
			allowed = append(allowed, m)
		}
	}
	if len(allowed) == 0 {
		WriteNotFound(w)
		return
	}
	w.Header().Set("Allow", strings.Join(allowed, ", "))
	WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Méthode non autorisée", nil)
}
