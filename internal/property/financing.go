package property

import (
	"fmt"

	"immo-calc/internal/loan"
)

// Financing is the « Financement » card of a property with a linked loan.
type Financing struct {
	MonthlyPaymentCents    int64 `json:"monthlyPaymentCents"`    // the loan's monthly total, insurance included
	CashFlowBeforeTaxCents int64 `json:"cashFlowBeforeTaxCents"` // R/12 − mensualité
	CashFlowAfterTaxCents  int64 `json:"cashFlowAfterTaxCents"`  // RN/12 − mensualité
	OwnContributionCents   int64 `json:"ownContributionCents"`   // apport: A + fees − principal
	// ReturnOnContributionPct is 12 × cash-flow after tax / apport, or nil
	// (shown « n/a ») when there is no positive own contribution.
	ReturnOnContributionPct *float64 `json:"returnOnContributionPct"`
}

// FiguresOf computes what the evaluation needs from a loan.
func FiguresOf(p loan.Params) (LoanFigures, error) {
	res, err := loan.Compute(p)
	if err != nil {
		return LoanFigures{}, fmt.Errorf("loan figures: %w", err)
	}
	f := LoanFigures{
		PrincipalCents:    p.PrincipalCents,
		FeesCents:         res.TotalFeesCents,
		MonthlyTotalCents: res.MonthlyTotalCents,
	}
	for _, inst := range res.Schedule[:min(12, len(res.Schedule))] {
		f.FirstYearInterest += inst.InterestCents
		f.FirstYearInsurance += inst.InsuranceCents
	}
	return f, nil
}

// financing computes the « Financement » card from an evaluated result.
func financing(r Result, l LoanFigures) *Financing {
	f := &Financing{
		MonthlyPaymentCents:    l.MonthlyTotalCents,
		CashFlowBeforeTaxCents: roundCents(float64(r.IncomeBeforeTaxCents)/12) - l.MonthlyTotalCents,
		CashFlowAfterTaxCents:  r.MonthlyNetIncomeCents - l.MonthlyTotalCents,
		OwnContributionCents:   r.AcquisitionCostCents + l.FeesCents - l.PrincipalCents,
	}
	if f.OwnContributionCents > 0 {
		v := pct(12*f.CashFlowAfterTaxCents, f.OwnContributionCents)
		f.ReturnOnContributionPct = &v
	}
	return f
}
