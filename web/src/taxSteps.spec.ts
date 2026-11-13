import { describe, expect, it } from 'vitest'
import type { PropertyResult, TaxRegime } from '@/api/properties'
import { taxSections, type TaxStepsInput } from '@/taxSteps'

const plain = (s: string) => s.replace(/[  ]/g, ' ')

// Reference flat of SPEC-property.md: 150 000 €, 700 €/month, TF 900 €,
// copropriété 1 200 €, PNO 150 €, 1 month of vacancy, TMI 30 %.
function input(regime: TaxRegime, patch: Partial<PropertyResult> = {}): TaxStepsInput {
  const micro = regime === 'micro_foncier' || regime === 'micro_bic'
  const share = regime === 'micro_bic' ? 0.5 : 0.7
  const taxable = micro ? 825_000 * share : 540_000 // rent 7 700 € + 550 € of reimbursed charges
  const ps = regime === 'micro_bic' || regime === 'lmnp_reel' ? 18.6 : 17.2
  const ir = Math.round((taxable * 30) / 100)
  const tax = Math.round((taxable * (30 + ps)) / 100)
  return {
    regime,
    monthlyRentCents: 70_000,
    propertyTaxCents: 90_000,
    condoFeesCents: 120_000,
    insuranceCents: 15_000,
    priceCents: 15_000_000,
    furnitureIncludedCents: 0,
    furnitureAddedCents: 0,
    agencyManaged: false,
    result: {
      acquisitionCostCents: 16_275_000,
      collectedRentCents: 770_000,
      reimbursedChargesCents: 55_000,
      agencyFeesCents: 0,
      vacancyChargesCents: 5_000,
      ownerChargesCents: 230_000,
      incomeBeforeTaxCents: 540_000,
      taxableIncomeCents: taxable,
      annualTaxCents: tax,
      netAnnualIncomeCents: 540_000 - tax,
      taxDetail: {
        regime,
        marginalTaxRatePct: 30,
        socialChargesPct: ps,
        loanInterestCents: 0,
        loanInsuranceCents: 0,
        loanFeesCents: 0,
        loanCostsCents: 0,
        reimbursedChargesCents: micro ? 55_000 : 0,
        rawTaxableCents: taxable,
        incomeTaxCents: ir,
        socialChargesCents: tax - ir,
      },
      ...patch,
    } as PropertyResult,
  }
}

const lines = (s: ReturnType<typeof taxSections>[number]) =>
  s.lines.map((l) => [l.op, plain(l.label), l.cents])

