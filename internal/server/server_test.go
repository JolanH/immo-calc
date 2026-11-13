package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	h := New(Config{Version: "1.2.3"})
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, localRequest(http.MethodGet, "/api/health"))

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

	h.ServeHTTP(rec, localRequest(http.MethodPost, "/api/health"))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestUnknownAPIRouteReturnsJSON404(t *testing.T) {
	h := New(Config{Version: "dev"})

	for _, path := range []string{"/api/unknown", "/api/loans/x/y", "/api"} {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, localRequest(http.MethodGet, path))

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

// localRequest builds a request addressed to 127.0.0.1, as the browser sends it.
func localRequest(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.Host = "127.0.0.1:8421"
	return req
}

func TestModuleRoutesAreRegistered(t *testing.T) {
	h := New(Config{Version: "dev", Routes: []func(*http.ServeMux){
		func(mux *http.ServeMux) {
			mux.HandleFunc("GET /api/things", func(w http.ResponseWriter, _ *http.Request) {
				WriteJSON(w, http.StatusOK, []string{"a"})
			})
		},
	}})
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, localRequest(http.MethodGet, "/api/things"))

	if rec.Code != http.StatusOK || rec.Body.String() != "[\"a\"]\n" {
		t.Errorf("status = %d, body = %q", rec.Code, rec.Body.String())
	}
}
