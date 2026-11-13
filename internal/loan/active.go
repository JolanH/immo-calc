package loan

import (
	"errors"
	"net/http"

	"immo-calc/internal/server"
)

// The active loan is picked on the home page and applies to the profitability
// shown there (SPEC-property.md, Home page). It lives in memory only: it
// survives page reloads but not a restart, and is never written to the data
// file.

// ActiveLoan returns the active loan's id, or "" when none is selected. A
// loan deleted since it was selected is no longer active.
func (s *Service) ActiveLoan() string {
	s.activeMu.Lock()
	id := s.active
	s.activeMu.Unlock()
	if id == "" {
		return ""
	}
	if _, err := s.Get(id); errors.Is(err, ErrNotFound) {
		s.clearActive(id)
		return ""
	}
	return id
}

// SetActiveLoan selects a saved loan as active; "" clears the selection.
func (s *Service) SetActiveLoan(id string) error {
	if id != "" {
		if _, err := s.Get(id); err != nil {
			return err
		}
	}
	s.activeMu.Lock()
	s.active = id
	s.activeMu.Unlock()
	return nil
}

// clearActive forgets the active loan if it is id.
func (s *Service) clearActive(id string) {
	s.activeMu.Lock()
	if s.active == id {
		s.active = ""
	}
	s.activeMu.Unlock()
}

type activeLoanBody struct {
	LoanID *string `json:"loanId"`
}

func activeBody(id string) activeLoanBody {
	if id == "" {
		return activeLoanBody{}
	}
	return activeLoanBody{LoanID: &id}
}

func (h *Handlers) getActive(w http.ResponseWriter, _ *http.Request) {
	server.WriteJSON(w, http.StatusOK, activeBody(h.svc.ActiveLoan()))
}

func (h *Handlers) putActive(w http.ResponseWriter, r *http.Request) {
	var body activeLoanBody
	if !server.DecodeJSON(w, r, &body) {
		return
	}
	id := ""
	if body.LoanID != nil {
		id = *body.LoanID
	}
	if err := h.svc.SetActiveLoan(id); err != nil {
		writeServiceError(w, err)
		return
	}
	server.WriteJSON(w, http.StatusOK, activeBody(id))
}
