// French (fr-FR) display formatting. Every amount or rate shown in the UI goes
// through these helpers.

const euros = new Intl.NumberFormat('fr-FR', {
  style: 'currency',
  currency: 'EUR',
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
})

const percent = new Intl.NumberFormat('fr-FR', {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
})

/** formatEuros(123456) → "1 234,56 €". Amounts are integer euro cents. */
export function formatEuros(cents: number): string {
  return euros.format(cents / 100)
}

/** formatPercent(3.5) → "3,50 %". The value is already a percentage. */
export function formatPercent(value: number): string {
  return `${percent.format(value)}\u00a0%`
}

const monthFormats = {
  short: new Intl.DateTimeFormat('fr-FR', { month: 'short', year: 'numeric' }),
  long: new Intl.DateTimeFormat('fr-FR', { month: 'long', year: 'numeric' }),
}

/** formatMonthDate("2032-01") → "01/01/2032": a month shown as its first day. */
export function formatMonthDate(month: string): string {
  const m = /^(\d{4})-(\d{2})$/.exec(month)
  return m ? `01/${m[2]}/${m[1]}` : month
}

/** formatMonth("2026-12") → "déc. 2026"; with "long" → "décembre 2026". */
export function formatMonth(month: string, style: 'short' | 'long' = 'short'): string {
  const m = /^(\d{4})-(\d{2})$/.exec(month)
  if (!m) return month
  return monthFormats[style].format(new Date(Number(m[1]), Number(m[2]) - 1, 1))
}
