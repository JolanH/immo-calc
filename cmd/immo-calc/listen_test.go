package main

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"syscall"
	"testing"
)

// fakeListener is a net.Listener that only reports its address.
type fakeListener struct{ addr string }

func (l fakeListener) Accept() (net.Conn, error) { return nil, errors.New("not implemented") }
func (l fakeListener) Close() error              { return nil }
func (l fakeListener) Addr() net.Addr {
	a, _ := net.ResolveTCPAddr("tcp", l.addr)
	return a
}

// fakeNet records every address it was asked to bind. Ports in busy fail.
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

func probeAnswering(answer bool) func(string) bool {
	return func(string) bool { return answer }
}

func TestListenOnFreePreferredPort(t *testing.T) {
	n := &fakeNet{}

	ln, err := listen(8421, n.listen, probeAnswering(false))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := ln.Addr().String(); got != "127.0.0.1:8421" {
		t.Errorf("addr = %s", got)
	}
}

func TestListenDetectsRunningInstance(t *testing.T) {
	n := &fakeNet{busy: map[string]bool{"127.0.0.1:8421": true}}

	_, err := listen(8421, n.listen, probeAnswering(true))

	var running *alreadyRunningError
	if !errors.As(err, &running) {
		t.Fatalf("err = %v, want alreadyRunningError", err)
	}
	if running.URL != "http://127.0.0.1:8421/" {
		t.Errorf("url = %s", running.URL)
	}
	if len(n.calls) != 1 {
		t.Errorf("bound %d times, want no second server: %v", len(n.calls), n.calls)
	}
}

func TestListenFallsBackToRandomPortWhenTakenByAnotherProgram(t *testing.T) {
	n := &fakeNet{busy: map[string]bool{"127.0.0.1:8421": true}}

	ln, err := listen(8421, n.listen, probeAnswering(false))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := ln.Addr().String(); got != "127.0.0.1:54321" {
		t.Errorf("addr = %s", got)
	}
}

func TestListenReturnsErrorWhenNothingCanBeBound(t *testing.T) {
	n := &fakeNet{busy: map[string]bool{"127.0.0.1:8421": true, "127.0.0.1:0": true}}

	if _, err := listen(8421, n.listen, probeAnswering(false)); err == nil {
		t.Fatal("expected an error")
	}
}

func TestListenOnlyEverBindsLoopback(t *testing.T) {
	n := &fakeNet{busy: map[string]bool{"127.0.0.1:8421": true}}

	_, _ = listen(8421, n.listen, probeAnswering(false))
	_, _ = listen(0, n.listen, probeAnswering(false))

	for _, addr := range n.calls {
		if !strings.HasPrefix(addr, "127.0.0.1:") {
			t.Errorf("bound %q, want 127.0.0.1 only", addr)
		}
	}
}

func TestProbeHealthRecognisesImmoCalc(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{"immo-calc", `{"status":"ok","app":"immo-calc","version":"dev"}`, true},
		{"another app", `{"status":"ok","app":"other"}`, false},
		{"not json", `<html>`, false},
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

			if got := probeHealth(srv.URL); got != tt.want {
				t.Errorf("probeHealth = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProbeHealthFalseWhenNothingAnswers(t *testing.T) {
	if probeHealth("http://127.0.0.1:1") {
		t.Error("expected false")
	}
}
