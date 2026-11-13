import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError, request, requestText } from './client'

function mockFetch(status: number, body?: unknown) {
  const fetchMock = vi.fn().mockResolvedValue(
    new Response(body === undefined ? null : JSON.stringify(body), {
      status,
      headers: { 'Content-Type': 'application/json' },
    }),
  )
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

afterEach(() => vi.unstubAllGlobals())

describe('request', () => {
  it('returns the decoded JSON body on success', async () => {
    mockFetch(200, { status: 'ok' })
    await expect(request('GET', '/api/health')).resolves.toEqual({ status: 'ok' })
  })

  it('sends a JSON body with the right headers', async () => {
    const fetchMock = mockFetch(201, { id: 'a' })

    await request('POST', '/api/loans', { name: 'Prêt 1' })

    expect(fetchMock).toHaveBeenCalledWith('/api/loans', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: '{"name":"Prêt 1"}',
    })
  })

  it('returns undefined for 204 No Content', async () => {
    mockFetch(204)
    await expect(request('DELETE', '/api/loans/a')).resolves.toBeUndefined()
  })

  it('throws an ApiError carrying the code, French message and field errors', async () => {
    mockFetch(400, {
      error: {
        code: 'validation_failed',
        message: 'Données invalides',
        fields: { name: 'Requis' },
      },
    })

    const err = await request('POST', '/api/loans', {}).catch((e: unknown) => e)

    expect(err).toBeInstanceOf(ApiError)
    expect(err).toMatchObject({
      status: 400,
      code: 'validation_failed',
      message: 'Données invalides',
      fields: { name: 'Requis' },
    })
  })

  it('throws a French ApiError when the server is unreachable', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Failed to fetch')))

    const err = await request('GET', '/api/health').catch((e: unknown) => e)

    expect(err).toMatchObject({ status: 0, code: 'network_error' })
    expect((err as ApiError).message).toContain('immo-calc')
  })

  it('throws a generic French ApiError when the error body is not JSON', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('oops', { status: 500 })))

    const err = await request('GET', '/api/x').catch((e: unknown) => e)

    expect(err).toMatchObject({ status: 500, code: 'unknown_error', fields: {} })
  })

  it('sends a raw text body with the given content type', async () => {
    const fetchMock = mockFetch(200, { ok: true })

    await requestText('POST', '/api/data/import', 'schema_version: 1\n', 'application/yaml')

    expect(fetchMock).toHaveBeenCalledWith('/api/data/import', {
      method: 'POST',
      headers: { 'Content-Type': 'application/yaml' },
      body: 'schema_version: 1\n',
    })
  })
})
