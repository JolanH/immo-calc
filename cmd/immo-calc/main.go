// Command immo-calc starts the local immo-calc web app.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

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

	// Stop on Ctrl+C / SIGTERM, or when the UI's "Quitter" calls /api/shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	srv := &http.Server{
		Handler:           server.New(server.Config{Version: version, Static: web.Dist(), Shutdown: stop}),
		ReadHeaderTimeout: 10 * time.Second,
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()
	openBrowser(cfg, url)

	select {
	case err := <-serveErr:
		log.Fatalf("server: %v", err)
	case <-ctx.Done():
	}
	log.Printf("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown: %v", err)
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
