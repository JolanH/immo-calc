package loan

import (
	"math"
	"strings"
	"unicode/utf8"
)

// Input is a loan as the user enters it, and as it is stored. Only inputs are
// stored; results are always recomputed. FirstPaymentMonth stays a string so
// a bad value becomes a field error rather than an unreadable request.
type Input struct {
	Name                string  `json:"name" yaml:"name"`
	PrincipalCents      int64   `json:"principalCents" yaml:"principal_cents"`
	AnnualRatePct       float64 `json:"annualRatePct" yaml:"annual_rate_pct"`
	DurationMonths      int     `json:"durationMonths" yaml:"duration_months"`
	InsuranceRatePct    float64 `json:"insuranceRatePct" yaml:"insurance_rate_pct"`
	ApplicationFeeCents int64   `json:"applicationFeeCents" yaml:"application_fee_cents"`
	GuaranteeFeeCents   int64   `json:"guaranteeFeeCents" yaml:"guarantee_fee_cents"`
	FirstPaymentMonth   string  `json:"firstPaymentMonth" yaml:"first_payment_month"`
}

// Limits from SPEC-loan.md.
const (
	maxNameLength       = 80
	minPrincipalCents   = 100_000     // 1 000 €
	maxPrincipalCents   = 500_000_000 // 5 000 000 €
	maxRatePct          = 15
	maxDurationYears    = 30
	maxInsuranceRatePct = 2
	maxApplicationFee   = 5_000_000  // 50 000 €
	maxGuaranteeFee     = 10_000_000 // 100 000 €
)

var (
	firstAllowedMonth = Month{2000, 1}
	lastAllowedMonth  = Month{2100, 12}
)

// Params converts a validated input to calculation parameters.
func (in Input) Params() Params {
	month, _ := ParseMonth(in.FirstPaymentMonth)
	return Params{
		PrincipalCents:      in.PrincipalCents,
		AnnualRatePct:       in.AnnualRatePct,
		DurationMonths:      in.DurationMonths,
		InsuranceRatePct:    in.InsuranceRatePct,
		ApplicationFeeCents: in.ApplicationFeeCents,
		GuaranteeFeeCents:   in.GuaranteeFeeCents,
		FirstPaymentMonth:   month,
	}
}

// Validate checks every field and returns French messages keyed by JSON
// field name. An empty map means the input is valid.
func Validate(in Input) map[string]string {
	errs := ValidateParams(in)
	name := strings.TrimSpace(in.Name)
	switch {
	case name == "":
		errs["name"] = "Le nom est obligatoire"
	case utf8.RuneCountInString(name) > maxNameLength:
		errs["name"] = "Le nom ne doit pas dépasser 80 caractères"
	}
	return errs
}

// ValidateParams checks the fields used by the calculation (everything but
// the name), for live simulations.
func ValidateParams(in Input) map[string]string {
	errs := map[string]string{}

	if in.PrincipalCents < minPrincipalCents || in.PrincipalCents > maxPrincipalCents {
		errs["principalCents"] = "Le montant doit être compris entre 1 000 € et 5 000 000 €"
	}
	if msg := checkRate(in.AnnualRatePct, maxRatePct, "Le taux doit être compris entre 0 et 15 %"); msg != "" {
		errs["annualRatePct"] = msg
	}
	switch {
	case in.DurationMonths < 12 || in.DurationMonths > maxDurationYears*12:
		errs["durationMonths"] = "La durée doit être comprise entre 1 et 30 ans"
	case in.DurationMonths%12 != 0:
		errs["durationMonths"] = "La durée doit être un nombre entier d'années"
	}
	if msg := checkRate(in.InsuranceRatePct, maxInsuranceRatePct, "Le taux d'assurance doit être compris entre 0 et 2 %"); msg != "" {
		errs["insuranceRatePct"] = msg
	}
	if in.ApplicationFeeCents < 0 || in.ApplicationFeeCents > maxApplicationFee {
		errs["applicationFeeCents"] = "Les frais de dossier doivent être compris entre 0 et 50 000 €"
	}
	if in.GuaranteeFeeCents < 0 || in.GuaranteeFeeCents > maxGuaranteeFee {
		errs["guaranteeFeeCents"] = "Les frais de garantie doivent être compris entre 0 et 100 000 €"
	}
	if msg := checkFirstPaymentMonth(in.FirstPaymentMonth); msg != "" {
		errs["firstPaymentMonth"] = msg
	}
	return errs
}

func checkRate(v, max float64, rangeMessage string) string {
	if v < 0 || v > max || math.IsNaN(v) {
		return rangeMessage
	}
	if scaled := v * 1000; math.Abs(scaled-math.Round(scaled)) > 1e-6 {
		return "Le taux accepte au plus 3 décimales"
	}
	return ""
}

func checkFirstPaymentMonth(s string) string {
	if strings.TrimSpace(s) == "" {
		return "La première échéance est obligatoire"
	}
	m, err := ParseMonth(s)
	if err != nil {
		return "La première échéance doit être un mois valide"
	}
	if before(m, firstAllowedMonth) || before(lastAllowedMonth, m) {
		return "La première échéance doit être comprise entre janvier 2000 et décembre 2100"
	}
	return ""
}

func before(a, b Month) bool {
	return a.Year < b.Year || (a.Year == b.Year && a.Month < b.Month)
}
