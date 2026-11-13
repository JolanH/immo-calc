package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"immo-calc/internal/server"
)

// loopback is the only interface immo-calc ever listens on.
const loopback = "127.0.0.1"

// How long to wait for a replaced instance to release the port: its graceful
// shutdown is bounded by 5 s.
const (
	releaseAttempts = 55
	releaseInterval = 100 * time.Millisecond
)

type listenFunc func(network, addr string) (net.Listener, error)

// alreadyRunningError means the same version of immo-calc already serves the
// preferred port.
type alreadyRunningError struct{ URL string }

func (e *alreadyRunningError) Error() string {
	return "immo-calc is already running at " + e.URL
}

// binder picks the listening socket. Its functions are injected for tests.
type binder struct {
	listen   listenFunc
	probe    func(baseURL string) (version string, ok bool)
	shutdown func(baseURL string) error
	sleep    func(time.Duration)
}

func newSystemBinder() binder {
	return binder{listen: net.Listen, probe: probeHealth, shutdown: shutdownInstance, sleep: time.Sleep}
}

// bind binds the preferred port on 127.0.0.1. When it's taken:
//   - by the same immo-calc version: returns *alreadyRunningError;
//   - by another immo-calc version: shuts it down and takes its port;
//   - by anything else (or the old version won't stop): binds any free port.
func (b binder) bind(port int, version string) (net.Listener, error) {
	addr := net.JoinHostPort(loopback, strconv.Itoa(port))
	ln, err := b.listen("tcp", addr)
	if err == nil || port == 0 {
		return ln, err
	}

	base := baseURL(port)
	if running, ok := b.probe(base); ok {
		if running == version {
			return nil, &alreadyRunningError{URL: base + "/"}
		}
		if ln := b.replace(addr, base, running, version); ln != nil {
			return ln, nil
		}
	}
	if ln, errAny := b.listen("tcp", net.JoinHostPort(loopback, "0")); errAny == nil {
		return ln, nil
	}
	return nil, fmt.Errorf("listen on %s port %d: %w", loopback, port, err)
}

// replace stops the other version running at base and binds addr once it's
// free. It returns nil if that didn't work in time.
func (b binder) replace(addr, base, oldVersion, newVersion string) net.Listener {
	slog.Info("another immo-calc version is running, replacing it", "running", oldVersion, "new", newVersion)
	if err := b.shutdown(base); err != nil {
		slog.Warn("could not stop the other version", "err", err)
		return nil
	}
	for range releaseAttempts {
		if ln, err := b.listen("tcp", addr); err == nil {
			return ln
		}
		b.sleep(releaseInterval)
	}
	slog.Warn("the other version did not release the port in time", "addr", addr)
	return nil
}

func baseURL(port int) string {
	return "http://" + net.JoinHostPort(loopback, strconv.Itoa(port))
}

// appURL is the URL to open in the browser for a bound listener.
func appURL(ln net.Listener) string {
	return baseURL(ln.Addr().(*net.TCPAddr).Port) + "/"
}

// probeHealth reports whether an immo-calc instance answers at base, and its version.
func probeHealth(base string) (string, bool) {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(base + "/api/health")
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	var health struct {
		App     string `json:"app"`
		Version string `json:"version"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&health) != nil ||
		health.App != server.AppName {
		return "", false
	}
	return health.Version, true
}

// shutdownInstance asks the immo-calc at base to stop, as its own UI would.
func shutdownInstance(base string) error {
	req, err := http.NewRequest(http.MethodPost, base+"/api/shutdown", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Origin", base)
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("shutdown %s: %w", base, err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("shutdown %s: status %d", base, resp.StatusCode)
	}
	return nil
}
