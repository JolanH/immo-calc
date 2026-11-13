package server

import (
	"io/fs"
	"log"
	"net/http"
	"strings"
)

// staticHandler serves files from the built frontend and falls back to
// index.html for every other path, so Vue Router deep links survive a reload.
func staticHandler(files fs.FS) http.Handler {
	fileServer := http.FileServerFS(files)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name != "" && name != "index.html" {
			if info, err := fs.Stat(files, name); err == nil && !info.IsDir() {
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		serveIndex(w, files)
	})
}

func serveIndex(w http.ResponseWriter, files fs.FS) {
	index, err := fs.ReadFile(files, "index.html")
	if err != nil {
		log.Printf("read embedded index.html: %v", err)
		http.Error(w, "index.html introuvable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(index)
}
