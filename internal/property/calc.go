package property

import (
	"math"

	"immo-calc/internal/loan"
)

// Regime is the tax regime of a property's rental income.
type Regime string

const (
	MicroFoncier Regime = "micro_foncier" // unfurnished, 30 % allowance
	ReelFoncier  Regime = "reel_foncier"  // unfurnished, actual charges
	MicroBIC     Regime = "micro_bic"     // furnished long-term, 50 % allowance
	LMNPReel     Regime = "lmnp_reel"     // furnished, actual charges and depreciation
)

// Allowances and ceilings of the micro regimes, checked 2026-11-13 (see
// tasks/legal-rates.md).
const (
	// Micro-foncier: 30 % allowance, up to 15 000 € of gross rent a year
	// (service-public F1991).
	microFoncierTaxableShare = 0.70
	MicroFoncierCeilingCents = 1_500_000
	microBICTaxableShare     = 0.50

	// LMNP au réel depreciation (common practice, docs/tax-regimes.md): the
	// building without its land share over 30 years, furniture over 7.
	lmnpBuildingShare    = 0.85
	lmnpBuildingYears    = 30
	lmnpFurnitureYears   = 7
	MicroBICCeilingCents = 8_360_000 // revenus 2026; 77 700 € for revenus 2025 (service-public F32744)
)

// Params are the inputs of a property evaluation (amounts in cents).
type Params struct {
	PriceCents                     int64
	MonthlyRentCents               int64 // L, excluding charges
	MonthlyRecoverableChargesCents int64 // C, passed on to the tenant
	AnnualPropertyTaxCents         int64 // TF
	AnnualCondoFeesCents           int64 // CP
	AnnualLandlordInsuranceCents   int64 // PNO
	AgencyManaged                  bool
	TaxRegime                      Regime
	DPE                            string
	// StartMonth starts the profitability timeline of a bien without a loan
	// (the month after today); zero means no timeline without a loan.
	StartMonth             loan.Month
	FurnitureIncludedCents int64 // furniture listed in the deed, part of the price
	FurnitureAddedCents    int64 // furniture bought on top of the price
}

// LoanFigures is what the evaluation needs from a linked loan.
type LoanFigures struct {
	PrincipalCents    int64
	FeesCents         int64 // application + guarantee fees
	MonthlyTotalCents int64 // monthly payment, insurance included
	// YearlyInterest and YearlyInsurance hold the interest and the borrower
	// insurance of each block of 12 installments (year 1 first).
	YearlyInterest    []int64
	YearlyInsurance   []int64
	FirstPaymentMonth loan.Month
	Payments          []int64 // each installment, insurance included
	TotalCostCents    int64   // coût total du crédit: interest + insurance + fees
	DurationMonths    int
}

// yearCosts are the loan costs deductible under the réel regimes in one year
// of the loan.
type yearCosts struct{ interest, insurance, fees int64 }

// costsOfYear returns the deductible loan costs of year (0 = the first year of
// the loan). Application and guarantee fees are deducted in year 1 only. A nil
// receiver (no loan) and the years after the loan cost nothing.
func (lf *LoanFigures) costsOfYear(year int) yearCosts {
	if lf == nil {
		return yearCosts{}
	}
	var c yearCosts
	if year < len(lf.YearlyInterest) {
		c.interest = lf.YearlyInterest[year]
	}
	if year < len(lf.YearlyInsurance) {
		c.insurance = lf.YearlyInsurance[year]
	}
	if year == 0 {
		c.fees = lf.FeesCents
	}
	return c
}

// Warning is shown above a property's results.
type Warning struct {
	Code     string `json:"code"`     // "dpe", "micro_ceiling"
	Severity string `json:"severity"` // "critical", "serious", "warning"
	Message  string `json:"message"`  // French
}

