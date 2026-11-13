import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import AmortizationTable from './AmortizationTable.vue'
import type { Installment, YearRow } from '@/api/loans'

const plain = (s: string) => s.replace(/[\u00a0\u202f]/g, ' ')

const schedule: Installment[] = [
  {
    index: 1,
    month: '2026-12',
    paymentCents: 115992,
    interestCents: 58333,
    principalCents: 57659,
    insuranceCents: 5000,
    remainingCents: 19942341,
  },
  {
    index: 2,
    month: '2027-01',
    paymentCents: 115992,
    interestCents: 58165,
    principalCents: 57827,
    insuranceCents: 5000,
    remainingCents: 19884514,
  },
]

const yearly: YearRow[] = [
  {
    year: 2026,
    installments: 1,
    paymentCents: 115992,
    interestCents: 58333,
    principalCents: 57659,
    insuranceCents: 5000,
    remainingCents: 19942341,
  },
  {
    year: 2027,
    installments: 12,
    paymentCents: 1391904,
    interestCents: 692000,
    principalCents: 699904,
    insuranceCents: 60000,
    remainingCents: 19242437,
  },
]

function rows(wrapper: ReturnType<typeof mount>) {
  return wrapper.findAll('tbody tr').map((tr) => tr.findAll('td, th').map((c) => plain(c.text())))
}

describe('AmortizationTable', () => {
  it('shows the yearly view by default', () => {
    const wrapper = mount(AmortizationTable, { props: { schedule, yearly } })

    expect(wrapper.get('h2').text()).toBe("Tableau d'amortissement")
    expect(wrapper.findAll('thead th').map((th) => th.text())).toEqual([
      'Année',
      'Échéance',
      'Intérêts',
      'Capital remboursé',
      'Assurance',
      'Capital restant dû',
    ])
    expect(rows(wrapper)).toEqual([
      ['2026 (1 échéance)', '1 209,92 €', '583,33 €', '576,59 €', '50,00 €', '199 423,41 €'],
      ['2027', '14 519,04 €', '6 920,00 €', '6 999,04 €', '600,00 €', '192 424,37 €'],
    ])
    expect(wrapper.get('[data-test="view-annual"]').attributes('aria-pressed')).toBe('true')
  })

  it('labels partial years with their number of installments', () => {
    const partial = [{ ...yearly[1], installments: 2 }]
    const wrapper = mount(AmortizationTable, { props: { schedule, yearly: partial } })

    expect(rows(wrapper)[0][0]).toBe('2027 (2 échéances)')
  })

  it('switches to the monthly view', async () => {
    const wrapper = mount(AmortizationTable, { props: { schedule, yearly } })

    await wrapper.get('[data-test="view-monthly"]').trigger('click')

    expect(wrapper.findAll('thead th')[0].text()).toBe('Mois')
    expect(rows(wrapper)).toEqual([
      ['déc. 2026', '1 209,92 €', '583,33 €', '576,59 €', '50,00 €', '199 423,41 €'],
      ['janv. 2027', '1 209,92 €', '581,65 €', '578,27 €', '50,00 €', '198 845,14 €'],
    ])
    expect(wrapper.get('[data-test="view-monthly"]').attributes('aria-pressed')).toBe('true')
    expect(wrapper.get('[data-test="view-annual"]').attributes('aria-pressed')).toBe('false')
  })
})
