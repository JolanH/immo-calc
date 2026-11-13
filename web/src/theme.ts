// The interface's main colour (« Couleur principale », settings.accentColor).
// From one chosen colour we derive a shade readable on the light surface and
// one readable on the dark surface (WCAG ≥ 4,5:1), plus the text colour to
// put on top of each, and expose them as CSS variables (see style.css).

export const DEFAULT_ACCENT = '#8b1a1a' // dark red
const LIGHT_SURFACE = '#ffffff'
const DARK_SURFACE = '#1f2228'
const MIN_CONTRAST = 4.5
const HEX = /^#[0-9a-f]{6}$/i

type RGB = [number, number, number]

function toRgb(hex: string): RGB {
  const n = parseInt(hex.slice(1), 16)
  return [(n >> 16) & 255, (n >> 8) & 255, n & 255]
}

function toHex([r, g, b]: RGB): string {
  return '#' + [r, g, b].map((c) => Math.round(c).toString(16).padStart(2, '0')).join('')
}

function luminance(hex: string): number {
  const [r, g, b] = toRgb(hex).map((c) => {
    const v = c / 255
    return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4
  })
  return 0.2126 * r + 0.7152 * g + 0.0722 * b
}

/** WCAG contrast ratio between two colours. */
export function contrast(a: string, b: string): number {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x)
  return (hi + 0.05) / (lo + 0.05)
}

function mix(hex: string, target: RGB, t: number): string {
  const c = toRgb(hex)
  return toHex([0, 1, 2].map((i) => c[i] + (target[i] - c[i]) * t) as RGB)
}

/**
 * The colour itself if text in it reads well on `surface`; otherwise the
 * closest lighter (dark surface) or darker (light surface) shade that does.
 */
export function readableOn(hex: string, surface: string): string {
  if (contrast(hex, surface) >= MIN_CONTRAST) return hex.toLowerCase()
  const target: RGB = luminance(surface) < 0.5 ? [255, 255, 255] : [0, 0, 0]
  for (let t = 0.05; t <= 1; t += 0.05) {
    const shade = mix(hex, target, t)
    if (contrast(shade, surface) >= MIN_CONTRAST) return shade
  }
  return toHex(target)
}

/** White or near-black text, whichever reads best on the colour. */
export function textOn(hex: string): string {
  return contrast('#ffffff', hex) >= contrast('#0b0b0b', hex) ? '#ffffff' : '#0b0b0b'
}

/** Applies the main colour to the whole interface (both themes). */
export function applyAccent(hex: string | undefined): void {
  const base = hex && HEX.test(hex) ? hex : DEFAULT_ACCENT
  const light = readableOn(base, LIGHT_SURFACE)
  const dark = readableOn(base, DARK_SURFACE)
  const style = document.documentElement.style
  style.setProperty('--brand', light)
  style.setProperty('--brand-text', textOn(light))
  style.setProperty('--brand-dark', dark)
  style.setProperty('--brand-dark-text', textOn(dark))
}
