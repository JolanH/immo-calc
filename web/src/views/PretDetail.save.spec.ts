import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory } from 'vue-router'
import { createAppRouter } from '@/router'
import App from '@/App.vue'

const result = {
  monthlyPaymentCents: 115992,
  monthlyInsuranceCents: 5000,
  monthlyTotalCents: 120992,
  totalInterestCents: 7838066,
  totalInsuranceCents: 1200000,
  totalFeesCents: 350000,
  totalCostCents: 9388066,
  totalRepaidCents: 29038066,
  schedule: [],
  yearly: [],
}

const saved = {
  id: 'aaaaaaaaaaaaaaaa',
  name: 'Appart Lyon',
  principalCents: 20_000_000,
  annualRatePct: 3.5,
  durationMonths: 240,
  insuranceRatePct: 0.3,
  applicationFeeCents: 100_000,
  guaranteeFeeCents: 250_000,
  firstPaymentMonth: '2026-12',
  createdAt: '2026-11-12T20:41:00+01:00',
  updatedAt: '2026-11-12T20:41:00+01:00',
  result,
}

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status })
}

/** Routes fetch calls by method and path; unmatched calls fail the test. */
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

async function mountAt(path: string) {
  const router = createAppRouter(createMemoryHistory())
  router.push(path)
  await router.isReady()
  const wrapper = mount(App, { global: { plugins: [router] }, attachTo: document.body })
  await flushPromises()
  return { wrapper, router }
}

beforeEach(() => {
  vi.useFakeTimers()
  vi.setSystemTime(new Date(2026, 10, 13))
})

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
  document.body.innerHTML = ''
})

