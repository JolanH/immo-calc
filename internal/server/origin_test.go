package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var okHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
})

func TestRequireSameOrigin(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		headers  map[string]string
		wantCode int
	}{
		{"same origin", "127.0.0.1:8421", map[string]string{"Origin": "http://127.0.0.1:8421"}, 204},
		{"same origin via localhost (vite proxy)", "localhost:5173", map[string]string{"Origin": "http://localhost:5173"}, 204},
		{"no origin, non-browser client", "127.0.0.1:8421", nil, 204},
		{"no origin, same-origin fetch metadata", "127.0.0.1:8421", map[string]string{"Sec-Fetch-Site": "same-origin"}, 204},
		{"foreign origin", "127.0.0.1:8421", map[string]string{"Origin": "http://evil.example"}, 403},
		{"other port on loopback", "127.0.0.1:8421", map[string]string{"Origin": "http://127.0.0.1:9999"}, 403},
		{"null origin", "127.0.0.1:8421", map[string]string{"Origin": "null"}, 403},
		{"no origin but cross-site fetch metadata", "127.0.0.1:8421", map[string]string{"Sec-Fetch-Site": "cross-site"}, 403},
		{"dns rebinding: matching origin on a foreign host", "evil.example:8421", map[string]string{"Origin": "http://evil.example:8421"}, 403},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/x", nil)
			req.Host = tt.host
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()

			RequireSameOrigin(okHandler).ServeHTTP(rec, req)

			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantCode)
			}
			if tt.wantCode == 403 && !strings.Contains(rec.Body.String(), `"forbidden_origin"`) {
				t.Errorf("body = %q", rec.Body.String())
			}
		})
	}
}

func TestAPIRequiresLoopbackHost(t *testing.T) {
	h := New(Config{Version: "dev"})
	for _, tt := range []struct {
		host string
		want int
	}{
		{"127.0.0.1:8421", 200},
		{"localhost:8421", 200},
		{"evil.example:8421", 403},
	} {
		t.Run(tt.host, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
			req.Host = tt.host
			rec := httptest.NewRecorder()

			h.ServeHTTP(rec, req)

			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}
