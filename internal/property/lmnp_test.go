package property

import (
	"strings"
	"testing"
)

// LMNP au réel reference table of SPEC-property.md (« SE appart 1 », no loan).
func TestLMNPReelReferenceTable(t *testing.T) {
	tests := []struct {
		name                    string
		included, added         int64
		acquisition             int64
		building, furniture     int64
		taxable, tax, netAnnual int64
		afterTaxYield           float64
	}{
		{"no furniture", 0, 0, 3_689_000, 105_967, 0, 150_033, 72_916, 183_084, 4.96},
		{"3 000 € included in the price", 300_000, 0, 3_663_500, 96_617, 42_857, 116_526, 56_632, 199_368, 5.44},
		{"2 000 € bought on top", 0, 200_000, 3_889_000, 105_967, 28_571, 121_462, 59_031, 196_969, 5.06},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := seAppart()
			p.TaxRegime = LMNPReel
			p.FurnitureIncludedCents = tt.included
			p.FurnitureAddedCents = tt.added

			r := Evaluate(p, referenceSettings, nil)

			got := []int64{r.AcquisitionCostCents, r.BuildingDepreciationCents, r.FurnitureDepreciationCents,
				r.TaxableIncomeCents, r.AnnualTaxCents, r.NetAnnualIncomeCents}
			want := []int64{tt.acquisition, tt.building, tt.furniture, tt.taxable, tt.tax, tt.netAnnual}
			for i := range got {
				if got[i] != want[i] {
					t.Errorf("A, bâti, mobilier, imposable, impôt, net = %v, want %v", got, want)
					break
				}
			}
			if r.NetAfterTaxYieldPct != tt.afterTaxYield {
				t.Errorf("rentabilité nette après impôts = %v, want %v", r.NetAfterTaxYieldPct, tt.afterTaxYield)
			}
		})
	}
}

func TestFurnitureInThePriceIsOutOfTheNotaryBaseForEveryRegime(t *testing.T) {
	p := seAppart()
	p.FurnitureIncludedCents = 300_000

	r := Evaluate(p, referenceSettings, nil) // micro-foncier

	// 34 000 + 8,5 % × 31 000.
	if r.AcquisitionCostCents != 3_663_500 || r.BuildingDepreciationCents != 0 {
		t.Errorf("A = %d, depreciation = %d", r.AcquisitionCostCents, r.BuildingDepreciationCents)
	}
}

func TestLMNPDeductsTheLoanBeforeDepreciationAndNeverCreatesADeficit(t *testing.T) {
	p := seAppart()
	p.TaxRegime = LMNPReel
	withLoan := evaluateWithLoan(t, p, consoImmo)
	// R 2 560 € − interest ≈ 678 € leaves ≈ 1 882 €, above the 1 059,67 € of depreciation.
	if withLoan.TaxableIncomeCents <= 0 || withLoan.TaxableIncomeCents >= 150_033 {
		t.Errorf("taxable with the loan = %d", withLoan.TaxableIncomeCents)
	}

	p.FurnitureAddedCents = 10_000_000 // 14 285,71 €/year of furniture depreciation
	capped := Evaluate(p, referenceSettings, nil)
	if capped.TaxableIncomeCents != 0 || capped.AnnualTaxCents != 0 || capped.NetAnnualIncomeCents != capped.IncomeBeforeTaxCents {
		t.Errorf("capped = taxable %d, tax %d", capped.TaxableIncomeCents, capped.AnnualTaxCents)
	}
}

func TestLMNPUsesFurnishedSocialCharges(t *testing.T) {
	s := referenceSettings
	s.SocialChargesFurnishedPct = 20
	s.SocialChargesUnfurnishedPct = 10
	p := seAppart()
	p.TaxRegime = LMNPReel

	r := Evaluate(p, s, nil)

	if r.AnnualTaxCents != roundCents(float64(150_033)*0.50) {
		t.Errorf("tax = %d, want 50 %% of 1 500,33 €", r.AnnualTaxCents)
	}
}