// Result is everything derived from Params and Settings. Amounts are cents,
// yields are percentages rounded to 2 decimals.
type Result struct {
	AcquisitionCostCents int64 `json:"acquisitionCostCents"` // A
	CollectedRentCents   int64 `json:"collectedRentCents"`   // E
	// ReimbursedChargesCents is C × (12 − v), the recoverable charges the tenant
	// pays back. They are receipts of the micro regimes (neutral under réel).
	ReimbursedChargesCents  int64 `json:"reimbursedChargesCents"`
	AgencyFeesCents         int64 `json:"agencyFeesCents"`         // E × g%
	VacancyChargesCents     int64 `json:"vacancyChargesCents"`     // C × v
	OwnerChargesCents       int64 `json:"ownerChargesCents"`       // CH
	IncomeBeforeTaxCents    int64 `json:"incomeBeforeTaxCents"`    // R
	TaxableIncomeCents      int64 `json:"taxableIncomeCents"`      // floored at 0
	AnnualTaxCents          int64 `json:"annualTaxCents"`          // I
	NetAnnualIncomeCents    int64 `json:"netAnnualIncomeCents"`    // RN
	MonthlyGrossIncomeCents int64 `json:"monthlyGrossIncomeCents"` // L
	MonthlyNetIncomeCents   int64 `json:"monthlyNetIncomeCents"`   // RN / 12

	GrossYieldPct       float64 `json:"grossYieldPct"`
	NetYieldPct         float64 `json:"netYieldPct"`
	NetAfterTaxYieldPct float64 `json:"netAfterTaxYieldPct"`
	// NetAfterCreditYieldPct is (RN − average annual credit cost) / A, the
	// headline KPI (rule 15b); equal to NetAfterTaxYieldPct without a loan.
	NetAfterCreditYieldPct float64 `json:"netAfterCreditYieldPct"`

	// LMNP au réel depreciation (0 under the other regimes).
	BuildingDepreciationCents  int64 `json:"buildingDepreciationCents"`
	FurnitureDepreciationCents int64 `json:"furnitureDepreciationCents"`
	// Regimes compares the four regimes on this bien (« Régime fiscal »).
	Regimes []RegimeComparison `json:"regimes"`
	// YearlyTax is the tax and the net income of each year (year 1 first), up to
	// a few years after the figures stop changing (« Impôt au fil des années »).
	YearlyTax []YearTax `json:"yearlyTax"`
	// TaxDetail explains how AnnualTaxCents is produced, step by step.
	TaxDetail TaxDetail `json:"taxDetail"`

	Financing *Financing `json:"financing"` // with a linked loan only
	Timeline  *Timeline  `json:"timeline"`  // profitability dates, with a dated loan only
	Warnings  []Warning  `json:"warnings"`
}

// YearTax is one year of the « Impôt au fil des années » chart.
type YearTax struct {
	Year                 int   `json:"year"` // 1 = the first year of the loan, or of the activity
	AnnualTaxCents       int64 `json:"annualTaxCents"`
	NetAnnualIncomeCents int64 `json:"netAnnualIncomeCents"` // R − tax
}

// TaxDetail is the breakdown of the annual tax (« Comment est calculé
// l'impôt »): every figure the taxable income and the tax are built from, so
// a human can redo the calculation. IncomeTax + SocialCharges = the annual tax.
type TaxDetail struct {
	Regime             Regime  `json:"regime"`
	MarginalTaxRatePct float64 `json:"marginalTaxRatePct"` // TMI
	SocialChargesPct   float64 `json:"socialChargesPct"`
	// Year-1 borrower interest and insurance, deducted under the réel regimes
	// only (0 under the micro regimes and without a loan).
	LoanInterestCents  int64 `json:"loanInterestCents"`
	LoanInsuranceCents int64 `json:"loanInsuranceCents"`
	LoanFeesCents      int64 `json:"loanFeesCents"`  // application + guarantee fees
	LoanCostsCents     int64 `json:"loanCostsCents"` // interest + insurance + fees
	// ReimbursedChargesCents is added to the rent under the micro regimes.
	ReimbursedChargesCents int64 `json:"reimbursedChargesCents"`
	// RawTaxableCents is the base before the floor at 0 (negative = deficit).
	RawTaxableCents    int64 `json:"rawTaxableCents"`
	IncomeTaxCents     int64 `json:"incomeTaxCents"`     // IR = taxable × TMI
	SocialChargesCents int64 `json:"socialChargesCents"` // PS = I − IR
}

