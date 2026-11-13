package server

import (
	"io/fs"
	"log"
	"net/http"
	"path"
	"strings"
)

// contentTypes fixes the Content-Type of the frontend's files. Without it
// net/http asks the OS, and on Windows that means the registry, which can map
// .js to text/plain; browsers then refuse to run the app's module scripts.
var contentTypes = map[string]string{
	".css":   "text/css; charset=utf-8",
	".html":  "text/html; charset=utf-8",
	".ico":   "image/vnd.microsoft.icon",
	".js":    "text/javascript; charset=utf-8",
	".json":  "application/json",
	".map":   "application/json",
	".mjs":   "text/javascript; charset=utf-8",
	".png":   "image/png",
	".svg":   "image/svg+xml",
	".webp":  "image/webp",
	".woff2": "font/woff2",
}

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
				// FileServer keeps a Content-Type that is already set.
				if ct, ok := contentTypes[strings.ToLower(path.Ext(name))]; ok {
					w.Header().Set("Content-Type", ct)
				}
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
