import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ProfitChart from './ProfitChart.vue'
import type { Timeline } from '@/api/properties'

const plain = (s: string) => s.replace(/[\u00a0\u202f]/g, ' ')

/** −1 000 € apport, −100 €/month for 12 months, then +200 €/month. */
function timeline(): Timeline {
  const points = []
  let cumulative = -100_000
  for (let k = 1; k <= 30; k++) {
    const cashFlow = k <= 12 ? -10_000 : 20_000
    cumulative += cashFlow
    const total = 2027 * 12 + (k - 1)
    const month = `${Math.floor(total / 12)}-${String((total % 12) + 1).padStart(2, '0')}`
    points.push({ month, cashFlowCents: cashFlow, cumulativeCents: cumulative })
  }
  return {
    startMonth: '2027-01',
    positiveCashFlowMonth: '2028-01',
    paybackMonth: '2028-12',
    paybackAtStart: false,
    points,
  }
}

describe('ProfitChart', () => {
  it('draws the cumulative position as one line, one point per month', () => {
    const wrapper = mount(ProfitChart, { props: { timeline: timeline() } })

    expect(wrapper.get('h2').text()).toBe('Trésorerie cumulée')
    const points = wrapper
      .get('[data-test="cumulative-line"]')
      .attributes('points')
      ?.trim()
      .split(/\s+/)
    expect(points).toHaveLength(30)
    expect(wrapper.find('[data-test="zero-line"]').exists()).toBe(true)
  })

  it('marks both dates with French labels', () => {
    const wrapper = mount(ProfitChart, { props: { timeline: timeline() } })

    const positive = wrapper.get('[data-test="marker-positive"]')
    expect(positive.text()).toContain('Cash-flow positif')
    const payback = wrapper.get('[data-test="marker-payback"]')
    expect(payback.text()).toContain('Apport récupéré')
  })

  it('says what the line adds up, with or without a loan', () => {
    const financed = mount(ProfitChart, { props: { timeline: timeline(), financed: true } })
    expect(financed.text()).toContain(
      'Apport de départ, puis chaque mois : revenus après impôts − mensualité du prêt.',
    )

    const unfinanced = mount(ProfitChart, { props: { timeline: timeline(), financed: false } })
    expect(unfinanced.text()).toContain(
      'Apport de départ (prix + notaire), puis chaque mois : revenus après impôts.',
    )
  })

  it('omits a marker for a date that is never reached', () => {
    const t = { ...timeline(), paybackMonth: '' }
    const wrapper = mount(ProfitChart, { props: { timeline: t } })

    expect(wrapper.find('[data-test="marker-payback"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="marker-positive"]').exists()).toBe(true)
  })

  it('has an accessible summary naming the dates', () => {
    const wrapper = mount(ProfitChart, { props: { timeline: timeline() } })

    const svg = wrapper.get('svg')
    expect(svg.attributes('role')).toBe('img')
    expect(svg.attributes('aria-label')).toContain('cash-flow mensuel positif le 01/01/2028')
    expect(svg.attributes('aria-label')).toContain('apport récupéré le 01/12/2028')
  })

  it('shows the month and both amounts on hover and with the keyboard', async () => {
    const wrapper = mount(ProfitChart, { props: { timeline: timeline() } })
    const frame = wrapper.get('[data-test="profit-chart"]')

    await frame.trigger('focus')
    let tip = plain(wrapper.get('[role="tooltip"]').text())
    expect(tip).toContain('janv. 2027')
    expect(tip).toContain('-1 100,00 €')
    expect(tip).toContain('-100,00 €')

    await frame.trigger('keydown', { key: 'ArrowRight' })
    tip = plain(wrapper.get('[role="tooltip"]').text())
    expect(tip).toContain('janv. 2028') // arrows move by a year
    await frame.trigger('keydown', { key: 'End' })
    expect(wrapper.get('[role="tooltip"]').text()).toContain('juin 2029')
    await frame.trigger('blur')
    expect(wrapper.find('[role="tooltip"]').exists()).toBe(false)
  })

  it('labels an immediately positive cash-flow « Immédiat », like the card', () => {
    const t = { ...timeline(), positiveCashFlowMonth: '2027-01' }
    const wrapper = mount(ProfitChart, { props: { timeline: t } })

    expect(wrapper.get('[data-test="marker-positive"]').text()).toBe('Cash-flow positif · Immédiat')
    expect(wrapper.get('svg').attributes('aria-label')).toContain(
      'cash-flow mensuel positif dès le premier mois',
    )
  })

  it('draws at its container width so labels keep their size', async () => {
    let notify: (entries: { contentRect: { width: number } }[]) => void = () => {}
    class FakeResizeObserver {
      constructor(cb: typeof notify) {
        notify = cb
      }
      observe() {}
      disconnect() {}
    }
    vi.stubGlobal('ResizeObserver', FakeResizeObserver)
    const wrapper = mount(ProfitChart, { props: { timeline: timeline() } })
    expect(wrapper.get('svg').attributes('viewBox')).toBe('0 0 1100 280')

    notify([{ contentRect: { width: 640 } }])
    await wrapper.vm.$nextTick()

    expect(wrapper.get('svg').attributes('viewBox')).toBe('0 0 640 280')
    vi.unstubAllGlobals()
  })
})
