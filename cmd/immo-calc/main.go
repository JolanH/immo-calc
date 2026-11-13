// Command immo-calc starts the local immo-calc web app.
package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"immo-calc/internal/server"
	"immo-calc/web"
)

// version is injected at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	cfg, err := parseConfig(os.Args[1:], executableDir(), os.Stderr)
	if err != nil {
		os.Exit(2)
	}
	if cfg.ShowVersion {
		fmt.Println("immo-calc", version)
		return
	}

	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(cfg.Port))
	log.Printf("immo-calc %s listening on http://%s (data: %s)", version, addr, cfg.DataPath)
	handler := server.New(server.Config{Version: version, Static: web.Dist()})
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// executableDir returns the directory holding the running binary, or the
// current directory if it can't be determined.
func executableDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return filepath.Dir(exe)
}
