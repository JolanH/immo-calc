import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory } from 'vue-router'
import { createAppRouter } from '@/router'
import App from '@/App.vue'

const plain = (s: string) => s.replace(/[\u00a0\u202f]/g, ' ')

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
  financing: null as null | Record<string, number | null>,
  warnings: [] as { code: string; severity: string; message: string }[],
}

const loans = [
  {
    id: 'aaaaaaaaaaaaaaaa',
    name: 'Petit prêt',
    principalCents: 5_000_000,
    annualRatePct: 3,
    durationMonths: 240,
    monthlyTotalCents: 28_980,
    totalCostCents: 1,
  },
]

type FetchLike = (url: string, init?: RequestInit) => Promise<Response>

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status })
}

function stubApi(evaluate: (body: Record<string, unknown>) => Response = () => json(result)) {
  const fetchMock = vi.fn<FetchLike>((url, init = {}) => {
    const key = `${init.method ?? 'GET'} ${url}`
    if (key === 'GET /api/loans') return Promise.resolve(json(loans))
    if (key === 'GET /api/properties') return Promise.resolve(json([]))
    if (key === 'POST /api/properties/evaluate')
      return Promise.resolve(evaluate(JSON.parse(String(init.body))))
    throw new Error(`unexpected call ${key}`)
  })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

async function mountNew() {
  const router = createAppRouter(createMemoryHistory())
  router.push('/biens/nouveau')
  await router.isReady()
  const wrapper = mount(App, { global: { plugins: [router] } })
  await flushPromises()
  return wrapper
}

async function fillReferenceFlat(wrapper: Awaited<ReturnType<typeof mountNew>>) {
  await wrapper.get('#bien-prix').setValue('150 000')
  await wrapper.get('#bien-dpe').setValue('D')
  await wrapper.get('#bien-loyer').setValue('700')
  await wrapper.get('#bien-charges').setValue('50')
  await wrapper.get('#bien-taxe-fonciere').setValue('900')
  await wrapper.get('#bien-copro').setValue('1 200')
  await wrapper.get('#bien-pno').setValue('150')
  vi.advanceTimersByTime(300)
  await flushPromises()
}

beforeEach(() => vi.useFakeTimers())
afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe('BienDetail (new property)', () => {
  it('has a breadcrumb and the documented defaults', async () => {
    stubApi()
    const wrapper = await mountNew()

    expect(
      wrapper
        .get('nav[aria-label="Fil d\'Ariane"]')
        .findAll('li')
        .map((li) => li.text()),
    ).toEqual(['Accueil', 'Biens', 'Nouveau bien'])
    expect(wrapper.get('h1').text()).toBe('Nouveau bien')
    expect((wrapper.get('#bien-regime').element as HTMLSelectElement).value).toBe('micro_foncier')
    expect(wrapper.findAll('#bien-regime option').map((o) => o.text())).toEqual([
      'Micro-foncier (location nue)',
      'Réel foncier (location nue)',
      'Micro-BIC (location meublée)',
    ])
    expect(wrapper.findAll('#bien-dpe option').map((o) => o.text())).toEqual([
      'Choisir…',
      'A',
      'B',
      'C',
      'D',
      'E',
      'F',
      'G',
    ])
    expect((wrapper.get('#bien-agence').element as HTMLInputElement).checked).toBe(false)
    expect(wrapper.findAll('#bien-pret option').map((o) => o.text())).toEqual([
      'Aucun',
      'Petit prêt',
    ])
  })

  it('waits for the required fields before evaluating', async () => {
    const fetchMock = stubApi()
    const wrapper = await mountNew()

    await wrapper.get('#bien-prix').setValue('150 000')
    await wrapper.get('#bien-loyer').setValue('700')
    vi.advanceTimersByTime(1000)
    await flushPromises()

    expect(fetchMock.mock.calls.some(([url]) => url === '/api/properties/evaluate')).toBe(false)
    expect(wrapper.get('[data-test="results"]').text()).toContain(
      'Saisissez le prix, le loyer, la taxe foncière et la classe DPE',
    )
  })

  it('evaluates while typing and shows the four outputs', async () => {
    const fetchMock = stubApi()
    const wrapper = await mountNew()

    await fillReferenceFlat(wrapper)

    const call = fetchMock.mock.calls.find(([url]) => url === '/api/properties/evaluate')
    expect(JSON.parse(String(call?.[1]?.body))).toEqual({
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
    })
    const results = plain(wrapper.get('[data-test="results"]').text())
    expect(results).toContain('Revenus mensuels bruts')
    expect(results).toContain('700,00 €')
    expect(results).toContain('Revenus mensuels après impôts')
    expect(results).toContain('237,99 €')
    expect(results).toContain('Revenu annuel net')
    expect(results).toContain('2 855,92 €')
    const yields = plain(wrapper.get('[data-test="yields"]').text())
    expect(yields).toContain('Brute 5,16 %')
    expect(yields).toContain('Nette 3,32 %')
    expect(yields).toContain('Après impôts 1,75 %')
  })

  it('lists every figure of the calculation in « Détail du calcul »', async () => {
    stubApi()
    const wrapper = await mountNew()
    await fillReferenceFlat(wrapper)

    const detail = wrapper.get('[data-test="calc-detail"]')
    expect(detail.get('summary').text()).toBe('Détail du calcul')
    const text = plain(detail.text())
    for (const [label, value] of [
      ["Coût d'acquisition (prix + notaire)", '162 750,00 €'],
      ['Loyers encaissés (vacance déduite)', '7 700,00 €'],
      ['Taxe foncière', '900,00 €'],
      ['Charges de copropriété', '1 200,00 €'],
      ['Assurance PNO', '150,00 €'],
      ['Charges non récupérées pendant la vacance', '50,00 €'],
      ['Charges du propriétaire', '2 300,00 €'],
      ['Revenu annuel avant impôts', '5 400,00 €'],
      ['Revenu imposable', '5 390,00 €'],
      ['Impôt annuel', '2 544,08 €'],
    ]) {
      expect(text).toContain(label)
      expect(text).toContain(value)
    }
  })

  it('shows the warnings above the results', async () => {
    stubApi(() =>
      json({
        ...result,
        warnings: [
          {
            code: 'dpe',
            severity: 'serious',
            message:
              'Logement F : interdit à la location à partir du 1er janvier 2028 ; loyer gelé',
          },
        ],
      }),
    )
    const wrapper = await mountNew()
    await fillReferenceFlat(wrapper)

    const warning = wrapper.get('[data-test="warnings"] [role="alert"]')
    expect(warning.text()).toContain('Logement F')
    expect(warning.classes()).toContain('warning--serious')
  })

  it('shows the Financement card when a loan is selected', async () => {
    const fetchMock = stubApi((body) =>
      json({
        ...result,
        financing: body.loanId
          ? {
              monthlyPaymentCents: 28_980,
              cashFlowBeforeTaxCents: 16_020,
              cashFlowAfterTaxCents: 1_170,
              ownContributionCents: 11_375_000,
              returnOnContributionPct: 0.12,
            }
          : null,
      }),
    )
    const wrapper = await mountNew()
    await fillReferenceFlat(wrapper)
    expect(wrapper.find('[data-test="financing"]').exists()).toBe(false)

    await wrapper.get('#bien-pret').setValue('aaaaaaaaaaaaaaaa')
    vi.advanceTimersByTime(300)
    await flushPromises()

    expect(JSON.parse(String(fetchMock.mock.calls.at(-1)?.[1]?.body)).loanId).toBe(
      'aaaaaaaaaaaaaaaa',
    )
    const card = plain(wrapper.get('[data-test="financing"]').text())
    expect(card).toContain('Financement')
    expect(card).toContain('Mensualité')
    expect(card).toContain('289,80 €')
    expect(card).toContain('Cash-flow avant impôts')
    expect(card).toContain('160,20 €')
    expect(card).toContain('Cash-flow après impôts')
    expect(card).toContain('11,70 €')
    expect(card).toContain('Apport')
    expect(card).toContain('113 750,00 €')
    expect(card).toContain('Rendement sur apport')
    expect(card).toContain('0,12 %')
  })

  it('shows « n/a » when there is no own contribution', async () => {
    stubApi(() =>
      json({
        ...result,
        financing: {
          monthlyPaymentCents: 115_992,
          cashFlowBeforeTaxCents: -70_992,
          cashFlowAfterTaxCents: -92_193,
          ownContributionCents: -3_725_000,
          returnOnContributionPct: null,
        },
      }),
    )
    const wrapper = await mountNew()
    await fillReferenceFlat(wrapper)

    expect(wrapper.get('[data-test="financing"]').text()).toContain('n/a')
  })

  it('shows French field errors next to the fields', async () => {
    stubApi(() =>
      json(
        {
          error: {
            code: 'validation_failed',
            message: 'Certains champs sont invalides',
            fields: { priceCents: 'Le prix doit être compris entre 10 000 € et 5 000 000 €' },
          },
        },
        400,
      ),
    )
    const wrapper = await mountNew()
    await fillReferenceFlat(wrapper)

    expect(wrapper.get('#bien-prix-error').text()).toBe(
      'Le prix doit être compris entre 10 000 € et 5 000 000 €',
    )
  })

  it('links to the selected loan', async () => {
    stubApi()
    const wrapper = await mountNew()
    expect(wrapper.find('[data-test="open-loan"]').exists()).toBe(false)

    await wrapper.get('#bien-pret').setValue('aaaaaaaaaaaaaaaa')

    const link = wrapper.get('[data-test="open-loan"]')
    expect(link.text()).toBe('Voir le prêt →')
    expect(link.attributes('href')).toBe('/prets/aaaaaaaaaaaaaaaa')
  })
})
