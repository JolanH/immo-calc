package property

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"immo-calc/internal/loan"
)

// fakeLoans is an in-memory LoanSource.
type fakeLoans map[string]loan.Loan

func (f fakeLoans) Get(id string) (loan.Loan, error) {
	l, ok := f[id]
	if !ok {
		return loan.Loan{}, loan.ErrNotFound
	}
	return l, nil
}

var knownLoans = fakeLoans{
	"aaaaaaaaaaaaaaaa": {ID: "aaaaaaaaaaaaaaaa", Input: loan.Input{
		Name: "Petit prêt", PrincipalCents: 5_000_000, AnnualRatePct: 3, DurationMonths: 240,
		InsuranceRatePct: 0.3, ApplicationFeeCents: 100_000, FirstPaymentMonth: "2026-12",
	}},
}

func validPropertyInput() Input {
	return Input{
		Name:                           "T2 Lyon 7e",
		PriceCents:                     15_000_000,
		DPE:                            "D",
		MonthlyRentCents:               70_000,
		MonthlyRecoverableChargesCents: 5_000,
		AnnualPropertyTaxCents:         90_000,
		AnnualCondoFeesCents:           120_000,
		AnnualLandlordInsuranceCents:   15_000,
		TaxRegime:                      MicroFoncier,
	}
}

func TestValidPropertyInput(t *testing.T) {
	if errs := ValidateInput(validPropertyInput(), knownLoans); len(errs) != 0 {
		t.Errorf("errors = %v", errs)
	}
}

func TestPropertyValidationRules(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Input)
		field   string
		message string
	}{
		{"empty name", func(in *Input) { in.Name = " " }, "name", "Le nom est obligatoire"},
		{"name too long", func(in *Input) { in.Name = strings.Repeat("a", 81) }, "name", "80 caractères"},
		{"price too low", func(in *Input) { in.PriceCents = 999_999 }, "priceCents", "Le prix doit être compris entre 10 000 € et 5 000 000 €"},
		{"price at minimum", func(in *Input) { in.PriceCents = 1_000_000 }, "", ""},
		{"price too high", func(in *Input) { in.PriceCents = 500_000_001 }, "priceCents", "Le prix doit être compris"},
		{"no DPE", func(in *Input) { in.DPE = "" }, "dpe", "Choisissez la classe DPE (A à G)"},
		{"unknown DPE", func(in *Input) { in.DPE = "H" }, "dpe", "Choisissez la classe DPE"},
		{"DPE G", func(in *Input) { in.DPE = "G" }, "", ""},
		{"rent zero", func(in *Input) { in.MonthlyRentCents = 99 }, "monthlyRentCents", "Le loyer doit être compris entre 1 € et 20 000 € par mois"},
		{"rent at maximum", func(in *Input) { in.MonthlyRentCents = 2_000_000 }, "", ""},
		{"rent too high", func(in *Input) { in.MonthlyRentCents = 2_000_001 }, "monthlyRentCents", "Le loyer"},
		{"negative charges", func(in *Input) { in.MonthlyRecoverableChargesCents = -1 }, "monthlyRecoverableChargesCents", "Les charges doivent être comprises entre 0 et 2 000 € par mois"},
		{"charges too high", func(in *Input) { in.MonthlyRecoverableChargesCents = 200_001 }, "monthlyRecoverableChargesCents", "2 000 €"},
		{"property tax too high", func(in *Input) { in.AnnualPropertyTaxCents = 5_000_001 }, "annualPropertyTaxCents", "La taxe foncière doit être comprise entre 0 et 50 000 € par an"},
		{"negative condo fees", func(in *Input) { in.AnnualCondoFeesCents = -1 }, "annualCondoFeesCents", "Les charges de copropriété doivent être comprises entre 0 et 50 000 € par an"},
		{"insurance too high", func(in *Input) { in.AnnualLandlordInsuranceCents = 500_001 }, "annualLandlordInsuranceCents", "L'assurance PNO doit être comprise entre 0 et 5 000 € par an"},
		{"unknown regime", func(in *Input) { in.TaxRegime = "lmnp_reel" }, "taxRegime", "Choisissez un régime fiscal"},
		{"réel", func(in *Input) { in.TaxRegime = ReelFoncier }, "", ""},
		{"micro-BIC", func(in *Input) { in.TaxRegime = MicroBIC }, "", ""},
		{"unknown loan", func(in *Input) { in.LoanID = "ffffffffffffffff" }, "loanId", "Ce prêt n'existe plus"},
		{"known loan", func(in *Input) { in.LoanID = "aaaaaaaaaaaaaaaa" }, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := validPropertyInput()
			tt.mutate(&in)

			errs := ValidateInput(in, knownLoans)

			if tt.field == "" {
				if len(errs) != 0 {
					t.Errorf("errors = %v", errs)
				}
				return
			}
			if !strings.Contains(errs[tt.field], tt.message) || len(errs) != 1 {
				t.Errorf("errors = %v, want %q on %s", errs, tt.message, tt.field)
			}
		})
	}
}

