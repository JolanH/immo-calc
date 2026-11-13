package property

import (
	"fmt"

	"immo-calc/internal/storage"
)

// ValidationError carries French field messages for invalid input.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return fmt.Sprintf("invalid input: %v", e.Fields) }

// Service reads and writes the property module's data.
type Service struct {
	store *storage.Store
	loans LoanSource
}

// NewService returns a service over store, whose sections must have been
// registered with RegisterSections. loans resolves linked loans.
func NewService(store *storage.Store, loans LoanSource) *Service {
	return &Service{store: store, loans: loans}
}

// loanFigures returns the figures of a linked loan, or nil without one.
func (s *Service) loanFigures(loanID string) (*LoanFigures, error) {
	if loanID == "" || s.loans == nil {
		return nil, nil
	}
	l, err := s.loans.Get(loanID)
	if err != nil {
		return nil, err
	}
	f, err := FiguresOf(l.Params())
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// EvaluateInput validates an unsaved input (name not required) and evaluates
// it with the current settings and its linked loan.
func (s *Service) EvaluateInput(in Input) (Result, error) {
	if errs := ValidateParams(in, s.loans); len(errs) > 0 {
		return Result{}, &ValidationError{Fields: errs}
	}
	settings, err := s.Settings()
	if err != nil {
		return Result{}, err
	}
	loanFig, err := s.loanFigures(in.LoanID)
	if err != nil {
		return Result{}, err
	}
	return Evaluate(in.Params(), settings, loanFig), nil
}

// Settings returns the global settings, or the defaults when the data file
// has no settings section.
func (s *Service) Settings() (Settings, error) {
	stored, err := storage.Get[*Settings](s.store, SettingsKey)
	if err != nil {
		return Settings{}, err
	}
	if stored == nil {
		return DefaultSettings(), nil
	}
	return *stored, nil
}

// UpdateSettings validates and saves the global settings.
func (s *Service) UpdateSettings(next Settings) (Settings, error) {
	if errs := ValidateSettings(next); len(errs) > 0 {
		return Settings{}, &ValidationError{Fields: errs}
	}
	err := storage.Update(s.store, SettingsKey, func(stored **Settings) error {
		*stored = &next
		return nil
	})
	if err != nil {
		return Settings{}, err
	}
	return next, nil
}
