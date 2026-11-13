import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory } from 'vue-router'
import { createAppRouter } from '@/router'
import App from '@/App.vue'

const result = {
  monthlyPaymentCents: 115992,
  monthlyInsuranceCents: 0,
  monthlyTotalCents: 115992,
  totalInterestCents: 7838066,
  totalInsuranceCents: 0,
  totalFeesCents: 0,
  totalCostCents: 7838066,
  totalRepaidCents: 27838066,
  schedule: [],
  yearly: [],
}

const saved = {
  id: 'aaaaaaaaaaaaaaaa',
  name: 'Appart Lyon',
  principalCents: 20_000_000,
  annualRatePct: 3.5,
  durationMonths: 240,
  insuranceRatePct: 0,
  applicationFeeCents: 0,
  guaranteeFeeCents: 0,
  firstPaymentMonth: '2026-12',
  createdAt: '2026-11-12T20:41:00+01:00',
  updatedAt: '2026-11-12T20:41:00+01:00',
  result,
}
const copy = { ...saved, id: 'bbbbbbbbbbbbbbbb', name: 'Appart Lyon (copie)' }

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status })
}

function stubApi(routes: Record<string, () => Response>) {
  const fetchMock = vi.fn((url: string, init: RequestInit = {}) => {
    const key = `${init.method ?? 'GET'} ${url}`
    const handler = routes[key]
    if (!handler) throw new Error(`unexpected call ${key}`)
    return Promise.resolve(handler())
  })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

async function mountAt(path: string) {
  const router = createAppRouter(createMemoryHistory())
  router.push(path)
  await router.isReady()
  const wrapper = mount(App, { global: { plugins: [router] } })
  await flushPromises()
  return { wrapper, router }
}

afterEach(() => vi.unstubAllGlobals())

describe('Dupliquer', () => {
  it('is not offered for an unsaved loan', async () => {
    stubApi({ 'GET /api/loans': () => json([]) })
    const { wrapper } = await mountAt('/prets/nouveau')

    expect(wrapper.find('[data-test="duplicate"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="delete"]').exists()).toBe(false)
  })

  it('creates a copy and opens it', async () => {
    const fetchMock = stubApi({
      'GET /api/loans/aaaaaaaaaaaaaaaa': () => json(saved),
      'POST /api/loans/simulate': () => json(result),
      'POST /api/loans/aaaaaaaaaaaaaaaa/duplicate': () => json(copy, 201),
      'GET /api/loans/bbbbbbbbbbbbbbbb': () => json(copy),
    })
    const { wrapper, router } = await mountAt('/prets/aaaaaaaaaaaaaaaa')

    await wrapper.get('[data-test="duplicate"]').trigger('click')
    await flushPromises()

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/loans/aaaaaaaaaaaaaaaa/duplicate',
      expect.objectContaining({ method: 'POST' }),
    )
    expect(router.currentRoute.value.path).toBe('/prets/bbbbbbbbbbbbbbbb')
    expect(wrapper.get('h1').text()).toBe('Appart Lyon (copie)')
    expect(wrapper.get('[role="status"]').text()).toBe('Copie créée')
  })
})

describe('Supprimer', () => {
  it('asks for confirmation, deletes and goes back to the list', async () => {
    const fetchMock = stubApi({
      'GET /api/loans/aaaaaaaaaaaaaaaa': () => json(saved),
      'POST /api/loans/simulate': () => json(result),
      'DELETE /api/loans/aaaaaaaaaaaaaaaa': () => new Response(null, { status: 204 }),
      'GET /api/loans': () => json([]),
    })
    const confirmMock = vi.fn().mockReturnValue(true)
    vi.stubGlobal('confirm', confirmMock)
    const { wrapper, router } = await mountAt('/prets/aaaaaaaaaaaaaaaa')

    await wrapper.get('[data-test="delete"]').trigger('click')
    await flushPromises()

    expect(confirmMock).toHaveBeenCalledWith(
      'Supprimer le prêt « Appart Lyon » ? Cette action est définitive.',
    )
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/loans/aaaaaaaaaaaaaaaa',
      expect.objectContaining({ method: 'DELETE' }),
    )
    expect(router.currentRoute.value.path).toBe('/prets')
  })

  it('does nothing when the user cancels', async () => {
    const fetchMock = stubApi({
      'GET /api/loans/aaaaaaaaaaaaaaaa': () => json(saved),
      'POST /api/loans/simulate': () => json(result),
    })
    vi.stubGlobal('confirm', vi.fn().mockReturnValue(false))
    const { wrapper, router } = await mountAt('/prets/aaaaaaaaaaaaaaaa')

    await wrapper.get('[data-test="delete"]').trigger('click')
    await flushPromises()

    expect(fetchMock.mock.calls.some(([, init]) => init?.method === 'DELETE')).toBe(false)
    expect(router.currentRoute.value.path).toBe('/prets/aaaaaaaaaaaaaaaa')
  })

  it('shows why the loan cannot be deleted', async () => {
    const inUse = {
      error: { code: 'loan_in_use', message: 'Ce prêt est utilisé par le bien « T2 Lyon 7e »' },
    }
    stubApi({
      'GET /api/loans/aaaaaaaaaaaaaaaa': () => json(saved),
      'POST /api/loans/simulate': () => json(result),
      'DELETE /api/loans/aaaaaaaaaaaaaaaa': () => json(inUse, 409),
    })
    vi.stubGlobal('confirm', vi.fn().mockReturnValue(true))
    const { wrapper, router } = await mountAt('/prets/aaaaaaaaaaaaaaaa')

    await wrapper.get('[data-test="delete"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toBe(
      'Ce prêt est utilisé par le bien « T2 Lyon 7e »',
    )
    expect(router.currentRoute.value.path).toBe('/prets/aaaaaaaaaaaaaaaa')
  })

  it('does not ask about unsaved changes once the loan is deleted', async () => {
    stubApi({
      'GET /api/loans/aaaaaaaaaaaaaaaa': () => json(saved),
      'POST /api/loans/simulate': () => json(result),
      'DELETE /api/loans/aaaaaaaaaaaaaaaa': () => new Response(null, { status: 204 }),
      'GET /api/loans': () => json([]),
    })
    const confirmMock = vi.fn().mockReturnValue(true)
    vi.stubGlobal('confirm', confirmMock)
    const { wrapper, router } = await mountAt('/prets/aaaaaaaaaaaaaaaa')
    await wrapper.get('#pret-taux').setValue('3,1')

    await wrapper.get('[data-test="delete"]').trigger('click')
    await flushPromises()

    expect(confirmMock).toHaveBeenCalledTimes(1) // only the delete confirmation
    expect(router.currentRoute.value.path).toBe('/prets')
  })
})
