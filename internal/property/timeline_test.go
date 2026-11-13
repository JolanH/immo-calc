package property

import (
	"testing"

	"immo-calc/internal/loan"
)

// The user's case (2026-11-13): « SE appart 1 » with « Conso Immo ».
func seAppart() Params {
	return Params{
		PriceCents: 3_400_000, MonthlyRentCents: 36_000, MonthlyRecoverableChargesCents: 5_000,
		AnnualPropertyTaxCents: 65_000, AnnualCondoFeesCents: 70_000, TaxRegime: MicroFoncier, DPE: "C",
	}
}

var consoImmo = loan.Params{
	PrincipalCents: 1_500_000, AnnualRatePct: 5, DurationMonths: 60, FirstPaymentMonth: loan.Month{Year: 2027, Month: 1},
}

func evaluateWithLoan(t *testing.T, p Params, lp loan.Params) Result {
	t.Helper()
	f, err := FiguresOf(lp)
	if err != nil {
		t.Fatal(err)
	}
	return Evaluate(p, referenceSettings, &f)
}

func TestProfitabilityDatesOfTheUsersCase(t *testing.T) {
	res := evaluateWithLoan(t, seAppart(), consoImmo)

	tl := res.Timeline
	if tl == nil {
		t.Fatal("no timeline")
	}
	// Independently computed: −178,77 €/month for 60 months, then +104,30 €.
	if tl.StartMonth != "2027-01" || tl.PositiveCashFlowMonth != "2032-01" || tl.PaybackMonth != "2058-01" || tl.PaybackAtStart {
		t.Errorf("timeline = start %s, positive %s, payback %s (at start %v)",
			tl.StartMonth, tl.PositiveCashFlowMonth, tl.PaybackMonth, tl.PaybackAtStart)
	}
	first, atLoanEnd, afterLoan := tl.Points[0], tl.Points[59], tl.Points[60]
	if first.Month != "2027-01" || first.CashFlowCents != -17_877 || first.CumulativeCents != -2_189_000-17_877 {
		t.Errorf("first point = %+v", first)
	}
	if atLoanEnd.CumulativeCents != -3_261_613 || afterLoan.CashFlowCents != 10_430 {
		t.Errorf("loan end = %+v, after = %+v", atLoanEnd, afterLoan)
	}
	payback := tl.Points[372]
	if payback.Month != "2058-01" || payback.CumulativeCents < 0 || tl.Points[371].CumulativeCents >= 0 {
		t.Errorf("payback point = %+v, before = %+v", payback, tl.Points[371])
	}
	// The chart runs a year past the last event.
	if got := len(tl.Points); got != 373+12 {
		t.Errorf("points = %d, want %d", got, 373+12)
	}
}

func TestPaybackAtStartWithoutOwnContribution(t *testing.T) {
	big := consoImmo
	big.PrincipalCents = 5_000_000 // more than the acquisition cost

	res := evaluateWithLoan(t, seAppart(), big)

	if !res.Timeline.PaybackAtStart || res.Timeline.PaybackMonth != "2027-01" {
		t.Errorf("timeline = %+v", res.Timeline.PaybackMonth)
	}
}

func TestNeverPositiveWhenTheRentDoesNotCoverTheCharges(t *testing.T) {
	p := seAppart()
	p.AnnualCondoFeesCents = 1_000_000 // charges above the rent

	res := evaluateWithLoan(t, p, consoImmo)

	tl := res.Timeline
	if tl.PositiveCashFlowMonth != "" || tl.PaybackMonth != "" {
		t.Errorf("positive %q, payback %q, want never", tl.PositiveCashFlowMonth, tl.PaybackMonth)
	}
	if got := len(tl.Points); got != 600 {
		t.Errorf("points = %d, want the 50-year horizon", got)
	}
}

func TestReelTaxAfterTheLoanLosesTheInterestDeduction(t *testing.T) {
	p := seAppart()
	p.TaxRegime = ReelFoncier
	withLoan := evaluateWithLoan(t, p, consoImmo)
	withoutLoan := Evaluate(p, referenceSettings, nil)

	during := withLoan.Timeline.Points[0].CashFlowCents + 28_307 // add back the installment
	after := withLoan.Timeline.Points[60].CashFlowCents
	if during != withLoan.MonthlyNetIncomeCents {
		t.Errorf("monthly net during the loan = %d, want %d", during, withLoan.MonthlyNetIncomeCents)
	}
	if after != withoutLoan.MonthlyNetIncomeCents || after >= during {
		t.Errorf("monthly net after the loan = %d, want %d (higher tax, lower than %d)",
			after, withoutLoan.MonthlyNetIncomeCents, during)
	}
}

func TestNoTimelineWithoutALoan(t *testing.T) {
	if res := Evaluate(seAppart(), referenceSettings, nil); res.Timeline != nil {
		t.Errorf("timeline = %+v", res.Timeline)
	}
}
