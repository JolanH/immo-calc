import { describe, expect, it } from 'vitest'
import { formatInputEuros, formatInputRate, parseEuros, parseRate } from './parse'

describe('parseEuros', () => {
  it.each([
    ['200 000', 20_000_000],
    ['200000', 20_000_000],
    ['200\u00a0000', 20_000_000],
    ['200\u202f000', 20_000_000],
    ['1 234,56', 123_456],
    ['1234.56', 123_456],
    ['0,5', 50],
    ['12,345', 1_235],
    ['  42  ', 4_200],
    ['42 €', 4_200],
    ['1500,', 150_000],
    ['0', 0],
  ])('parses %j as %i cents', (text, cents) => {
    expect(parseEuros(text)).toBe(cents)
  })

  it.each(['', '   ', 'abc', '12,3,4', '-5', '1e5', '12a'])('rejects %j', (text) => {
    expect(parseEuros(text)).toBeNull()
  })
})

describe('parseRate', () => {
  it.each([
    ['3,5', 3.5],
    ['3.5', 3.5],
    ['3,125', 3.125],
    ['0', 0],
    [' 4 ', 4],
    ['3,5 %', 3.5],
    ['3,', 3],
  ])('parses %j as %d', (text, value) => {
    expect(parseRate(text)).toBe(value)
  })

  it.each(['', 'x', '3,5,1', '-1'])('rejects %j', (text) => {
    expect(parseRate(text)).toBeNull()
  })
})

describe('input formatting', () => {
  it.each([
    [20_000_000, '200 000'],
    [123_456, '1 234,56'],
    [50, '0,50'],
    [0, '0'],
  ])('formats %i cents for an input as %j', (cents, text) => {
    expect(formatInputEuros(cents).replace(/[\u00a0\u202f]/g, ' ')).toBe(text)
  })

  it.each([
    [3.5, '3,5'],
    [3.125, '3,125'],
    [0, '0'],
  ])('formats rate %d for an input as %j', (value, text) => {
    expect(formatInputRate(value)).toBe(text)
  })
})
