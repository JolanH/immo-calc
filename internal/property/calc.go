package property

import "math"

// Regime is the tax regime of a property's rental income.
type Regime string

const (
	MicroFoncier Regime = "micro_foncier" // unfurnished, 30 % allowance
	ReelFoncier  Regime = "reel_foncier"  // unfurnished, actual charges
	MicroBIC     Regime = "micro_bic"     // furnished long-term, 50 % allowance
)

// Allowances and ceilings of the micro regimes, checked 2026-11-13 (see
// tasks/legal-rates.md).
const (
	// Micro-foncier: 30 % allowance, up to 15 000 € of gross rent a year
	// (service-public F1991).
	microFoncierTaxableShare = 0.70
	MicroFoncierCeilingCents = 1_500_000
	microBICTaxableShare     = 0.50
	MicroBICCeilingCents     = 8_360_000 // revenus 2026; 77 700 € for revenus 2025 (service-public F32744)
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
}

// LoanFigures is what the evaluation needs from a linked loan.
type LoanFigures struct {
	PrincipalCents     int64
	FeesCents          int64 // application + guarantee fees
	MonthlyTotalCents  int64 // monthly payment, insurance included
	FirstYearInterest  int64 // interest of the first 12 installments
	FirstYearInsurance int64 // borrower insurance of the first 12 installments
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

	Financing *Financing `json:"financing"` // with a linked loan only
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
	if r == MicroBIC {
		return s.SocialChargesFurnishedPct
	}
	return s.SocialChargesUnfurnishedPct
}

// Evaluate applies the SPEC-property.md calculation rules. Amounts are
// rounded to the cent as they are computed, so the figures shown add up
// (R = E − CH, RN = R − I). loan is nil without a linked loan.
func Evaluate(p Params, s Settings, loan *LoanFigures) Result {
	var r Result
	r.AcquisitionCostCents = roundCents(float64(p.PriceCents) * (1 + s.NotaryFeePct/100))
	r.CollectedRentCents = roundCents(float64(p.MonthlyRentCents) * (12 - s.VacancyMonthsPerYear))
	if p.AgencyManaged {
		r.AgencyFeesCents = roundCents(float64(r.CollectedRentCents) * s.AgencyFeePct / 100)
	}
	// During vacancy, the owner pays the recoverable charges no tenant reimburses.
	r.VacancyChargesCents = roundCents(float64(p.MonthlyRecoverableChargesCents) * s.VacancyMonthsPerYear)
	r.OwnerChargesCents = p.AnnualPropertyTaxCents + p.AnnualCondoFeesCents + p.AnnualLandlordInsuranceCents +
		r.AgencyFeesCents + r.VacancyChargesCents
	r.IncomeBeforeTaxCents = r.CollectedRentCents - r.OwnerChargesCents

	switch p.TaxRegime {
	case MicroBIC:
		r.TaxableIncomeCents = roundCents(float64(r.CollectedRentCents) * microBICTaxableShare)
	case ReelFoncier:
		// Year-1 interest and borrower insurance of the linked loan are
		// deductible; a deficit is floored at 0 below, not carried over.
		r.TaxableIncomeCents = r.IncomeBeforeTaxCents
		if loan != nil {
			r.TaxableIncomeCents -= loan.FirstYearInterest + loan.FirstYearInsurance
		}
	default:
		r.TaxableIncomeCents = roundCents(float64(r.CollectedRentCents) * microFoncierTaxableShare)
	}
	r.TaxableIncomeCents = max(r.TaxableIncomeCents, 0)

	rate := s.MarginalTaxRatePct + socialChargesPct(p.TaxRegime, s)
	r.AnnualTaxCents = roundCents(float64(r.TaxableIncomeCents) * rate / 100)
	r.NetAnnualIncomeCents = r.IncomeBeforeTaxCents - r.AnnualTaxCents
	r.MonthlyGrossIncomeCents = p.MonthlyRentCents
	r.MonthlyNetIncomeCents = roundCents(float64(r.NetAnnualIncomeCents) / 12)

	r.GrossYieldPct = pct(12*p.MonthlyRentCents, r.AcquisitionCostCents)
	r.NetYieldPct = pct(r.IncomeBeforeTaxCents, r.AcquisitionCostCents)
	r.NetAfterTaxYieldPct = pct(r.NetAnnualIncomeCents, r.AcquisitionCostCents)

	if loan != nil {
		r.Financing = financing(r, *loan)
	}

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
