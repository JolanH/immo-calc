import { request } from './client'

/** A loan as entered (and stored). Amounts in cents, rates in %, months. */
export interface LoanInput {
  name: string
  principalCents: number
  annualRatePct: number
  durationMonths: number
  insuranceRatePct: number
  applicationFeeCents: number
  guaranteeFeeCents: number
  firstPaymentMonth: string // "YYYY-MM"
}

export interface Installment {
  index: number
  month: string
  paymentCents: number
  interestCents: number
  principalCents: number
  insuranceCents: number
  remainingCents: number
}

export interface YearRow {
  year: number
  installments: number
  paymentCents: number
  interestCents: number
  principalCents: number
  insuranceCents: number
  remainingCents: number
}

export interface LoanResult {
  monthlyPaymentCents: number
  monthlyInsuranceCents: number
  monthlyTotalCents: number
  totalInterestCents: number
  totalInsuranceCents: number
  totalFeesCents: number
  totalCostCents: number
  totalRepaidCents: number
  schedule: Installment[]
  yearly: YearRow[]
}

/** Live simulation: computes without saving. */
export function simulateLoan(input: Omit<LoanInput, 'name'>): Promise<LoanResult> {
  return request<LoanResult>('POST', '/api/loans/simulate', input)
}

/** A saved loan as listed. */
export interface LoanSummary {
  id: string
  name: string
  principalCents: number
  annualRatePct: number
  durationMonths: number
  monthlyTotalCents: number
  totalCostCents: number
}

/** A saved loan with its recomputed results. */
export interface LoanDetail extends LoanInput {
  id: string
  createdAt: string
  updatedAt: string
  result: LoanResult
}

export function listLoans(): Promise<LoanSummary[]> {
  return request<LoanSummary[]>('GET', '/api/loans')
}

export function getLoan(id: string): Promise<LoanDetail> {
  return request<LoanDetail>('GET', `/api/loans/${encodeURIComponent(id)}`)
}

export function createLoan(input: LoanInput): Promise<LoanDetail> {
  return request<LoanDetail>('POST', '/api/loans', input)
}

export function updateLoan(id: string, input: LoanInput): Promise<LoanDetail> {
  return request<LoanDetail>('PUT', `/api/loans/${encodeURIComponent(id)}`, input)
}

/** Saves a copy named « <nom> (copie) ». */
export function duplicateLoan(id: string): Promise<LoanDetail> {
  return request<LoanDetail>('POST', `/api/loans/${encodeURIComponent(id)}/duplicate`)
}

/** Deletes a loan; fails with code loan_in_use when a property uses it. */
export function deleteLoan(id: string): Promise<void> {
  return request<void>('DELETE', `/api/loans/${encodeURIComponent(id)}`)
}
