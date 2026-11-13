package loan

import (
	"strings"
	"testing"
)

func validInput() Input {
	return Input{
		Name:                "Appart Lyon — Crédit Mutuel",
		PrincipalCents:      20_000_000,
		AnnualRatePct:       3.5,
		DurationMonths:      240,
		InsuranceRatePct:    0.3,
		ApplicationFeeCents: 100_000,
		GuaranteeFeeCents:   250_000,
		FirstPaymentMonth:   "2026-12",
	}
}

func TestValidInputHasNoErrors(t *testing.T) {
	if errs := Validate(validInput()); len(errs) != 0 {
		t.Errorf("errors = %v", errs)
	}
}

func TestValidationRules(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Input)
		field   string
		message string // substring of the French message; "" means valid
	}{
		{"empty name", func(in *Input) { in.Name = "  " }, "name", "Le nom est obligatoire"},
		{"name too long", func(in *Input) { in.Name = strings.Repeat("é", 81) }, "name", "80 caractères"},
		{"name of 80 accented chars", func(in *Input) { in.Name = strings.Repeat("é", 80) }, "name", ""},

		{"principal too small", func(in *Input) { in.PrincipalCents = 99_999 }, "principalCents", "Le montant doit être compris entre 1 000 € et 5 000 000 €"},
		{"principal at minimum", func(in *Input) { in.PrincipalCents = 100_000 }, "principalCents", ""},
		{"principal at maximum", func(in *Input) { in.PrincipalCents = 500_000_000 }, "principalCents", ""},
		{"principal too large", func(in *Input) { in.PrincipalCents = 500_000_001 }, "principalCents", "Le montant doit être compris"},

		{"negative rate", func(in *Input) { in.AnnualRatePct = -0.001 }, "annualRatePct", "Le taux doit être compris entre 0 et 15 %"},
		{"zero rate", func(in *Input) { in.AnnualRatePct = 0 }, "annualRatePct", ""},
		{"rate at maximum", func(in *Input) { in.AnnualRatePct = 15 }, "annualRatePct", ""},
		{"rate too high", func(in *Input) { in.AnnualRatePct = 15.001 }, "annualRatePct", "Le taux doit être compris"},
		{"rate with 3 decimals", func(in *Input) { in.AnnualRatePct = 3.125 }, "annualRatePct", ""},
		{"rate with 4 decimals", func(in *Input) { in.AnnualRatePct = 3.1255 }, "annualRatePct", "3 décimales"},

		{"duration of 0", func(in *Input) { in.DurationMonths = 0 }, "durationMonths", "La durée doit être comprise entre 1 et 30 ans"},
		{"duration of 1 year", func(in *Input) { in.DurationMonths = 12 }, "durationMonths", ""},
		{"duration of 30 years", func(in *Input) { in.DurationMonths = 360 }, "durationMonths", ""},
		{"duration of 31 years", func(in *Input) { in.DurationMonths = 372 }, "durationMonths", "entre 1 et 30 ans"},
		{"duration not whole years", func(in *Input) { in.DurationMonths = 18 }, "durationMonths", "nombre entier d'années"},

		{"negative insurance", func(in *Input) { in.InsuranceRatePct = -0.1 }, "insuranceRatePct", "Le taux d'assurance doit être compris entre 0 et 2 %"},
		{"insurance too high", func(in *Input) { in.InsuranceRatePct = 2.001 }, "insuranceRatePct", "entre 0 et 2 %"},
		{"insurance with 4 decimals", func(in *Input) { in.InsuranceRatePct = 0.3125 }, "insuranceRatePct", "3 décimales"},
		{"no insurance", func(in *Input) { in.InsuranceRatePct = 0 }, "insuranceRatePct", ""},

		{"negative application fee", func(in *Input) { in.ApplicationFeeCents = -1 }, "applicationFeeCents", "Les frais de dossier doivent être compris entre 0 et 50 000 €"},
		{"application fee too high", func(in *Input) { in.ApplicationFeeCents = 5_000_001 }, "applicationFeeCents", "entre 0 et 50 000 €"},
		{"negative guarantee fee", func(in *Input) { in.GuaranteeFeeCents = -1 }, "guaranteeFeeCents", "Les frais de garantie doivent être compris entre 0 et 100 000 €"},
		{"guarantee fee too high", func(in *Input) { in.GuaranteeFeeCents = 10_000_001 }, "guaranteeFeeCents", "entre 0 et 100 000 €"},

		{"missing first payment", func(in *Input) { in.FirstPaymentMonth = "" }, "firstPaymentMonth", "La première échéance est obligatoire"},
		{"malformed first payment", func(in *Input) { in.FirstPaymentMonth = "12/2026" }, "firstPaymentMonth", "mois valide"},
		{"first payment too early", func(in *Input) { in.FirstPaymentMonth = "1999-12" }, "firstPaymentMonth", "entre janvier 2000 et décembre 2100"},
		{"first payment at the start", func(in *Input) { in.FirstPaymentMonth = "2000-01" }, "firstPaymentMonth", ""},
		{"first payment at the end", func(in *Input) { in.FirstPaymentMonth = "2100-12" }, "firstPaymentMonth", ""},
		{"first payment too late", func(in *Input) { in.FirstPaymentMonth = "2101-01" }, "firstPaymentMonth", "entre janvier 2000 et décembre 2100"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := validInput()
			tt.mutate(&in)

			errs := Validate(in)

			if tt.message == "" {
				if len(errs) != 0 {
					t.Errorf("errors = %v, want none", errs)
				}
				return
			}
			if !strings.Contains(errs[tt.field], tt.message) {
				t.Errorf("errors[%q] = %q, want it to contain %q (all: %v)", tt.field, errs[tt.field], tt.message, errs)
			}
			if len(errs) != 1 {
				t.Errorf("errors = %v, want only %q", errs, tt.field)
			}
		})
	}
}

func TestValidateParamsIgnoresTheName(t *testing.T) {
	in := validInput()
	in.Name = ""

	if errs := ValidateParams(in); len(errs) != 0 {
		t.Errorf("errors = %v", errs)
	}
}

func TestInputParams(t *testing.T) {
	p := validInput().Params()

	want := Params{
		PrincipalCents: 20_000_000, AnnualRatePct: 3.5, DurationMonths: 240, InsuranceRatePct: 0.3,
		ApplicationFeeCents: 100_000, GuaranteeFeeCents: 250_000, FirstPaymentMonth: Month{2026, 12},
	}
	if p != want {
		t.Errorf("params = %+v", p)
	}
}
