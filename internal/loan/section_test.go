package loan

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"immo-calc/internal/storage"
)

const validLoanYAML = `
  - id: 9f2c4e1a7b3d5f60
    name: Appart Lyon
    principal_cents: 20000000
    annual_rate_pct: 3.5
    duration_months: 240
    insurance_rate_pct: 0.3
    application_fee_cents: 100000
    guarantee_fee_cents: 250000
    first_payment_month: 2026-12
    created_at: 2026-11-12T20:41:00+01:00
    updated_at: 2026-11-12T20:41:00+01:00
`

func loadLoans(t *testing.T, loansYAML string) error {
	t.Helper()
	path := filepath.Join(t.TempDir(), "immo-calc.yaml")
	if err := os.WriteFile(path, []byte("schema_version: 1\nloans:"+loansYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	store := storage.New()
	RegisterSection(store)
	return store.Load(path)
}

func TestSectionAcceptsTheSpecExample(t *testing.T) {
	if err := loadLoans(t, validLoanYAML); err != nil {
		t.Fatalf("load: %v", err)
	}
}

func TestSectionAcceptsAnEmptyList(t *testing.T) {
	if err := loadLoans(t, " []\n"); err != nil {
		t.Fatalf("load: %v", err)
	}
}

func TestSectionRejectsInvalidLoansWithNameAndLine(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		wantLine string
		wantText string
	}{
		{"negative amount", strings.Replace(validLoanYAML, "principal_cents: 20000000", "principal_cents: -5", 1),
			"(ligne 5)", "Le montant doit être compris entre 1 000 € et 5 000 000 € (prêt « Appart Lyon »)"},
		{"bad month", strings.Replace(validLoanYAML, "first_payment_month: 2026-12", "first_payment_month: 2026-13", 1),
			"(ligne 11)", "prêt « Appart Lyon »"},
		{"empty name", strings.Replace(validLoanYAML, "name: Appart Lyon", "name: ''", 1),
			"(ligne 4)", "Le nom est obligatoire"},
		{"bad id", strings.Replace(validLoanYAML, "id: 9f2c4e1a7b3d5f60", "id: xyz", 1),
			"(ligne 3)", "identifiant"},
		{"duplicate id", validLoanYAML + strings.Replace(validLoanYAML, "Appart Lyon", "Copie", 1),
			"(ligne 15)", "identifiant 9f2c4e1a7b3d5f60 en double"},
		{"unknown field", strings.Replace(validLoanYAML, "name: Appart Lyon", "name: Appart Lyon\n    nmae: typo", 1),
			"", "nmae"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := loadLoans(t, tt.yaml)

			var loadErr *storage.LoadError
			if !errors.As(err, &loadErr) {
				t.Fatalf("err = %v, want a LoadError", err)
			}
			if tt.wantLine != "" && !strings.Contains(err.Error(), tt.wantLine) {
				t.Errorf("error %q lacks %q", err, tt.wantLine)
			}
			if !strings.Contains(err.Error(), tt.wantText) {
				t.Errorf("error %q lacks %q", err, tt.wantText)
			}
		})
	}
}
