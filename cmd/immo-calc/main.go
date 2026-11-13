// Command immo-calc starts the local immo-calc web app.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"immo-calc/internal/applog"
	"immo-calc/internal/browser"
	"immo-calc/internal/loan"
	"immo-calc/internal/notify"
	"immo-calc/internal/server"
	"immo-calc/internal/storage"
	"immo-calc/web"
)

// version is injected at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	exeDir := executableDir()
	cfg, err := parseConfig(os.Args[1:], os.Stderr)
	if err != nil {
		os.Exit(2)
	}
	if cfg.ShowVersion {
		fmt.Println("immo-calc", version)
		return
	}

	logPath := setupLogging(exeDir, cfg.Debug)
	fatal := func(err error) {
		logError := func(format string, args ...any) { slog.Error(fmt.Sprintf(format, args...)) }
		reportFatal(notify.Fatal, logError, err, logPath)
		os.Exit(1)
	}

	ln, err := newSystemBinder().bind(cfg.Port, version)
	var running *alreadyRunningError
	if errors.As(err, &running) {
		slog.Info("immo-calc is already running, opening it", "url", running.URL)
		openBrowser(cfg, running.URL)
		return
	}
	if err != nil {
		fatal(err)
	}

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		userConfigDir = os.TempDir()
	}
	data, err := openStore(cfg, exeDir, userConfigDir, loan.RegisterSection)
	if err != nil {
		fatal(err)
	}
	dataHandlers := storage.NewHandlers(data.store, storage.HandlerConfig{
		CountKeys:  []string{"properties", "loans"},
		Warning:    data.warning,
		ConfigPath: data.configPath,
		Picker:     storage.ZenityPicker{},
	})

	url := appURL(ln)
	slog.Info("immo-calc started", "version", version, "url", url,
		"data", data.store.Path(), "config", data.configPath, "log", logPath)

	// Stop on Ctrl+C / SIGTERM, or when the UI's "Quitter" calls /api/shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	srv := &http.Server{
		Handler: server.New(server.Config{
			Version:  version,
			Static:   web.Dist(),
			Shutdown: stop,
			Routes:   []func(*http.ServeMux){dataHandlers.Register, loan.NewHandlers(loan.NewService(data.store)).Register},
		}),
		ReadHeaderTimeout: 10 * time.Second,
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()
	openBrowser(cfg, url)

	select {
	case err := <-serveErr:
		fatal(fmt.Errorf("server: %w", err))
	case <-ctx.Done():
	}
	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Warn("graceful shutdown", "err", err)
	}
}

// setupLogging sends logs to immo-calc.log (exe dir, else temp dir) and to
// stdout when there is one. It returns the log file path, or "" if no log
// file could be created.
func setupLogging(exeDir string, debug bool) string {
	var out io.Writer = os.Stdout
	f, path, err := applog.Open(exeDir, os.TempDir())
	if err == nil {
		// The file goes first: on Windows (no console) writes to stdout fail,
		// and MultiWriter stops at the first failing writer.
		out = io.MultiWriter(f, os.Stdout)
	}
	slog.SetDefault(applog.NewLogger(out, debug))
	if err != nil {
		slog.Warn("no log file", "err", err)
	}
	return path
}

func openBrowser(cfg Config, url string) {
	if cfg.NoBrowser {
		return
	}
	if err := browser.Open(url); err != nil {
		slog.Warn("could not open the browser, open the URL by hand", "url", url, "err", err)
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
