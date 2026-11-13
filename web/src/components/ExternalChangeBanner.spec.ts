import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ExternalChangeBanner from './ExternalChangeBanner.vue'
import { request } from '@/api/client'
import { externalChange } from '@/api/externalChange'

const conflict = {
  error: {
    code: 'file_changed_externally',
    message: "Le fichier a été modifié en dehors d'immo-calc",
  },
}

beforeEach(() => {
  externalChange.detected = false
})

afterEach(() => vi.unstubAllGlobals())

describe('ExternalChangeBanner', () => {
  it('is hidden until a request reports an external change', async () => {
    const wrapper = mount(ExternalChangeBanner)
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)

    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response(JSON.stringify(conflict), { status: 409 })),
    )
    await request('PUT', '/api/loans/a', {}).catch(() => {})
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain(
      "Le fichier a été modifié en dehors d'immo-calc",
    )
  })

  it('is not shown for other errors', async () => {
    const wrapper = mount(ExternalChangeBanner)
    const other = { error: { code: 'loan_in_use', message: 'Prêt utilisé' } }
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response(JSON.stringify(other), { status: 409 })),
    )

    await request('DELETE', '/api/loans/a').catch(() => {})
    await flushPromises()

    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })

  it('reloads from disk then refreshes the page on Recharger', async () => {
    externalChange.detected = true
    const fetchMock = vi.fn().mockResolvedValue(new Response('{}', { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)
    const reload = vi.fn()
    vi.stubGlobal('location', { ...window.location, reload })
    const wrapper = mount(ExternalChangeBanner)

    await wrapper.get('button').trigger('click')
    await flushPromises()

    expect(fetchMock).toHaveBeenCalledWith('/api/data/reload', { method: 'POST' })
    expect(reload).toHaveBeenCalled()
  })

  it('keeps the banner and shows why when the edited file is invalid', async () => {
    externalChange.detected = true
    const invalid = {
      error: { code: 'invalid_file', message: 'Le fichier /x.yaml est invalide (ligne 3) : …' },
    }
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response(JSON.stringify(invalid), { status: 400 })),
    )
    const wrapper = mount(ExternalChangeBanner)

    await wrapper.get('button').trigger('click')
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain('(ligne 3)')
  })
})