func roundCents(v float64) int64 { return int64(math.Round(v)) }

func roundPct(v float64) float64 { return math.Round(v*100) / 100 }

// pct returns part / whole as a percentage, 0 when whole is 0.
func pct(part, whole int64) float64 {
	if whole == 0 {
		return 0
	}
	return roundPct(float64(part) / float64(whole) * 100)
}

// socialChargesPct is the social charges rate of a regime.
func socialChargesPct(r Regime, s Settings) float64 {
	if r == MicroBIC || r == LMNPReel {
		return s.SocialChargesFurnishedPct
	}
	return s.SocialChargesUnfurnishedPct
}

// Evaluate applies the SPEC-property.md calculation rules. Amounts are
// rounded to the cent as they are computed, so the figures shown add up
// (R = E − CH, RN = R − I). lf (the loan's figures) is nil without a loan.
func Evaluate(p Params, s Settings, lf *LoanFigures) Result {
	var r Result
	// Furniture listed in the deed is outside the notary-fee base; furniture
	// bought on top is paid with own funds.
	r.AcquisitionCostCents = p.PriceCents + notaryFees(p, s) + p.FurnitureAddedCents
	r.CollectedRentCents = roundCents(float64(p.MonthlyRentCents) * (12 - s.VacancyMonthsPerYear))
	r.ReimbursedChargesCents = roundCents(float64(p.MonthlyRecoverableChargesCents) * (12 - s.VacancyMonthsPerYear))
	if p.AgencyManaged {
		r.AgencyFeesCents = roundCents(float64(r.CollectedRentCents) * s.AgencyFeePct / 100)
	}
	// During vacancy, the owner pays the recoverable charges no tenant reimburses.
	r.VacancyChargesCents = roundCents(float64(p.MonthlyRecoverableChargesCents) * s.VacancyMonthsPerYear)
	r.OwnerChargesCents = p.AnnualPropertyTaxCents + p.AnnualCondoFeesCents + p.AnnualLandlordInsuranceCents +
		r.AgencyFeesCents + r.VacancyChargesCents
	r.IncomeBeforeTaxCents = r.CollectedRentCents - r.OwnerChargesCents

	net := yearlyNet(p, s, r, lf)
	r.YearlyTax = yearlyTax(r.IncomeBeforeTaxCents, net)
	t := taxFor(p.TaxRegime, p, s, r, lf, 0)
	r.TaxableIncomeCents, r.AnnualTaxCents = t.taxable, t.tax
	r.BuildingDepreciationCents, r.FurnitureDepreciationCents = t.building, t.furniture
	r.TaxDetail = taxDetail(p.TaxRegime, s, r, t)
	r.NetAnnualIncomeCents = r.IncomeBeforeTaxCents - r.AnnualTaxCents
	r.MonthlyGrossIncomeCents = p.MonthlyRentCents
	r.MonthlyNetIncomeCents = roundCents(float64(r.NetAnnualIncomeCents) / 12)

	r.GrossYieldPct = pct(12*p.MonthlyRentCents, r.AcquisitionCostCents)
	r.NetYieldPct = pct(r.IncomeBeforeTaxCents, r.AcquisitionCostCents)
	r.NetAfterTaxYieldPct = pct(r.NetAnnualIncomeCents, r.AcquisitionCostCents)

	r.NetAfterCreditYieldPct = r.NetAfterTaxYieldPct
	if lf != nil {
		r.Financing = financing(r, *lf)
		r.NetAfterCreditYieldPct = pct(r.NetAnnualIncomeCents-r.Financing.AnnualCreditCostCents, r.AcquisitionCostCents)
		if !lf.FirstPaymentMonth.IsZero() && len(lf.Payments) > 0 {
			r.Timeline = timeline(net, r.Financing.OwnContributionCents,
				lf.Payments, lf.FirstPaymentMonth)
		}
	} else if !p.StartMonth.IsZero() {
		// Without a loan everything is paid with own funds: apport = A.
		r.Timeline = timeline(net, r.AcquisitionCostCents, nil, p.StartMonth)
	}

	r.Regimes = compareRegimes(p, s, r, lf)

	r.Warnings = []Warning{}
	if w := DPEWarning(p.DPE); w != nil {
		r.Warnings = append(r.Warnings, *w)
	}
	return r
}

