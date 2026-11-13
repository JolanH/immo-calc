import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory } from 'vue-router'
import { createAppRouter } from '@/router'
import App from '@/App.vue'

const plain = (s: string) => s.replace(/[\u00a0\u202f]/g, ' ')

const propertyResult = {
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
const property = {
  id: 'pppppppppppppppp',
  name: 'T2 Lyon 7e',
  priceCents: 15_000_000,
  dpe: 'D',
  monthlyRentCents: 70_000,
  monthlyRecoverableChargesCents: 5_000,
  annualPropertyTaxCents: 90_000,
  annualCondoFeesCents: 120_000,
  annualLandlordInsuranceCents: 15_000,
  agencyManaged: false,
  taxRegime: 'micro_foncier',
  loanId: '',
  createdAt: '',
  updatedAt: '',
  result: propertyResult,
  loan: null,
}

const loanResult = {
  monthlyPaymentCents: 1,
  monthlyInsuranceCents: 0,
  monthlyTotalCents: 1,
  totalInterestCents: 1,
  totalInsuranceCents: 0,
  totalFeesCents: 0,
  totalCostCents: 1,
  totalRepaidCents: 1,
  schedule: [],
  yearly: [],
}
const savedLoan = {
  id: 'llllllllllllllll',
  name: 'Prêt – T2 Lyon 7e',
  principalCents: 16_275_000,
  annualRatePct: 3.5,
  durationMonths: 240,
  insuranceRatePct: 0,
  applicationFeeCents: 0,
  guaranteeFeeCents: 0,
  firstPaymentMonth: '2026-12',
  createdAt: '',
  updatedAt: '',
  result: loanResult,
}

type FetchLike = (url: string, init?: RequestInit) => Promise<Response>

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status })
}

function stubApi(routes: Record<string, (init: RequestInit) => Response>) {
  const fetchMock = vi.fn<FetchLike>((url, init = {}) => {
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

describe('« Créer un prêt pour ce bien »', () => {
  it('is offered on a saved property and opens a prefilled loan form', async () => {
    stubApi({
      'GET /api/loans': () => json([]),
      'GET /api/properties/pppppppppppppppp': () => json(property),
      'POST /api/properties/evaluate?id=pppppppppppppppp': () => json(propertyResult),
      'POST /api/loans/simulate': () => json(loanResult),
    })
    const { wrapper, router } = await mountAt('/biens/pppppppppppppppp')

    await wrapper.get('[data-test="create-loan"]').trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.fullPath).toBe('/prets/nouveau?bien=pppppppppppppppp')
    expect((wrapper.get('#pret-nom').element as HTMLInputElement).value).toBe('Prêt – T2 Lyon 7e')
    expect(plain((wrapper.get('#pret-montant').element as HTMLInputElement).value)).toBe('162 750')
  })

  it('creates the loan, links it to the property and offers a way back', async () => {
    const fetchMock = stubApi({
      'GET /api/loans': () => json([]),
      'GET /api/properties/pppppppppppppppp': () => json(property),
      'POST /api/loans/simulate': () => json(loanResult),
      'POST /api/loans': () => json(savedLoan, 201),
      'PUT /api/properties/pppppppppppppppp': (init) =>
        json({ ...property, ...JSON.parse(String(init.body)) }),
      'GET /api/properties?loanId=llllllllllllllll': () =>
        json([{ id: 'pppppppppppppppp', name: 'T2 Lyon 7e' }]),
    })
    const { wrapper, router } = await mountAt('/prets/nouveau?bien=pppppppppppppppp')
    await wrapper.get('#pret-taux').setValue('3,5')

    await wrapper.get('[data-test="save"]').trigger('click')
    await flushPromises()

    const put = fetchMock.mock.calls.find(([, init]) => init?.method === 'PUT')
    expect(JSON.parse(String(put?.[1]?.body))).toMatchObject({
      name: 'T2 Lyon 7e',
      loanId: 'llllllllllllllll',
      dpe: 'D',
    })
    expect(router.currentRoute.value.fullPath).toBe('/prets/llllllllllllllll?bien=pppppppppppppppp')
    const back = wrapper.get('[data-test="back-to-property"]')
    expect(back.text()).toBe('← Retour au bien')
    expect(back.attributes('href')).toBe('/biens/pppppppppppppppp')
    expect(wrapper.get('[role="status"]').text()).toBe(
      'Prêt enregistré et associé au bien « T2 Lyon 7e »',
    )
  })
})

describe('« Biens liés »', () => {
  const loanRoutes = {
    'GET /api/loans/llllllllllllllll': () => json(savedLoan),
    'POST /api/loans/simulate': () => json(loanResult),
  }

  it('lists the properties using the loan as links', async () => {
    stubApi({
      ...loanRoutes,
      'GET /api/properties?loanId=llllllllllllllll': () =>
        json([
          { id: 'pppppppppppppppp', name: 'T2 Lyon 7e' },
          { id: 'qqqqqqqqqqqqqqqq', name: 'Studio' },
        ]),
    })
    const { wrapper } = await mountAt('/prets/llllllllllllllll')

    const section = wrapper.get('[data-test="linked-properties"]')
    expect(section.get('h2').text()).toBe('Biens liés')
    expect(section.findAll('a').map((a) => [a.text(), a.attributes('href')])).toEqual([
      ['T2 Lyon 7e', '/biens/pppppppppppppppp'],
      ['Studio', '/biens/qqqqqqqqqqqqqqqq'],
    ])
  })

  it('says so when no property uses the loan', async () => {
    stubApi({ ...loanRoutes, 'GET /api/properties?loanId=llllllllllllllll': () => json([]) })
    const { wrapper } = await mountAt('/prets/llllllllllllllll')

    expect(wrapper.get('[data-test="linked-properties"]').text()).toContain(
      "Aucun bien n'utilise ce prêt.",
    )
  })

  it('is not shown for an unsaved loan', async () => {
    stubApi({ 'GET /api/loans': () => json([]) })
    const { wrapper } = await mountAt('/prets/nouveau')

    expect(wrapper.find('[data-test="linked-properties"]').exists()).toBe(false)
  })
})
