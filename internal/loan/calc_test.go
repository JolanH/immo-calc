package loan

import (
	"reflect"
	"testing"
)

// Reference values from SPEC-loan.md (no insurance, no fees).
//
// Computed with the spec's rules (monthly rate = annual/12, interest rounded
// to the cent each month on the remaining principal, last installment
// adjusted), recomputed independently, and cross-checked on 2026-11-13
// against a public French simulator for the first row: moicombien.fr
// (https://moicombien.fr/simulateur/pret-immobilier-200000) gives a monthly
// payment of 1 159,92 € and about 78 381 € of interest for 200 000 € at
// 3,5 % over 20 years. Simulators round totals differently (±1 € expected).
var referenceLoans = []struct {
	name           string
	principal      int64
	rate           float64
	years          int
	monthly        int64
	totalInterest  int64
	lastInstalment int64
}{
	{"200k at 3.5% over 20y", 20_000_000, 3.5, 20, 115_992, 7_838_066, 115_978},
	{"150k at 4% over 25y", 15_000_000, 4.0, 25, 79_176, 8_752_543, 78_919},
	{"250k at 3.2% over 15y", 25_000_000, 3.2, 15, 175_060, 6_510_861, 175_121},
	{"100k at 0% over 10y", 10_000_000, 0, 10, 83_333, 0, 83_373},
}

func TestReferenceLoans(t *testing.T) {
	for _, tt := range referenceLoans {
		t.Run(tt.name, func(t *testing.T) {
			res, err := Compute(Params{PrincipalCents: tt.principal, AnnualRatePct: tt.rate, DurationMonths: tt.years * 12})
			if err != nil {
				t.Fatalf("compute: %v", err)
			}
			if res.MonthlyPaymentCents != tt.monthly {
				t.Errorf("monthly payment = %d, want %d", res.MonthlyPaymentCents, tt.monthly)
			}
			if res.TotalInterestCents != tt.totalInterest {
				t.Errorf("total interest = %d, want %d", res.TotalInterestCents, tt.totalInterest)
			}
			last := res.Schedule[len(res.Schedule)-1]
			if last.PaymentCents != tt.lastInstalment {
				t.Errorf("last installment = %d, want %d", last.PaymentCents, tt.lastInstalment)
			}
			if len(res.Schedule) != tt.years*12 {
				t.Errorf("schedule length = %d", len(res.Schedule))
			}
		})
	}
}

func TestMonthlyPayment(t *testing.T) {
	tests := []struct {
		name      string
		principal int64
		rate      float64
		months    int
		want      int64
	}{
		{"reference", 20_000_000, 3.5, 240, 115_992},
		{"zero rate", 10_000_000, 0, 120, 83_333},
		{"one month at zero", 100_000, 0, 1, 100_000},
		{"one month with interest", 100_000, 12, 1, 101_000},
		{"30 years", 30_000_000, 4.25, 360, 147_582},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MonthlyPayment(tt.principal, tt.rate, tt.months)
			if err != nil || got != tt.want {
				t.Errorf("MonthlyPayment = %d, %v; want %d", got, err, tt.want)
			}
		})
	}
}

func TestMonthlyPaymentRejectsInvalidInput(t *testing.T) {
	for _, tt := range []struct {
		principal int64
		rate      float64
		months    int
	}{
		{0, 3, 240}, {-1, 3, 240}, {100_000, -0.1, 240}, {100_000, 3, 0}, {100_000, 3, -12},
	} {
		if _, err := MonthlyPayment(tt.principal, tt.rate, tt.months); err == nil {
			t.Errorf("MonthlyPayment(%d, %v, %d) returned no error", tt.principal, tt.rate, tt.months)
		}
	}
}

func TestScheduleFirstRowsFollowTheRules(t *testing.T) {
	res, err := Compute(Params{PrincipalCents: 20_000_000, AnnualRatePct: 3.5, DurationMonths: 240})
	if err != nil {
		t.Fatal(err)
	}

	// Values from the result JSON example in SPEC-loan.md.
	first := res.Schedule[0]
	want := Installment{Index: 1, PaymentCents: 115_992, InterestCents: 58_333, PrincipalCents: 57_659, RemainingCents: 19_942_341}
	if first != want {
		t.Errorf("first installment = %+v, want %+v", first, want)
	}
	second := res.Schedule[1]
	if second.Index != 2 || second.InterestCents != 58_165 || second.RemainingCents != 19_942_341-(115_992-58_165) {
		t.Errorf("second installment = %+v", second)
	}
}

func TestScheduleInvariants(t *testing.T) {
	for _, tt := range referenceLoans {
		t.Run(tt.name, func(t *testing.T) {
			res, _ := Compute(Params{PrincipalCents: tt.principal, AnnualRatePct: tt.rate, DurationMonths: tt.years * 12})
			assertInvariants(t, Params{PrincipalCents: tt.principal}, res)
		})
	}
}

