package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
