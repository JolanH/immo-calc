import type { PropertyResult, TaxRegime } from '@/api/properties'
import { formatEuros } from '@/format'

// « Comment est calculé l'impôt »: the annual tax rebuilt from the bien's own
// figures, step by step, so a human can check every line (SPEC-property.md
// rule 5, docs/tax-regimes.md). The amounts come from the result (computed by
// the server); only the presentation and the formulas shown live here.

export interface TaxLine {
  /** '−', '+' or '=' before the amount; '' for a starting figure. */
  op: '' | '−' | '+' | '='
  label: string
  /** The formula with the actual figures, when there is one. */
  detail?: string
  cents: number
}

export interface TaxSection {
  title: string
  lines: TaxLine[]
  note?: string
}

export interface TaxStepsInput {
  regime: TaxRegime
  result: PropertyResult
  monthlyRentCents: number
  propertyTaxCents: number
  condoFeesCents: number
  insuranceCents: number
  priceCents: number
  furnitureIncludedCents: number
  furnitureAddedCents: number
  agencyManaged: boolean
}

const MICRO_ALLOWANCE_PCT: Partial<Record<TaxRegime, number>> = {
  micro_foncier: 30,
  micro_bic: 50,
}

/** 30 → "30 %", 17.2 → "17,2 %". */
function rate(pct: number): string {
  return `${new Intl.NumberFormat('fr-FR', { maximumFractionDigits: 2 }).format(pct)} %`
}

export function taxSections(i: TaxStepsInput): TaxSection[] {
  const r = i.result
  const d = r.taxDetail
  if (!d) return []

  const income: TaxLine[] = [
    {
      op: '',
      label: `Loyer annuel théorique (12 × ${formatEuros(i.monthlyRentCents)})`,
      cents: 12 * i.monthlyRentCents,
    },
    { op: '−', label: 'Vacance locative', cents: 12 * i.monthlyRentCents - r.collectedRentCents },
    { op: '=', label: 'Loyers encaissés', cents: r.collectedRentCents },
    { op: '−', label: 'Taxe foncière', cents: i.propertyTaxCents },
    { op: '−', label: 'Charges de copropriété non récupérables', cents: i.condoFeesCents },
    { op: '−', label: 'Assurance PNO', cents: i.insuranceCents },
  ]
  if (i.agencyManaged) {
    income.push({ op: '−', label: 'Frais de gestion locative', cents: r.agencyFeesCents })
  }
  income.push(
    { op: '−', label: 'Charges non récupérées pendant la vacance', cents: r.vacancyChargesCents },
    { op: '=', label: 'Revenu annuel avant impôts', cents: r.incomeBeforeTaxCents },
  )

  const taxable = taxableSection(i, d)

  const base = formatEuros(r.taxableIncomeCents)
  const tax: TaxLine[] = [
    {
      op: '',
      label: 'Impôt sur le revenu (IR)',
      detail: `${base} × ${rate(d.marginalTaxRatePct)}`,
      cents: d.incomeTaxCents,
    },
    {
      op: '+',
      label: 'Prélèvements sociaux (PS)',
      detail: `${base} × ${rate(d.socialChargesPct)}`,
      cents: d.socialChargesCents,
    },
    { op: '=', label: 'Impôt annuel', cents: r.annualTaxCents },
  ]

  return [
    { title: '1. Revenu annuel avant impôts', lines: income },
    taxable,
    {
      title: '3. Impôt : IR + prélèvements sociaux',
      lines: tax,
      note: 'Le taux marginal (TMI) et les prélèvements sociaux se règlent dans Paramètres.',
    },
    {
      title: '4. Revenu annuel net',
      lines: [
        { op: '', label: 'Revenu annuel avant impôts', cents: r.incomeBeforeTaxCents },
        { op: '−', label: 'Impôt annuel', cents: r.annualTaxCents },
        { op: '=', label: 'Revenu annuel net', cents: r.netAnnualIncomeCents },
      ],
    },
  ]
}

function taxableSection(i: TaxStepsInput, d: NonNullable<PropertyResult['taxDetail']>): TaxSection {
  const r = i.result
  const title = '2. Revenu imposable'
  const result: TaxLine = { op: '=', label: 'Revenu imposable', cents: r.taxableIncomeCents }

  const allowance = MICRO_ALLOWANCE_PCT[i.regime]
  if (allowance !== undefined) {
    return {
      title,
      lines: [
        { op: '', label: 'Loyers encaissés', cents: r.collectedRentCents },
        {
          op: '+',
          label: 'Charges remboursées par le locataire',
          cents: d.reimbursedChargesCents,
        },
        {
          op: '−',
          label: `Abattement forfaitaire de ${allowance} %`,
          cents: r.collectedRentCents + d.reimbursedChargesCents - r.taxableIncomeCents,
        },
        result,
      ],
      note: `Avec le micro, les recettes comprennent les charges que le locataire rembourse, et les charges réelles ne sont pas déduites : l'abattement de ${allowance} % les remplace.`,
    }
  }

  const lines: TaxLine[] = [
    { op: '', label: 'Revenu annuel avant impôts', cents: r.incomeBeforeTaxCents },
  ]
  const notes: string[] = []
  if (d.loanCostsCents > 0) {
    lines.push(
      { op: '−', label: "Intérêts de l'emprunt (1re année)", cents: d.loanInterestCents },
      { op: '−', label: "Assurance de l'emprunt (1re année)", cents: d.loanInsuranceCents },
    )
    if (d.loanFeesCents > 0) {
      lines.push({
        op: '−',
        label: "Frais de dossier et de garantie de l'emprunt (1re année)",
        cents: d.loanFeesCents,
      })
    }
  } else {
    notes.push("Aucun prêt lié : pas d'intérêts ni d'assurance d'emprunt à déduire.")
  }

  if (i.regime === 'lmnp_reel') {
    const notary = r.acquisitionCostCents - i.priceCents - i.furnitureAddedCents
    lines.push(
      {
        op: '−',
        label: 'Amortissement du bâti',
        detail: `((${formatEuros(i.priceCents)} − ${formatEuros(i.furnitureIncludedCents)}) × 85 % + ${formatEuros(notary)} de frais de notaire) ÷ 30 ans`,
        cents: r.buildingDepreciationCents ?? 0,
      },
      {
        op: '−',
        label: 'Amortissement du mobilier',
        detail: `(${formatEuros(i.furnitureIncludedCents)} + ${formatEuros(i.furnitureAddedCents)}) ÷ 7 ans`,
        cents: r.furnitureDepreciationCents ?? 0,
      },
    )
    notes.push('Le terrain (15 % du bâti) ne s’amortit pas.')
  }

  if (d.rawTaxableCents < 0) {
    notes.push(
      `Le résultat est un déficit (${formatEuros(d.rawTaxableCents)}) : il n'est pas reporté, le revenu imposable est ramené à 0.`,
    )
  }
  lines.push(result)
  return { title, lines, note: notes.length ? notes.join(' ') : undefined }
}
