import { describe, expect, it } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory } from 'vue-router'
import { createAppRouter } from '@/router'
import AppBreadcrumb from './AppBreadcrumb.vue'

async function mountCrumbs(items: { label: string; to?: string }[]) {
  const router = createAppRouter(createMemoryHistory())
  router.push('/prets/aaaaaaaaaaaaaaaa')
  await router.isReady()
  const wrapper = mount(AppBreadcrumb, { props: { items }, global: { plugins: [router] } })
  return { wrapper, router }
}

describe('AppBreadcrumb', () => {
  it('is a labelled navigation landmark', async () => {
    const { wrapper } = await mountCrumbs([
      { label: 'Prêts', to: '/prets' },
      { label: 'Appart Lyon' },
    ])

    expect(wrapper.get('nav').attributes('aria-label')).toBe("Fil d'Ariane")
    expect(wrapper.findAll('li').map((li) => li.text())).toEqual(['Prêts', 'Appart Lyon'])
  })

  it('links the parent items and marks the current page', async () => {
    const { wrapper } = await mountCrumbs([
      { label: 'Prêts', to: '/prets' },
      { label: 'Appart Lyon' },
    ])

    const links = wrapper.findAll('a')
    expect(links).toHaveLength(1)
    expect(links[0].attributes('href')).toBe('/prets')
    const current = wrapper.get('[aria-current="page"]')
    expect(current.text()).toBe('Appart Lyon')
    expect(current.element.tagName).not.toBe('A')
  })

  it('navigates back to the list', async () => {
    const { wrapper, router } = await mountCrumbs([
      { label: 'Prêts', to: '/prets' },
      { label: 'Appart Lyon' },
    ])

    await wrapper.get('a').trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.path).toBe('/prets')
  })
})
