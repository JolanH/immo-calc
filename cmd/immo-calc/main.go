// Command immo-calc starts the local immo-calc web app.
package main

import (
	"log"
	"net/http"

	"immo-calc/internal/server"
)

// version is injected at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	addr := "127.0.0.1:8421"
	log.Printf("immo-calc %s listening on http://%s", version, addr)
	if err := http.ListenAndServe(addr, server.New(server.Config{Version: version})); err != nil {
		log.Fatalf("server: %v", err)
	}
}
