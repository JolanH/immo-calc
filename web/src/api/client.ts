// Typed fetch wrapper for the Go API. Errors always come back as ApiError with
// a French message that can be shown as is.

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
    message: string,
    readonly fields: Record<string, string> = {},
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

type Method = 'GET' | 'POST' | 'PUT' | 'DELETE'

interface ErrorBody {
  error?: { code?: string; message?: string; fields?: Record<string, string> }
}

/** Sends body (if any) as JSON and decodes the JSON answer. */
export function request<T = unknown>(method: Method, path: string, body?: unknown): Promise<T> {
  const init: RequestInit = { method }
  if (body !== undefined) {
    init.headers = { 'Content-Type': 'application/json' }
    init.body = JSON.stringify(body)
  }
  return send<T>(path, init)
}

/** Sends a raw text body (e.g. an imported YAML file) and decodes the JSON answer. */
export function requestText<T = unknown>(
  method: Method,
  path: string,
  text: string,
  contentType: string,
): Promise<T> {
  return send<T>(path, { method, headers: { 'Content-Type': contentType }, body: text })
}

async function send<T>(path: string, init: RequestInit): Promise<T> {
  let response: Response
  try {
    response = await fetch(path, init)
  } catch {
    throw new ApiError(
      0,
      'network_error',
      'Impossible de joindre immo-calc. Est-il toujours lancé ?',
    )
  }

  if (response.status === 204) return undefined as T
  const text = await response.text()

  if (!response.ok) {
    let parsed: ErrorBody = {}
    try {
      parsed = JSON.parse(text) as ErrorBody
    } catch {
      // not JSON: fall through to the generic message
    }
    throw new ApiError(
      response.status,
      parsed.error?.code ?? 'unknown_error',
      parsed.error?.message ?? `Erreur inattendue (${response.status})`,
      parsed.error?.fields ?? {},
    )
  }
  return (text ? JSON.parse(text) : undefined) as T
}
