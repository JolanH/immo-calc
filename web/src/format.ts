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
  return `${percent.format(value)} %`
}
