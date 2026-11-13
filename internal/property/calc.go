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
	PrincipalCents     int64
	FeesCents          int64 // application + guarantee fees
	MonthlyTotalCents  int64 // monthly payment, insurance included
	FirstYearInterest  int64 // interest of the first 12 installments
	FirstYearInsurance int64 // borrower insurance of the first 12 installments
	FirstPaymentMonth  loan.Month
	Payments           []int64 // each installment, insurance included
	TotalCostCents     int64   // coût total du crédit: interest + insurance + fees
	DurationMonths     int
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
	AcquisitionCostCents    int64 `json:"acquisitionCostCents"`    // A
	CollectedRentCents      int64 `json:"collectedRentCents"`      // E
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

	Financing *Financing `json:"financing"` // with a linked loan only
	Timeline  *Timeline  `json:"timeline"`  // profitability dates, with a dated loan only
	Warnings  []Warning  `json:"warnings"`
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
	if p.AgencyManaged {
		r.AgencyFeesCents = roundCents(float64(r.CollectedRentCents) * s.AgencyFeePct / 100)
	}
	// During vacancy, the owner pays the recoverable charges no tenant reimburses.
	r.VacancyChargesCents = roundCents(float64(p.MonthlyRecoverableChargesCents) * s.VacancyMonthsPerYear)
	r.OwnerChargesCents = p.AnnualPropertyTaxCents + p.AnnualCondoFeesCents + p.AnnualLandlordInsuranceCents +
		r.AgencyFeesCents + r.VacancyChargesCents
	r.IncomeBeforeTaxCents = r.CollectedRentCents - r.OwnerChargesCents

	t := taxFor(p.TaxRegime, p, s, r, lf)
	r.TaxableIncomeCents, r.AnnualTaxCents = t.taxable, t.tax
	r.BuildingDepreciationCents, r.FurnitureDepreciationCents = t.building, t.furniture
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
			unfinanced := p
			unfinanced.StartMonth = loan.Month{} // only its monthly income is needed
			after := Evaluate(unfinanced, s, nil).MonthlyNetIncomeCents
			r.Timeline = timeline(r.MonthlyNetIncomeCents, after, r.Financing.OwnContributionCents,
				lf.Payments, lf.FirstPaymentMonth)
		}
	} else if !p.StartMonth.IsZero() {
		// Without a loan everything is paid with own funds: apport = A.
		r.Timeline = timeline(r.MonthlyNetIncomeCents, r.MonthlyNetIncomeCents, r.AcquisitionCostCents,
			nil, p.StartMonth)
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

// taxOutcome is a regime's taxable income and tax for one bien.
type taxOutcome struct {
	taxable, tax        int64
	building, furniture int64 // LMNP au réel depreciation
}

// taxFor applies a regime's rule (SPEC-property.md rule 5) to an evaluated
// bien (E, CH and R already computed in r).
func taxFor(regime Regime, p Params, s Settings, r Result, lf *LoanFigures) taxOutcome {
	var t taxOutcome
	loanCosts := int64(0)
	if lf != nil {
		loanCosts = lf.FirstYearInterest + lf.FirstYearInsurance
	}
	switch regime {
	case MicroBIC:
		t.taxable = roundCents(float64(r.CollectedRentCents) * microBICTaxableShare)
	case ReelFoncier:
		// Year-1 interest and borrower insurance are deductible; a deficit is
		// floored at 0 below, not carried over.
		t.taxable = r.IncomeBeforeTaxCents - loanCosts
	case LMNPReel:
		t.building = roundCents((float64(p.PriceCents-p.FurnitureIncludedCents)*lmnpBuildingShare +
			float64(notaryFees(p, s))) / lmnpBuildingYears)
		t.furniture = roundCents(float64(p.FurnitureIncludedCents+p.FurnitureAddedCents) / lmnpFurnitureYears)
		// Depreciation is used only up to the result: it never creates a deficit.
		t.taxable = r.IncomeBeforeTaxCents - loanCosts - t.building - t.furniture
	default:
		t.taxable = roundCents(float64(r.CollectedRentCents) * microFoncierTaxableShare)
	}
	t.taxable = max(t.taxable, 0)
	t.tax = roundCents(float64(t.taxable) * (s.MarginalTaxRatePct + socialChargesPct(regime, s)) / 100)
	return t
}

// RegimeComparison is one regime applied to the bien (« Régime fiscal »).
type RegimeComparison struct {
	Regime               Regime  `json:"regime"`
	TaxableIncomeCents   int64   `json:"taxableIncomeCents"`
	AnnualTaxCents       int64   `json:"annualTaxCents"`
	NetAnnualIncomeCents int64   `json:"netAnnualIncomeCents"`
	NetAfterTaxYieldPct  float64 `json:"netAfterTaxYieldPct"`
	Current              bool    `json:"current"` // the bien's own regime
	Best                 bool    `json:"best"`    // lowest tax (first one on a tie)
}

var allRegimes = []Regime{MicroFoncier, ReelFoncier, MicroBIC, LMNPReel}

func compareRegimes(p Params, s Settings, r Result, lf *LoanFigures) []RegimeComparison {
	out := make([]RegimeComparison, len(allRegimes))
	best := 0
	for i, regime := range allRegimes {
		t := taxFor(regime, p, s, r, lf)
		net := r.IncomeBeforeTaxCents - t.tax
		out[i] = RegimeComparison{
			Regime: regime, TaxableIncomeCents: t.taxable, AnnualTaxCents: t.tax, NetAnnualIncomeCents: net,
			NetAfterTaxYieldPct: pct(net, r.AcquisitionCostCents), Current: regime == p.TaxRegime,
		}
		if t.tax < out[best].AnnualTaxCents {
			best = i
		}
	}
	out[best].Best = true
	return out
}
