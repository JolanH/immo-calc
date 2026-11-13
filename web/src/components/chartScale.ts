// Small helpers for the hand-written SVG charts.

const NICE = [1, 2, 2.5, 5]
const MAX_INTERVALS = 5

/**
 * Round axis ticks from 0 covering max: the smallest step of the form
 * {1, 2, 2.5, 5} × 10^k that needs at most five intervals.
 */
export function niceTicks(max: number): number[] {
  if (max <= 0) return [0, 1]
  let step = 1
  for (let magnitude = 1; ; magnitude *= 10) {
    const found = NICE.map((n) => n * magnitude).find((s) => Math.ceil(max / s) <= MAX_INTERVALS)
    if (found !== undefined) {
      step = found
      break
    }
  }
  const ticks: number[] = []
  for (let v = 0; v < max + step; v += step) {
    ticks.push(v)
    if (v >= max) break
  }
  return ticks
}

const compact = new Intl.NumberFormat('fr-FR', { notation: 'compact', maximumFractionDigits: 1 })
const full = new Intl.NumberFormat('fr-FR', { maximumFractionDigits: 0 })

/**
 * Formats axis ticks given in cents: in full below 10 000 € ("2 000 €"),
 * compact on an axis that reaches it ("50 k€", "1,5 M€"). One style per axis.
 */
export function tickFormatter(ticks: number[]): (cents: number) => string {
  const top = (ticks.at(-1) ?? 0) / 100
  if (top < 10_000) return (cents) => `${full.format(cents / 100)}\u00a0€`
  return (cents) => {
    const text = compact.format(cents / 100).replace(/\s/g, '\u00a0')
    // "15 k€" but "0 €": a bare number keeps its space before the unit.
    return /\d$/.test(text) ? `${text}\u00a0€` : `${text}€`
  }
}
