import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory } from 'vue-router'
import { createAppRouter } from '@/router'
import App from '@/App.vue'

const result = {
  acquisitionCostCents: 16_275_000,
  collectedRentCents: 770_000,
  agencyFeesCents: 0,
  vacancyChargesCents: 5_000,
  ownerChargesCents: 230_000,
  incomeBeforeTaxCents: 540_000,
  taxableIncomeCents: 539_000,
  annualTaxCents: 254_408,
  netAnnualIncomeCents: 285_592,
  monthlyGrossIncomeCents: 70_000,
  monthlyNetIncomeCents: 23_799,
  grossYieldPct: 5.16,
  netYieldPct: 3.32,
  netAfterTaxYieldPct: 1.75,
  financing: null,
  warnings: [],
}

const saved = {
  id: 'aaaaaaaaaaaaaaaa',
  name: 'T2 Lyon 7e',
  priceCents: 15_000_000,
  dpe: 'D',
  monthlyRentCents: 70_000,
  monthlyRecoverableChargesCents: 5_000,
  annualPropertyTaxCents: 90_000,
  annualCondoFeesCents: 120_000,
  annualLandlordInsuranceCents: 15_000,
  furnitureIncludedCents: 0,
  furnitureAddedCents: 0,
  agencyManaged: true,
  taxRegime: 'reel_foncier',
  loanId: '',
  createdAt: '2026-11-13T10:00:00+01:00',
  updatedAt: '2026-11-13T10:00:00+01:00',
  result,
  loan: null,
}
const copy = { ...saved, id: 'bbbbbbbbbbbbbbbb', name: 'T2 Lyon 7e (copie)' }

type FetchLike = (url: string, init?: RequestInit) => Promise<Response>

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status })
}

