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
	write := func(f http.HandlerFunc) http.Handler { return server.RequireSameOrigin(f) }
	mux.Handle("POST /api/properties/evaluate", write(h.evaluate))
	mux.HandleFunc("GET /api/properties", h.list)
	mux.Handle("POST /api/properties", write(h.create))
	mux.HandleFunc("GET /api/properties/{id}", h.get)
	mux.Handle("PUT /api/properties/{id}", write(h.update))
	mux.Handle("DELETE /api/properties/{id}", write(h.delete))
	mux.Handle("POST /api/properties/{id}/duplicate", write(h.duplicate))
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	props, err := h.svc.List(r.URL.Query().Get("loanId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	server.WriteJSON(w, http.StatusOK, props)
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	h.writeDetail(w, http.StatusOK, r.PathValue("id"))
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	var in Input
	if !server.DecodeJSON(w, r, &in) {
		return
	}
	p, err := h.svc.Create(in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	h.writeDetail(w, http.StatusCreated, p.ID)
}

func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	var in Input
	if !server.DecodeJSON(w, r, &in) {
		return
	}
	p, err := h.svc.Update(r.PathValue("id"), in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	h.writeDetail(w, http.StatusOK, p.ID)
}

func (h *Handlers) duplicate(w http.ResponseWriter, r *http.Request) {
	p, err := h.svc.Duplicate(r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	h.writeDetail(w, http.StatusCreated, p.ID)
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.PathValue("id")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) writeDetail(w http.ResponseWriter, status int, id string) {
	d, err := h.svc.Detail(id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	server.WriteJSON(w, status, d)
}

func (h *Handlers) evaluate(w http.ResponseWriter, r *http.Request) {
	var in Input
	if !server.DecodeJSON(w, r, &in) {
		return
	}
	res, err := h.svc.EvaluateFor(r.URL.Query().Get("id"), in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	server.WriteJSON(w, http.StatusOK, res)
}

// writeServiceError maps service errors to API answers.
func writeServiceError(w http.ResponseWriter, err error) {
	var verr *ValidationError
	switch {
	case errors.As(err, &verr):
		server.WriteError(w, http.StatusBadRequest, "validation_failed",
			"Certains champs sont invalides, corrigez-les pour continuer", verr.Fields)
	case errors.Is(err, ErrNotFound):
		server.WriteError(w, http.StatusNotFound, "not_found", "Bien introuvable", nil)
	default:
		storage.WriteSaveError(w, err)
	}
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
