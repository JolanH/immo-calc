package property

import (
	"testing"

	"immo-calc/internal/loan"
)

func figures(t *testing.T, p loan.Params) LoanFigures {
	t.Helper()
	f, err := FiguresOf(p)
	if err != nil {
		t.Fatalf("figures: %v", err)
	}
	return f
}

// Reference loan 2 of SPEC-loan.md: 150 000 € at 4 % over 25 years, no
// insurance, no fees. Values below were computed independently with the
// spec rules (first-year interest = sum of the first 12 installments).
var bigLoan = loan.Params{PrincipalCents: 15_000_000, AnnualRatePct: 4, DurationMonths: 300}

// A smaller loan where the réel deduction is partial: 50 000 € at 3 % over
// 20 years, insurance 0,30 %, 1 000 € of fees.
var smallLoan = loan.Params{
	PrincipalCents: 5_000_000, AnnualRatePct: 3, DurationMonths: 240,
	InsuranceRatePct: 0.3, ApplicationFeeCents: 100_000,
}

func TestFiguresOfALoan(t *testing.T) {
	big := figures(t, bigLoan)
	if big.MonthlyTotalCents != 79_176 || big.FirstYearInterest != 593_509 || big.FirstYearInsurance != 0 ||
		big.PrincipalCents != 15_000_000 || big.FeesCents != 0 {
		t.Errorf("big loan figures = %+v", big)
	}
	small := figures(t, smallLoan)
	if small.MonthlyTotalCents != 28_980 || small.FirstYearInterest != 147_466 || small.FirstYearInsurance != 15_000 ||
		small.FeesCents != 100_000 {
		t.Errorf("small loan figures = %+v", small)
	}
}

func TestFiguresOfAOneYearLoanUseItsOwnInstallments(t *testing.T) {
	f := figures(t, loan.Params{PrincipalCents: 1_200_000, AnnualRatePct: 0, DurationMonths: 12})
	if f.FirstYearInterest != 0 || f.MonthlyTotalCents != 100_000 {
		t.Errorf("figures = %+v", f)
	}
}

func TestFiguresOfAnInvalidLoan(t *testing.T) {
	if _, err := FiguresOf(loan.Params{}); err == nil {
		t.Error("expected an error")
	}
}

func TestReelDeductsFirstYearInterestAndInsurance(t *testing.T) {
	small := figures(t, smallLoan)

	res := Evaluate(referenceFlat(ReelFoncier, false), referenceSettings, &small)

	// R 5 400 € − interest 1 474,66 € − insurance 150 € = 3 775,34 €.
	if res.TaxableIncomeCents != 377_534 || res.AnnualTaxCents != 178_196 || res.NetAnnualIncomeCents != 361_804 {
		t.Errorf("taxable = %d, tax = %d, RN = %d", res.TaxableIncomeCents, res.AnnualTaxCents, res.NetAnnualIncomeCents)
	}
}

func TestReelDeductionIsFlooredAtZero(t *testing.T) {
	big := figures(t, bigLoan)

	res := Evaluate(referenceFlat(ReelFoncier, false), referenceSettings, &big)

	// First-year interest (5 935,09 €) is above R (5 400 €): no tax, no carry-over.
	if res.TaxableIncomeCents != 0 || res.AnnualTaxCents != 0 || res.NetAnnualIncomeCents != 540_000 {
		t.Errorf("taxable = %d, tax = %d, RN = %d", res.TaxableIncomeCents, res.AnnualTaxCents, res.NetAnnualIncomeCents)
	}
}

func TestMicroRegimesIgnoreTheLoanForTax(t *testing.T) {
	small := figures(t, smallLoan)
	for _, regime := range []Regime{MicroFoncier, MicroBIC} {
		with := Evaluate(referenceFlat(regime, false), referenceSettings, &small)
		without := Evaluate(referenceFlat(regime, false), referenceSettings, nil)
		if with.AnnualTaxCents != without.AnnualTaxCents {
			t.Errorf("%s: tax %d with a loan, %d without", regime, with.AnnualTaxCents, without.AnnualTaxCents)
		}
	}
}

func TestFinancingFigures(t *testing.T) {
	small := figures(t, smallLoan)

	res := Evaluate(referenceFlat(ReelFoncier, false), referenceSettings, &small)

	f := res.Financing
	if f == nil {
		t.Fatal("no financing figures")
	}
	want := Financing{
		MonthlyPaymentCents:    28_980,
		CashFlowBeforeTaxCents: 16_020,     // 450 € − 289,80 €
		CashFlowAfterTaxCents:  1_170,      // 301,50 € − 289,80 €
		OwnContributionCents:   11_375_000, // 162 750 € + 1 000 € fees − 50 000 €
		// (16 551,57 € interest + 3 000 € insurance + 1 000 € fees) / 20 years.
		AnnualCreditCostCents: 102_758,
	}
	got := *f
	got.ReturnOnContributionPct = nil
	if got != want {
		t.Errorf("financing = %+v\nwant        %+v", got, want)
	}
	if f.ReturnOnContributionPct == nil || *f.ReturnOnContributionPct != 0.12 {
		t.Errorf("rendement sur apport = %v, want 0,12", f.ReturnOnContributionPct)
	}
}

func TestReferenceCaseWithTheReferenceLoan(t *testing.T) {
	big := figures(t, bigLoan)

	res := Evaluate(referenceFlat(MicroFoncier, false), referenceSettings, &big)

	f := res.Financing
	if f.MonthlyPaymentCents != 79_176 || f.CashFlowBeforeTaxCents != -34_176 || f.CashFlowAfterTaxCents != -55_377 ||
		f.OwnContributionCents != 1_275_000 {
		t.Errorf("financing = %+v", *f)
	}
	if f.ReturnOnContributionPct == nil || *f.ReturnOnContributionPct != -52.12 {
		t.Errorf("rendement sur apport = %v, want -52,12", f.ReturnOnContributionPct)
	}
}

func TestReturnOnContributionIsNotApplicableWithoutAContribution(t *testing.T) {
	full := figures(t, loan.Params{PrincipalCents: 20_000_000, AnnualRatePct: 3.5, DurationMonths: 240})

	res := Evaluate(referenceFlat(MicroFoncier, false), referenceSettings, &full)

	if res.Financing.OwnContributionCents > 0 || res.Financing.ReturnOnContributionPct != nil {
		t.Errorf("financing = %+v", *res.Financing)
	}
}

func TestNoFinancingWithoutALoan(t *testing.T) {
	if res := Evaluate(referenceFlat(MicroFoncier, false), referenceSettings, nil); res.Financing != nil {
		t.Errorf("financing = %+v", res.Financing)
	}
}
