import { describe, expect, it } from 'vitest'
import { formatEuros, formatPercent } from './format'

// Intl uses non-breaking spaces (U+00A0, U+202F) in fr-FR; compare with plain spaces.
const plain = (s: string) => s.replace(/[\u00a0\u202f]/g, ' ')

describe('formatEuros', () => {
  it.each([
    [123456, '1 234,56 €'],
    [0, '0,00 €'],
    [5, '0,05 €'],
    [100, '1,00 €'],
    [-123456, '-1 234,56 €'],
    [500_000_000, '5 000 000,00 €'],
  ])('formats %i cents as %s', (cents, expected) => {
    expect(plain(formatEuros(cents))).toBe(expected)
  })

  it('keeps the amount and the euro sign on one line (non-breaking spaces)', () => {
    expect(formatEuros(123456)).not.toContain(' ')
  })
})

describe('formatPercent', () => {
  it.each([
    [3.5, '3,50 %'],
    [0, '0,00 %'],
    [5.164, '5,16 %'],
    [1.756, '1,76 %'],
    [-2.5, '-2,50 %'],
    [100, '100,00 %'],
  ])('formats %d as %s', (value, expected) => {
    expect(plain(formatPercent(value))).toBe(expected)
  })
})
