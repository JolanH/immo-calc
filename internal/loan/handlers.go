package loan

import (
	"log/slog"
	"net/http"

	"immo-calc/internal/server"
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
	mux.Handle("POST /api/loans/simulate", server.RequireSameOrigin(http.HandlerFunc(h.simulate)))
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
