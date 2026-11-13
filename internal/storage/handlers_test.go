package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func serve(t *testing.T, h *Handlers, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	h.Register(mux)
	req := httptest.NewRequest(method, path, nil)
	req.Host = "127.0.0.1:8421"
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

type infoBody struct {
	Path          string         `json:"path"`
	SchemaVersion int            `json:"schemaVersion"`
	Counts        map[string]int `json:"counts"`
	Warning       *string        `json:"warning"`
}

func TestDataInfo(t *testing.T) {
	s := New()
	s.RegisterSection("loans", func([]byte) error { return nil })
	path := writeFile(t, "schema_version: 1\nloans:\n  - a: 1\n  - a: 2\n")
	if err := s.Load(path); err != nil {
		t.Fatal(err)
	}
	h := NewHandlers(s, HandlerConfig{CountKeys: []string{"properties", "loans"}})

	rec := serve(t, h, http.MethodGet, "/api/data/info")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body infoBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Path != path || body.SchemaVersion != 1 {
		t.Errorf("body = %+v", body)
	}
	if body.Counts["loans"] != 2 || body.Counts["properties"] != 0 || len(body.Counts) != 2 {
		t.Errorf("counts = %v", body.Counts)
	}
	if body.Warning != nil {
		t.Errorf("warning = %q, want none", *body.Warning)
	}
}

func TestDataInfoIncludesTheStartupWarning(t *testing.T) {
	s := newTestStore()
	_ = s.Load(writeFile(t, "schema_version: 1\n"))
	h := NewHandlers(s, HandlerConfig{Warning: "Le fichier /x est introuvable"})

	var body infoBody
	_ = json.Unmarshal(serve(t, h, http.MethodGet, "/api/data/info").Body.Bytes(), &body)

	if body.Warning == nil || *body.Warning != "Le fichier /x est introuvable" {
		t.Errorf("warning = %v", body.Warning)
	}
}

func TestWriteSaveError(t *testing.T) {
	tests := []struct {
		err      error
		wantCode int
		wantBody string
	}{
		{fmt.Errorf("save: %w", ErrFileChangedExternally), http.StatusConflict, "file_changed_externally"},
		{errors.New("disk full"), http.StatusInternalServerError, "save_failed"},
	}
	for _, tt := range tests {
		t.Run(tt.wantBody, func(t *testing.T) {
			rec := httptest.NewRecorder()

			WriteSaveError(rec, tt.err)

			var body struct {
				Error struct{ Code, Message string } `json:"error"`
			}
			_ = json.Unmarshal(rec.Body.Bytes(), &body)
			if rec.Code != tt.wantCode || body.Error.Code != tt.wantBody || body.Error.Message == "" {
				t.Errorf("status = %d, body = %s", rec.Code, rec.Body.String())
			}
		})
	}
}
