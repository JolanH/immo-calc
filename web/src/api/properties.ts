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

/**
 * Live evaluation: computes without saving. id is the saved property being
 * edited, so the micro ceilings don't count it twice.
 */
export function evaluateProperty(
  input: Omit<PropertyInput, 'name'>,
  id?: string,
): Promise<PropertyResult> {
  const query = id ? `?id=${encodeURIComponent(id)}` : ''
  return request<PropertyResult>('POST', `/api/properties/evaluate${query}`, input)
}

/** A saved property as listed. */
export interface PropertySummary {
  id: string
  name: string
  priceCents: number
  monthlyRentCents: number
  dpe: string
  netAfterTaxYieldPct: number
  monthlyNetIncomeCents: number
  cashFlowAfterTaxCents: number | null
  loanId: string
  loanName: string
  warnings: PropertyWarning[]
}

/** A saved property with its results and linked loan. */
export interface PropertyDetail extends PropertyInput {
  id: string
  createdAt: string
  updatedAt: string
  result: PropertyResult
  loan: { id: string; name: string } | null
}

const path = (id: string) => `/api/properties/${encodeURIComponent(id)}`

export function listProperties(loanId?: string): Promise<PropertySummary[]> {
  const query = loanId ? `?loanId=${encodeURIComponent(loanId)}` : ''
  return request<PropertySummary[]>('GET', `/api/properties${query}`)
}

export function getProperty(id: string): Promise<PropertyDetail> {
  return request<PropertyDetail>('GET', path(id))
}

export function createProperty(input: PropertyInput): Promise<PropertyDetail> {
  return request<PropertyDetail>('POST', '/api/properties', input)
}

export function updateProperty(id: string, input: PropertyInput): Promise<PropertyDetail> {
  return request<PropertyDetail>('PUT', path(id), input)
}

export function duplicateProperty(id: string): Promise<PropertyDetail> {
  return request<PropertyDetail>('POST', `${path(id)}/duplicate`)
}

export function deleteProperty(id: string): Promise<void> {
  return request<void>('DELETE', path(id))
}