describe('taxSections', () => {
  it('rebuilds the income before tax from the rent and the charges', () => {
    const [income] = taxSections(input('micro_foncier'))
    expect(lines(income)).toEqual([
      ['', 'Loyer annuel théorique (12 × 700,00 €)', 840_000],
      ['−', 'Vacance locative', 70_000],
      ['=', 'Loyers encaissés', 770_000],
      ['−', 'Taxe foncière', 90_000],
      ['−', 'Charges de copropriété non récupérables', 120_000],
      ['−', 'Assurance PNO', 15_000],
      ['−', 'Charges non récupérées pendant la vacance', 5_000],
      ['=', 'Revenu annuel avant impôts', 540_000],
    ])
  })

  it('lists the agency fees only when the bien is agency-managed', () => {
    const withAgency = input('micro_foncier', {
      agencyFeesCents: 53_900,
      incomeBeforeTaxCents: 486_100,
    })
    withAgency.agencyManaged = true
    const [income] = taxSections(withAgency)
    expect(income.lines.map((l) => l.label)).toContain('Frais de gestion locative')
    const [without] = taxSections(input('micro_foncier'))
    expect(without.lines.map((l) => l.label)).not.toContain('Frais de gestion locative')
  })

  it('micro-foncier: a 30 % allowance on the rent and the reimbursed charges, real charges not deducted', () => {
    const taxable = taxSections(input('micro_foncier'))[1]
    expect(lines(taxable)).toEqual([
      ['', 'Loyers encaissés', 770_000],
      ['+', 'Charges remboursées par le locataire', 55_000],
      ['−', 'Abattement forfaitaire de 30 %', 247_500],
      ['=', 'Revenu imposable', 577_500],
    ])
    expect(plain(taxable.note ?? '')).toContain('charges réelles ne sont pas déduites')
  })

  it('micro-BIC: a 50 % allowance', () => {
    const taxable = taxSections(input('micro_bic'))[1]
    expect(lines(taxable)).toEqual([
      ['', 'Loyers encaissés', 770_000],
      ['+', 'Charges remboursées par le locataire', 55_000],
      ['−', 'Abattement forfaitaire de 50 %', 412_500],
      ['=', 'Revenu imposable', 412_500],
    ])
  })

  it('réel foncier: deducts the first-year loan interest and insurance', () => {
    const i = input('reel_foncier')
    i.result.taxDetail = {
      ...i.result.taxDetail!,
      loanInterestCents: 100_000,
      loanInsuranceCents: 20_000,
      loanFeesCents: 30_000,
      loanCostsCents: 150_000,
      rawTaxableCents: 390_000,
    }
    i.result.taxableIncomeCents = 390_000
    const taxable = taxSections(i)[1]
    expect(lines(taxable)).toEqual([
      ['', 'Revenu annuel avant impôts', 540_000],
      ['−', "Intérêts de l'emprunt (1re année)", 100_000],
      ['−', "Assurance de l'emprunt (1re année)", 20_000],
      ['−', "Frais de dossier et de garantie de l'emprunt (1re année)", 30_000],
      ['=', 'Revenu imposable', 390_000],
    ])
  })

  it('réel: a deficit is brought back to 0 and says so', () => {
    const i = input('reel_foncier')
    i.result.taxDetail = { ...i.result.taxDetail!, rawTaxableCents: -50_000 }
    i.result.taxableIncomeCents = 0
    const taxable = taxSections(i)[1]
    expect(plain(taxable.note ?? '')).toContain('déficit')
    expect(taxable.lines.at(-1)).toMatchObject({ op: '=', label: 'Revenu imposable', cents: 0 })
  })

  it('LMNP au réel: shows both depreciations with their formula', () => {
    const i = input('lmnp_reel')
    i.furnitureIncludedCents = 500_000
    i.furnitureAddedCents = 140_000
    i.result.buildingDepreciationCents = 442_833
    i.result.furnitureDepreciationCents = 91_429
    i.result.acquisitionCostCents = 15_000_000 + 1_232_500 + 140_000 // notary on price − furniture
    i.result.taxDetail = { ...i.result.taxDetail!, rawTaxableCents: 5_738 }
    i.result.taxableIncomeCents = 5_738
    const taxable = taxSections(i)[1]
    const building = taxable.lines.find((l) => l.label === 'Amortissement du bâti')!
    const furniture = taxable.lines.find((l) => l.label === 'Amortissement du mobilier')!
    expect(building).toMatchObject({ op: '−', cents: 442_833 })
    expect(plain(building.detail ?? '')).toBe(
      '((150 000,00 € − 5 000,00 €) × 85 % + 12 325,00 € de frais de notaire) ÷ 30 ans',
    )
    expect(furniture).toMatchObject({ op: '−', cents: 91_429 })
    expect(plain(furniture.detail ?? '')).toBe('(5 000,00 € + 1 400,00 €) ÷ 7 ans')
  })

  it('splits the tax into IR and prélèvements sociaux, then the net income', () => {
    const sections = taxSections(input('micro_foncier'))
    const tax = sections[2]
    expect(lines(tax)).toEqual([
      ['', 'Impôt sur le revenu (IR)', 173_250],
      ['+', 'Prélèvements sociaux (PS)', 99_330],
      ['=', 'Impôt annuel', 272_580],
    ])
    expect(plain(tax.lines[0].detail ?? '')).toBe('5 775,00 € × 30 %')
    expect(plain(tax.lines[1].detail ?? '')).toBe('5 775,00 € × 17,2 %')
    const net = sections[3]
    expect(lines(net)).toEqual([
      ['', 'Revenu annuel avant impôts', 540_000],
      ['−', 'Impôt annuel', 272_580],
      ['=', 'Revenu annuel net', 267_420],
    ])
  })
})
