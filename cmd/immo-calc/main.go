// Command immo-calc starts the local immo-calc web app.
package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"immo-calc/internal/browser"
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

	ln, err := listen(cfg.Port, net.Listen, probeHealth)
	var running *alreadyRunningError
	if errors.As(err, &running) {
		log.Printf("immo-calc is already running at %s", running.URL)
		openBrowser(cfg, running.URL)
		return
	}
	if err != nil {
		log.Fatalf("start: %v", err)
	}

	url := appURL(ln)
	log.Printf("immo-calc %s listening on %s (data: %s)", version, url, cfg.DataPath)
	openBrowser(cfg, url)
	handler := server.New(server.Config{Version: version, Static: web.Dist()})
	if err := http.Serve(ln, handler); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func openBrowser(cfg Config, url string) {
	if cfg.NoBrowser {
		return
	}
	if err := browser.Open(url); err != nil {
		log.Printf("could not open the browser, open %s by hand: %v", url, err)
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
