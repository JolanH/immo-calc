package property

import "testing"

// Hand-checked on referenceFlat: R = 5 400 €, E = 7 700 €, reimbursed charges
// 50 € × 11 = 550 €, T = 30 % + 17,2 % (nue) or 18,6 % (meublée).

func TestMicroRegimesTaxTheReimbursedChargesToo(t *testing.T) {
	for _, tt := range []struct {
		regime  Regime
		taxable int64
	}{
		{MicroFoncier, 577_500}, // (7 700 + 550) × 70 %
		{MicroBIC, 412_500},     // (7 700 + 550) × 50 %
		{ReelFoncier, 540_000},  // réel: receipt and charge cancel out, R
	} {
		r := Evaluate(referenceFlat(tt.regime, false), referenceSettings, nil)
		if r.ReimbursedChargesCents != 55_000 || r.TaxableIncomeCents != tt.taxable {
			t.Errorf("%s: reimbursed = %d, taxable = %d, want 550 € and %d", tt.regime, r.ReimbursedChargesCents, r.TaxableIncomeCents, tt.taxable)
		}
	}
}

func TestMicroCeilingCountsReimbursedCharges(t *testing.T) {
	p := referenceFlat(MicroFoncier, false)
	p.MonthlyRentCents, p.MonthlyRecoverableChargesCents = 130_000, 10_000 // E = 14 300 €, + 1 100 € = 15 400 €
	p.AnnualCondoFeesCents, p.AnnualPropertyTaxCents, p.AnnualLandlordInsuranceCents = 0, 0, 0

	r := Evaluate(p, referenceSettings, nil)

	if r.CollectedRentCents > MicroFoncierCeilingCents {
		t.Fatalf("rent alone is already above the ceiling: %d", r.CollectedRentCents)
	}
	if w := ceilingWarning(MicroFoncier, microReceipts(r)); w == nil {
		t.Error("no warning although rent + reimbursed charges are above 15 000 €")
	}
}

func TestIneligibleMicroRegimeIsNeverTheBestOne(t *testing.T) {
	p := referenceFlat(ReelFoncier, false)
	p.MonthlyRentCents, p.MonthlyRecoverableChargesCents = 130_000, 10_000
	p.AnnualCondoFeesCents, p.AnnualPropertyTaxCents, p.AnnualLandlordInsuranceCents = 0, 0, 0

	r := Evaluate(p, referenceSettings, nil)

	byRegime := map[Regime]RegimeComparison{}
	for _, c := range r.Regimes {
		byRegime[c.Regime] = c
	}
	micro, reel := byRegime[MicroFoncier], byRegime[ReelFoncier]
	if micro.Eligible || !reel.Eligible {
		t.Errorf("eligible: micro-foncier %v, réel %v", micro.Eligible, reel.Eligible)
	}
	if micro.AnnualTaxCents >= reel.AnnualTaxCents {
		t.Fatalf("the test needs micro to be the cheapest: %d vs %d", micro.AnnualTaxCents, reel.AnnualTaxCents)
	}
	if micro.Best || !reel.Best {
		t.Errorf("best: micro-foncier %v, réel %v", micro.Best, reel.Best)
	}
}

func TestBestIsChosenAmongTheSameKindOfRental(t *testing.T) {
	r := Evaluate(referenceFlat(MicroFoncier, false), referenceSettings, nil)

	var nue, meuble []string
	for _, c := range r.Regimes {
		if c.Best && c.Furnished {
			meuble = append(meuble, string(c.Regime))
		}
		if c.Best && !c.Furnished {
			nue = append(nue, string(c.Regime))
		}
	}
	if len(nue) != 1 || len(meuble) != 1 {
		t.Errorf("best nue = %v, best meublée = %v, want one of each", nue, meuble)
	}
}

func TestLoanFeesAreDeductedInTheFirstYearOnly(t *testing.T) {
	lf := &LoanFigures{FeesCents: 50_000, YearlyInterest: []int64{100_000, 90_000}}
	p := referenceFlat(ReelFoncier, false)
	r := Evaluate(p, referenceSettings, lf)

	for year, wantTaxable := range []int64{390_000, 450_000, 540_000} { // R − interest − fees (year 1)
		if got := taxFor(ReelFoncier, p, referenceSettings, r, lf, year).taxable; got != wantTaxable {
			t.Errorf("year %d: taxable = %d, want %d", year+1, got, wantTaxable)
		}
	}
}

