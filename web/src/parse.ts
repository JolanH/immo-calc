// Parsing of numbers typed the French way ("200 000", "3,5") in form inputs,
// and formatting of values back into those inputs. Display formatting (with
// "€" and "%") lives in format.ts.

const spaces = /[\s\u00a0\u202f]/g
const decimal = /^(\d+)(?:[.,](\d*))?$/ // a trailing "," is accepted while typing

function clean(text: string, unit: string): RegExpMatchArray | null {
  return text.replaceAll(unit, '').replace(spaces, '').match(decimal)
}

/** parseEuros("1 234,56") → 123456 cents; null when the text isn't an amount. */
export function parseEuros(text: string): number | null {
  const m = clean(text, '€')
  if (!m) return null
  const whole = Number(m[1])
  // Work on the digits to avoid float error; round to the cent, half up.
  const fraction = ((m[2] ?? '') + '000').slice(0, 3)
  return whole * 100 + Math.round(Number(fraction) / 10)
}

/** parseRate("3,5") → 3.5; null when the text isn't a number. */
export function parseRate(text: string): number | null {
  const m = clean(text, '%')
  if (!m) return null
  return Number(m[2] ? `${m[1]}.${m[2]}` : m[1])
}

const wholeEuros = new Intl.NumberFormat('fr-FR', { maximumFractionDigits: 0 })
const centEuros = new Intl.NumberFormat('fr-FR', {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
})

/** formatInputEuros(20000000) → "200 000" (no unit, cents only when needed). */
export function formatInputEuros(cents: number): string {
  return cents % 100 === 0 ? wholeEuros.format(cents / 100) : centEuros.format(cents / 100)
}

/** formatInputRate(3.5) → "3,5". */
export function formatInputRate(value: number): string {
  return String(value).replace('.', ',')
}
