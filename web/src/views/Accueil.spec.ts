import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory } from 'vue-router'
import { createAppRouter } from '@/router'
import App from '@/App.vue'

const plain = (s: string) => s.replace(/[\u00a0\u202f]/g, ' ')

const loans = [
  {
    id: 'aaaaaaaaaaaaaaaa',
    name: 'Crédit Mutuel',
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

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status })
}

function stubApi(routes: Record<string, (init: RequestInit) => Response>) {
  const fetchMock = vi.fn((url: string, init: RequestInit = {}) => {
    const key = `${init.method ?? 'GET'} ${url}`
    if (!(key in routes) && url.startsWith('/api/properties')) return Promise.resolve(json([]))
    const handler = routes[key]
    if (!handler) throw new Error(`unexpected call ${key}`)
    return Promise.resolve(handler(init))
  })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

async function mountHome() {
  const router = createAppRouter(createMemoryHistory())
  router.push('/')
  await router.isReady()
  const wrapper = mount(App, { global: { plugins: [router] } })
  await flushPromises()
  return { wrapper, router }
}

afterEach(() => vi.unstubAllGlobals())

describe('Accueil', () => {
  it('is the page at / with a Biens pane and a Prêts pane', async () => {
    stubApi({
      'GET /api/loans': () => json(loans),
      'GET /api/active-loan': () => json({ loanId: null }),
    })
    const { wrapper, router } = await mountHome()

    expect(router.currentRoute.value.path).toBe('/')
    expect(wrapper.get('h1').text()).toBe('Accueil')
    expect(wrapper.get('[data-test="home-biens"] h2').text()).toBe('Biens')
    expect(wrapper.get('[data-test="home-prets"] h2').text()).toBe('Prêts')
    expect(wrapper.find('nav[aria-label="Fil d\'Ariane"]').exists()).toBe(false)
  })

  it('shows each loan with its amount, duration and monthly total, and a link to it', async () => {
    stubApi({
      'GET /api/loans': () => json(loans),
      'GET /api/active-loan': () => json({ loanId: null }),
    })
    const { wrapper } = await mountHome()

    const rows = wrapper.findAll('[data-test="loan-row"]')
    expect(rows).toHaveLength(2)
    const first = plain(rows[0].text())
    expect(first).toContain('Crédit Mutuel')
    expect(first).toContain('200 000 € · 20 ans')
    expect(first).toContain('1 209,92 € / mois')
    expect(plain(rows[1].text())).toContain('150 000 € · 25 ans')
    const link = rows[1].get('a')
    expect(link.attributes('href')).toBe('/prets/bbbbbbbbbbbbbbbb')
    expect(link.attributes('aria-label')).toBe('Voir le prêt BNP')
  })

  it('selects a loan when anywhere on its row is clicked, and highlights it', async () => {
    const fetchMock = stubApi({
      'GET /api/loans': () => json(loans),
      'GET /api/active-loan': () => json({ loanId: null }),
      'PUT /api/active-loan': (init) => json(JSON.parse(String(init.body))),
    })
    const { wrapper } = await mountHome()
    const rows = () => wrapper.findAll('[data-test="loan-row"]')
    expect(rows().some((r) => r.classes('home__loan--active'))).toBe(false)
    expect(wrapper.get('[data-test="no-loan-row"]').classes()).toContain('home__loan--active')

    // The row is the <label> of the radio: a click on its text selects it.
    expect(rows()[1].element.tagName).toBe('LABEL')
    await rows()[1].get('input[type="radio"]').setValue(true)
    await flushPromises()

    expect(rows()[1].classes()).toContain('home__loan--active')
    expect(rows()[0].classes()).not.toContain('home__loan--active')
    expect(wrapper.get('[data-test="no-loan-row"]').classes()).not.toContain('home__loan--active')
    const put = fetchMock.mock.calls.find(([, init]) => init?.method === 'PUT')
    expect(JSON.parse(String(put?.[1]?.body))).toEqual({ loanId: 'bbbbbbbbbbbbbbbb' })
  })

  it('opening a loan from its link does not change the selection', async () => {
    const fetchMock = stubApi({
      'GET /api/loans': () => json(loans),
      'GET /api/active-loan': () => json({ loanId: null }),
      'GET /api/loans/aaaaaaaaaaaaaaaa': () =>
        json({ error: { code: 'not_found', message: 'x' } }, 404),
    })
    const { wrapper } = await mountHome()

    await wrapper.findAll('[data-test="loan-row"]')[0].get('a').trigger('click')
    await flushPromises()

    expect(fetchMock.mock.calls.some(([, init]) => init?.method === 'PUT')).toBe(false)
  })

  it('selects « Aucun » until a loan is active', async () => {
    stubApi({
      'GET /api/loans': () => json(loans),
      'GET /api/active-loan': () => json({ loanId: null }),
    })
    const { wrapper } = await mountHome()

    const radios = wrapper.findAll('input[type="radio"]')
    expect(radios).toHaveLength(3)
    expect((wrapper.get('input[value=""]').element as HTMLInputElement).checked).toBe(true)
    expect(wrapper.get('fieldset legend').text()).toBe('Prêt actif')
  })

  it('shows the loan that is already active', async () => {
    stubApi({
      'GET /api/loans': () => json(loans),
      'GET /api/active-loan': () => json({ loanId: 'bbbbbbbbbbbbbbbb' }),
    })
    const { wrapper } = await mountHome()

    expect(
      (wrapper.get('input[value="bbbbbbbbbbbbbbbb"]').element as HTMLInputElement).checked,
    ).toBe(true)
  })

  it('saves the active loan when one is picked', async () => {
    const fetchMock = stubApi({
      'GET /api/loans': () => json(loans),
      'GET /api/active-loan': () => json({ loanId: null }),
      'PUT /api/active-loan': (init) => json(JSON.parse(String(init.body))),
    })
    const { wrapper } = await mountHome()

    await wrapper.get('input[value="aaaaaaaaaaaaaaaa"]').setValue(true)
    await flushPromises()

    const put = fetchMock.mock.calls.find(([, init]) => init?.method === 'PUT')
    expect(JSON.parse(String(put?.[1]?.body))).toEqual({ loanId: 'aaaaaaaaaaaaaaaa' })

    await wrapper.get('input[value=""]').setValue(true)
    await flushPromises()
    const lastPut = fetchMock.mock.calls.filter(([, init]) => init?.method === 'PUT').at(-1)
    expect(JSON.parse(String(lastPut?.[1]?.body))).toEqual({ loanId: null })
  })

  it('invites to create a loan when there is none', async () => {
    stubApi({
      'GET /api/loans': () => json([]),
      'GET /api/active-loan': () => json({ loanId: null }),
    })
    const { wrapper } = await mountHome()

    const pane = wrapper.get('[data-test="home-prets"]')
    expect(pane.text()).toContain('Aucune simulation de prêt.')
    expect(pane.get('a').attributes('href')).toBe('/prets/nouveau')
    expect(pane.find('fieldset').exists()).toBe(false)
  })

  it('links the app name in the header to the home page', async () => {
    stubApi({
      'GET /api/loans': () => json([]),
      'GET /api/active-loan': () => json({ loanId: null }),
    })
    const { wrapper } = await mountHome()

    expect(wrapper.get('header a.app-header__name').attributes('href')).toBe('/')
  })

  describe('Biens pane', () => {
    const property = {
      id: 'pppppppppppppppp',
      name: 'T2 Lyon 7e',
      priceCents: 15_000_000,
      monthlyRentCents: 70_000,
      dpe: 'D',
      netAfterTaxYieldPct: 1.75,
      monthlyNetIncomeCents: 23_799,
      cashFlowAfterTaxCents: null,
      loanId: '',
      loanName: '',
      warnings: [],
    }

    it('lists the biens with their own loans when no loan is active', async () => {
      const fetchMock = stubApi({
        'GET /api/loans': () => json(loans),
        'GET /api/active-loan': () => json({ loanId: null }),
        'GET /api/properties': () => json([property]),
      })
      const { wrapper } = await mountHome()

      const pane = wrapper.get('[data-test="home-biens"]')
      expect(pane.get('tbody a').text()).toBe('T2 Lyon 7e')
      expect(pane.find('[data-test="active-loan-note"]').exists()).toBe(false)
      expect(pane.get('[data-test="new-property"]').attributes('href')).toBe('/biens/nouveau')
      expect(fetchMock.mock.calls.some(([url]) => url === '/api/properties')).toBe(true)
    })

    it('applies the active loan to every bien', async () => {
      const financed = {
        ...property,
        cashFlowAfterTaxCents: -40_000,
        loanId: 'bbbbbbbbbbbbbbbb',
        loanName: 'BNP',
      }
      stubApi({
        'GET /api/loans': () => json(loans),
        'GET /api/active-loan': () => json({ loanId: 'bbbbbbbbbbbbbbbb' }),
        'GET /api/properties?activeLoanId=bbbbbbbbbbbbbbbb': () => json([financed]),
      })
      const { wrapper } = await mountHome()

      const pane = wrapper.get('[data-test="home-biens"]')
      expect(pane.get('[data-test="active-loan-note"]').text()).toBe(
        'Financement calculé avec le prêt « BNP » appliqué à chaque bien (montant emprunté : prix + frais de notaire).',
      )
      expect(plain(pane.text())).toContain('-400,00 €')
    })

    it('reloads the biens when the active loan changes', async () => {
      const fetchMock = stubApi({
        'GET /api/loans': () => json(loans),
        'GET /api/active-loan': () => json({ loanId: null }),
        'GET /api/properties': () => json([property]),
        'PUT /api/active-loan': (init) => json(JSON.parse(String(init.body))),
        'GET /api/properties?activeLoanId=aaaaaaaaaaaaaaaa': () => json([property]),
      })
      const { wrapper } = await mountHome()

      await wrapper.get('input[value="aaaaaaaaaaaaaaaa"]').setValue(true)
      await flushPromises()

      expect(
        fetchMock.mock.calls.some(
          ([url]) => url === '/api/properties?activeLoanId=aaaaaaaaaaaaaaaa',
        ),
      ).toBe(true)
      expect(wrapper.get('[data-test="active-loan-note"]').text()).toContain('« Crédit Mutuel »')
    })
  })
})
