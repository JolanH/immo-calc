import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import DataSection from './DataSection.vue'

function stubInfo(body: unknown, status = 200) {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify(body), { status })))
}

afterEach(() => vi.unstubAllGlobals())

describe('DataSection', () => {
  it('shows the current file and the counts', async () => {
    stubInfo({
      path: '/home/me/immo-calc.yaml',
      schemaVersion: 1,
      counts: { properties: 3, loans: 1 },
    })

    const wrapper = mount(DataSection)
    await flushPromises()

    expect(wrapper.find('h2').text()).toBe('Données')
    expect(wrapper.get('[data-test="data-path"]').text()).toBe('/home/me/immo-calc.yaml')
    expect(wrapper.text()).toContain('3 biens')
    expect(wrapper.text()).toContain('1 prêt')
    expect(wrapper.find('[data-test="data-warning"]').exists()).toBe(false)
  })

  it('uses singular and plural forms', async () => {
    stubInfo({ path: '/x.yaml', schemaVersion: 1, counts: { properties: 1, loans: 0 } })

    const wrapper = mount(DataSection)
    await flushPromises()

    expect(wrapper.text()).toContain('1 bien')
    expect(wrapper.text()).not.toContain('1 biens')
    expect(wrapper.text()).toContain('0 prêt')
  })

  it('shows the startup warning and lets the user dismiss it', async () => {
    stubInfo({
      path: '/opt/immo-calc.yaml',
      schemaVersion: 1,
      counts: { properties: 0, loans: 0 },
      warning:
        'Le fichier /home/me/gone.yaml est introuvable, immo-calc utilise /opt/immo-calc.yaml',
    })

    const wrapper = mount(DataSection)
    await flushPromises()

    const banner = wrapper.get('[data-test="data-warning"]')
    expect(banner.text()).toContain('introuvable')
    await banner.get('button[aria-label="Fermer"]').trigger('click')
    expect(wrapper.find('[data-test="data-warning"]').exists()).toBe(false)
  })

  it('shows the API error message when the info call fails', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Failed to fetch')))

    const wrapper = mount(DataSection)
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain('immo-calc')
  })
})
