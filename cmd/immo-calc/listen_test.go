package main

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"syscall"
	"testing"
	"time"
)

// fakeListener is a net.Listener that only reports its address.
type fakeListener struct{ addr string }

func (l fakeListener) Accept() (net.Conn, error) { return nil, errors.New("not implemented") }
func (l fakeListener) Close() error              { return nil }
func (l fakeListener) Addr() net.Addr {
	a, _ := net.ResolveTCPAddr("tcp", l.addr)
	return a
}

// fakeNet records every address it was asked to bind. Addresses in busy fail.
type fakeNet struct {
	busy  map[string]bool
	calls []string
}

func (f *fakeNet) listen(network, addr string) (net.Listener, error) {
	f.calls = append(f.calls, addr)
	if f.busy[addr] {
		return nil, &net.OpError{Op: "listen", Net: network, Err: syscall.EADDRINUSE}
	}
	if strings.HasSuffix(addr, ":0") {
		addr = strings.TrimSuffix(addr, "0") + "54321"
	}
	return fakeListener{addr: addr}, nil
}

// fakeInstance is what answers on the preferred port, if anything.
type fakeInstance struct {
	running    bool
	version    string
	net        *fakeNet
	stops      bool // shutting it down frees the port
	shutdowns  int
	shutdownTo string
}

func (f *fakeInstance) probe(string) (string, bool) { return f.version, f.running }

func (f *fakeInstance) shutdown(base string) error {
	f.shutdowns++
	f.shutdownTo = base
	if f.stops {
		delete(f.net.busy, "127.0.0.1:8421")
	}
	return nil
}

func newBinder(n *fakeNet, inst *fakeInstance) binder {
	inst.net = n
	return binder{listen: n.listen, probe: inst.probe, shutdown: inst.shutdown, sleep: func(time.Duration) {}}
}

const ourVersion = "v2"

func TestBindOnFreePreferredPort(t *testing.T) {
	n := &fakeNet{}

	ln, err := newBinder(n, &fakeInstance{}).bind(8421, ourVersion)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := ln.Addr().String(); got != "127.0.0.1:8421" {
		t.Errorf("addr = %s", got)
	}
}

func TestBindDetectsTheSameVersionAlreadyRunning(t *testing.T) {
	n := &fakeNet{busy: map[string]bool{"127.0.0.1:8421": true}}
	inst := &fakeInstance{running: true, version: ourVersion}

	_, err := newBinder(n, inst).bind(8421, ourVersion)

	var running *alreadyRunningError
	if !errors.As(err, &running) {
		t.Fatalf("err = %v, want alreadyRunningError", err)
	}
	if running.URL != "http://127.0.0.1:8421/" {
		t.Errorf("url = %s", running.URL)
	}
	if inst.shutdowns != 0 || len(n.calls) != 1 {
		t.Errorf("shutdowns = %d, binds = %v: the running instance must be left alone", inst.shutdowns, n.calls)
	}
}

func TestBindReplacesADifferentVersion(t *testing.T) {
	n := &fakeNet{busy: map[string]bool{"127.0.0.1:8421": true}}
	inst := &fakeInstance{running: true, version: "v1", stops: true}

	ln, err := newBinder(n, inst).bind(8421, ourVersion)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inst.shutdowns != 1 || inst.shutdownTo != "http://127.0.0.1:8421" {
		t.Errorf("shutdowns = %d to %q", inst.shutdowns, inst.shutdownTo)
	}
	if got := ln.Addr().String(); got != "127.0.0.1:8421" {
		t.Errorf("addr = %s, want the preferred port taken over", got)
	}
}

func TestBindFallsBackWhenTheOldVersionDoesNotStop(t *testing.T) {
	n := &fakeNet{busy: map[string]bool{"127.0.0.1:8421": true}}
	inst := &fakeInstance{running: true, version: "v1", stops: false}

	ln, err := newBinder(n, inst).bind(8421, ourVersion)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := ln.Addr().String(); got != "127.0.0.1:54321" {
		t.Errorf("addr = %s, want a random port", got)
	}
}

func TestBindFallsBackToRandomPortWhenTakenByAnotherProgram(t *testing.T) {
	n := &fakeNet{busy: map[string]bool{"127.0.0.1:8421": true}}
	inst := &fakeInstance{running: false}

	ln, err := newBinder(n, inst).bind(8421, ourVersion)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := ln.Addr().String(); got != "127.0.0.1:54321" {
		t.Errorf("addr = %s", got)
	}
	if inst.shutdowns != 0 {
		t.Error("tried to shut down a program that isn't immo-calc")
	}
}

func TestBindReturnsErrorWhenNothingCanBeBound(t *testing.T) {
	n := &fakeNet{busy: map[string]bool{"127.0.0.1:8421": true, "127.0.0.1:0": true}}

	if _, err := newBinder(n, &fakeInstance{}).bind(8421, ourVersion); err == nil {
		t.Fatal("expected an error")
	}
}

func TestBindOnlyEverBindsLoopback(t *testing.T) {
	n := &fakeNet{busy: map[string]bool{"127.0.0.1:8421": true}}
	b := newBinder(n, &fakeInstance{running: true, version: "v1"})

	_, _ = b.bind(8421, ourVersion)
	_, _ = b.bind(0, ourVersion)

	for _, addr := range n.calls {
		if !strings.HasPrefix(addr, "127.0.0.1:") {
			t.Errorf("bound %q, want 127.0.0.1 only", addr)
		}
	}
}

func TestProbeHealthReadsTheVersion(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantVersion string
		wantOK      bool
	}{
		{"immo-calc", `{"status":"ok","app":"immo-calc","version":"871b22c"}`, "871b22c", true},
		{"another app", `{"status":"ok","app":"other","version":"1"}`, "", false},
		{"not json", `<html>`, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/health" {
					http.NotFound(w, r)
					return
				}
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			version, ok := probeHealth(srv.URL)

			if version != tt.wantVersion || ok != tt.wantOK {
				t.Errorf("probeHealth = (%q, %v), want (%q, %v)", version, ok, tt.wantVersion, tt.wantOK)
			}
		})
	}
}

func TestProbeHealthFalseWhenNothingAnswers(t *testing.T) {
	if _, ok := probeHealth("http://127.0.0.1:1"); ok {
		t.Error("expected false")
	}
}

func TestShutdownInstanceSendsItsOwnOrigin(t *testing.T) {
	var gotMethod, gotPath, gotOrigin string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath, gotOrigin = r.Method, r.URL.Path, r.Header.Get("Origin")
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()

	if err := shutdownInstance(srv.URL); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/shutdown" || gotOrigin != srv.URL {
		t.Errorf("got %s %s with Origin %q", gotMethod, gotPath, gotOrigin)
	}
}

func TestShutdownInstanceReportsRefusal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	if err := shutdownInstance(srv.URL); err == nil {
		t.Error("expected an error")
	}
}
