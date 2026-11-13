package server

import (
	"encoding/json"
	"log"
	"net/http"
)

// WriteJSON writes v as a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json response: %v", err)
	}
}

// APIError is the error body shared by every API endpoint. Message and the
// field messages are in French because the UI shows them as they are.
type APIError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// WriteError writes {"error":{...}} with the given status. fields may be nil.
func WriteError(w http.ResponseWriter, status int, code, message string, fields map[string]string) {
	WriteJSON(w, status, struct {
		Error APIError `json:"error"`
	}{APIError{Code: code, Message: message, Fields: fields}})
}

// WriteNotFound writes the standard 404 API error.
func WriteNotFound(w http.ResponseWriter) {
	WriteError(w, http.StatusNotFound, "not_found", "Ressource introuvable", nil)
}

// maxJSONBytes bounds request bodies; immo-calc's forms are tiny.
const maxJSONBytes = 1 << 20

// DecodeJSON decodes the request body into v. On failure it writes a 400
// invalid_json answer and returns false.
func DecodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxJSONBytes))
	if err := dec.Decode(v); err != nil || dec.More() {
		WriteError(w, http.StatusBadRequest, "invalid_json", "Requête illisible", nil)
		return false
	}
	return true
}
