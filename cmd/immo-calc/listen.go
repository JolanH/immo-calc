package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"immo-calc/internal/server"
)

// loopback is the only interface immo-calc ever listens on.
const loopback = "127.0.0.1"

type listenFunc func(network, addr string) (net.Listener, error)

// alreadyRunningError means another immo-calc already serves the preferred port.
type alreadyRunningError struct{ URL string }

func (e *alreadyRunningError) Error() string {
	return "immo-calc is already running at " + e.URL
}

// listen binds the preferred port on 127.0.0.1. If it's taken by another
// immo-calc it returns *alreadyRunningError; if it's taken by something else
// it binds any free port instead.
func listen(port int, listenFn listenFunc, probe func(baseURL string) bool) (net.Listener, error) {
	ln, err := listenFn("tcp", net.JoinHostPort(loopback, strconv.Itoa(port)))
	if err == nil {
		return ln, nil
	}
	if port != 0 {
		if base := baseURL(port); probe(base) {
			return nil, &alreadyRunningError{URL: base + "/"}
		}
		if ln, errAny := listenFn("tcp", net.JoinHostPort(loopback, "0")); errAny == nil {
			return ln, nil
		}
	}
	return nil, fmt.Errorf("listen on %s port %d: %w", loopback, port, err)
}

func baseURL(port int) string {
	return "http://" + net.JoinHostPort(loopback, strconv.Itoa(port))
}

// appURL is the URL to open in the browser for a bound listener.
func appURL(ln net.Listener) string {
	return baseURL(ln.Addr().(*net.TCPAddr).Port) + "/"
}

// probeHealth reports whether an immo-calc instance answers at base.
func probeHealth(base string) bool {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(base + "/api/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	var health struct {
		App string `json:"app"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&health) != nil {
		return false
	}
	return health.App == server.AppName
}
