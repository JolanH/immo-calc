import { afterEach, describe, expect, it } from 'vitest'
import { applyAccent, contrast, DEFAULT_ACCENT, readableOn, textOn } from './theme'

const LIGHT_SURFACE = '#ffffff'
const DARK_SURFACE = '#1f2228'

afterEach(() => document.documentElement.removeAttribute('style'))

describe('contrast', () => {
  it('follows WCAG', () => {
    expect(contrast('#ffffff', '#000000')).toBeCloseTo(21, 1)
    expect(contrast('#8b1a1a', '#ffffff')).toBeGreaterThan(9)
  })
})

describe('readableOn', () => {
  it('keeps a colour that is already readable', () => {
    expect(readableOn(DEFAULT_ACCENT, LIGHT_SURFACE)).toBe('#8b1a1a')
  })

  it('lightens dark red for the dark theme until text is readable', () => {
    const shade = readableOn(DEFAULT_ACCENT, DARK_SURFACE)
    expect(shade).not.toBe('#8b1a1a')
    expect(contrast(shade, DARK_SURFACE)).toBeGreaterThanOrEqual(4.5)
  })

  it('darkens a colour too pale for the light theme', () => {
    const shade = readableOn('#ffd0d0', LIGHT_SURFACE)
    expect(contrast(shade, LIGHT_SURFACE)).toBeGreaterThanOrEqual(4.5)
  })
})

describe('textOn', () => {
  it('picks white or near-black, whichever reads best on the colour', () => {
    expect(textOn('#8b1a1a')).toBe('#ffffff')
    expect(textOn('#ffd98a')).toBe('#0b0b0b')
  })
})

describe('applyAccent', () => {
  it('sets the light and dark shades as CSS variables', () => {
    applyAccent('#8b1a1a')

    const style = document.documentElement.style
    expect(style.getPropertyValue('--brand')).toBe('#8b1a1a')
    expect(style.getPropertyValue('--brand-text')).toBe('#ffffff')
    expect(contrast(style.getPropertyValue('--brand-dark'), DARK_SURFACE)).toBeGreaterThanOrEqual(
      4.5,
    )
    expect(style.getPropertyValue('--brand-dark-text')).not.toBe('')
  })

  it('falls back to dark red for an invalid colour', () => {
    applyAccent('nope')
    expect(document.documentElement.style.getPropertyValue('--brand')).toBe('#8b1a1a')
  })
})
