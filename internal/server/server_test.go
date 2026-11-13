package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	h := New(Config{Version: "1.2.3"})
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	want := `{"status":"ok","app":"immo-calc","version":"1.2.3"}` + "\n"
	if got := rec.Body.String(); got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

func TestHealthRejectsOtherMethods(t *testing.T) {
	h := New(Config{Version: "dev"})
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/health", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestUnknownAPIRouteReturnsJSON404(t *testing.T) {
	h := New(Config{Version: "dev"})

	for _, path := range []string{"/api/unknown", "/api/loans/x/y", "/api"} {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d", rec.Code)
			}
			want := `{"error":{"code":"not_found","message":"Ressource introuvable"}}` + "\n"
			if got := rec.Body.String(); got != want {
				t.Errorf("body = %q", got)
			}
		})
	}
}
