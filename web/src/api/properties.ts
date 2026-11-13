import { request } from './client'

export type TaxRegime = 'micro_foncier' | 'reel_foncier' | 'micro_bic'

/** A property as entered (and stored). Amounts in cents. */
export interface PropertyInput {
  name: string
  priceCents: number
  dpe: string
  monthlyRentCents: number
  monthlyRecoverableChargesCents: number
  annualPropertyTaxCents: number
  annualCondoFeesCents: number
  annualLandlordInsuranceCents: number
  agencyManaged: boolean
  taxRegime: TaxRegime
  loanId: string // '' = no linked loan
}

export interface Financing {
  monthlyPaymentCents: number
  cashFlowBeforeTaxCents: number
  cashFlowAfterTaxCents: number
  ownContributionCents: number
  returnOnContributionPct: number | null // null = n/a
}

export interface PropertyWarning {
  code: string
  severity: 'critical' | 'serious' | 'warning'
  message: string
}

export interface PropertyResult {
  acquisitionCostCents: number
  collectedRentCents: number
  agencyFeesCents: number
  vacancyChargesCents: number
  ownerChargesCents: number
  incomeBeforeTaxCents: number
  taxableIncomeCents: number
  annualTaxCents: number
  netAnnualIncomeCents: number
  monthlyGrossIncomeCents: number
  monthlyNetIncomeCents: number
  grossYieldPct: number
  netYieldPct: number
  netAfterTaxYieldPct: number
  financing: Financing | null
  warnings: PropertyWarning[]
}

/** Live evaluation: computes without saving. */
export function evaluateProperty(input: Omit<PropertyInput, 'name'>): Promise<PropertyResult> {
  return request<PropertyResult>('POST', '/api/properties/evaluate', input)
}
