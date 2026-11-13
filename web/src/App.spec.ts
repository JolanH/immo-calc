import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory } from 'vue-router'
import App from './App.vue'
import { createAppRouter } from './router'

async function mountApp() {
  const router = createAppRouter(createMemoryHistory())
  // A page that loads nothing, so fetch only sees the shutdown call.
  router.push('/page-sans-donnees')
  await router.isReady()
  return mount(App, { global: { plugins: [router] } })
}

afterEach(() => vi.unstubAllGlobals())

describe('Quitter', () => {
  it('calls the shutdown API and shows the stopped message', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response('{}', { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)
    const wrapper = await mountApp()

    await wrapper.get('[data-test="quit"]').trigger('click')
    await flushPromises()

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/shutdown',
      expect.objectContaining({ method: 'POST' }),
    )
    expect(wrapper.text()).toContain('immo-calc est arrêté, vous pouvez fermer cet onglet')
    expect(wrapper.find('nav').exists()).toBe(false)
  })

  it('keeps the app and shows an error when the shutdown call fails', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Failed to fetch')))
    const wrapper = await mountApp()

    await wrapper.get('[data-test="quit"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('nav').exists()).toBe(true)
    expect(wrapper.get('[role="alert"]').text()).toContain('immo-calc')
  })
})
