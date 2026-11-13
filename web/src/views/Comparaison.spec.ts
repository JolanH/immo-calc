import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory } from 'vue-router'
import { createAppRouter } from '@/router'
import App from '@/App.vue'

const plain = (s: string) => s.replace(/[\u00a0\u202f]/g, ' ')

function bien(id: string, name: string, loanId = '') {
  return {
    id,
    name,
    priceCents: 15_000_000,
    dpe: 'D',
    monthlyRentCents: 70_000,
    monthlyRecoverableChargesCents: 5_000,
    annualPropertyTaxCents: 90_000,
    annualCondoFeesCents: 120_000,
    annualLandlordInsuranceCents: 15_000,
    furnitureIncludedCents: 0,
    furnitureAddedCents: 0,
    agencyManaged: false,
    taxRegime: 'micro_foncier',
    loanId,
    createdAt: '',
    updatedAt: '',
    result: {},
    loan: null,
  }
}

const A = 'aaaaaaaaaaaaaaaa'
const B = 'bbbbbbbbbbbbbbbb'
const C = 'cccccccccccccccc'
const D = 'dddddddddddddddd'
const E = 'eeeeeeeeeeeeeeee'
const LOAN1 = '1111111111111111'
const LOAN2 = '2222222222222222'

const loans = [
  {
    id: LOAN1,
    name: 'Crédit Mutuel',
    principalCents: 1,
    annualRatePct: 3,
    durationMonths: 240,
    monthlyTotalCents: 1,
    totalCostCents: 1,
  },
  {
    id: LOAN2,
    name: 'BNP',
    principalCents: 1,
    annualRatePct: 4,
    durationMonths: 300,
    monthlyTotalCents: 1,
    totalCostCents: 1,
  },
]

/** An evaluation whose figures reveal which loan was used. */
function evaluation(loanId: string) {
  const financed = loanId !== ''
  return {
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
    netAfterCreditYieldPct: loanId === LOAN1 ? 1.12 : loanId === LOAN2 ? 0.4 : 1.75,
    financing: financed
      ? {
          monthlyPaymentCents: loanId === LOAN1 ? 28_980 : 50_000,
          cashFlowBeforeTaxCents: 1,
          cashFlowAfterTaxCents: loanId === LOAN1 ? 1_170 : -26_201,
          ownContributionCents: 11_375_000,
          returnOnContributionPct: 0.12,
        }
      : null,
    timeline: financed
      ? {
          startMonth: '2026-12',
          positiveCashFlowMonth: '2046-12',
          paybackMonth: '2060-03',
          paybackAtStart: false,
          points: [],
        }
      : null,
    warnings: [],
  }
}

type FetchLike = (url: string, init?: RequestInit) => Promise<Response>