func TestRegimeComparison(t *testing.T) {
	r := Evaluate(seAppart(), referenceSettings, nil) // micro-foncier

	if len(r.Regimes) != 4 {
		t.Fatalf("regimes = %+v", r.Regimes)
	}
	order := []Regime{MicroFoncier, ReelFoncier, MicroBIC, LMNPReel}
	taxes := map[Regime]int64{}
	for i, c := range r.Regimes {
		if c.Regime != order[i] {
			t.Errorf("regimes[%d] = %s, want %s", i, c.Regime, order[i])
		}
		taxes[c.Regime] = c.AnnualTaxCents
		if c.Current != (c.Regime == MicroFoncier) {
			t.Errorf("%s: current = %v", c.Regime, c.Current)
		}
		if c.Best != (c.Regime == LMNPReel) {
			t.Errorf("%s: best = %v", c.Regime, c.Best)
		}
	}
	// The bien's own regime matches the main result.
	if r.Regimes[0].AnnualTaxCents != r.AnnualTaxCents || r.Regimes[0].NetAnnualIncomeCents != r.NetAnnualIncomeCents {
		t.Errorf("micro-foncier entry = %+v, result tax %d", r.Regimes[0], r.AnnualTaxCents)
	}
	// Values from docs/tax-regimes.md.
	if taxes[MicroFoncier] != 130_838 || taxes[ReelFoncier] != 120_832 || taxes[MicroBIC] != 96_228 || taxes[LMNPReel] != 72_916 {
		t.Errorf("taxes = %v", taxes)
	}
	if r.Regimes[3].NetAfterTaxYieldPct != 4.96 {
		t.Errorf("LMNP yield = %v", r.Regimes[3].NetAfterTaxYieldPct)
	}
}

func TestRegimeComparisonUsesTheLoan(t *testing.T) {
	r := evaluateWithLoan(t, seAppart(), consoImmo)

	for _, c := range r.Regimes {
		if c.Regime == ReelFoncier && c.TaxableIncomeCents >= 256_000 {
			t.Errorf("réel ignores the loan interest: %+v", c)
		}
	}
}

func TestFurnitureValidation(t *testing.T) {
	in := validPropertyInput()
	in.FurnitureIncludedCents = in.PriceCents + 1
	in.FurnitureAddedCents = 10_000_001
	in.TaxRegime = LMNPReel

	errs := ValidateInput(in, nil)

	if !strings.Contains(errs["furnitureIncludedCents"], "ne peut pas dépasser le prix") {
		t.Errorf("included: %v", errs)
	}
	if !strings.Contains(errs["furnitureAddedCents"], "entre 0 et 100 000 €") {
		t.Errorf("added: %v", errs)
	}
	if _, bad := errs["taxRegime"]; bad {
		t.Errorf("lmnp_reel refused: %v", errs)
	}
}

func TestFurnitureIsSavedAndOldFilesStillLoad(t *testing.T) {
	f := newFixture(t)
	in := validPropertyInput()
	in.TaxRegime = LMNPReel
	in.FurnitureIncludedCents = 300_000
	p, err := f.svc.Create(in)
	if err != nil {
		t.Fatal(err)
	}

	got, _ := reopen(t, f.path).svc.Get(p.ID)
	if got.FurnitureIncludedCents != 300_000 || got.TaxRegime != LMNPReel {
		t.Errorf("after restart = %+v", got.Input)
	}

	// A property written before the furniture fields existed.
	path := writeData(t, `schema_version: 1
properties:
  - id: 3a7e0c91d24b6f58
    name: T2
    price_cents: 15000000
    dpe: D
    monthly_rent_cents: 70000
    monthly_recoverable_charges_cents: 0
    annual_property_tax_cents: 90000
    annual_condo_fees_cents: 0
    annual_landlord_insurance_cents: 0
    agency_managed: false
    tax_regime: micro_foncier
    created_at: 2026-11-12T20:41:00+01:00
    updated_at: 2026-11-12T20:41:00+01:00
`)
	if _, err := reopen(t, path).svc.Get("3a7e0c91d24b6f58"); err != nil {
		t.Errorf("old file: %v", err)
	}
}
