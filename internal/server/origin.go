package server

import (
	"net"
	"net/http"
)

// isLoopbackHost reports whether a Host header names this machine. Requiring
// it on the API stops DNS-rebinding pages (evil.example → 127.0.0.1) from
// talking to immo-calc.
func isLoopbackHost(hostport string) bool {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport
	}
	return host == loopbackIP || host == "localhost"
}

const loopbackIP = "127.0.0.1"

func writeForbiddenOrigin(w http.ResponseWriter) {
	WriteError(w, http.StatusForbidden, "forbidden_origin", "Requête refusée : origine non autorisée", nil)
}

// requireLoopbackHost rejects requests whose Host isn't 127.0.0.1 or localhost.
func requireLoopbackHost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isLoopbackHost(r.Host) {
			writeForbiddenOrigin(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireSameOrigin protects state-changing endpoints against cross-site
// requests: the Origin header, when present, must be this server's own origin,
// and requests browsers flag as cross-site are refused. Every module wraps its
// POST/PUT/DELETE handlers with it.
func RequireSameOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isLoopbackHost(r.Host) {
			writeForbiddenOrigin(w)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			if origin != "http://"+r.Host {
				writeForbiddenOrigin(w)
				return
			}
		} else if site := r.Header.Get("Sec-Fetch-Site"); site == "cross-site" || site == "same-site" {
			writeForbiddenOrigin(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}
