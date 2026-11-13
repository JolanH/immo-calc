package property

import (
	"testing"
)

// referenceFlat is the reference example of SPEC-property.md: Prix 150 000 €,
// Loyer 700 €, Charges 50 €, TF 900 €, CP 1 200 €, PNO 150 €, no loan.
func referenceFlat(regime Regime, agency bool) Params {
	return Params{
		PriceCents:                     15_000_000,
		MonthlyRentCents:               70_000,
		MonthlyRecoverableChargesCents: 5_000,
		AnnualPropertyTaxCents:         90_000,
		AnnualCondoFeesCents:           120_000,
		AnnualLandlordInsuranceCents:   15_000,
		AgencyManaged:                  agency,
		TaxRegime:                      regime,
		DPE:                            "D",
	}
}

// Settings of the reference example: TMI 30 %, PS 17,2 % / 18,6 %, notaire
// 8,5 %, vacance 1 mois, gestion 7 % (the defaults).
var referenceSettings = DefaultSettings()

func TestReferenceCommonFigures(t *testing.T) {
	res := Evaluate(referenceFlat(MicroFoncier, false), referenceSettings, nil)

	if res.AcquisitionCostCents != 16_275_000 {
		t.Errorf("A = %d, want 162 750 €", res.AcquisitionCostCents)
	}
	if res.CollectedRentCents != 770_000 {
		t.Errorf("E = %d, want 7 700 €", res.CollectedRentCents)
	}
	if res.GrossYieldPct != 5.16 {
		t.Errorf("rentabilité brute = %v, want 5,16", res.GrossYieldPct)
	}
	if res.MonthlyGrossIncomeCents != 70_000 {
		t.Errorf("revenus mensuels bruts = %d", res.MonthlyGrossIncomeCents)
	}
}

// Both reference tables of SPEC-property.md, three regimes each.
func TestReferenceTables(t *testing.T) {
	tests := []struct {
		name          string
		regime        Regime
		agency        bool
		ownerCharges  int64
		income        int64 // R
		netYield      float64
		taxable       int64
		tax           int64
		netAnnual     int64
		netMonthly    int64
		afterTaxYield float64
	}{
		{"micro-foncier", MicroFoncier, false, 230_000, 540_000, 3.32, 539_000, 254_408, 285_592, 23_799, 1.75},
		{"réel", ReelFoncier, false, 230_000, 540_000, 3.32, 540_000, 254_880, 285_120, 23_760, 1.75},
		{"micro-BIC", MicroBIC, false, 230_000, 540_000, 3.32, 385_000, 187_110, 352_890, 29_408, 2.17},
		{"micro-foncier with agency", MicroFoncier, true, 283_900, 486_100, 2.99, 539_000, 254_408, 231_692, 19_308, 1.42},
		{"réel with agency", ReelFoncier, true, 283_900, 486_100, 2.99, 486_100, 229_439, 256_661, 21_388, 1.58},
		{"micro-BIC with agency", MicroBIC, true, 283_900, 486_100, 2.99, 385_000, 187_110, 298_990, 24_916, 1.84},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := Evaluate(referenceFlat(tt.regime, tt.agency), referenceSettings, nil)

			check := func(label string, got, want int64) {
				t.Helper()
				if got != want {
					t.Errorf("%s = %d, want %d", label, got, want)
				}
			}
			check("charges du propriétaire", res.OwnerChargesCents, tt.ownerCharges)
			check("revenu avant impôts", res.IncomeBeforeTaxCents, tt.income)
			check("revenu imposable", res.TaxableIncomeCents, tt.taxable)
			check("impôt annuel", res.AnnualTaxCents, tt.tax)
			check("revenu annuel net", res.NetAnnualIncomeCents, tt.netAnnual)
			check("revenus mensuels après impôts", res.MonthlyNetIncomeCents, tt.netMonthly)
			if res.NetYieldPct != tt.netYield {
				t.Errorf("rentabilité nette = %v, want %v", res.NetYieldPct, tt.netYield)
			}
			if res.NetAfterTaxYieldPct != tt.afterTaxYield {
				t.Errorf("rentabilité nette après impôts = %v, want %v", res.NetAfterTaxYieldPct, tt.afterTaxYield)
			}
		})
	}
}