// DPEWarning returns the rental-ban warning of a DPE class (loi Climat et
// résilience), or nil for A–D. Dates checked on ecologie.gouv.fr and
// service-public F2042 (2026-11-13). Informational: no effect on the maths.
func DPEWarning(dpe string) *Warning {
	switch dpe {
	case "G":
		return &Warning{Code: "dpe", Severity: "critical", Message: "Logement G : interdit à la location depuis le 1er janvier 2025"}
	case "F":
		return &Warning{Code: "dpe", Severity: "serious", Message: "Logement F : interdit à la location à partir du 1er janvier 2028 ; loyer gelé"}
	case "E":
		return &Warning{Code: "dpe", Severity: "warning", Message: "Logement E : interdit à la location à partir du 1er janvier 2034"}
	}
	return nil
}

// notaryFees applies the notary rate to the price excluding furniture.
func notaryFees(p Params, s Settings) int64 {
	return roundCents(float64(p.PriceCents-p.FurnitureIncludedCents) * s.NotaryFeePct / 100)
}

// taxOutcome is a regime's taxable income and tax for one bien and one year.
type taxOutcome struct {
	taxable, tax        int64
	building, furniture int64 // LMNP au réel depreciation
	rawTaxable          int64 // before the floor at 0
	costs               yearCosts
}

// taxDetail builds the breakdown of a regime's tax outcome.
func taxDetail(regime Regime, s Settings, r Result, t taxOutcome) TaxDetail {
	ir := roundCents(float64(t.taxable) * s.MarginalTaxRatePct / 100)
	d := TaxDetail{
		Regime: regime, MarginalTaxRatePct: s.MarginalTaxRatePct, SocialChargesPct: socialChargesPct(regime, s),
		LoanInterestCents: t.costs.interest, LoanInsuranceCents: t.costs.insurance, LoanFeesCents: t.costs.fees,
		LoanCostsCents:  t.costs.interest + t.costs.insurance + t.costs.fees,
		RawTaxableCents: t.rawTaxable, IncomeTaxCents: ir, SocialChargesCents: t.tax - ir,
	}
	if isMicro(regime) {
		d.ReimbursedChargesCents = r.ReimbursedChargesCents
	}
	return d
}

func isMicro(r Regime) bool { return r == MicroFoncier || r == MicroBIC }

// microReceipts are the gross receipts a micro regime taxes: the rent
// collected plus the charges the tenant reimburses (the allowance replaces
// every deduction).
func microReceipts(r Result) int64 { return r.CollectedRentCents + r.ReimbursedChargesCents }

// taxFor applies a regime's rule (SPEC-property.md rule 5) to an evaluated
// bien (E, CH and R already computed in r) for one year (0 = the first year
// of the loan, or of the activity without a loan).
func taxFor(regime Regime, p Params, s Settings, r Result, lf *LoanFigures, year int) taxOutcome {
	var t taxOutcome
	switch regime {
	case MicroBIC:
		t.taxable = roundCents(float64(microReceipts(r)) * microBICTaxableShare)
	case ReelFoncier:
		// Interest, borrower insurance and loan fees are deductible; a deficit is
		// floored at 0 below, not carried over.
		t.costs = lf.costsOfYear(year)
		t.taxable = r.IncomeBeforeTaxCents - t.costs.interest - t.costs.insurance - t.costs.fees
	case LMNPReel:
		t.costs = lf.costsOfYear(year)
		// Depreciation stops when its period is over.
		if year < lmnpBuildingYears {
			t.building = roundCents((float64(p.PriceCents-p.FurnitureIncludedCents)*lmnpBuildingShare +
				float64(notaryFees(p, s))) / lmnpBuildingYears)
		}
		if year < lmnpFurnitureYears {
			t.furniture = roundCents(float64(p.FurnitureIncludedCents+p.FurnitureAddedCents) / lmnpFurnitureYears)
		}
		// Depreciation is used only up to the result: it never creates a deficit.
		t.taxable = r.IncomeBeforeTaxCents - t.costs.interest - t.costs.insurance - t.costs.fees - t.building - t.furniture
	default:
		t.taxable = roundCents(float64(microReceipts(r)) * microFoncierTaxableShare)
	}
	t.rawTaxable = t.taxable
	t.taxable = max(t.taxable, 0)
	t.tax = roundCents(float64(t.taxable) * (s.MarginalTaxRatePct + socialChargesPct(regime, s)) / 100)
	return t
}

