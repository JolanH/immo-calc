package property

import "immo-calc/internal/loan"

// horizonMonths bounds the timeline: past 50 years « au-delà de 50 ans ».
const horizonMonths = 600

// TimelinePoint is one month of the « Trésorerie cumulée » chart.
type TimelinePoint struct {
	Month           string `json:"month"`           // "YYYY-MM"
	CashFlowCents   int64  `json:"cashFlowCents"`   // that month's cash-flow after tax
	CumulativeCents int64  `json:"cumulativeCents"` // −apport + cash-flows so far
}

// Timeline gives the profitability dates (SPEC-property.md rules 16–20).
type Timeline struct {
	StartMonth            string          `json:"startMonth"`
	PositiveCashFlowMonth string          `json:"positiveCashFlowMonth"` // "" = never
	PaybackMonth          string          `json:"paybackMonth"`          // "" = not within 50 years
	PaybackAtStart        bool            `json:"paybackAtStart"`        // no positive apport
	Points                []TimelinePoint `json:"points"`
}

// timeline walks month by month from start (the loan's first installment,
// or the month after today without a loan): the monthly net income of that
// month's year (yearlyNet: the tax changes as the interest falls) minus that
// month's installment while payments last. The points run a year past the
// last date found (or to the horizon).
func timeline(yearlyNet []int64, apport int64, payments []int64, start loan.Month) *Timeline {
	t := &Timeline{StartMonth: start.String(), PaybackAtStart: apport <= 0}
	if t.PaybackAtStart {
		t.PaybackMonth = t.StartMonth
	}
	cumulative := -apport
	last := horizonMonths
	for k := 1; k <= last && k <= horizonMonths; k++ {
		cashFlow := roundCents(float64(yearlyNet[(k-1)/12]) / 12)
		if k <= len(payments) {
			cashFlow -= payments[k-1]
		}
		cumulative += cashFlow
		month := start.Add(k - 1).String()
		t.Points = append(t.Points, TimelinePoint{Month: month, CashFlowCents: cashFlow, CumulativeCents: cumulative})
		if t.PositiveCashFlowMonth == "" && cashFlow > 0 {
			t.PositiveCashFlowMonth = month
		}
		if t.PaybackMonth == "" && cumulative >= 0 {
			t.PaybackMonth = month
		}
		// Once both dates are known (and the loan is over), show one more year.
		if last == horizonMonths && t.PositiveCashFlowMonth != "" && t.PaybackMonth != "" && k >= len(payments) {
			last = k + 12
		}
	}
	return t
}

// paymentsOf lists each installment's total (principal + interest + insurance).
func paymentsOf(res loan.Result) []int64 {
	out := make([]int64, len(res.Schedule))
	for i, inst := range res.Schedule {
		out[i] = inst.PaymentCents + inst.InsuranceCents
	}
	return out
}
