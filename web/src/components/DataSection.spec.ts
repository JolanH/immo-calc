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

  it('offers the export as a download link', async () => {
    stubInfo({ path: '/x.yaml', schemaVersion: 1, counts: { properties: 0, loans: 0 } })

    const wrapper = mount(DataSection)
    await flushPromises()

    const link = wrapper.get('[data-test="export"]')
    expect(link.text()).toBe('Exporter')
    expect(link.attributes('href')).toBe('/api/data/export')
    expect(link.attributes('download')).toBeDefined()
  })

  describe('Importer…', () => {
    const info = { path: '/x.yaml', schemaVersion: 1, counts: { properties: 2, loans: 1 } }

    async function chooseFile(wrapper: ReturnType<typeof mount>, content: string) {
      const input = wrapper.get('input[type="file"]')
      const file = new File([content], 'export.yaml', { type: 'application/yaml' })
      Object.defineProperty(input.element, 'files', { value: [file], configurable: true })
      await input.trigger('change')
      await flushPromises()
    }

    it('asks for confirmation, sends the file and shows the new counts', async () => {
      const fetchMock = vi
        .fn()
        .mockResolvedValueOnce(new Response(JSON.stringify(info)))
        .mockResolvedValueOnce(
          new Response(JSON.stringify({ ...info, counts: { properties: 5, loans: 4 } })),
        )
      vi.stubGlobal('fetch', fetchMock)
      const confirmMock = vi.fn().mockReturnValue(true)
      vi.stubGlobal('confirm', confirmMock)
      const wrapper = mount(DataSection)
      await flushPromises()

      await chooseFile(wrapper, 'schema_version: 1\n')

      expect(confirmMock).toHaveBeenCalledWith(
        'Toutes les données actuelles (2 biens, 1 prêt) seront remplacées par celles du fichier importé. Une sauvegarde du fichier actuel sera créée. Continuer ?',
      )
      expect(fetchMock).toHaveBeenLastCalledWith('/api/data/import', {
        method: 'POST',
        headers: { 'Content-Type': 'application/yaml' },
        body: 'schema_version: 1\n',
      })
      expect(wrapper.get('[role="status"]').text()).toBe('Import terminé')
      expect(wrapper.text()).toContain('5 biens')
    })

    it('does nothing when the user cancels', async () => {
      const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify(info)))
      vi.stubGlobal('fetch', fetchMock)
      vi.stubGlobal('confirm', vi.fn().mockReturnValue(false))
      const wrapper = mount(DataSection)
      await flushPromises()

      await chooseFile(wrapper, 'schema_version: 1\n')

      expect(fetchMock).toHaveBeenCalledTimes(1)
      expect(wrapper.find('[role="status"]').exists()).toBe(false)
    })

    it('shows the server message when the file is invalid', async () => {
      const error = {
        error: {
          code: 'invalid_import',
          message: 'Le fichier importé est invalide : section inconnue « foo »',
        },
      }
      vi.stubGlobal(
        'fetch',
        vi
          .fn()
          .mockResolvedValueOnce(new Response(JSON.stringify(info)))
          .mockResolvedValueOnce(new Response(JSON.stringify(error), { status: 400 })),
      )
      vi.stubGlobal('confirm', vi.fn().mockReturnValue(true))
      const wrapper = mount(DataSection)
      await flushPromises()

      await chooseFile(wrapper, 'foo: 1\n')

      expect(wrapper.get('[role="alert"]').text()).toContain('section inconnue « foo »')
      expect(wrapper.text()).toContain('2 biens')
    })
  })
})
