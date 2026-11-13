package loan

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"immo-calc/internal/storage"
)

// SectionKey is the top-level key of the loans in the data file.
const SectionKey = "loans"

// Loan is a saved simulation: its inputs plus identity and timestamps.
type Loan struct {
	ID        string `json:"id" yaml:"id"`
	Input     `yaml:",inline"`
	CreatedAt time.Time `json:"createdAt" yaml:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" yaml:"updated_at"`
}

// RegisterSection declares the loans section and its validator. Call it
// before the store is loaded.
func RegisterSection(store *storage.Store) {
	store.RegisterSection(SectionKey, validateSection)
}

var idPattern = regexp.MustCompile(`^[0-9a-f]{16}$`)

func validID(id string) bool { return idPattern.MatchString(id) }

// fieldOrder lists input fields in form order, with their YAML key, so the
// first error reported for a loan is deterministic.
var fieldOrder = []struct{ json, yaml string }{
	{"name", "name"},
	{"principalCents", "principal_cents"},
	{"annualRatePct", "annual_rate_pct"},
	{"durationMonths", "duration_months"},
	{"insuranceRatePct", "insurance_rate_pct"},
	{"applicationFeeCents", "application_fee_cents"},
	{"guaranteeFeeCents", "guarantee_fee_cents"},
	{"firstPaymentMonth", "first_payment_month"},
}

// validateSection checks a hand-editable loans section: known fields only,
// valid unique ids and every input rule. Errors name the loan.
func validateSection(raw []byte) error {
	var loans []Loan
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	if err := dec.Decode(&loans); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	seen := map[string]bool{}
	for i, l := range loans {
		label := fmt.Sprintf("prêt « %s »", strings.TrimSpace(l.Name))
		if strings.TrimSpace(l.Name) == "" {
			label = fmt.Sprintf("prêt n° %d", i+1)
		}
		if !validID(l.ID) {
			return &storage.FieldError{Path: fmt.Sprintf("[%d].id", i),
				Message: fmt.Sprintf("identifiant invalide %q, 16 caractères hexadécimaux attendus (%s)", l.ID, label)}
		}
		if seen[l.ID] {
			return &storage.FieldError{Path: fmt.Sprintf("[%d].id", i),
				Message: fmt.Sprintf("identifiant %s en double (%s)", l.ID, label)}
		}
		seen[l.ID] = true
		errs := Validate(l.Input)
		for _, f := range fieldOrder {
			if msg, bad := errs[f.json]; bad {
				return &storage.FieldError{Path: fmt.Sprintf("[%d].%s", i, f.yaml), Message: fmt.Sprintf("%s (%s)", msg, label)}
			}
		}
	}
	return nil
}
