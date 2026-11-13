import { describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createMemoryHistory } from 'vue-router'
import App from './App.vue'
import { createAppRouter } from './router'

async function mountAt(path: string) {
  const router = createAppRouter(createMemoryHistory())
  router.push(path)
  await router.isReady()
  const wrapper = mount(App, { global: { plugins: [router] } })
  await flushPromises()
  return { wrapper, router }
}

describe('router', () => {
  it('redirects / to /biens', async () => {
    const { router, wrapper } = await mountAt('/')
    expect(router.currentRoute.value.path).toBe('/biens')
    expect(wrapper.find('h1').text()).toBe('Biens')
  })

  it.each([
    ['/biens', 'Biens'],
    ['/prets', 'Prêts'],
  ])('shows the placeholder page for %s', async (path, title) => {
    const { wrapper } = await mountAt(path)
    expect(wrapper.find('h1').text()).toBe(title)
    expect(wrapper.text()).toContain('Fonctionnalité à venir')
  })

  it('shows the Paramètres page with its Données section', async () => {
    const info = { path: '/x.yaml', schemaVersion: 1, counts: { properties: 0, loans: 0 } }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify(info))))
    const { wrapper } = await mountAt('/parametres')
    expect(wrapper.find('h1').text()).toBe('Paramètres')
    expect(wrapper.find('h2').text()).toBe('Données')
    vi.unstubAllGlobals()
  })

  it('shows a not-found page with a link back to Biens for unknown routes', async () => {
    const { wrapper, router } = await mountAt('/nimporte/quoi')
    expect(wrapper.find('h1').text()).toBe('Page introuvable')

    await wrapper.get('[data-test="back-to-biens"]').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/biens')
  })

  it('renders the header with the app name, the navigation and the Quitter button', async () => {
    const { wrapper } = await mountAt('/biens')
    const header = wrapper.get('header')
    expect(header.text()).toContain('immo-calc')
    expect(header.text()).toContain('Quitter')
    const links = wrapper.findAll('nav a').map((a) => [a.text(), a.attributes('href')])
    expect(links).toEqual([
      ['Biens', '/biens'],
      ['Prêts', '/prets'],
      ['Paramètres', '/parametres'],
    ])
  })
})
