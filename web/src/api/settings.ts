import { request } from './client'

/** Global hypotheses and tax settings (rates in %). */
export interface Settings {
  marginalTaxRatePct: number
  socialChargesUnfurnishedPct: number
  socialChargesFurnishedPct: number
  notaryFeePct: number
  vacancyMonthsPerYear: number
  agencyFeePct: number
  accentColor: string // "#rrggbb", the interface's main colour
  themeMode: 'system' | 'light' | 'dark'
}

export function getSettings(): Promise<Settings> {
  return request<Settings>('GET', '/api/settings')
}

export function updateSettings(settings: Settings): Promise<Settings> {
  return request<Settings>('PUT', '/api/settings', settings)
}
