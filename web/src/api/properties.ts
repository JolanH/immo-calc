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
  annualCreditCostCents: number
  returnOnContributionPct: number | null // null = n/a
}

export interface PropertyWarning {
  code: string
  severity: 'critical' | 'serious' | 'warning'
  message: string
}

/** One month of the « Trésorerie cumulée » chart. */
export interface TimelinePoint {
  month: string // "YYYY-MM"
  cashFlowCents: number
  cumulativeCents: number
}

/** Profitability dates (with a linked loan). */
export interface Timeline {
  startMonth: string
  positiveCashFlowMonth: string // '' = never
  paybackMonth: string // '' = not within 50 years
  paybackAtStart: boolean
  points: TimelinePoint[]
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
  /** (RN − average annual credit cost) / A; equals netAfterTaxYieldPct without a loan. */
  netAfterCreditYieldPct: number
  financing: Financing | null
  timeline?: Timeline | null
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
  netAfterCreditYieldPct: number
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

/**
 * Lists saved properties: only those using `loanId` if given; with
 * `activeLoanId`, every property is financed by that loan exactly as saved
 * (home page).
 */
export function listProperties(
  options: { loanId?: string; activeLoanId?: string } = {},
): Promise<PropertySummary[]> {
  const params = new URLSearchParams()
  if (options.loanId) params.set('loanId', options.loanId)
  if (options.activeLoanId) params.set('activeLoanId', options.activeLoanId)
  const query = params.size ? `?${params}` : ''
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
