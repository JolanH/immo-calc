package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteJSON(rec, http.StatusCreated, map[string]int{"answer": 42})

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("content type = %q", ct)
	}
	if got := rec.Body.String(); got != "{\"answer\":42}\n" {
		t.Errorf("body = %q", got)
	}
}

func TestWriteErrorWithoutFields(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteError(rec, http.StatusNotFound, "not_found", "Ressource introuvable", nil)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Body.String(); got != `{"error":{"code":"not_found","message":"Ressource introuvable"}}`+"\n" {
		t.Errorf("body = %q", got)
	}
}

func TestWriteErrorWithFields(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteError(rec, http.StatusBadRequest, "validation_failed", "Données invalides",
		map[string]string{"name": "Le nom est obligatoire"})

	var body struct {
		Error struct {
			Code    string            `json:"code"`
			Message string            `json:"message"`
			Fields  map[string]string `json:"fields"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Code != "validation_failed" || body.Error.Fields["name"] != "Le nom est obligatoire" {
		t.Errorf("unexpected body: %+v", body)
	}
}

func TestDecodeJSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}
	tests := []struct {
		body   string
		wantOK bool
	}{
		{`{"name":"a"}`, true},
		{`{"name":"a","extra":1}`, true},
		{`{"name":`, false},
		{`{"name":1}`, false},
		{``, false},
		{`{"name":"a"} {"name":"b"}`, false},
	}
	for _, tt := range tests {
		t.Run(tt.body, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			var v payload

			ok := DecodeJSON(rec, req, &v)

			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok && (rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), `"invalid_json"`)) {
				t.Errorf("status = %d, body = %s", rec.Code, rec.Body.String())
			}
		})
	}
}
