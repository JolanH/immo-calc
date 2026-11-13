import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import DataSection from './DataSection.vue'

const current = {
  path: '/home/me/immo-calc.yaml',
  schemaVersion: 1,
  counts: { properties: 1, loans: 1 },
}
const switched = {
  path: '/home/me/autre.yaml',
  schemaVersion: 1,
  counts: { properties: 4, loans: 2 },
}

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status })
}

const pickerUnavailable = {
  error: {
    code: 'picker_unavailable',
    message:
      "Aucune fenêtre de sélection de fichier n'est disponible : saisissez le chemin du fichier.",
  },
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

describe('Ouvrir un autre fichier…', () => {
  it('opens the native dialog through the API and shows the new file', async () => {
    const { wrapper, fetchMock } = await mountWith(json(switched))

    await wrapper.get('[data-test="open"]').trigger('click')
    await flushPromises()

    expect(fetchMock).toHaveBeenLastCalledWith('/api/data/open', { method: 'POST' })
    expect(wrapper.get('[data-test="data-path"]').text()).toBe('/home/me/autre.yaml')
    expect(wrapper.get('[role="status"]').text()).toBe('Fichier ouvert : /home/me/autre.yaml')
  })

  it('changes nothing when the dialog is cancelled', async () => {
    const { wrapper } = await mountWith(new Response(null, { status: 204 }))

    await wrapper.get('[data-test="open"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-test="data-path"]').text()).toBe('/home/me/immo-calc.yaml')
    expect(wrapper.find('[role="status"]').exists()).toBe(false)
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })

  it('shows why an invalid file was refused', async () => {
    const invalid = {
      error: { code: 'invalid_file', message: 'Le fichier /x.yaml est invalide (ligne 2) : …' },
    }
    const { wrapper } = await mountWith(json(invalid, 400))

    await wrapper.get('[data-test="open"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain('(ligne 2)')
    expect(wrapper.get('[data-test="data-path"]').text()).toBe('/home/me/immo-calc.yaml')
  })

  it('falls back to a typed path when no native dialog is available', async () => {
    const { wrapper, fetchMock } = await mountWith(json(pickerUnavailable, 400), json(switched))

    await wrapper.get('[data-test="open"]').trigger('click')
    await flushPromises()

    const form = wrapper.get('[data-test="path-form"]')
    expect(form.text()).toContain('saisissez le chemin du fichier')
    await form.get('input').setValue('/home/me/autre.yaml')
    await form.trigger('submit')
    await flushPromises()

    expect(fetchMock).toHaveBeenLastCalledWith('/api/data/open', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: '{"path":"/home/me/autre.yaml"}',
    })
    expect(wrapper.get('[data-test="data-path"]').text()).toBe('/home/me/autre.yaml')
    expect(wrapper.find('[data-test="path-form"]').exists()).toBe(false)
  })

  it('shows field errors on the typed path and keeps the form open', async () => {
    const relative = {
      error: {
        code: 'invalid_path',
        message: 'Indiquez le chemin complet du fichier',
        fields: { path: 'Chemin complet requis' },
      },
    }
    const { wrapper } = await mountWith(json(pickerUnavailable, 400), json(relative, 400))

    await wrapper.get('[data-test="open"]').trigger('click')
    await flushPromises()
    const form = wrapper.get('[data-test="path-form"]')
    await form.get('input').setValue('autre.yaml')
    await form.trigger('submit')
    await flushPromises()

    expect(wrapper.get('[data-test="path-form"]').text()).toContain('Chemin complet requis')
  })

  it('can cancel the typed-path form', async () => {
    const { wrapper } = await mountWith(json(pickerUnavailable, 400))

    await wrapper.get('[data-test="open"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-test="path-cancel"]').trigger('click')

    expect(wrapper.find('[data-test="path-form"]').exists()).toBe(false)
  })
})
