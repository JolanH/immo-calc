import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory } from 'vue-router'
import { createAppRouter } from '@/router'
import App from '@/App.vue'

const plain = (s: string) => s.replace(/[\u00a0\u202f]/g, ' ')

const properties = [
  {
    id: 'aaaaaaaaaaaaaaaa',
    name: 'T2 Lyon 7e',
    priceCents: 15_000_000,
    monthlyRentCents: 70_000,
    dpe: 'F',
    netAfterTaxYieldPct: 1.75,
    monthlyNetIncomeCents: 23_799,
    cashFlowAfterTaxCents: null,
    loanId: '',
    loanName: '',
    warnings: [],
  },
  {
    id: 'bbbbbbbbbbbbbbbb',
    name: 'Studio',
    priceCents: 9_000_000,
    monthlyRentCents: 50_000,
    dpe: 'C',
    netAfterTaxYieldPct: 2.1,
    monthlyNetIncomeCents: 15_000,
    cashFlowAfterTaxCents: -12_345,
    loanId: 'cccccccccccccccc',
    loanName: 'Petit prêt',
    warnings: [],
  },
]

async function mountList(body: unknown) {
  // Only the list is served; pages opened from it get a 404.
  const notFound = { error: { code: 'not_found', message: 'introuvable' } }
  vi.stubGlobal(
    'fetch',
    vi.fn((url: string) =>
      Promise.resolve(
        url === '/api/properties'
          ? new Response(JSON.stringify(body))
          : new Response(JSON.stringify(notFound), { status: 404 }),
      ),
    ),
  )
  const router = createAppRouter(createMemoryHistory())
  router.push('/biens')
  await router.isReady()
  const wrapper = mount(App, { global: { plugins: [router] } })
  await flushPromises()
  return { wrapper, router }
}

afterEach(() => vi.unstubAllGlobals())

describe('BiensListe', () => {
  it('lists the properties with their key figures', async () => {
    const { wrapper } = await mountList(properties)

    expect(wrapper.findAll('thead th').map((th) => th.text())).toEqual([
      'Nom',
      'Prix',
      'Loyer',
      'DPE',
      'Rentabilité nette après impôts',
      'Revenus mensuels après impôts',
      'Cash-flow après impôts',
      'Prêt',
    ])
    const rows = wrapper
      .findAll('tbody tr')
      .map((tr) => tr.findAll('td').map((td) => plain(td.text())))
    expect(rows[0]).toEqual([
      'T2 Lyon 7e',
      '150 000,00 €',
      '700,00 €',
      'F',
      '1,75 %',
      '237,99 €',
      '—',
      '—',
    ])
    expect(rows[1]).toEqual([
      'Studio',
      '90 000,00 €',
      '500,00 €',
      'C',
      '2,10 %',
      '150,00 €',
      '-123,45 €',
      'Petit prêt',
    ])
  })

  it('shows the DPE as a labelled badge', async () => {
    const { wrapper } = await mountList(properties)

    const badge = wrapper.get('tbody [data-test="dpe-badge"]')
    expect(badge.text()).toBe('F')
    expect(badge.attributes('aria-label')).toBe('DPE F')
    expect(badge.classes()).toContain('dpe--f')
  })

  it('links each property and its loan', async () => {
    const { wrapper, router } = await mountList(properties)

    const links = wrapper.findAll('tbody a').map((a) => a.attributes('href'))
    expect(links).toEqual([
      '/biens/aaaaaaaaaaaaaaaa',
      '/biens/bbbbbbbbbbbbbbbb',
      '/prets/cccccccccccccccc',
    ])
    await wrapper.findAll('tbody tr')[0].trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/biens/aaaaaaaaaaaaaaaa')
  })

  it('hides the cash-flow column when no property has a loan', async () => {
    const { wrapper } = await mountList([properties[0]])

    expect(wrapper.findAll('thead th').map((th) => th.text())).not.toContain(
      'Cash-flow après impôts',
    )
  })

  it('shows the empty state with « Nouveau bien »', async () => {
    const { wrapper } = await mountList([])

    expect(wrapper.text()).toContain('Aucun bien enregistré.')
    expect(wrapper.get('[data-test="new-property"]').attributes('href')).toBe('/biens/nouveau')
    expect(wrapper.find('table').exists()).toBe(false)
  })

  it('has a breadcrumb', async () => {
    const { wrapper } = await mountList([])

    expect(
      wrapper
        .get('nav[aria-label="Fil d\'Ariane"]')
        .findAll('li')
        .map((li) => li.text()),
    ).toEqual(['Accueil', 'Biens'])
  })
})
