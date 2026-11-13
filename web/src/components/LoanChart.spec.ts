import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import LoanChart from './LoanChart.vue'
import { niceRange, niceTicks, tickFormatter } from './chartScale'
import type { YearRow } from '@/api/loans'

const plain = (s: string) => s.replace(/[\u00a0\u202f]/g, ' ')

function year(
  y: number,
  principal: number,
  interest: number,
  remaining: number,
  installments = 12,
): YearRow {
  return {
    year: y,
    installments,
    paymentCents: principal + interest,
    interestCents: interest,
    principalCents: principal,
    insuranceCents: 0,
    remainingCents: remaining,
  }
}

const yearly: YearRow[] = [
  year(2026, 57_659, 58_333, 19_942_341, 1),
  year(2027, 700_000, 692_000, 19_242_341),
  year(2028, 725_000, 667_000, 18_517_341),
  year(2029, 1_000_000, 0, 17_517_341),
]

const num = (el: { attributes: (k: string) => string | undefined }, k: string) =>
  Number(el.attributes(k))

describe('LoanChart', () => {
  it('renders nothing without yearly data', () => {
    const wrapper = mount(LoanChart, { props: { yearly: [] } })
    expect(wrapper.find('svg').exists()).toBe(false)
  })

  it('draws one stacked bar per calendar year', () => {
    const wrapper = mount(LoanChart, { props: { yearly } })

    expect(wrapper.findAll('[data-test="bar"]')).toHaveLength(4)
    expect(wrapper.findAll('[data-test="segment-principal"]')).toHaveLength(4)
    expect(wrapper.findAll('[data-test="segment-interest"]')).toHaveLength(4)
  })

  it('makes segment heights proportional to the yearly sums', () => {
    const wrapper = mount(LoanChart, { props: { yearly } })
    const principal = wrapper
      .findAll('[data-test="segment-principal"]')
      .map((s) => num(s, 'data-height'))
    const interest = wrapper
      .findAll('[data-test="segment-interest"]')
      .map((s) => num(s, 'data-height'))

    // 2027 vs 2029 principal: 700 000 / 1 000 000.
    expect(principal[1] / principal[3]).toBeCloseTo(0.7, 5)
    // 2027: interest / principal = 692 000 / 700 000.
    expect(interest[1] / principal[1]).toBeCloseTo(692_000 / 700_000, 5)
    // No interest in 2029: no top segment drawn.
    expect(interest[3]).toBe(0)
  })

  it('draws the remaining principal as a line with one point per year, in its own chart', () => {
    const wrapper = mount(LoanChart, { props: { yearly } })

    const svgs = wrapper.findAll('svg')
    expect(svgs).toHaveLength(2)
    const line = wrapper.get('[data-test="remaining-line"]')
    expect(line.attributes('points')?.trim().split(/\s+/)).toHaveLength(4)
    expect(wrapper.text()).toContain('Capital restant dû')
  })

  it('has a French legend and an accessible summary', () => {
    const wrapper = mount(LoanChart, { props: { yearly } })

    const legend = wrapper.get('[data-test="legend"]').text()
    expect(legend).toContain('Capital remboursé')
    expect(legend).toContain('Intérêts')
    const bars = wrapper.findAll('svg')[0]
    expect(bars.attributes('role')).toBe('img')
    expect(bars.attributes('aria-label')).toContain('2026 à 2029')
    expect(bars.get('title').text()).toContain('Remboursement par année')
  })

  it('shows the year and the three amounts on hover', async () => {
    const wrapper = mount(LoanChart, { props: { yearly } })
    expect(wrapper.find('[role="tooltip"]').exists()).toBe(false)

    await wrapper.findAll('[data-test="hit"]')[1].trigger('pointerenter')

    const tip = plain(wrapper.get('[role="tooltip"]').text())
    expect(tip).toContain('2027')
    expect(tip).toContain('7 000,00 €')
    expect(tip).toContain('Capital remboursé')
    expect(tip).toContain('6 920,00 €')
    expect(tip).toContain('Intérêts')
    expect(tip).toContain('192 423,41 €')
    expect(tip).toContain('Capital restant dû')

    await wrapper.findAll('[data-test="hit"]')[1].trigger('pointerleave')
    expect(wrapper.find('[role="tooltip"]').exists()).toBe(false)
  })

  it('labels partial years in the tooltip', async () => {
    const wrapper = mount(LoanChart, { props: { yearly } })

    await wrapper.findAll('[data-test="hit"]')[0].trigger('pointerenter')

    expect(wrapper.get('[role="tooltip"]').text()).toContain('2026 (1 échéance)')
  })

  it('moves through the years with the keyboard', async () => {
    const wrapper = mount(LoanChart, { props: { yearly } })
    const chart = wrapper.get('[data-test="chart"]')

    await chart.trigger('focus')
    expect(wrapper.get('[role="tooltip"]').text()).toContain('2026')
    await chart.trigger('keydown', { key: 'ArrowRight' })
    expect(wrapper.get('[role="tooltip"]').text()).toContain('2027')
    await chart.trigger('keydown', { key: 'End' })
    expect(wrapper.get('[role="tooltip"]').text()).toContain('2029')
    await chart.trigger('keydown', { key: 'ArrowRight' })
    expect(wrapper.get('[role="tooltip"]').text()).toContain('2029')
    await chart.trigger('keydown', { key: 'Home' })
    expect(wrapper.get('[role="tooltip"]').text()).toContain('2026')
    await chart.trigger('blur')
    expect(wrapper.find('[role="tooltip"]').exists()).toBe(false)
  })
})

describe('niceTicks', () => {
  it.each([
    [1_391_904, [0, 500_000, 1_000_000, 1_500_000]],
    [20_000_000, [0, 5_000_000, 10_000_000, 15_000_000, 20_000_000]],
    [83_373, [0, 20_000, 40_000, 60_000, 80_000, 100_000]],
    [1, [0, 1]],
  ])('covers %i with round ticks', (max, expected) => {
    expect(niceTicks(max)).toEqual(expected)
  })

  it('handles zero', () => {
    expect(niceTicks(0)).toEqual([0, 1])
  })
})

describe('tickFormatter', () => {
  it('writes ticks in full below 10 000 €', () => {
    const format = tickFormatter([0, 200_000, 400_000])
    expect(plain(format(0))).toBe('0 €')
    expect(plain(format(200_000))).toBe('2 000 €')
  })

  it('writes ticks compactly on an axis reaching 10 000 €', () => {
    const format = tickFormatter([0, 1_000_000, 1_500_000, 20_000_000])
    expect(plain(format(0))).toBe('0 €')
    expect(plain(format(1_500_000))).toBe('15 k€')
    expect(plain(format(20_000_000))).toBe('200 k€')
  })
})

describe('niceRange', () => {
  it('covers a range crossing zero with round ticks', () => {
    expect(niceRange(-3_261_613, 2_977)).toEqual([
      -4_000_000, -3_000_000, -2_000_000, -1_000_000, 0, 1_000_000,
    ])
  })

  it('always includes zero', () => {
    expect(niceRange(500_000, 2_000_000)).toEqual([0, 500_000, 1_000_000, 1_500_000, 2_000_000])
    expect(niceRange(-2_000_000, -500_000)).toEqual([
      -2_000_000, -1_500_000, -1_000_000, -500_000, 0,
    ])
  })
})
