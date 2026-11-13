package loan

import (
	"errors"
	"log/slog"
	"net/http"

	"immo-calc/internal/server"
	"immo-calc/internal/storage"
)

// Handlers serves the /api/loans endpoints.
type Handlers struct {
	svc *Service
}

// NewHandlers returns the loan API handlers. svc may be nil for routes that
// don't touch saved loans (simulate).
func NewHandlers(svc *Service) *Handlers {
	return &Handlers{svc: svc}
}

// Register adds the routes to mux.
func (h *Handlers) Register(mux *http.ServeMux) {
	write := func(f http.HandlerFunc) http.Handler { return server.RequireSameOrigin(f) }
	mux.Handle("POST /api/loans/simulate", write(h.simulate))
	mux.HandleFunc("GET /api/loans", h.list)
	mux.Handle("POST /api/loans", write(h.create))
	mux.HandleFunc("GET /api/loans/{id}", h.get)
	mux.Handle("PUT /api/loans/{id}", write(h.update))
	mux.Handle("DELETE /api/loans/{id}", write(h.delete))
	mux.Handle("POST /api/loans/{id}/duplicate", write(h.duplicate))
	mux.HandleFunc("GET /api/active-loan", h.getActive)
	mux.Handle("PUT /api/active-loan", write(h.putActive))
}

func (h *Handlers) duplicate(w http.ResponseWriter, r *http.Request) {
	l, err := h.svc.Duplicate(r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	h.writeDetail(w, http.StatusCreated, l)
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.PathValue("id")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// writeServiceError maps service errors to API answers.
func writeServiceError(w http.ResponseWriter, err error) {
	var verr *ValidationError
	var inUse *InUseError
	switch {
	case errors.As(err, &verr):
		writeValidationError(w, verr.Fields)
	case errors.As(err, &inUse):
		server.WriteError(w, http.StatusConflict, "loan_in_use", inUse.Message, nil)
	case errors.Is(err, ErrNotFound):
		server.WriteError(w, http.StatusNotFound, "not_found", "Prêt introuvable", nil)
	default:
		storage.WriteSaveError(w, err)
	}
}

func (h *Handlers) list(w http.ResponseWriter, _ *http.Request) {
	loans, err := h.svc.List()
	if err != nil {
		writeServiceError(w, err)
		return
	}
	server.WriteJSON(w, http.StatusOK, loans)
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	detail, err := h.svc.Detail(r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	server.WriteJSON(w, http.StatusOK, detail)
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	var in Input
	if !server.DecodeJSON(w, r, &in) {
		return
	}
	l, err := h.svc.Create(in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	h.writeDetail(w, http.StatusCreated, l)
}

func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	var in Input
	if !server.DecodeJSON(w, r, &in) {
		return
	}
	l, err := h.svc.Update(r.PathValue("id"), in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	h.writeDetail(w, http.StatusOK, l)
}

func (h *Handlers) writeDetail(w http.ResponseWriter, status int, l Loan) {
	detail, err := detailOf(l)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	server.WriteJSON(w, status, detail)
}

// writeValidationError answers 400 with French field messages.
func writeValidationError(w http.ResponseWriter, fields map[string]string) {
	server.WriteError(w, http.StatusBadRequest, "validation_failed",
		"Certains champs sont invalides, corrigez-les pour continuer", fields)
}

func (h *Handlers) simulate(w http.ResponseWriter, r *http.Request) {
	var in Input
	if !server.DecodeJSON(w, r, &in) {
		return
	}
	if errs := ValidateParams(in); len(errs) > 0 {
		writeValidationError(w, errs)
		return
	}
	res, err := Compute(in.Params())
	if err != nil {
		// Unreachable after validation; keep the API honest if rules drift.
		slog.Error("simulate after validation", "err", err)
		server.WriteError(w, http.StatusInternalServerError, "compute_failed", "Le calcul a échoué", nil)
		return
	}
	server.WriteJSON(w, http.StatusOK, res)
}
