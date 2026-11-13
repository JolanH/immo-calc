import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import DataSection from './DataSection.vue'

const current = {
  path: '/home/me/immo-calc.yaml',
  schemaVersion: 1,
  counts: { properties: 3, loans: 2 },
}
const created = {
  path: '/home/me/nouveau.yaml',
  schemaVersion: 1,
  counts: { properties: 0, loans: 0 },
}

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status })
}

async function mountWith(...responses: Response[]) {
  const fetchMock = vi.fn()
  fetchMock.mockResolvedValueOnce(json(current))
  for (const r of responses) fetchMock.mockResolvedValueOnce(r)
  vi.stubGlobal('fetch', fetchMock)
  const wrapper = mount(DataSection)
  await flushPromises()
  return { wrapper, fetchMock }
}

afterEach(() => vi.unstubAllGlobals())

describe('Nouveau fichier…', () => {
  it('creates the file through the native dialog and shows it empty', async () => {
    const { wrapper, fetchMock } = await mountWith(json(created))

    await wrapper.get('[data-test="new"]').trigger('click')
    await flushPromises()

    expect(fetchMock).toHaveBeenLastCalledWith('/api/data/new', { method: 'POST' })
    expect(wrapper.get('[data-test="data-path"]').text()).toBe('/home/me/nouveau.yaml')
    expect(wrapper.text()).toContain('0 bien, 0 prêt')
    expect(wrapper.get('[role="status"]').text()).toBe(
      'Nouveau fichier créé : /home/me/nouveau.yaml',
    )
  })

  it('shows the refusal when the chosen file already exists', async () => {
    const exists = {
      error: {
        code: 'file_exists',
        message: 'Ce fichier existe déjà, choisissez un autre nom ou utilisez « Ouvrir »',
        fields: { path: 'Ce fichier existe déjà' },
      },
    }
    const { wrapper } = await mountWith(json(exists, 400))

    await wrapper.get('[data-test="new"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain('utilisez « Ouvrir »')
    expect(wrapper.get('[data-test="data-path"]').text()).toBe('/home/me/immo-calc.yaml')
  })

  it('falls back to a typed path and creates the file there', async () => {
    const unavailable = {
      error: {
        code: 'picker_unavailable',
        message: 'Aucune fenêtre… saisissez le chemin du fichier.',
      },
    }
    const { wrapper, fetchMock } = await mountWith(json(unavailable, 400), json(created))

    await wrapper.get('[data-test="new"]').trigger('click')
    await flushPromises()
    const form = wrapper.get('[data-test="path-form"]')
    expect(form.get('button[type="submit"]').text()).toBe('Créer')
    await form.get('input').setValue('/home/me/nouveau.yaml')
    await form.trigger('submit')
    await flushPromises()

    expect(fetchMock).toHaveBeenLastCalledWith('/api/data/new', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: '{"path":"/home/me/nouveau.yaml"}',
    })
    expect(wrapper.get('[data-test="data-path"]').text()).toBe('/home/me/nouveau.yaml')
  })
})
