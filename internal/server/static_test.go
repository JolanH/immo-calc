package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func testStatic() fstest.MapFS {
	return fstest.MapFS{
		"index.html":       {Data: []byte("<!doctype html><title>immo-calc</title>")},
		"assets/app-1.js":  {Data: []byte("console.log('app')")},
		"assets/app-1.css": {Data: []byte("body{}")},
		"favicon.ico":      {Data: []byte("ico")},
	}
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestSPAFallbackServesIndex(t *testing.T) {
	h := New(Config{Version: "dev", Static: testStatic()})

	for _, path := range []string{"/", "/biens", "/prets/123", "/index.html", "/assets/missing.js"} {
		t.Run(path, func(t *testing.T) {
			rec := get(t, h, path)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d", rec.Code)
			}
			if !strings.Contains(rec.Body.String(), "<title>immo-calc</title>") {
				t.Errorf("body = %q, want index.html", rec.Body.String())
			}
			if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
				t.Errorf("content type = %q", ct)
			}
		})
	}
}

func TestStaticAssetsAreServed(t *testing.T) {
	h := New(Config{Version: "dev", Static: testStatic()})

	tests := []struct{ path, body, contentType string }{
		{"/assets/app-1.js", "console.log('app')", "text/javascript; charset=utf-8"},
		{"/assets/app-1.css", "body{}", "text/css; charset=utf-8"},
		{"/favicon.ico", "ico", "image/vnd.microsoft.icon"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rec := get(t, h, tt.path)
			if rec.Code != http.StatusOK || rec.Body.String() != tt.body {
				t.Fatalf("status = %d, body = %q", rec.Code, rec.Body.String())
			}
			if ct := rec.Header().Get("Content-Type"); ct != tt.contentType {
				t.Errorf("content type = %q, want %q", ct, tt.contentType)
			}
		})
	}
}

func TestStaticRejectsWrites(t *testing.T) {
	h := New(Config{Version: "dev", Static: testStatic()})
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/biens", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestUnknownAPIRouteStaysJSONWithStaticFiles(t *testing.T) {
	h := New(Config{Version: "dev", Static: testStatic()})

	rec := get(t, h, "/api/nope")

	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), `"not_found"`) {
		t.Errorf("status = %d, body = %q", rec.Code, rec.Body.String())
	}
}
