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

  it('lists the loans with their monthly total and a link to each', async () => {
    stubApi({
      'GET /api/loans': () => json(loans),
      'GET /api/active-loan': () => json({ loanId: null }),
    })
    const { wrapper } = await mountHome()

    const pane = wrapper.get('[data-test="home-prets"]')
    expect(plain(pane.text())).toContain('Crédit Mutuel')
    expect(plain(pane.text())).toContain('1 209,92 €')
    expect(pane.findAll('a').map((a) => a.attributes('href'))).toContain('/prets/bbbbbbbbbbbbbbbb')
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
    expect(JSON.parse(String(fetchMock.mock.calls.at(-1)?.[1]?.body))).toEqual({ loanId: null })
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
})
