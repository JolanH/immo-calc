package property

import (
	"errors"
	"net/http"

	"immo-calc/internal/server"
	"immo-calc/internal/storage"
)

// Handlers serves the property module's API.
type Handlers struct {
	svc *Service
}

// NewHandlers returns the property API handlers.
func NewHandlers(svc *Service) *Handlers {
	return &Handlers{svc: svc}
}

// Register adds the routes to mux.
func (h *Handlers) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/settings", h.getSettings)
	mux.Handle("PUT /api/settings", server.RequireSameOrigin(http.HandlerFunc(h.putSettings)))
	mux.Handle("POST /api/properties/evaluate", server.RequireSameOrigin(http.HandlerFunc(h.evaluate)))
}

func (h *Handlers) evaluate(w http.ResponseWriter, r *http.Request) {
	var in Input
	if !server.DecodeJSON(w, r, &in) {
		return
	}
	res, err := h.svc.EvaluateInput(in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	server.WriteJSON(w, http.StatusOK, res)
}

// writeServiceError maps service errors to API answers.
func writeServiceError(w http.ResponseWriter, err error) {
	var verr *ValidationError
	if errors.As(err, &verr) {
		server.WriteError(w, http.StatusBadRequest, "validation_failed",
			"Certains champs sont invalides, corrigez-les pour continuer", verr.Fields)
		return
	}
	storage.WriteSaveError(w, err)
}

func (h *Handlers) getSettings(w http.ResponseWriter, _ *http.Request) {
	s, err := h.svc.Settings()
	if err != nil {
		writeServiceError(w, err)
		return
	}
	server.WriteJSON(w, http.StatusOK, s)
}

func (h *Handlers) putSettings(w http.ResponseWriter, r *http.Request) {
	var next Settings
	if !server.DecodeJSON(w, r, &next) {
		return
	}
	saved, err := h.svc.UpdateSettings(next)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	server.WriteJSON(w, http.StatusOK, saved)
}