describe('saving a new loan', () => {
  it('proposes « Prêt N » as the name', async () => {
    stubApi({ 'GET /api/loans': () => json([{ id: 'x' }, { id: 'y' }]) })
    const { wrapper } = await mountAt('/prets/nouveau')

    expect((wrapper.get('#pret-nom').element as HTMLInputElement).value).toBe('Prêt 3')
  })

  it('creates the loan and moves to its page', async () => {
    const fetchMock = stubApi({
      'GET /api/loans': () => json([]),
      'POST /api/loans/simulate': () => json(result),
      'POST /api/loans': () => json(saved, 201),
      'GET /api/loans/aaaaaaaaaaaaaaaa': () => json(saved),
    })
    const { wrapper, router } = await mountAt('/prets/nouveau')
    await wrapper.get('#pret-nom').setValue('Appart Lyon')
    await wrapper.get('#pret-montant').setValue('200 000')
    await wrapper.get('#pret-taux').setValue('3,5')

    await wrapper.get('[data-test="save"]').trigger('click')
    await flushPromises()

    const post = fetchMock.mock.calls.find(
      ([url, init]) => url === '/api/loans' && init?.method === 'POST',
    )
    expect(JSON.parse(String(post?.[1]?.body))).toMatchObject({
      name: 'Appart Lyon',
      principalCents: 20_000_000,
      annualRatePct: 3.5,
      durationMonths: 240,
      firstPaymentMonth: '2026-12',
    })
    expect(router.currentRoute.value.path).toBe('/prets/aaaaaaaaaaaaaaaa')
    expect(wrapper.get('[role="status"]').text()).toBe('Prêt enregistré')
  })

  it('shows field errors from the save and stays on the page', async () => {
    const invalid = {
      error: {
        code: 'validation_failed',
        message: 'Certains champs sont invalides',
        fields: { name: 'Le nom est obligatoire' },
      },
    }
    stubApi({
      'GET /api/loans': () => json([]),
      'POST /api/loans/simulate': () => json(result),
      'POST /api/loans': () => json(invalid, 400),
    })
    const { wrapper, router } = await mountAt('/prets/nouveau')
    await wrapper.get('#pret-nom').setValue('')
    await wrapper.get('#pret-montant').setValue('200 000')
    await wrapper.get('#pret-taux').setValue('3,5')

    await wrapper.get('[data-test="save"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('#pret-nom-error').text()).toBe('Le nom est obligatoire')
    expect(router.currentRoute.value.path).toBe('/prets/nouveau')
  })
})

describe('an existing loan', () => {
  it('loads its inputs and results', async () => {
    stubApi({
      'GET /api/loans/aaaaaaaaaaaaaaaa': () => json(saved),
      'POST /api/loans/simulate': () => json(result),
    })
    const { wrapper } = await mountAt('/prets/aaaaaaaaaaaaaaaa')

    expect(wrapper.get('h1').text()).toBe('Appart Lyon')
    expect((wrapper.get('#pret-nom').element as HTMLInputElement).value).toBe('Appart Lyon')
    expect(
      (wrapper.get('#pret-montant').element as HTMLInputElement).value.replace(/\s/g, ' '),
    ).toBe('200 000')
    expect((wrapper.get('#pret-assurance').element as HTMLInputElement).value).toBe('0,3')
    expect(wrapper.get('[data-test="results"]').text()).toContain('Coût total du crédit')
  })

  it('saves changes with PUT', async () => {
    const fetchMock = stubApi({
      'GET /api/loans/aaaaaaaaaaaaaaaa': () => json(saved),
      'POST /api/loans/simulate': () => json(result),
      'PUT /api/loans/aaaaaaaaaaaaaaaa': (init) =>
        json({ ...saved, ...JSON.parse(String(init.body)) }),
    })
    const { wrapper } = await mountAt('/prets/aaaaaaaaaaaaaaaa')
    await wrapper.get('#pret-taux').setValue('3,1')

    await wrapper.get('[data-test="save"]').trigger('click')
    await flushPromises()

    const put = fetchMock.mock.calls.find(([, init]) => init?.method === 'PUT')
    expect(JSON.parse(String(put?.[1]?.body))).toMatchObject({
      annualRatePct: 3.1,
      name: 'Appart Lyon',
    })
    expect(wrapper.get('[role="status"]').text()).toBe('Prêt enregistré')
  })

  it('shows « Prêt introuvable » for an unknown id', async () => {
    const notFound = { error: { code: 'not_found', message: 'Prêt introuvable' } }
    stubApi({ 'GET /api/loans/ffffffffffffffff': () => json(notFound, 404) })
    const { wrapper } = await mountAt('/prets/ffffffffffffffff')

    expect(wrapper.get('[role="alert"]').text()).toBe('Prêt introuvable')
    expect(wrapper.find('form').exists()).toBe(false)
  })
})

describe('unsaved changes', () => {
  it('asks before leaving with unsaved changes, and stays when refused', async () => {
    stubApi({
      'GET /api/loans/aaaaaaaaaaaaaaaa': () => json(saved),
      'POST /api/loans/simulate': () => json(result),
      'GET /api/loans': () => json([]),
    })
    const confirmMock = vi.fn().mockReturnValue(false)
    vi.stubGlobal('confirm', confirmMock)
    const { wrapper, router } = await mountAt('/prets/aaaaaaaaaaaaaaaa')
    await wrapper.get('#pret-taux').setValue('3,1')

    await router.push('/prets')
    await flushPromises()

    expect(confirmMock).toHaveBeenCalledWith(
      'Des modifications ne sont pas enregistrées. Quitter quand même ?',
    )
    expect(router.currentRoute.value.path).toBe('/prets/aaaaaaaaaaaaaaaa')
  })

  it('leaves without asking when nothing changed', async () => {
    stubApi({
      'GET /api/loans/aaaaaaaaaaaaaaaa': () => json(saved),
      'POST /api/loans/simulate': () => json(result),
      'GET /api/loans': () => json([]),
    })
    const confirmMock = vi.fn()
    vi.stubGlobal('confirm', confirmMock)
    const { router } = await mountAt('/prets/aaaaaaaaaaaaaaaa')

    await router.push('/prets')
    await flushPromises()

    expect(confirmMock).not.toHaveBeenCalled()
    expect(router.currentRoute.value.path).toBe('/prets')
  })

  it('does not ask after saving', async () => {
    stubApi({
      'GET /api/loans/aaaaaaaaaaaaaaaa': () => json(saved),
      'POST /api/loans/simulate': () => json(result),
      'PUT /api/loans/aaaaaaaaaaaaaaaa': (init) =>
        json({ ...saved, ...JSON.parse(String(init.body)) }),
      'GET /api/loans': () => json([]),
    })
    const confirmMock = vi.fn()
    vi.stubGlobal('confirm', confirmMock)
    const { wrapper, router } = await mountAt('/prets/aaaaaaaaaaaaaaaa')
    await wrapper.get('#pret-taux').setValue('3,1')
    await wrapper.get('[data-test="save"]').trigger('click')
    await flushPromises()

    await router.push('/prets')
    await flushPromises()

    expect(confirmMock).not.toHaveBeenCalled()
  })

  it('warns the browser before closing the tab with unsaved changes', async () => {
    stubApi({
      'GET /api/loans/aaaaaaaaaaaaaaaa': () => json(saved),
      'POST /api/loans/simulate': () => json(result),
    })
    const { wrapper } = await mountAt('/prets/aaaaaaaaaaaaaaaa')
    await wrapper.get('#pret-taux').setValue('3,1')

    const event = new Event('beforeunload', { cancelable: true })
    window.dispatchEvent(event)

    expect(event.defaultPrevented).toBe(true)
  })
})
