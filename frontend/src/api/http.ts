import { API_BASE_URL } from './config'
import { emitAuthChanged } from '../lib/authEvents'

export class ApiError extends Error {
  readonly status: number
  readonly payload: unknown

  constructor(message: string, status: number, payload: unknown) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.payload = payload
  }
}

type JsonValue = null | boolean | number | string | JsonValue[] | { [key: string]: JsonValue }

type RequestJsonOptions = {
  baseUrl?: string
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  path: string
  query?: Record<string, string | number | boolean | undefined>
  body?: JsonValue
  headers?: Record<string, string>
  signal?: AbortSignal
  credentials?: RequestCredentials
}

function buildUrl(baseUrl: string, path: string, query?: RequestJsonOptions['query']): string {
  const url = new URL(path, baseUrl || window.location.origin)

  if (query) {
    for (const [key, value] of Object.entries(query)) {
      if (value === undefined) continue
      url.searchParams.set(key, String(value))
    }
  }

  return url.toString()
}

export async function requestJson<T>(options: RequestJsonOptions): Promise<T> {
  const baseUrl = options.baseUrl ?? API_BASE_URL
  const url = buildUrl(baseUrl, options.path, options.query)

  const response = await fetch(url, {
    method: options.method ?? 'GET',
    headers: {
      Accept: 'application/json',
      ...(options.body ? { 'Content-Type': 'application/json' } : {}),
      ...(options.headers ?? {}),
    },
    body: options.body ? JSON.stringify(options.body) : undefined,
    signal: options.signal,
    credentials: options.credentials,
  })

  const contentType = response.headers.get('content-type') || ''
  const isJson = contentType.includes('application/json')

  const payload = isJson ? await response.json().catch(() => null) : await response.text().catch(() => '')

  if (response.status === 401 && options.path !== '/api/users/me') {
    // If a protected request fails, broadcast auth state change so the app can refresh UI.
    emitAuthChanged()
  }

  if (!response.ok) {
    const message = `API request failed: ${response.status} ${response.statusText}`
    throw new ApiError(message, response.status, payload)
  }

  return payload as T
}
