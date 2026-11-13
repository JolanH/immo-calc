import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import SettingsForm from './SettingsForm.vue'

const defaults = {
  marginalTaxRatePct: 30,
  socialChargesUnfurnishedPct: 17.2,
  socialChargesFurnishedPct: 18.6,
  notaryFeePct: 8.5,
  vacancyMonthsPerYear: 1,
  agencyFeePct: 7,
  accentColor: '#8b1a1a',
}

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status })
}

function stubApi(put?: (body: unknown) => Response) {
  const fetchMock = vi.fn((url: string, init: RequestInit = {}) => {
    if ((init.method ?? 'GET') === 'GET') return Promise.resolve(json(defaults))
    return Promise.resolve(put ? put(JSON.parse(String(init.body))) : json({}))
  })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

const value = (wrapper: ReturnType<typeof mount>, id: string) =>
  (wrapper.get(`#${id}`).element as HTMLInputElement).value

afterEach(() => vi.unstubAllGlobals())

describe('SettingsForm', () => {
  it('shows the current settings in French', async () => {
    stubApi()
    const wrapper = mount(SettingsForm)
    await flushPromises()

    expect(wrapper.get('h2').text()).toBe('Hypothèses et fiscalité')
    expect(value(wrapper, 'param-tmi')).toBe('30')
    expect(wrapper.findAll('#param-tmi option').map((o) => o.text())).toEqual([
      '0 %',
      '11 %',
      '30 %',
      '41 %',
      '45 %',
    ])
    expect(value(wrapper, 'param-ps-nue')).toBe('17,2')
    expect(value(wrapper, 'param-ps-meublee')).toBe('18,6')
    expect(value(wrapper, 'param-notaire')).toBe('8,5')
    expect(value(wrapper, 'param-vacance')).toBe('1')
    expect(value(wrapper, 'param-gestion')).toBe('7')
    expect(wrapper.text()).toContain('Prélèvements sociaux — location nue')
    expect(wrapper.text()).toContain('Prélèvements sociaux — location meublée')
  })

  it('saves the edited settings', async () => {
    const fetchMock = stubApi((body) => json(body))
    const wrapper = mount(SettingsForm)
    await flushPromises()

    await wrapper.get('#param-tmi').setValue('41')
    await wrapper.get('#param-vacance').setValue('1,5')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    const put = fetchMock.mock.calls.find(([, init]) => init?.method === 'PUT')
    expect(put?.[0]).toBe('/api/settings')
    expect(JSON.parse(String(put?.[1]?.body))).toEqual({
      ...defaults,
      marginalTaxRatePct: 41,
      vacancyMonthsPerYear: 1.5,
    })
    expect(wrapper.get('[role="status"]').text()).toBe('Paramètres enregistrés')
  })

  it('shows French field errors', async () => {
    stubApi(() =>
      json(
        {
          error: {
            code: 'validation_failed',
            message: 'Certains champs sont invalides',
            fields: {
              vacancyMonthsPerYear:
                'La vacance doit être comprise entre 0 et 12 mois, par demi-mois',
            },
          },
        },
        400,
      ),
    )
    const wrapper = mount(SettingsForm)
    await flushPromises()

    await wrapper.get('#param-vacance').setValue('1,3')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(wrapper.get('#param-vacance-error').text()).toContain('par demi-mois')
    expect(wrapper.find('[role="status"]').exists()).toBe(false)
  })

  it('does not send a field left empty', async () => {
    const fetchMock = stubApi()
    const wrapper = mount(SettingsForm)
    await flushPromises()

    await wrapper.get('#param-notaire').setValue('')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(fetchMock.mock.calls.some(([, init]) => init?.method === 'PUT')).toBe(false)
    expect(wrapper.get('#param-notaire-error').text()).toBe('Ce champ est obligatoire')
  })

  describe('Couleur principale', () => {
    afterEach(() => document.documentElement.removeAttribute('style'))

    it('previews the colour live and saves it', async () => {
      const fetchMock = stubApi((body) => json(body))
      const wrapper = mount(SettingsForm)
      await flushPromises()
      const picker = wrapper.get('#param-couleur')
      expect(picker.attributes('type')).toBe('color')
      expect(value(wrapper, 'param-couleur')).toBe('#8b1a1a')

      await picker.setValue('#1f6feb')
      expect(document.documentElement.style.getPropertyValue('--brand')).toBe('#1f6feb')

      await wrapper.get('form').trigger('submit')
      await flushPromises()
      const put = fetchMock.mock.calls.find(([, init]) => init?.method === 'PUT')
      expect(JSON.parse(String(put?.[1]?.body)).accentColor).toBe('#1f6feb')
    })

    it('resets to dark red', async () => {
      stubApi()
      const wrapper = mount(SettingsForm)
      await flushPromises()
      await wrapper.get('#param-couleur').setValue('#1f6feb')

      await wrapper.get('[data-test="reset-colour"]').trigger('click')

      expect(value(wrapper, 'param-couleur')).toBe('#8b1a1a')
      expect(document.documentElement.style.getPropertyValue('--brand')).toBe('#8b1a1a')
    })
  })
})
