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
	// Independently computed: −193,91 €/month for 60 months, then +89,16 €.
	if tl.StartMonth != "2027-01" || tl.PositiveCashFlowMonth != "2032-01" || tl.PaybackMonth != "2063-05" || tl.PaybackAtStart {
		t.Errorf("timeline = start %s, positive %s, payback %s (at start %v)",
			tl.StartMonth, tl.PositiveCashFlowMonth, tl.PaybackMonth, tl.PaybackAtStart)
	}
	first, atLoanEnd, afterLoan := tl.Points[0], tl.Points[59], tl.Points[60]
	if first.Month != "2027-01" || first.CashFlowCents != -19_391 || first.CumulativeCents != -2_189_000-19_391 {
		t.Errorf("first point = %+v", first)
	}
	if atLoanEnd.CumulativeCents != -3_352_453 || afterLoan.CashFlowCents != 8_916 {
		t.Errorf("loan end = %+v, after = %+v", atLoanEnd, afterLoan)
	}
	payback := tl.Points[436]
	if payback.Month != "2063-05" || payback.CumulativeCents < 0 || tl.Points[435].CumulativeCents >= 0 {
		t.Errorf("payback point = %+v, before = %+v", payback, tl.Points[435])
	}
	// The chart runs a year past the last event.
	if got := len(tl.Points); got != 437+12 {
		t.Errorf("points = %d, want %d", got, 437+12)
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

func TestNoTimelineWithoutAStartMonth(t *testing.T) {
	if res := Evaluate(seAppart(), referenceSettings, nil); res.Timeline != nil {
		t.Errorf("timeline = %+v", res.Timeline)
	}
}

func TestProfitabilityDatesWithoutALoan(t *testing.T) {
	p := seAppart()
	p.StartMonth = loan.Month{Year: 2026, Month: 12}

	res := Evaluate(p, referenceSettings, nil)

	tl := res.Timeline
	if tl == nil {
		t.Fatal("no timeline")
	}
	// Apport = A (36 890 €), +89,16 € every month from 12/2026: 414 months.
	if tl.StartMonth != "2026-12" || tl.PositiveCashFlowMonth != "2026-12" || tl.PaybackMonth != "2061-05" {
		t.Errorf("timeline = start %s, positive %s, payback %s", tl.StartMonth, tl.PositiveCashFlowMonth, tl.PaybackMonth)
	}
	if first := tl.Points[0]; first.CashFlowCents != 8_916 || first.CumulativeCents != -3_689_000+8_916 {
		t.Errorf("first point = %+v", first)
	}
}

func TestNoLoanAndNoIncomeNeverRecovers(t *testing.T) {
	p := seAppart()
	p.StartMonth = loan.Month{Year: 2026, Month: 12}
	p.AnnualCondoFeesCents = 1_000_000

	tl := Evaluate(p, referenceSettings, nil).Timeline

	if tl.PositiveCashFlowMonth != "" || tl.PaybackMonth != "" {
		t.Errorf("positive %q, payback %q", tl.PositiveCashFlowMonth, tl.PaybackMonth)
	}
}

func TestNetAfterCreditYield(t *testing.T) {
	withLoan := evaluateWithLoan(t, seAppart(), consoImmo)
	withoutLoan := Evaluate(seAppart(), referenceSettings, nil)

	// (1 069,90 − 1 984,13 × 12 / 60) / 36 890 = 1,82 %.
	if withLoan.Financing.AnnualCreditCostCents != 39_683 {
		t.Errorf("annual credit cost = %d, want 396,83 €", withLoan.Financing.AnnualCreditCostCents)
	}
	if withLoan.NetAfterCreditYieldPct != 1.82 {
		t.Errorf("with the loan = %v, want 1,82", withLoan.NetAfterCreditYieldPct)
	}
	if withoutLoan.NetAfterCreditYieldPct != withoutLoan.NetAfterTaxYieldPct || withoutLoan.NetAfterCreditYieldPct != 2.90 {
		t.Errorf("without a loan = %v, want the after-tax yield 2,90", withoutLoan.NetAfterCreditYieldPct)
	}
}

func TestListsCarryTheNetAfterCreditYield(t *testing.T) {
	f := newFixture(t)
	l := saveLoan(t, f, "Petit prêt")
	in := validPropertyInput()
	in.LoanID = l.ID
	_, _ = f.svc.Create(in)

	own, _ := f.svc.List("")
	active, _ := f.svc.ListWith(ListOptions{ActiveLoanID: l.ID})
	d, _ := f.svc.Detail(own[0].ID)

	if own[0].NetAfterCreditYieldPct != d.Result.NetAfterCreditYieldPct || own[0].NetAfterCreditYieldPct >= own[0].NetAfterTaxYieldPct {
		t.Errorf("list = %v, detail = %v, after tax = %v", own[0].NetAfterCreditYieldPct, d.Result.NetAfterCreditYieldPct, own[0].NetAfterTaxYieldPct)
	}
	if active[0].NetAfterCreditYieldPct != own[0].NetAfterCreditYieldPct {
		t.Errorf("active-loan list = %v", active[0].NetAfterCreditYieldPct)
	}
}
