import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory } from 'vue-router'
import { createAppRouter } from '@/router'
import PretsListe from './PretsListe.vue'

const plain = (s: string) => s.replace(/[\u00a0\u202f]/g, ' ')

const loans = [
  {
    id: 'aaaaaaaaaaaaaaaa',
    name: 'Appart Lyon — Crédit Mutuel',
    principalCents: 20_000_000,
    annualRatePct: 3.5,
    durationMonths: 240,
    monthlyTotalCents: 120_992,
    totalCostCents: 9_388_066,
  },
  {
    id: 'bbbbbbbbbbbbbbbb',
    name: 'BNP',
    principalCents: 15_000_000,
    annualRatePct: 4,
    durationMonths: 300,
    monthlyTotalCents: 79_176,
    totalCostCents: 8_752_543,
  },
]

async function mountList(body: unknown) {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify(body))))
  const router = createAppRouter(createMemoryHistory())
  router.push('/prets')
  await router.isReady()
  const wrapper = mount(PretsListe, { global: { plugins: [router] } })
  await flushPromises()
  return { wrapper, router }
}

afterEach(() => vi.unstubAllGlobals())

describe('PretsListe', () => {
  it('lists the saved loans with their key figures', async () => {
    const { wrapper } = await mountList(loans)

    const headers = wrapper.findAll('th').map((th) => th.text())
    expect(headers).toEqual([
      'Nom',
      'Montant',
      'Taux',
      'Durée',
      'Mensualité (assurance incluse)',
      'Coût total du crédit',
    ])
    const cells = wrapper
      .findAll('tbody tr')[0]
      .findAll('td')
      .map((td) => plain(td.text()))
    expect(cells).toEqual([
      'Appart Lyon — Crédit Mutuel',
      '200 000,00 €',
      '3,50 %',
      '20 ans',
      '1 209,92 €',
      '93 880,66 €',
    ])
    expect(wrapper.findAll('tbody tr')).toHaveLength(2)
  })

  it('opens a loan when its row is clicked', async () => {
    const { wrapper, router } = await mountList(loans)

    await wrapper.findAll('tbody tr')[1].trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.path).toBe('/prets/bbbbbbbbbbbbbbbb')
  })

  it('links each name to its loan for keyboard users', async () => {
    const { wrapper } = await mountList(loans)

    expect(wrapper.get('tbody a').attributes('href')).toBe('/prets/aaaaaaaaaaaaaaaa')
  })

  it('offers a new simulation', async () => {
    const { wrapper } = await mountList(loans)

    expect(wrapper.get('[data-test="new-loan"]').text()).toBe('Nouvelle simulation')
    expect(wrapper.get('[data-test="new-loan"]').attributes('href')).toBe('/prets/nouveau')
  })

  it('shows the empty state', async () => {
    const { wrapper } = await mountList([])

    expect(wrapper.text()).toContain('Aucune simulation de prêt.')
    expect(wrapper.find('table').exists()).toBe(false)
    expect(wrapper.find('[data-test="new-loan"]').exists()).toBe(true)
  })

  it('shows API errors', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Failed to fetch')))
    const router = createAppRouter(createMemoryHistory())
    router.push('/prets')
    await router.isReady()
    const wrapper = mount(PretsListe, { global: { plugins: [router] } })
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain('immo-calc')
  })
})