function stubApi(routes: Record<string, (init: RequestInit) => Response>) {
  const fetchMock = vi.fn<FetchLike>((url, init = {}) => {
    const key = `${init.method ?? 'GET'} ${url}`
    if (key === 'GET /api/loans') return Promise.resolve(json([]))
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
  // Attached so that clicking « Enregistrer » submits the form, as in a browser.
  const wrapper = mount(App, { global: { plugins: [router] }, attachTo: document.body })
  await flushPromises()
  return { wrapper, router }
}

const evaluateSaved = { 'POST /api/properties/evaluate?id=aaaaaaaaaaaaaaaa': () => json(result) }

afterEach(() => {
  vi.unstubAllGlobals()
  document.body.innerHTML = ''
})

describe('an existing property', () => {
  it('loads its inputs, results and breadcrumb', async () => {
    stubApi({ 'GET /api/properties/aaaaaaaaaaaaaaaa': () => json(saved), ...evaluateSaved })
    const { wrapper } = await mountAt('/biens/aaaaaaaaaaaaaaaa')

    expect(wrapper.get('h1').text()).toBe('T2 Lyon 7e')
    expect(
      wrapper
        .get('nav[aria-label="Fil d\'Ariane"]')
        .findAll('li')
        .map((li) => li.text()),
    ).toEqual(['Accueil', 'Biens', 'T2 Lyon 7e'])
    expect((wrapper.get('#bien-nom').element as HTMLInputElement).value).toBe('T2 Lyon 7e')
    expect((wrapper.get('#bien-dpe').element as HTMLSelectElement).value).toBe('D')
    expect((wrapper.get('#bien-agence').element as HTMLInputElement).checked).toBe(true)
    expect((wrapper.get('#bien-regime').element as HTMLSelectElement).value).toBe('reel_foncier')
    expect(wrapper.get('[data-test="results"]').text()).toContain('Revenu annuel net')
  })

  it('evaluates with its id so the ceilings count it once', async () => {
    const fetchMock = stubApi({
      'GET /api/properties/aaaaaaaaaaaaaaaa': () => json(saved),
      ...evaluateSaved,
    })
    await mountAt('/biens/aaaaaaaaaaaaaaaa')

    expect(
      fetchMock.mock.calls.some(([url]) => url === '/api/properties/evaluate?id=aaaaaaaaaaaaaaaa'),
    ).toBe(true)
  })

  it('saves changes with PUT', async () => {
    const fetchMock = stubApi({
      'GET /api/properties/aaaaaaaaaaaaaaaa': () => json(saved),
      ...evaluateSaved,
      'PUT /api/properties/aaaaaaaaaaaaaaaa': (init) =>
        json({ ...saved, ...JSON.parse(String(init.body)) }),
    })
    const { wrapper } = await mountAt('/biens/aaaaaaaaaaaaaaaa')
    await wrapper.get('#bien-loyer').setValue('750')

    await wrapper.get('[data-test="save"]').trigger('click')
    await flushPromises()

    const put = fetchMock.mock.calls.find(([, init]) => init?.method === 'PUT')
    expect(JSON.parse(String(put?.[1]?.body))).toMatchObject({
      name: 'T2 Lyon 7e',
      monthlyRentCents: 75_000,
    })
    expect(wrapper.get('[role="status"]').text()).toBe('Bien enregistré')
  })

  it('duplicates and opens the copy', async () => {
    stubApi({
      'GET /api/properties/aaaaaaaaaaaaaaaa': () => json(saved),
      ...evaluateSaved,
      'POST /api/properties/aaaaaaaaaaaaaaaa/duplicate': () => json(copy, 201),
      'POST /api/properties/evaluate?id=bbbbbbbbbbbbbbbb': () => json(result),
    })
    const { wrapper, router } = await mountAt('/biens/aaaaaaaaaaaaaaaa')

    await wrapper.get('[data-test="duplicate"]').trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.path).toBe('/biens/bbbbbbbbbbbbbbbb')
    expect(wrapper.get('h1').text()).toBe('T2 Lyon 7e (copie)')
    expect(wrapper.get('[role="status"]').text()).toBe('Copie créée')
  })

  it('deletes after confirmation and goes back to the list', async () => {
    const fetchMock = stubApi({
      'GET /api/properties/aaaaaaaaaaaaaaaa': () => json(saved),
      ...evaluateSaved,
      'DELETE /api/properties/aaaaaaaaaaaaaaaa': () => new Response(null, { status: 204 }),
      'GET /api/properties': () => json([]),
    })
    const confirmMock = vi.fn().mockReturnValue(true)
    vi.stubGlobal('confirm', confirmMock)
    const { wrapper, router } = await mountAt('/biens/aaaaaaaaaaaaaaaa')

    await wrapper.get('[data-test="delete"]').trigger('click')
    await flushPromises()

    expect(confirmMock).toHaveBeenCalledWith(
      'Supprimer le bien « T2 Lyon 7e » ? Cette action est définitive.',
    )
    expect(fetchMock.mock.calls.some(([, init]) => init?.method === 'DELETE')).toBe(true)
    expect(router.currentRoute.value.path).toBe('/biens')
  })

  it('asks before leaving with unsaved changes', async () => {
    stubApi({
      'GET /api/properties/aaaaaaaaaaaaaaaa': () => json(saved),
      ...evaluateSaved,
      'GET /api/properties': () => json([]),
    })
    const confirmMock = vi.fn().mockReturnValue(false)
    vi.stubGlobal('confirm', confirmMock)
    const { wrapper, router } = await mountAt('/biens/aaaaaaaaaaaaaaaa')
    await wrapper.get('#bien-loyer').setValue('750')

    await router.push('/biens')
    await flushPromises()

    expect(confirmMock).toHaveBeenCalledWith(
      'Des modifications ne sont pas enregistrées. Quitter quand même ?',
    )
    expect(router.currentRoute.value.path).toBe('/biens/aaaaaaaaaaaaaaaa')
  })

  it('shows « Bien introuvable » for an unknown id', async () => {
    stubApi({
      'GET /api/properties/ffffffffffffffff': () =>
        json({ error: { code: 'not_found', message: 'Bien introuvable' } }, 404),
    })
    const { wrapper } = await mountAt('/biens/ffffffffffffffff')

    expect(wrapper.get('[role="alert"]').text()).toBe('Bien introuvable')
    expect(wrapper.find('form').exists()).toBe(false)
  })
})

describe('saving a new property', () => {
  it('creates it and stays on its page at its new URL', async () => {
    vi.useFakeTimers()
    const fetchMock = stubApi({
      'POST /api/properties/evaluate': () => json(result),
      'POST /api/properties': () => json(saved, 201),
    })
    const { wrapper, router } = await mountAt('/biens/nouveau')
    await wrapper.get('#bien-nom').setValue('T2 Lyon 7e')
    await wrapper.get('#bien-prix').setValue('150 000')
    await wrapper.get('#bien-dpe').setValue('D')
    await wrapper.get('#bien-loyer').setValue('700')
    await wrapper.get('#bien-taxe-fonciere').setValue('900')

    await wrapper.get('[data-test="save"]').trigger('click')
    await flushPromises()
    vi.useRealTimers()

    const post = fetchMock.mock.calls.find(
      ([url, init]) => url === '/api/properties' && init?.method === 'POST',
    )
    expect(JSON.parse(String(post?.[1]?.body))).toMatchObject({
      name: 'T2 Lyon 7e',
      priceCents: 15_000_000,
      dpe: 'D',
    })
    expect(router.currentRoute.value.path).toBe('/biens/aaaaaaaaaaaaaaaa')
    expect(wrapper.get('[role="status"]').text()).toBe('Bien enregistré')
    expect(wrapper.find('[data-test="duplicate"]').exists()).toBe(true)
  })

  it('shows field errors from the save', async () => {
    stubApi({
      'POST /api/properties': () =>
        json(
          {
            error: {
              code: 'validation_failed',
              message: 'x',
              fields: { name: 'Le nom est obligatoire' },
            },
          },
          400,
        ),
    })
    const { wrapper } = await mountAt('/biens/nouveau')

    await wrapper.get('[data-test="save"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('#bien-nom-error').text()).toBe('Le nom est obligatoire')
  })
})
