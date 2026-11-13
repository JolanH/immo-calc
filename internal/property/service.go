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
}

// NewService returns a service over store, whose sections must have been
// registered with RegisterSections.
func NewService(store *storage.Store) *Service {
	return &Service{store: store}
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