func TestValidateParamsIgnoresTheName(t *testing.T) {
	in := validPropertyInput()
	in.Name = ""
	if errs := ValidateParams(in, knownLoans); len(errs) != 0 {
		t.Errorf("errors = %v", errs)
	}
}

func evaluateRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/properties/evaluate", strings.NewReader(body))
	req.Host = "127.0.0.1:8421"
	req.Header.Set("Origin", localOrigin)
	return req
}

func newEvaluateHandlers(t *testing.T) *Handlers {
	t.Helper()
	store := openStore(t, filepath.Join(t.TempDir(), "immo-calc.yaml"))
	return NewHandlers(NewService(store, knownLoans))
}

const evaluateBody = `{"priceCents":15000000,"dpe":"F","monthlyRentCents":70000,"monthlyRecoverableChargesCents":5000,
	"annualPropertyTaxCents":90000,"annualCondoFeesCents":120000,"annualLandlordInsuranceCents":15000,
	"agencyManaged":false,"taxRegime":"micro_foncier"}`

func TestEvaluateEndpoint(t *testing.T) {
	rec := serveProperty(newEvaluateHandlers(t), evaluateRequest(evaluateBody))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	for key, want := range map[string]float64{
		"acquisitionCostCents": 16275000, "collectedRentCents": 770000, "ownerChargesCents": 230000,
		"incomeBeforeTaxCents": 540000, "taxableIncomeCents": 539000, "annualTaxCents": 254408,
		"netAnnualIncomeCents": 285592, "monthlyGrossIncomeCents": 70000, "monthlyNetIncomeCents": 23799,
		"grossYieldPct": 5.16, "netYieldPct": 3.32, "netAfterTaxYieldPct": 1.75,
	} {
		if body[key] != want {
			t.Errorf("%s = %v, want %v", key, body[key], want)
		}
	}
	if body["financing"] != nil {
		t.Errorf("financing = %v, want null without a loan", body["financing"])
	}
	warnings := body["warnings"].([]any)
	if len(warnings) != 1 || warnings[0].(map[string]any)["severity"] != "serious" {
		t.Errorf("warnings = %v", warnings)
	}
}

func TestEvaluateWithALoanUsesCurrentSettings(t *testing.T) {
	h := newEvaluateHandlers(t)
	s := DefaultSettings()
	s.MarginalTaxRatePct = 11
	if _, err := h.svc.UpdateSettings(s); err != nil {
		t.Fatal(err)
	}
	body := strings.Replace(evaluateBody, `"taxRegime":"micro_foncier"`, `"taxRegime":"reel_foncier","loanId":"aaaaaaaaaaaaaaaa"`, 1)

	rec := serveProperty(h, evaluateRequest(body))

	var res Result
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	// Taxable 3 775,34 € at 11 % + 17,2 % = 1 064,65 €.
	if rec.Code != http.StatusOK || res.TaxableIncomeCents != 377_534 || res.AnnualTaxCents != 106_465 {
		t.Errorf("status = %d, res = %+v", rec.Code, res)
	}
	if res.Financing == nil || res.Financing.MonthlyPaymentCents != 28_980 {
		t.Errorf("financing = %+v", res.Financing)
	}
}

func TestEvaluateReturnsFieldErrors(t *testing.T) {
	body := strings.Replace(evaluateBody, `"dpe":"F"`, `"dpe":""`, 1)
	body = strings.Replace(body, `"loanId"`, `"x"`, 1)

	rec := serveProperty(newEvaluateHandlers(t), evaluateRequest(body))

	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), `"dpe"`) ||
		!strings.Contains(rec.Body.String(), "validation_failed") {
		t.Errorf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestEvaluateRejectsMalformedJSONAndForeignOrigins(t *testing.T) {
	h := newEvaluateHandlers(t)
	if rec := serveProperty(h, evaluateRequest(`{`)); rec.Code != http.StatusBadRequest {
		t.Errorf("malformed: status = %d", rec.Code)
	}
	req := evaluateRequest(evaluateBody)
	req.Header.Set("Origin", "http://evil.example")
	if rec := serveProperty(h, req); rec.Code != http.StatusForbidden {
		t.Errorf("foreign origin: status = %d", rec.Code)
	}
}
