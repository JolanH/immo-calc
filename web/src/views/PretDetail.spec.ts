import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory } from 'vue-router'
import { createAppRouter } from '@/router'
import PretDetail from './PretDetail.vue'

const plain = (s: string) => s.replace(/[\u00a0\u202f]/g, ' ')

function simulation(overrides: Record<string, number> = {}) {
  return {
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
    ...overrides,
  }
}

async function mountPage() {
  const router = createAppRouter(createMemoryHistory())
  router.push('/prets/nouveau')
  await router.isReady()
  const wrapper = mount(PretDetail, { global: { plugins: [router] } })
  await flushPromises()
  return wrapper
}

beforeEach(() => {
  vi.useFakeTimers()
  vi.setSystemTime(new Date(2026, 10, 13)) // 13 November 2026
})

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe('PretDetail (new loan)', () => {
  it('starts with the documented defaults', async () => {
    vi.stubGlobal('fetch', vi.fn())
    const wrapper = await mountPage()

    expect(wrapper.get('h1').text()).toBe('Nouvelle simulation de prêt')
    expect((wrapper.get('#pret-duree').element as HTMLInputElement).value).toBe('20')
    expect((wrapper.get('#pret-echeance').element as HTMLSelectElement).value).toBe('12')
    expect((wrapper.get('#pret-echeance-annee').element as HTMLInputElement).value).toBe('2026')
    expect(wrapper.get('details summary').text()).toBe('Assurance et frais')
  })

  it('simulates while typing, without saving, and shows the results', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify(simulation())))
    vi.stubGlobal('fetch', fetchMock)
    const wrapper = await mountPage()

    await wrapper.get('#pret-montant').setValue('200 000')
    await wrapper.get('#pret-taux').setValue('3,5')
    vi.advanceTimersByTime(300)
    await flushPromises()

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/loans/simulate')
    expect(JSON.parse(init.body)).toMatchObject({
      principalCents: 20_000_000,
      annualRatePct: 3.5,
      durationMonths: 240,
      insuranceRatePct: 0,
      applicationFeeCents: 0,
      guaranteeFeeCents: 0,
      firstPaymentMonth: '2026-12',
    })
    const results = plain(wrapper.get('[data-test="results"]').text())
    expect(results).toContain('Mensualité totale')
    expect(results).toContain('1 159,92 €')
    expect(results).toContain('Coût total du crédit')
    expect(results).toContain('78 380,66 €')
    for (const label of [
      'Mensualité hors assurance',
      'Assurance mensuelle',
      'Intérêts totaux',
      "Coût de l'assurance",
      'Frais',
      'Montant total remboursé',
    ]) {
      expect(results).toContain(label)
    }
  })

  it('does not call the API while the amount is missing', async () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    const wrapper = await mountPage()

    await wrapper.get('#pret-taux').setValue('3,5')
    vi.advanceTimersByTime(1000)
    await flushPromises()

    expect(fetchMock).not.toHaveBeenCalled()
    expect(wrapper.get('[data-test="results"]').text()).toContain('Saisissez')
  })

  it('shows French field errors next to the fields', async () => {
    const error = {
      error: {
        code: 'validation_failed',
        message: 'Certains champs sont invalides',
        fields: { principalCents: 'Le montant doit être compris entre 1 000 € et 5 000 000 €' },
      },
    }
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response(JSON.stringify(error), { status: 400 })),
    )
    const wrapper = await mountPage()

    await wrapper.get('#pret-montant').setValue('50')
    await wrapper.get('#pret-taux').setValue('3,5')
    vi.advanceTimersByTime(300)
    await flushPromises()

    expect(wrapper.get('#pret-montant-error').text()).toBe(
      'Le montant doit être compris entre 1 000 € et 5 000 000 €',
    )
  })

  it('sends insurance and fees from the collapsible section', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify(simulation())))
    vi.stubGlobal('fetch', fetchMock)
    const wrapper = await mountPage()

    await wrapper.get('#pret-montant').setValue('200 000')
    await wrapper.get('#pret-taux').setValue('3,5')
    await wrapper.get('#pret-assurance').setValue('0,3')
    await wrapper.get('#pret-frais-dossier').setValue('1 000')
    await wrapper.get('#pret-frais-garantie').setValue('2 500')
    vi.advanceTimersByTime(300)
    await flushPromises()

    expect(JSON.parse(fetchMock.mock.calls.at(-1)![1].body)).toMatchObject({
      insuranceRatePct: 0.3,
      applicationFeeCents: 100_000,
      guaranteeFeeCents: 250_000,
    })
  })
})
