package property

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

// PropertiesKey is the top-level key of the properties in the data file.
const PropertiesKey = "properties"

// Property is a saved property: its inputs plus identity and timestamps.
type Property struct {
	ID        string `json:"id" yaml:"id"`
	Input     `yaml:",inline"`
	CreatedAt time.Time `json:"createdAt" yaml:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" yaml:"updated_at"`
}

var idPattern = regexp.MustCompile(`^[0-9a-f]{16}$`)

func propertyLabel(p Property, i int) string {
	if name := strings.TrimSpace(p.Name); name != "" {
		return fmt.Sprintf("bien « %s »", name)
	}
	return fmt.Sprintf("bien n° %d", i+1)
}

// validatePropertiesSection checks a hand-editable properties section:
// known fields only, valid unique ids and every input rule. Whether linked
// loans exist is checked across sections (checkLoanLinks).
func validatePropertiesSection(raw []byte) error {
	var props []Property
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	if err := dec.Decode(&props); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	seen := map[string]bool{}
	for i, p := range props {
		label := propertyLabel(p, i)
		if !idPattern.MatchString(p.ID) {
			return &storage.FieldError{Path: fmt.Sprintf("[%d].id", i),
				Message: fmt.Sprintf("identifiant invalide %q, 16 caractères hexadécimaux attendus (%s)", p.ID, label)}
		}
		if seen[p.ID] {
			return &storage.FieldError{Path: fmt.Sprintf("[%d].id", i),
				Message: fmt.Sprintf("identifiant %s en double (%s)", p.ID, label)}
		}
		seen[p.ID] = true
		errs := ValidateInput(p.Input, nil)
		for _, f := range inputFields {
			if msg, bad := errs[f.json]; bad {
				return &storage.FieldError{Path: fmt.Sprintf("[%d].%s", i, f.yaml), Message: fmt.Sprintf("%s (%s)", msg, label)}
			}
		}
	}
	return nil
}

// checkLoanLinks is a document check: every property's loan_id must name a
// saved loan. It guards startup, import, open, reload and every save.
func checkLoanLinks(get func(key string) []byte) error {
	var loans []struct {
		ID string `yaml:"id"`
	}
	if raw := get("loans"); raw != nil {
		if err := yaml.Unmarshal(raw, &loans); err != nil {
			return err
		}
	}
	known := map[string]bool{}
	for _, l := range loans {
		known[l.ID] = true
	}
	var props []Property
	if raw := get(PropertiesKey); raw != nil {
		if err := yaml.Unmarshal(raw, &props); err != nil {
			return err
		}
	}
	for i, p := range props {
		if p.LoanID != "" && !known[p.LoanID] {
			return &storage.DocumentError{Section: PropertiesKey, Path: fmt.Sprintf("[%d].loan_id", i),
				Message: fmt.Sprintf("le %s utilise un prêt qui n'existe pas (loan_id %s)", propertyLabel(p, i), p.LoanID)}
		}
	}
	return nil
}

// RegisterSections declares the property module's sections, validators and
// cross-section checks. Call it before the store is loaded.
func RegisterSections(store *storage.Store) {
	store.RegisterSection(SettingsKey, validateSettingsSection)
	store.RegisterSection(PropertiesKey, validatePropertiesSection)
	store.RegisterDocumentCheck(checkLoanLinks)
}
