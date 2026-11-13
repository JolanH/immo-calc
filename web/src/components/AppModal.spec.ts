import { afterEach, describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import AppModal from '@/components/AppModal.vue'

afterEach(() => {
  document.body.innerHTML = ''
})

function mountModal(open: boolean) {
  return mount(AppModal, {
    attachTo: document.body,
    props: { open, title: 'Mon titre' },
    slots: { default: '<p data-test="body">contenu</p>' },
  })
}

describe('AppModal', () => {
  it('renders nothing while closed', () => {
    const wrapper = mountModal(false)
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
  })

  it('is a labelled modal dialog showing its content', () => {
    const wrapper = mountModal(true)
    const dialog = wrapper.get('[role="dialog"]')
    expect(dialog.attributes('aria-modal')).toBe('true')
    const labelledBy = dialog.attributes('aria-labelledby')!
    expect(wrapper.get(`#${labelledBy}`).text()).toBe('Mon titre')
    expect(wrapper.get('[data-test="body"]').text()).toBe('contenu')
  })

  it('moves the focus to the close button when it opens', async () => {
    const wrapper = mountModal(false)
    await wrapper.setProps({ open: true })
    await nextTick()
    expect(document.activeElement).toBe(wrapper.get('[data-test="modal-close"]').element)
  })

  it('asks to close with the button, Escape or a click on the backdrop', async () => {
    const wrapper = mountModal(true)
    await wrapper.get('[data-test="modal-close"]').trigger('click')
    await wrapper.get('[role="dialog"]').trigger('keydown', { key: 'Escape' })
    await wrapper.get('[data-test="modal-backdrop"]').trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(3)
  })

  it('does not close when the click is inside the dialog', async () => {
    const wrapper = mountModal(true)
    await wrapper.get('[data-test="body"]').trigger('click')
    expect(wrapper.emitted('close')).toBeUndefined()
  })
})