// assertInvariants checks what must hold for every schedule.
func assertInvariants(t *testing.T, p Params, res Result) {
	t.Helper()
	var sumPrincipal, sumInterest, sumInsurance int64
	for i, row := range res.Schedule {
		sumPrincipal += row.PrincipalCents
		sumInterest += row.InterestCents
		sumInsurance += row.InsuranceCents
		if row.PaymentCents != row.PrincipalCents+row.InterestCents {
			t.Fatalf("row %d: payment %d != principal %d + interest %d", i+1, row.PaymentCents, row.PrincipalCents, row.InterestCents)
		}
		if row.PrincipalCents < 0 || row.RemainingCents < 0 {
			t.Fatalf("row %d: negative amounts %+v", i+1, row)
		}
	}
	if last := res.Schedule[len(res.Schedule)-1]; last.RemainingCents != 0 {
		t.Errorf("remaining after last installment = %d, want 0", last.RemainingCents)
	}
	if sumPrincipal != p.PrincipalCents {
		t.Errorf("sum of principal repaid = %d, want %d", sumPrincipal, p.PrincipalCents)
	}
	if sumInterest != res.TotalInterestCents {
		t.Errorf("sum of interest = %d, total = %d", sumInterest, res.TotalInterestCents)
	}
	if sumInsurance != res.TotalInsuranceCents {
		t.Errorf("sum of insurance = %d, total = %d", sumInsurance, res.TotalInsuranceCents)
	}
}

func TestInsuranceAndFees(t *testing.T) {
	// SPEC-loan.md: 200 000 € at 0,30 % insurance gives 50,00 €/month,
	// 12 000,00 € over 20 years.
	p := Params{
		PrincipalCents:      20_000_000,
		AnnualRatePct:       3.5,
		DurationMonths:      240,
		InsuranceRatePct:    0.3,
		ApplicationFeeCents: 100_000,
		GuaranteeFeeCents:   250_000,
	}

	res, err := Compute(p)
	if err != nil {
		t.Fatal(err)
	}

	// Matches the result JSON example in SPEC-loan.md.
	want := Result{
		MonthlyPaymentCents:   115_992,
		MonthlyInsuranceCents: 5_000,
		MonthlyTotalCents:     120_992,
		TotalInterestCents:    7_838_066,
		TotalInsuranceCents:   1_200_000,
		TotalFeesCents:        350_000,
		TotalCostCents:        9_388_066,
		TotalRepaidCents:      29_038_066,
	}
	got := res
	got.Schedule = nil
	if !reflect.DeepEqual(got, want) {
		t.Errorf("result = %+v\nwant     %+v", got, want)
	}
	if res.Schedule[0].InsuranceCents != 5_000 || res.Schedule[239].InsuranceCents != 5_000 {
		t.Errorf("insurance per installment = %d … %d", res.Schedule[0].InsuranceCents, res.Schedule[239].InsuranceCents)
	}
	assertInvariants(t, p, res)
}

func TestOneMonthLoan(t *testing.T) {
	res, err := Compute(Params{PrincipalCents: 100_000, AnnualRatePct: 12, DurationMonths: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Schedule) != 1 || res.Schedule[0] != (Installment{Index: 1, PaymentCents: 101_000, InterestCents: 1_000, PrincipalCents: 100_000}) {
		t.Errorf("schedule = %+v", res.Schedule)
	}
}

func TestComputeRejectsInvalidInput(t *testing.T) {
	valid := Params{PrincipalCents: 100_000, AnnualRatePct: 3, DurationMonths: 12}
	for name, mutate := range map[string]func(*Params){
		"no principal":       func(p *Params) { p.PrincipalCents = 0 },
		"negative rate":      func(p *Params) { p.AnnualRatePct = -1 },
		"no duration":        func(p *Params) { p.DurationMonths = 0 },
		"negative insurance": func(p *Params) { p.InsuranceRatePct = -0.1 },
		"negative fee":       func(p *Params) { p.ApplicationFeeCents = -1 },
		"negative guarantee": func(p *Params) { p.GuaranteeFeeCents = -1 },
	} {
		t.Run(name, func(t *testing.T) {
			p := valid
			mutate(&p)
			if _, err := Compute(p); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

func TestRoundCentsHalfAwayFromZero(t *testing.T) {
	for _, tt := range []struct {
		in   float64
		want int64
	}{
		{12.5, 13}, {12.4999, 12}, {-12.5, -13}, {0, 0}, {0.5, 1}, {83_333.333, 83_333},
	} {
		if got := roundCents(tt.in); got != tt.want {
			t.Errorf("roundCents(%v) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestHalfCentInterestRoundsUp(t *testing.T) {
	// 1 000 € at 0,6 % a year: the first month's interest is 50 cents exactly
	// (100 000 × 0,0005); 1 001 € gives 50,05 cents, which rounds to 50.
	if got := roundCents(100_100 * 0.6 / 100 / 12); got != 50 {
		t.Errorf("interest = %d, want 50", got)
	}
	res, _ := Compute(Params{PrincipalCents: 100_000, AnnualRatePct: 0.6, DurationMonths: 12})
	if res.Schedule[0].InterestCents != 50 {
		t.Errorf("first interest = %d, want 50", res.Schedule[0].InterestCents)
	}
}