function stubApi() {
  const biens: Record<string, ReturnType<typeof bien>> = {
    [A]: bien(A, 'T2 Lyon 7e', LOAN1),
    [B]: bien(B, 'Studio'),
    [C]: bien(C, 'Maison'),
    [D]: bien(D, 'Duplex'),
    [E]: bien(E, 'Loft'),
  }
  const fetchMock = vi.fn<FetchLike>((url, init = {}) => {
    const method = init.method ?? 'GET'
    const json = (body: unknown, status = 200) =>
      Promise.resolve(new Response(JSON.stringify(body), { status }))
    if (method === 'GET' && url === '/api/loans') return json(loans)
    if (method === 'GET' && url === '/api/properties')
      return json(Object.values(biens).map((b) => ({ id: b.id, name: b.name })))
    const detail = /^\/api\/properties\/(\w+)$/.exec(url)
    if (method === 'GET' && detail) {
      const b = biens[detail[1]]
      return b ? json(b) : json({ error: { code: 'not_found', message: 'Bien introuvable' } }, 404)
    }
    if (method === 'POST' && url.startsWith('/api/properties/evaluate?id=')) {
      return json(evaluation(JSON.parse(String(init.body)).loanId))
    }
    throw new Error(`unexpected call ${method} ${url}`)
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

const column = (wrapper: Awaited<ReturnType<typeof mountAt>>['wrapper'], i: number) =>
  wrapper.findAll('[data-test="compare-column"]')[i]

const cell = (wrapper: Awaited<ReturnType<typeof mountAt>>['wrapper'], row: string, i: number) =>
  plain(wrapper.get(`[data-test="row-${row}"]`).findAll('td')[i].text())

afterEach(() => vi.unstubAllGlobals())

describe('Comparaison', () => {
  it('has a breadcrumb under Biens', async () => {
    stubApi()
    const { wrapper } = await mountAt(`/biens/comparer?biens=${A},${B}`)

    expect(wrapper.get('h1').text()).toBe('Comparaison')
    expect(
      wrapper
        .get('nav[aria-label="Fil d\'Ariane"]')
        .findAll('li')
        .map((li) => li.text()),
    ).toEqual(['Accueil', 'Biens', 'Comparaison'])
  })

  it('shows one column per bien with every basic result', async () => {
    stubApi()
    const { wrapper } = await mountAt(`/biens/comparer?biens=${A},${B}`)

    expect(wrapper.findAll('[data-test="compare-column"]')).toHaveLength(2)
    expect(column(wrapper, 0).get('a').text()).toBe('T2 Lyon 7e')
    expect(column(wrapper, 0).get('a').attributes('href')).toBe(`/biens/${A}`)
    const labels = wrapper.findAll('tbody th').map((th) => th.text())
    for (const label of [
      'Prix',
      "Coût d'acquisition",
      'Loyer',
      'Régime fiscal',
      'DPE',
      'Revenus mensuels bruts',
      'Revenus mensuels après impôts',
      'Revenu annuel net',
      'Rentabilité brute',
      'Rentabilité nette',
      'Rentabilité nette après impôts',
      'Prêt',
      'Mensualité',
      'Cash-flow avant impôts',
      'Cash-flow après impôts',
      'Apport',
      'Rendement sur apport',
      'Cash-flow mensuel positif',
      'Apport récupéré',
      'Avertissements',
    ]) {
      expect(labels).toContain(label)
    }
    expect(cell(wrapper, 'price', 0)).toBe('150 000,00 €')
    expect(cell(wrapper, 'net-monthly', 1)).toBe('237,99 €')
    expect(cell(wrapper, 'after-tax-yield', 0)).toBe('1,75 %')
    expect(cell(wrapper, 'regime', 0)).toBe('Micro-foncier')
  })

  it("highlights « Rentabilité nette après crédit », following each column's loan", async () => {
    stubApi()
    const { wrapper } = await mountAt(`/biens/comparer?biens=${A},${B}`)

    const row = wrapper.get('[data-test="row-after-credit-yield"]')
    expect(row.get('th').text()).toBe('Rentabilité nette après crédit')
    expect(row.classes()).toContain('compare-kpi')
    expect(cell(wrapper, 'after-credit-yield', 0)).toBe('1,12 %')
    expect(cell(wrapper, 'after-credit-yield', 1)).toBe('1,75 %')

    await wrapper.findAll('[data-test="row-loan"] select')[1].setValue(LOAN2)
    await flushPromises()
    expect(cell(wrapper, 'after-credit-yield', 1)).toBe('0,40 %')
  })

  it("starts each column on the bien's own loan", async () => {
    const fetchMock = stubApi()
    const { wrapper } = await mountAt(`/biens/comparer?biens=${A},${B}`)

    const selects = wrapper.findAll('[data-test="row-loan"] select')
    expect((selects[0].element as HTMLSelectElement).value).toBe(LOAN1)
    expect((selects[1].element as HTMLSelectElement).value).toBe('')
    expect(cell(wrapper, 'cash-flow-after', 0)).toBe('11,70 €')
    expect(cell(wrapper, 'cash-flow-after', 1)).toBe('—')
    expect(cell(wrapper, 'payback', 0)).toBe('01/03/2060')
    const evaluated = fetchMock.mock.calls.filter(([url]) =>
      url.startsWith('/api/properties/evaluate'),
    )
    expect(evaluated.map(([url]) => url)).toEqual([
      `/api/properties/evaluate?id=${A}`,
      `/api/properties/evaluate?id=${B}`,
    ])
  })

  it('applies a distinct loan to one column only and keeps it in the URL', async () => {
    const fetchMock = stubApi()
    const { wrapper, router } = await mountAt(`/biens/comparer?biens=${A},${B}`)
    fetchMock.mockClear()

    await wrapper.findAll('[data-test="row-loan"] select')[1].setValue(LOAN2)
    await flushPromises()

    const calls = fetchMock.mock.calls
    expect(calls).toHaveLength(1)
    expect(calls[0][0]).toBe(`/api/properties/evaluate?id=${B}`)
    expect(JSON.parse(String(calls[0][1]?.body)).loanId).toBe(LOAN2)
    expect(calls.some(([, init]) => init?.method === 'PUT')).toBe(false) // the bien isn't changed
    expect(cell(wrapper, 'cash-flow-after', 1)).toBe('-262,01 €')
    expect(cell(wrapper, 'cash-flow-after', 0)).toBe('11,70 €')
    expect(router.currentRoute.value.query.prets).toBe(`${LOAN1},${LOAN2}`)
  })

  it('restores the loans from the URL', async () => {
    stubApi()
    const { wrapper } = await mountAt(`/biens/comparer?biens=${A},${B}&prets=,${LOAN2}`)

    const selects = wrapper.findAll('[data-test="row-loan"] select')
    expect((selects[0].element as HTMLSelectElement).value).toBe('')
    expect((selects[1].element as HTMLSelectElement).value).toBe(LOAN2)
    expect(cell(wrapper, 'cash-flow-after', 0)).toBe('—')
  })

  it('removes a column and adds another bien', async () => {
    stubApi()
    const { wrapper, router } = await mountAt(`/biens/comparer?biens=${A},${B}`)

    await column(wrapper, 0).get('[data-test="remove-column"]').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.query.biens).toBe(B)
    expect(wrapper.findAll('[data-test="compare-column"]')).toHaveLength(1)

    const add = wrapper.get('[data-test="add-bien"]')
    expect(add.findAll('option').map((o) => o.text())).toEqual([
      'Ajouter un bien…',
      'T2 Lyon 7e',
      'Maison',
      'Duplex',
      'Loft',
    ])
    await add.setValue(C)
    await flushPromises()
    expect(router.currentRoute.value.query.biens).toBe(`${B},${C}`)
    expect(column(wrapper, 1).get('a').text()).toBe('Maison')
  })

  it('keeps at most 4 biens and hides « Ajouter » when full', async () => {
    stubApi()
    const { wrapper } = await mountAt(`/biens/comparer?biens=${[A, B, C, D, E].join(',')}`)

    expect(wrapper.findAll('[data-test="compare-column"]')).toHaveLength(4)
    expect(wrapper.find('[data-test="add-bien"]').exists()).toBe(false)
  })

  it('drops duplicate biens', async () => {
    stubApi()
    const { wrapper } = await mountAt(`/biens/comparer?biens=${[A, B, A].join(',')}`)

    expect(wrapper.findAll('[data-test="compare-column"]')).toHaveLength(2)
  })

  it('shows « Bien introuvable » for a bien that no longer exists', async () => {
    stubApi()
    const { wrapper } = await mountAt(`/biens/comparer?biens=${A},ffffffffffffffff`)

    expect(column(wrapper, 1).text()).toContain('Bien introuvable')
  })

  it('invites to pick biens when none are given', async () => {
    stubApi()
    const { wrapper } = await mountAt('/biens/comparer')

    expect(wrapper.text()).toContain(
      'Sélectionnez de 2 à 4 biens dans la liste des biens pour les comparer.',
    )
  })
})
