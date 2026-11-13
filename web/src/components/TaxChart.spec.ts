import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import TaxChart from './TaxChart.vue'
import type { YearTax } from '@/api/properties'

const plain = (s: string) => s.replace(/[  ]/g, ' ')

/** Tax 500 € for 7 years, then 800 €; income before tax 3 000 €. */
const years = (): YearTax[] =>
  Array.from({ length: 10 }, (_, i) => {
    const tax = i < 7 ? 50_000 : 80_000
    return { year: i + 1, annualTaxCents: tax, netAnnualIncomeCents: 300_000 - tax }
  })

describe('TaxChart', () => {
  it('draws the tax and the net income, one point per year', () => {
    const wrapper = mount(TaxChart, { props: { years: years() } })

    expect(wrapper.get('h2').text()).toBe('Impôt au fil des années')
    for (const line of ['tax-line', 'net-line']) {
      expect(
        wrapper.get(`[data-test="${line}"]`).attributes('points')?.trim().split(/\s+/),
      ).toHaveLength(10)
    }
    expect(plain(wrapper.get('[data-test="tax-chart-legend"]').text())).toContain('Impôt annuel')
  })

  it('describes the first and the last year for assistive technologies', () => {
    const wrapper = mount(TaxChart, { props: { years: years() } })

    const label = plain(wrapper.get('svg').attributes('aria-label') ?? '')
    expect(label).toContain('500,00 € la 1re année')
    expect(label).toContain('800,00 € la 10e')
    expect(label).toContain('2 500,00 € puis 2 200,00 €')
  })

  it('shows a year with the keyboard', async () => {
    const wrapper = mount(TaxChart, { props: { years: years() } })
    const frame = wrapper.get('[data-test="tax-chart"]')

    await frame.trigger('focus')
    await frame.trigger('keydown', { key: 'ArrowRight' })

    const tip = plain(wrapper.get('[role="tooltip"]').text())
    expect(tip).toContain('Année 2')
    expect(tip).toContain('2 500,00 € Revenu net')
    expect(tip).toContain('500,00 € Impôt')
  })

  it('draws nothing for a single year', () => {
    const wrapper = mount(TaxChart, { props: { years: years().slice(0, 1) } })

    expect(wrapper.find('.tax-chart').exists()).toBe(false)
  })
})
