// Package loan holds the maths of a French fixed-rate amortizing mortgage
// (prêt amortissable à mensualités constantes) and the saved simulations.
//
// Amounts are integer euro cents, rates are percentages (3.5 means 3.5 %),
// durations are months. Intermediate maths use float64; every amount is
// rounded to the cent, half away from zero.
package loan

import (
	"fmt"
	"math"
)

// Params are the inputs of a loan simulation.
type Params struct {
	PrincipalCents      int64
	AnnualRatePct       float64 // nominal annual rate
	DurationMonths      int
	InsuranceRatePct    float64 // borrower insurance, % of the initial principal per year
	ApplicationFeeCents int64   // frais de dossier
	GuaranteeFeeCents   int64   // frais de garantie (caution or hypothèque)
	FirstPaymentMonth   Month   // month of installment 1; zero means undated
}

// Installment is one row of the amortization schedule. PaymentCents is
// principal + interest, insurance excluded.
type Installment struct {
	Index          int
	Month          Month // zero when the loan has no first payment month
	PaymentCents   int64
	InterestCents  int64
	PrincipalCents int64
	InsuranceCents int64
	RemainingCents int64
}

// Result is everything derived from Params.
type Result struct {
	MonthlyPaymentCents   int64 // mensualité hors assurance
	MonthlyInsuranceCents int64
	MonthlyTotalCents     int64 // mensualité totale
	TotalInterestCents    int64
	TotalInsuranceCents   int64
	TotalFeesCents        int64
	TotalCostCents        int64 // coût total du crédit: interest + insurance + fees
	TotalRepaidCents      int64 // principal + interest + insurance
	Schedule              []Installment
	Yearly                []YearRow // by calendar year; empty for an undated loan
}

// YearRow sums the installments of one calendar year. The first and last
// years can be partial (Installments < 12).
type YearRow struct {
	Year           int
	Installments   int
	PaymentCents   int64
	InterestCents  int64
	PrincipalCents int64
	InsuranceCents int64
	RemainingCents int64 // remaining principal after the year's last installment
}

// roundCents rounds to the nearest cent, half away from zero.
func roundCents(v float64) int64 {
	return int64(math.Round(v))
}

func monthlyRate(annualRatePct float64) float64 {
	return annualRatePct / 100 / 12
}

// MonthlyPayment returns the constant monthly installment (in cents) for a
// fully amortizing loan. annualRatePct is a percentage, e.g. 3.5 for 3.5 %.
func MonthlyPayment(principalCents int64, annualRatePct float64, months int) (int64, error) {
	if principalCents <= 0 || months <= 0 || annualRatePct < 0 {
		return 0, fmt.Errorf("monthly payment: invalid input (principal=%d, rate=%v, months=%d)",
			principalCents, annualRatePct, months)
	}
	if annualRatePct == 0 {
		return roundCents(float64(principalCents) / float64(months)), nil
	}
	r := monthlyRate(annualRatePct)
	p := float64(principalCents) * r / (1 - math.Pow(1+r, -float64(months)))
	return roundCents(p), nil
}

// Compute builds the schedule and the totals. Interest is charged each month
// on the remaining principal; the last installment absorbs rounding so the
// remaining principal ends at exactly 0.
func Compute(p Params) (Result, error) {
	if p.InsuranceRatePct < 0 || p.ApplicationFeeCents < 0 || p.GuaranteeFeeCents < 0 {
		return Result{}, fmt.Errorf("compute loan: negative insurance or fees (%+v)", p)
	}
	payment, err := MonthlyPayment(p.PrincipalCents, p.AnnualRatePct, p.DurationMonths)
	if err != nil {
		return Result{}, fmt.Errorf("compute loan: %w", err)
	}

	r := monthlyRate(p.AnnualRatePct)
	insurance := roundCents(float64(p.PrincipalCents) * p.InsuranceRatePct / 100 / 12)
	res := Result{
		MonthlyPaymentCents:   payment,
		MonthlyInsuranceCents: insurance,
		MonthlyTotalCents:     payment + insurance,
		TotalFeesCents:        p.ApplicationFeeCents + p.GuaranteeFeeCents,
		Schedule:              make([]Installment, 0, p.DurationMonths),
	}

	remaining := p.PrincipalCents
	for k := 1; k <= p.DurationMonths; k++ {
		interest := roundCents(float64(remaining) * r)
		principal := payment - interest
		if k == p.DurationMonths || principal > remaining {
			principal = remaining // last installment: repay exactly what's left
		}
		remaining -= principal
		var month Month
		if !p.FirstPaymentMonth.IsZero() {
			month = p.FirstPaymentMonth.Add(k - 1)
		}
		res.Schedule = append(res.Schedule, Installment{
			Index:          k,
			Month:          month,
			PaymentCents:   principal + interest,
			InterestCents:  interest,
			PrincipalCents: principal,
			InsuranceCents: insurance,
			RemainingCents: remaining,
		})
		res.TotalInterestCents += interest
		res.TotalInsuranceCents += insurance
		if remaining == 0 {
			break
		}
	}

	if !p.FirstPaymentMonth.IsZero() {
		res.Yearly = yearly(res.Schedule)
	}
	res.TotalCostCents = res.TotalInterestCents + res.TotalInsuranceCents + res.TotalFeesCents
	res.TotalRepaidCents = p.PrincipalCents + res.TotalInterestCents + res.TotalInsuranceCents
	return res, nil
}

// yearly groups dated installments by calendar year.
func yearly(schedule []Installment) []YearRow {
	var rows []YearRow
	for _, inst := range schedule {
		if len(rows) == 0 || rows[len(rows)-1].Year != inst.Month.Year {
			rows = append(rows, YearRow{Year: inst.Month.Year})
		}
		row := &rows[len(rows)-1]
		row.Installments++
		row.PaymentCents += inst.PaymentCents
		row.InterestCents += inst.InterestCents
		row.PrincipalCents += inst.PrincipalCents
		row.InsuranceCents += inst.InsuranceCents
		row.RemainingCents = inst.RemainingCents
	}
	return rows
}
