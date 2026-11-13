package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func shutdownRequest(origin string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/shutdown", nil)
	req.Host = "127.0.0.1:8421"
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	return req
}

func TestShutdownAnswersThenCallsTheCallback(t *testing.T) {
	called := make(chan struct{}, 1)
	h := New(Config{Version: "dev", Shutdown: func() { called <- struct{}{} }})
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, shutdownRequest("http://127.0.0.1:8421"))

	if rec.Code != http.StatusOK || rec.Body.String() != "{}\n" {
		t.Fatalf("status = %d, body = %q", rec.Code, rec.Body.String())
	}
	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("shutdown callback not called")
	}
}

func TestShutdownRejectsForeignOrigin(t *testing.T) {
	called := make(chan struct{}, 1)
	h := New(Config{Version: "dev", Shutdown: func() { called <- struct{}{} }})
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, shutdownRequest("http://evil.example"))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	select {
	case <-called:
		t.Fatal("shutdown callback called for a foreign origin")
	case <-time.After(50 * time.Millisecond):
	}
}