// yearlyTax lists the tax and net income of each year, keeping the years up
// to three after the last change (at least 10), so the chart shows the
// plateau without 50 identical years.
func yearlyTax(income int64, net []int64) []YearTax {
	last := 0
	for y := 1; y < len(net); y++ {
		if net[y] != net[y-1] {
			last = y
		}
	}
	n := min(len(net), max(10, last+4))
	out := make([]YearTax, n)
	for y := range out {
		out[y] = YearTax{Year: y + 1, AnnualTaxCents: income - net[y], NetAnnualIncomeCents: net[y]}
	}
	return out
}

// yearlyNet is the net annual income of each year of the timeline (50 years):
// the tax changes with the loan's interest and with the end of the LMNP
// depreciation. Rents and charges are not indexed.
func yearlyNet(p Params, s Settings, r Result, lf *LoanFigures) []int64 {
	out := make([]int64, horizonMonths/12)
	for y := range out {
		out[y] = r.IncomeBeforeTaxCents - taxFor(p.TaxRegime, p, s, r, lf, y).tax
	}
	return out
}

// RegimeComparison is one regime applied to the bien (« Régime fiscal »).
type RegimeComparison struct {
	Regime               Regime  `json:"regime"`
	Furnished            bool    `json:"furnished"` // location meublée (micro-BIC, LMNP) or nue
	TaxableIncomeCents   int64   `json:"taxableIncomeCents"`
	AnnualTaxCents       int64   `json:"annualTaxCents"`
	NetAnnualIncomeCents int64   `json:"netAnnualIncomeCents"`
	NetAfterTaxYieldPct  float64 `json:"netAfterTaxYieldPct"`
	Current              bool    `json:"current"` // the bien's own regime
	// Eligible is false when the bien's receipts are above the regime's micro ceiling.
	Eligible bool `json:"eligible"`
	// Best marks the lowest tax among the eligible regimes of the same kind of
	// rental (nue or meublée; first one on a tie): the two kinds are not
	// interchangeable, so they are not ranked against each other.
	Best bool `json:"best"`
}

var allRegimes = []Regime{MicroFoncier, ReelFoncier, MicroBIC, LMNPReel}

func isFurnished(r Regime) bool { return r == MicroBIC || r == LMNPReel }

// microEligible reports whether the bien's receipts fit the regime's ceiling.
func microEligible(regime Regime, r Result) bool {
	switch regime {
	case MicroFoncier:
		return microReceipts(r) <= MicroFoncierCeilingCents
	case MicroBIC:
		return microReceipts(r) <= MicroBICCeilingCents
	}
	return true
}

func compareRegimes(p Params, s Settings, r Result, lf *LoanFigures) []RegimeComparison {
	out := make([]RegimeComparison, len(allRegimes))
	best := map[bool]int{false: -1, true: -1} // by furnished
	for i, regime := range allRegimes {
		t := taxFor(regime, p, s, r, lf, 0)
		net := r.IncomeBeforeTaxCents - t.tax
		out[i] = RegimeComparison{
			Regime: regime, Furnished: isFurnished(regime), TaxableIncomeCents: t.taxable, AnnualTaxCents: t.tax,
			NetAnnualIncomeCents: net, NetAfterTaxYieldPct: pct(net, r.AcquisitionCostCents),
			Current: regime == p.TaxRegime, Eligible: microEligible(regime, r),
		}
		if !out[i].Eligible {
			continue
		}
		if b := best[out[i].Furnished]; b < 0 || t.tax < out[b].AnnualTaxCents {
			best[out[i].Furnished] = i
		}
	}
	for _, b := range best {
		if b >= 0 {
			out[b].Best = true
		}
	}
	return out
}
