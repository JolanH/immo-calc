package loan

import (
	"encoding/json"
	"math/rand/v2"
	"reflect"
	"testing"
)

func mustMonth(t *testing.T, s string) Month {
	t.Helper()
	m, err := ParseMonth(s)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestParseMonth(t *testing.T) {
	for _, tt := range []struct {
		in    string
		want  Month
		valid bool
	}{
		{"2026-12", Month{2026, 12}, true},
		{"2000-01", Month{2000, 1}, true},
		{"2100-12", Month{2100, 12}, true},
		{"2026-13", Month{}, false},
		{"2026-00", Month{}, false},
		{"2026-1", Month{}, false},
		{"26-01", Month{}, false},
		{"", Month{}, false},
		{"2026/12", Month{}, false},
	} {
		got, err := ParseMonth(tt.in)
		if (err == nil) != tt.valid || got != tt.want {
			t.Errorf("ParseMonth(%q) = %v, %v", tt.in, got, err)
		}
	}
}

func TestMonthArithmeticAndFormatting(t *testing.T) {
	m := Month{2026, 11}
	for _, tt := range []struct {
		add  int
		want string
	}{{0, "2026-11"}, {1, "2026-12"}, {2, "2027-01"}, {14, "2028-01"}, {-11, "2025-12"}} {
		if got := m.Add(tt.add).String(); got != tt.want {
			t.Errorf("%v + %d = %s, want %s", m, tt.add, got, tt.want)
		}
	}
	if !(Month{}).IsZero() || m.IsZero() {
		t.Error("IsZero is wrong")
	}
	if got := m.Next(); got != (Month{2026, 12}) {
		t.Errorf("Next = %v", got)
	}
}

func TestMonthJSONRoundTrip(t *testing.T) {
	data, err := json.Marshal(struct{ M Month }{Month{2026, 12}})
	if err != nil || string(data) != `{"M":"2026-12"}` {
		t.Fatalf("marshal = %s, %v", data, err)
	}
	var back struct{ M Month }
	if err := json.Unmarshal(data, &back); err != nil || back.M != (Month{2026, 12}) {
		t.Errorf("unmarshal = %v, %v", back.M, err)
	}
	if err := json.Unmarshal([]byte(`{"M":"2026-13"}`), &back); err == nil {
		t.Error("expected an error for an invalid month")
	}
}

func TestInstallmentsAreDatedFromTheFirstPaymentMonth(t *testing.T) {
	res, err := Compute(Params{PrincipalCents: 20_000_000, AnnualRatePct: 3.5, DurationMonths: 240,
		FirstPaymentMonth: mustMonth(t, "2026-12")})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Schedule[0].Month.String(); got != "2026-12" {
		t.Errorf("first month = %s", got)
	}
	if got := res.Schedule[1].Month.String(); got != "2027-01" {
		t.Errorf("second month = %s", got)
	}
	if got := res.Schedule[239].Month.String(); got != "2046-11" {
		t.Errorf("last month = %s", got)
	}
}

func TestYearlyGroupsByCalendarYearWithPartialYears(t *testing.T) {
	res, err := Compute(Params{PrincipalCents: 20_000_000, AnnualRatePct: 3.5, DurationMonths: 240,
		InsuranceRatePct: 0.3, FirstPaymentMonth: mustMonth(t, "2026-11")})
	if err != nil {
		t.Fatal(err)
	}

	years := res.Yearly
	if len(years) != 21 {
		t.Fatalf("years = %d, want 21 (2026 … 2046)", len(years))
	}
	first, last := years[0], years[len(years)-1]
	if first.Year != 2026 || first.Installments != 2 {
		t.Errorf("first year = %d with %d installments, want 2026 with 2", first.Year, first.Installments)
	}
	if last.Year != 2046 || last.Installments != 10 {
		t.Errorf("last year = %d with %d installments, want 2046 with 10", last.Year, last.Installments)
	}
	if years[1].Installments != 12 {
		t.Errorf("full year has %d installments", years[1].Installments)
	}
	if last.RemainingCents != 0 {
		t.Errorf("remaining at the end = %d", last.RemainingCents)
	}
	assertYearlyMatchesMonths(t, res)
}

// assertYearlyMatchesMonths checks each yearly row is the sum of its months.
func assertYearlyMatchesMonths(t *testing.T, res Result) {
	t.Helper()
	sums := map[int]*YearRow{}
	var order []int
	for _, row := range res.Schedule {
		y := row.Month.Year
		if sums[y] == nil {
			sums[y] = &YearRow{Year: y}
			order = append(order, y)
		}
		s := sums[y]
		s.Installments++
		s.PaymentCents += row.PaymentCents
		s.InterestCents += row.InterestCents
		s.PrincipalCents += row.PrincipalCents
		s.InsuranceCents += row.InsuranceCents
		s.RemainingCents = row.RemainingCents
	}
	var want []YearRow
	for _, y := range order {
		want = append(want, *sums[y])
	}
	if !reflect.DeepEqual(res.Yearly, want) {
		t.Errorf("yearly rows differ from the monthly sums:\n got %+v\nwant %+v", res.Yearly, want)
	}
}

func TestYearlyIsEmptyWithoutAFirstPaymentMonth(t *testing.T) {
	res, _ := Compute(Params{PrincipalCents: 100_000, AnnualRatePct: 3, DurationMonths: 12})
	if len(res.Yearly) != 0 {
		t.Errorf("yearly = %+v", res.Yearly)
	}
}

// randomParams draws inputs across the whole valid range of SPEC-loan.md.
func randomParams(r *rand.Rand) Params {
	return Params{
		PrincipalCents:      100_000 + r.Int64N(500_000_000-100_000+1),
		AnnualRatePct:       float64(r.IntN(15_001)) / 1000,
		DurationMonths:      12 * (1 + r.IntN(30)),
		InsuranceRatePct:    float64(r.IntN(2_001)) / 1000,
		ApplicationFeeCents: r.Int64N(5_000_001),
		GuaranteeFeeCents:   r.Int64N(10_000_001),
		FirstPaymentMonth:   Month{2000 + r.IntN(70), 1 + r.IntN(12)},
	}
}

func TestPropertiesOverGeneratedLoans(t *testing.T) {
	r := rand.New(rand.NewPCG(2026, 11))
	for i := range 2_000 {
		p := randomParams(r)
		res, err := Compute(p)
		if err != nil {
			t.Fatalf("case %d %+v: %v", i, p, err)
		}
		if len(res.Schedule) != p.DurationMonths {
			t.Fatalf("case %d %+v: %d installments, want %d", i, p, len(res.Schedule), p.DurationMonths)
		}
		assertInvariants(t, p, res)
		assertYearlyMatchesMonths(t, res)
		if res.TotalCostCents != res.TotalInterestCents+res.TotalInsuranceCents+res.TotalFeesCents {
			t.Fatalf("case %d: total cost inconsistent", i)
		}
		if t.Failed() {
			t.Fatalf("case %d failed with %+v", i, p)
		}
	}
}

func TestPaymentIsMonotonicInRateAndAmount(t *testing.T) {
	r := rand.New(rand.NewPCG(7, 7))
	for range 2_000 {
		p := randomParams(r)
		base, _ := MonthlyPayment(p.PrincipalCents, p.AnnualRatePct, p.DurationMonths)

		higherRate, _ := MonthlyPayment(p.PrincipalCents, p.AnnualRatePct+0.125, p.DurationMonths)
		if higherRate < base {
			t.Fatalf("payment dropped when the rate rose: %+v (%d → %d)", p, base, higherRate)
		}
		higherAmount, _ := MonthlyPayment(p.PrincipalCents+100_000, p.AnnualRatePct, p.DurationMonths)
		if higherAmount < base {
			t.Fatalf("payment dropped when the amount rose: %+v (%d → %d)", p, base, higherAmount)
		}
	}
}