func TestAgencyFeesAreOnCollectedRent(t *testing.T) {
	with := Evaluate(referenceFlat(MicroFoncier, true), referenceSettings, nil)
	without := Evaluate(referenceFlat(MicroFoncier, false), referenceSettings, nil)

	if with.AgencyFeesCents != 53_900 || without.AgencyFeesCents != 0 {
		t.Errorf("agency fees = %d / %d, want 539 € / 0", with.AgencyFeesCents, without.AgencyFeesCents)
	}
}

func TestVacancy(t *testing.T) {
	noVacancy := referenceSettings
	noVacancy.VacancyMonthsPerYear = 0
	full := Evaluate(referenceFlat(MicroFoncier, false), noVacancy, nil)
	if full.CollectedRentCents != 840_000 || full.VacancyChargesCents != 0 {
		t.Errorf("no vacancy: E = %d, vacancy charges = %d", full.CollectedRentCents, full.VacancyChargesCents)
	}

	allYear := referenceSettings
	allYear.VacancyMonthsPerYear = 12
	empty := Evaluate(referenceFlat(MicroFoncier, true), allYear, nil)
	if empty.CollectedRentCents != 0 || empty.AgencyFeesCents != 0 || empty.VacancyChargesCents != 60_000 {
		t.Errorf("12 months: %+v", empty)
	}
	// Owner pays everything and earns nothing: negative income, no tax.
	if empty.IncomeBeforeTaxCents != -(90_000+120_000+15_000+60_000) || empty.TaxableIncomeCents != 0 || empty.AnnualTaxCents != 0 {
		t.Errorf("12 months: R = %d, taxable = %d, tax = %d", empty.IncomeBeforeTaxCents, empty.TaxableIncomeCents, empty.AnnualTaxCents)
	}
	if empty.NetAnnualIncomeCents != empty.IncomeBeforeTaxCents {
		t.Errorf("RN = %d, want R", empty.NetAnnualIncomeCents)
	}

	half := referenceSettings
	half.VacancyMonthsPerYear = 1.5
	if got := Evaluate(referenceFlat(MicroFoncier, false), half, nil).CollectedRentCents; got != 735_000 {
		t.Errorf("1.5 months: E = %d, want 7 350 €", got)
	}
}

func TestReelTaxableIncomeIsFlooredAtZero(t *testing.T) {
	p := referenceFlat(ReelFoncier, false)
	p.AnnualCondoFeesCents = 2_000_000 // charges above the rent

	res := Evaluate(p, referenceSettings, nil)

	if res.IncomeBeforeTaxCents >= 0 || res.TaxableIncomeCents != 0 || res.AnnualTaxCents != 0 {
		t.Errorf("R = %d, taxable = %d, tax = %d", res.IncomeBeforeTaxCents, res.TaxableIncomeCents, res.AnnualTaxCents)
	}
}

func TestSocialChargesFollowTheRegime(t *testing.T) {
	s := referenceSettings
	s.SocialChargesUnfurnishedPct = 10
	s.SocialChargesFurnishedPct = 20

	foncier := Evaluate(referenceFlat(MicroFoncier, false), s, nil)
	bic := Evaluate(referenceFlat(MicroBIC, false), s, nil)

	if foncier.AnnualTaxCents != 539_000*40/100 {
		t.Errorf("micro-foncier tax = %d, want 40 %% of 5 390 €", foncier.AnnualTaxCents)
	}
	if bic.AnnualTaxCents != 385_000*50/100 {
		t.Errorf("micro-BIC tax = %d, want 50 %% of 3 850 €", bic.AnnualTaxCents)
	}
}

func TestNotaryRateChangesTheAcquisitionCostAndYields(t *testing.T) {
	s := referenceSettings
	s.NotaryFeePct = 0

	res := Evaluate(referenceFlat(MicroFoncier, false), s, nil)

	if res.AcquisitionCostCents != 15_000_000 || res.GrossYieldPct != 5.6 {
		t.Errorf("A = %d, brute = %v", res.AcquisitionCostCents, res.GrossYieldPct)
	}
}

