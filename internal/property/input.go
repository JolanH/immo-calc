package property

import (
	"strings"
	"unicode/utf8"

	"immo-calc/internal/loan"
)

// Input is a property as the user enters it, and as it is stored. Only
// inputs are stored; results are always recomputed.
type Input struct {
	Name                           string `json:"name" yaml:"name"`
	PriceCents                     int64  `json:"priceCents" yaml:"price_cents"`
	DPE                            string `json:"dpe" yaml:"dpe"`
	MonthlyRentCents               int64  `json:"monthlyRentCents" yaml:"monthly_rent_cents"`
	MonthlyRecoverableChargesCents int64  `json:"monthlyRecoverableChargesCents" yaml:"monthly_recoverable_charges_cents"`
	AnnualPropertyTaxCents         int64  `json:"annualPropertyTaxCents" yaml:"annual_property_tax_cents"`
	AnnualCondoFeesCents           int64  `json:"annualCondoFeesCents" yaml:"annual_condo_fees_cents"`
	AnnualLandlordInsuranceCents   int64  `json:"annualLandlordInsuranceCents" yaml:"annual_landlord_insurance_cents"`
	AgencyManaged                  bool   `json:"agencyManaged" yaml:"agency_managed"`
	TaxRegime                      Regime `json:"taxRegime" yaml:"tax_regime"`
	LoanID                         string `json:"loanId" yaml:"loan_id,omitempty"`
}

// LoanSource finds saved loans; *loan.Service implements it.
type LoanSource interface {
	Get(id string) (loan.Loan, error)
}

// Params converts an input to calculation parameters.
func (in Input) Params() Params {
	return Params{
		PriceCents:                     in.PriceCents,
		MonthlyRentCents:               in.MonthlyRentCents,
		MonthlyRecoverableChargesCents: in.MonthlyRecoverableChargesCents,
		AnnualPropertyTaxCents:         in.AnnualPropertyTaxCents,
		AnnualCondoFeesCents:           in.AnnualCondoFeesCents,
		AnnualLandlordInsuranceCents:   in.AnnualLandlordInsuranceCents,
		AgencyManaged:                  in.AgencyManaged,
		TaxRegime:                      in.TaxRegime,
		DPE:                            in.DPE,
	}
}

// inputFields lists fields in form order with their YAML key, so the first
// error reported for a hand-edited property is deterministic.
var inputFields = []struct{ json, yaml string }{
	{"name", "name"},
	{"priceCents", "price_cents"},
	{"dpe", "dpe"},
	{"monthlyRentCents", "monthly_rent_cents"},
	{"monthlyRecoverableChargesCents", "monthly_recoverable_charges_cents"},
	{"annualPropertyTaxCents", "annual_property_tax_cents"},
	{"annualCondoFeesCents", "annual_condo_fees_cents"},
	{"annualLandlordInsuranceCents", "annual_landlord_insurance_cents"},
	{"taxRegime", "tax_regime"},
	{"loanId", "loan_id"},
}

func inRange(v, lo, hi int64) bool { return v >= lo && v <= hi }

// ValidateInput checks every field and returns French messages keyed by
// JSON field name. loans resolves loanId; nil skips that check.
func ValidateInput(in Input, loans LoanSource) map[string]string {
	errs := ValidateParams(in, loans)
	name := strings.TrimSpace(in.Name)
	switch {
	case name == "":
		errs["name"] = "Le nom est obligatoire"
	case utf8.RuneCountInString(name) > 80:
		errs["name"] = "Le nom ne doit pas dépasser 80 caractères"
	}
	return errs
}

// ValidateParams checks everything but the name, for live evaluations.
func ValidateParams(in Input, loans LoanSource) map[string]string {
	errs := map[string]string{}
	if !inRange(in.PriceCents, 1_000_000, 500_000_000) {
		errs["priceCents"] = "Le prix doit être compris entre 10 000 € et 5 000 000 €"
	}
	if len(in.DPE) != 1 || !strings.Contains("ABCDEFG", in.DPE) {
		errs["dpe"] = "Choisissez la classe DPE (A à G)"
	}
	if !inRange(in.MonthlyRentCents, 100, 2_000_000) {
		errs["monthlyRentCents"] = "Le loyer doit être compris entre 1 € et 20 000 € par mois"
	}
	if !inRange(in.MonthlyRecoverableChargesCents, 0, 200_000) {
		errs["monthlyRecoverableChargesCents"] = "Les charges doivent être comprises entre 0 et 2 000 € par mois"
	}
	if !inRange(in.AnnualPropertyTaxCents, 0, 5_000_000) {
		errs["annualPropertyTaxCents"] = "La taxe foncière doit être comprise entre 0 et 50 000 € par an"
	}
	if !inRange(in.AnnualCondoFeesCents, 0, 5_000_000) {
		errs["annualCondoFeesCents"] = "Les charges de copropriété doivent être comprises entre 0 et 50 000 € par an"
	}
	if !inRange(in.AnnualLandlordInsuranceCents, 0, 500_000) {
		errs["annualLandlordInsuranceCents"] = "L'assurance PNO doit être comprise entre 0 et 5 000 € par an"
	}
	switch in.TaxRegime {
	case MicroFoncier, ReelFoncier, MicroBIC:
	default:
		errs["taxRegime"] = "Choisissez un régime fiscal"
	}
	if in.LoanID != "" && loans != nil {
		if _, err := loans.Get(in.LoanID); err != nil {
			errs["loanId"] = "Ce prêt n'existe plus, choisissez-en un autre"
		}
	}
	return errs
}
