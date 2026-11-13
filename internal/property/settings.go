// Package property evaluates the profitability of a flat bought to rent out
// and stores the properties and the global settings (hypotheses and tax).
package property

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"

	"immo-calc/internal/storage"
)

// SettingsKey is the top-level key of the global settings in the data file.
const SettingsKey = "settings"

// Settings are the global hypotheses every property's results depend on.
// Rates are percentages; sources for the defaults are in tasks/legal-rates.md.
type Settings struct {
	MarginalTaxRatePct          float64 `json:"marginalTaxRatePct" yaml:"marginal_tax_rate_pct"`
	SocialChargesUnfurnishedPct float64 `json:"socialChargesUnfurnishedPct" yaml:"social_charges_unfurnished_pct"`
	SocialChargesFurnishedPct   float64 `json:"socialChargesFurnishedPct" yaml:"social_charges_furnished_pct"`
	NotaryFeePct                float64 `json:"notaryFeePct" yaml:"notary_fee_pct"`
	VacancyMonthsPerYear        float64 `json:"vacancyMonthsPerYear" yaml:"vacancy_months_per_year"`
	AgencyFeePct                float64 `json:"agencyFeePct" yaml:"agency_fee_pct"`
	// AccentColor is the UI's main colour, "#rrggbb" (not a financial input;
	// kept here so it travels with export/import).
	AccentColor string `json:"accentColor" yaml:"accent_color,omitempty"`
}

// DefaultSettings applies when the data file has no settings section.
func DefaultSettings() Settings {
	return Settings{
		MarginalTaxRatePct: 30,
		// Prélèvements sociaux on revenus fonciers (location nue): 17,2 %
		// (service-public F2329, checked 2026-11-13).
		SocialChargesUnfurnishedPct: 17.2,
		// Location meublée non professionnelle: 18,6 % since the LFSS 2026 raised
		// the CSG to 10,6 % (service-public F2329, checked 2026-11-13).
		SocialChargesFurnishedPct: 18.6,
		NotaryFeePct:              8.5,
		VacancyMonthsPerYear:      1,
		AgencyFeePct:              7,
		AccentColor:               DefaultAccentColor,
	}
}

// DefaultAccentColor is dark red (chosen by the user, 2026-11-13).
const DefaultAccentColor = "#8b1a1a"

var accentPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// withDefaults fills values a settings section may lack (written before
// they existed) and normalises the colour.
func withDefaults(s Settings) Settings {
	if s.AccentColor == "" {
		s.AccentColor = DefaultAccentColor
	}
	s.AccentColor = strings.ToLower(s.AccentColor)
	return s
}

var taxBrackets = []float64{0, 11, 30, 41, 45}

// settingsFields lists fields in form order with their YAML key.
var settingsFields = []struct{ json, yaml string }{
	{"marginalTaxRatePct", "marginal_tax_rate_pct"},
	{"socialChargesUnfurnishedPct", "social_charges_unfurnished_pct"},
	{"socialChargesFurnishedPct", "social_charges_furnished_pct"},
	{"notaryFeePct", "notary_fee_pct"},
	{"vacancyMonthsPerYear", "vacancy_months_per_year"},
	{"agencyFeePct", "agency_fee_pct"},
	{"accentColor", "accent_color"},
}

func between(v, lo, hi float64) bool { return v >= lo && v <= hi && !math.IsNaN(v) }

// ValidateSettings returns French messages keyed by JSON field name.
func ValidateSettings(s Settings) map[string]string {
	errs := map[string]string{}
	bracket := false
	for _, b := range taxBrackets {
		bracket = bracket || s.MarginalTaxRatePct == b
	}
	if !bracket {
		errs["marginalTaxRatePct"] = "La TMI doit être l'une des tranches : 0, 11, 30, 41 ou 45 %"
	}
	const socialMsg = "Les prélèvements sociaux doivent être compris entre 0 et 30 %"
	if !between(s.SocialChargesUnfurnishedPct, 0, 30) {
		errs["socialChargesUnfurnishedPct"] = socialMsg
	}
	if !between(s.SocialChargesFurnishedPct, 0, 30) {
		errs["socialChargesFurnishedPct"] = socialMsg
	}
	if !between(s.NotaryFeePct, 0, 15) {
		errs["notaryFeePct"] = "Les frais de notaire doivent être compris entre 0 et 15 %"
	}
	if !between(s.VacancyMonthsPerYear, 0, 12) || s.VacancyMonthsPerYear*2 != math.Round(s.VacancyMonthsPerYear*2) {
		errs["vacancyMonthsPerYear"] = "La vacance doit être comprise entre 0 et 12 mois, par demi-mois"
	}
	if !between(s.AgencyFeePct, 0, 15) {
		errs["agencyFeePct"] = "Les frais de gestion doivent être compris entre 0 et 15 %"
	}
	if s.AccentColor != "" && !accentPattern.MatchString(s.AccentColor) {
		errs["accentColor"] = "La couleur doit être au format #rrggbb"
	}
	return errs
}

// validateSettingsSection checks a hand-editable settings section.
func validateSettingsSection(raw []byte) error {
	var s Settings
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	if err := dec.Decode(&s); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	errs := ValidateSettings(s)
	for _, f := range settingsFields {
		if msg, bad := errs[f.json]; bad {
			return &storage.FieldError{Path: f.yaml, Message: fmt.Sprintf("%s (paramètres)", msg)}
		}
	}
	return nil
}
