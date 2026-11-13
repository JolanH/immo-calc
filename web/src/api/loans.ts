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