func TestYieldsAreZeroWithoutAnAcquisitionCost(t *testing.T) {
	p := referenceFlat(MicroFoncier, false)
	p.PriceCents = 0

	res := Evaluate(p, referenceSettings, nil)

	if res.GrossYieldPct != 0 || res.NetYieldPct != 0 || res.NetAfterTaxYieldPct != 0 {
		t.Errorf("yields = %v / %v / %v", res.GrossYieldPct, res.NetYieldPct, res.NetAfterTaxYieldPct)
	}
}

func TestDPEWarnings(t *testing.T) {
	tests := []struct {
		dpe      string
		severity string
		message  string
	}{
		{"G", "critical", "Logement G : interdit à la location depuis le 1er janvier 2025"},
		{"F", "serious", "Logement F : interdit à la location à partir du 1er janvier 2028 ; loyer gelé"},
		{"E", "warning", "Logement E : interdit à la location à partir du 1er janvier 2034"},
	}
	for _, tt := range tests {
		w := DPEWarning(tt.dpe)
		if w == nil || w.Severity != tt.severity || w.Message != tt.message || w.Code != "dpe" {
			t.Errorf("DPEWarning(%s) = %+v", tt.dpe, w)
		}
	}
	for _, dpe := range []string{"A", "B", "C", "D"} {
		if w := DPEWarning(dpe); w != nil {
			t.Errorf("DPEWarning(%s) = %+v, want none", dpe, w)
		}
	}
}

func TestEvaluateIncludesTheDPEWarning(t *testing.T) {
	p := referenceFlat(MicroFoncier, false)
	p.DPE = "G"

	res := Evaluate(p, referenceSettings, nil)

	if len(res.Warnings) != 1 || res.Warnings[0].Severity != "critical" {
		t.Errorf("warnings = %+v", res.Warnings)
	}
	if got := Evaluate(referenceFlat(MicroFoncier, false), referenceSettings, nil).Warnings; len(got) != 0 {
		t.Errorf("DPE D warnings = %+v", got)
	}
}

// The tax breakdown shown under « Comment est calculé l'impôt » adds up to
// the annual tax: IR + PS = I.
func TestTaxDetailSplitsIncomeTaxAndSocialCharges(t *testing.T) {
	res := Evaluate(referenceFlat(MicroFoncier, false), referenceSettings, nil)
	d := res.TaxDetail

	if d.MarginalTaxRatePct != 30 || d.SocialChargesPct != 17.2 {
		t.Errorf("rates = %v / %v, want 30 / 17,2", d.MarginalTaxRatePct, d.SocialChargesPct)
	}
	if d.IncomeTaxCents != 161_700 { // 5 390 € × 30 %
		t.Errorf("IR = %d, want 1 617 €", d.IncomeTaxCents)
	}
	if d.IncomeTaxCents+d.SocialChargesCents != res.AnnualTaxCents {
		t.Errorf("IR %d + PS %d != I %d", d.IncomeTaxCents, d.SocialChargesCents, res.AnnualTaxCents)
	}
	if d.LoanCostsCents != 0 {
		t.Errorf("loan costs = %d without a loan", d.LoanCostsCents)
	}
}

func TestTaxDetailReportsDeductedLoanCostsUnderTheReelRegimes(t *testing.T) {
	lf := &LoanFigures{FirstYearInterest: 300_000, FirstYearInsurance: 20_000}
	for _, tt := range []struct {
		regime Regime
		want   int64
		ps     float64
	}{{ReelFoncier, 320_000, 17.2}, {LMNPReel, 320_000, 18.6}, {MicroFoncier, 0, 17.2}, {MicroBIC, 0, 18.6}} {
		d := Evaluate(referenceFlat(tt.regime, false), referenceSettings, lf).TaxDetail
		if d.LoanCostsCents != tt.want || d.SocialChargesPct != tt.ps {
			t.Errorf("%s: loan costs = %d, PS = %v; want %d, %v", tt.regime, d.LoanCostsCents, d.SocialChargesPct, tt.want, tt.ps)
		}
	}
}
