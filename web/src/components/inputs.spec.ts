import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import MoneyInput from './MoneyInput.vue'
import RateInput from './RateInput.vue'
import MonthPicker from './MonthPicker.vue'
import ResultCard from './ResultCard.vue'

const plain = (s: string) => s.replace(/[\u00a0\u202f]/g, ' ')

describe('MoneyInput', () => {
  it('shows the amount in French and emits cents while typing', async () => {
    const wrapper = mount(MoneyInput, {
      props: { id: 'montant', label: 'Montant', modelValue: 20_000_000 },
    })
    const input = wrapper.get('input')
    expect(plain(input.element.value)).toBe('200 000')
    expect(wrapper.get('label').attributes('for')).toBe('montant')

    await input.setValue('150 000')

    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([15_000_000])
  })

  it('emits null for text that is not an amount', async () => {
    const wrapper = mount(MoneyInput, { props: { id: 'm', label: 'Montant', modelValue: 100 } })

    await wrapper.get('input').setValue('abc')

    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([null])
  })

  it('reformats on blur', async () => {
    const wrapper = mount(MoneyInput, { props: { id: 'm', label: 'Montant', modelValue: null } })
    const input = wrapper.get('input')

    await input.setValue('150000')
    await wrapper.setProps({ modelValue: 15_000_000 })
    await input.trigger('blur')

    expect(plain(input.element.value)).toBe('150 000')
  })

  it('does not rewrite what the user is typing when the value comes back', async () => {
    const wrapper = mount(MoneyInput, { props: { id: 'm', label: 'Montant', modelValue: null } })
    const input = wrapper.get('input')

    await input.setValue('1500,')
    await wrapper.setProps({ modelValue: 150_000 })

    expect(input.element.value).toBe('1500,')
  })

  it('shows the field error and marks the input invalid', () => {
    const wrapper = mount(MoneyInput, {
      props: {
        id: 'm',
        label: 'Montant',
        modelValue: 5000,
        error: 'Le montant doit être compris…',
      },
    })

    expect(wrapper.get('[role="alert"]').text()).toBe('Le montant doit être compris…')
    expect(wrapper.get('input').attributes('aria-invalid')).toBe('true')
    expect(wrapper.get('input').attributes('aria-describedby')).toBe('m-error')
  })
})

describe('RateInput', () => {
  it('shows the rate in French and emits the number', async () => {
    const wrapper = mount(RateInput, { props: { id: 't', label: 'Taux', modelValue: 3.5 } })
    const input = wrapper.get('input')
    expect(input.element.value).toBe('3,5')
    expect(wrapper.text()).toContain('%')

    await input.setValue('4,25')

    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([4.25])
  })
})

describe('MonthPicker', () => {
  it('shows the month in French and emits YYYY-MM', async () => {
    const wrapper = mount(MonthPicker, {
      props: { id: 'p', label: 'Première échéance', modelValue: '2026-12' },
    })
    const select = wrapper.get('select')
    expect(select.element.value).toBe('12')
    expect(select.find('option:checked').text()).toBe('décembre')
    expect((wrapper.get('input').element as HTMLInputElement).value).toBe('2026')

    await select.setValue('3')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['2026-03'])

    await wrapper.get('input').setValue('2027')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['2027-12'])
  })

  it('emits an empty value for an unusable year', async () => {
    const wrapper = mount(MonthPicker, {
      props: { id: 'p', label: 'Première échéance', modelValue: '2026-12' },
    })

    await wrapper.get('input').setValue('')

    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([''])
  })
})

describe('ResultCard', () => {
  it('shows the label and the formatted amount', () => {
    const wrapper = mount(ResultCard, { props: { label: 'Mensualité totale', cents: 120_992 } })

    expect(wrapper.text()).toContain('Mensualité totale')
    expect(plain(wrapper.text())).toContain('1 209,92 €')
  })

  it('can be highlighted', () => {
    const wrapper = mount(ResultCard, { props: { label: 'X', cents: 1, highlight: true } })
    expect(wrapper.classes()).toContain('result-card--highlight')
  })
})