func TestTheTaxRisesAsTheInterestFalls(t *testing.T) {
	f := figures(t, smallLoan)
	p := referenceFlat(ReelFoncier, false)
	p.StartMonth = f.FirstPaymentMonth
	r := Evaluate(p, referenceSettings, &f)

	net := yearlyNet(p, referenceSettings, r, &f)

	// 20-year loan: the net income falls every year, then settles once it is repaid.
	for y := 1; y < 20; y++ {
		if net[y] >= net[y-1] {
			t.Errorf("year %d net %d is not below year %d net %d", y+1, net[y], y, net[y-1])
		}
	}
	if net[20] != net[21] || net[20] >= net[19] {
		t.Errorf("after the loan: net[19..21] = %d, %d, %d", net[19], net[20], net[21])
	}
	if net[0] != r.NetAnnualIncomeCents {
		t.Errorf("year 1 net = %d, want the result's %d", net[0], r.NetAnnualIncomeCents)
	}
}

func TestLMNPDepreciationStopsAfterItsPeriod(t *testing.T) {
	p := referenceFlat(LMNPReel, false)
	p.FurnitureAddedCents = 700_000
	r := Evaluate(p, referenceSettings, nil)

	cases := []struct {
		year                int
		building, furniture bool
	}{{0, true, true}, {6, true, true}, {7, true, false}, {29, true, false}, {30, false, false}}
	for _, c := range cases {
		o := taxFor(LMNPReel, p, referenceSettings, r, nil, c.year)
		if (o.building > 0) != c.building || (o.furniture > 0) != c.furniture {
			t.Errorf("year %d: building %d, furniture %d", c.year+1, o.building, o.furniture)
		}
	}
	// Furniture 7 000 € / 7 = 1 000 € a year.
	if o := taxFor(LMNPReel, p, referenceSettings, r, nil, 0); o.furniture != 100_000 {
		t.Errorf("furniture depreciation = %d, want 1 000 €", o.furniture)
	}
}

func TestYearlyTaxFollowsTheLMNPDepreciation(t *testing.T) {
	p := referenceFlat(LMNPReel, false)
	p.FurnitureIncludedCents = 700_000 // 7 000 € over 7 years = 1 000 € a year

	r := Evaluate(p, referenceSettings, nil)

	// Shape only: the tax is flat for 7 years, steps up when the furniture is fully
	// depreciated, steps up again after 30 years (the building), then the series
	// stops 3 years later.
	y := r.YearlyTax
	if len(y) != 34 || y[0].Year != 1 || y[33].Year != 34 {
		t.Fatalf("years = %d, want 34 (30 years of building + 3 + the step)", len(y))
	}
	if y[6].AnnualTaxCents != y[0].AnnualTaxCents || y[7].AnnualTaxCents <= y[6].AnnualTaxCents {
		t.Errorf("tax years 1, 7, 8 = %d, %d, %d", y[0].AnnualTaxCents, y[6].AnnualTaxCents, y[7].AnnualTaxCents)
	}
	if y[30].AnnualTaxCents <= y[29].AnnualTaxCents {
		t.Errorf("tax years 30, 31 = %d, %d", y[29].AnnualTaxCents, y[30].AnnualTaxCents)
	}
	for _, c := range y {
		if c.AnnualTaxCents+c.NetAnnualIncomeCents != r.IncomeBeforeTaxCents {
			t.Fatalf("year %d: tax %d + net %d != R %d", c.Year, c.AnnualTaxCents, c.NetAnnualIncomeCents, r.IncomeBeforeTaxCents)
		}
	}
	if y[0].AnnualTaxCents != r.AnnualTaxCents || y[0].NetAnnualIncomeCents != r.NetAnnualIncomeCents {
		t.Errorf("year 1 = %+v, result tax %d", y[0], r.AnnualTaxCents)
	}
}

func TestYearlyTaxOfAMicroRegimeIsFlatAndShort(t *testing.T) {
	r := Evaluate(referenceFlat(MicroFoncier, false), referenceSettings, nil)

	if len(r.YearlyTax) != 10 {
		t.Errorf("years = %d, want the 10-year minimum", len(r.YearlyTax))
	}
}
