import type { ApiError, HttpMethod } from '../types/api'
import { useRequestLog } from '../composables/useRequestLog'

interface RequestOptions {
  method: HttpMethod
  url: string
  body?: unknown
}

const cloneLogPayload = (value: unknown): unknown => {
  if (value === null || typeof value !== 'object') {
    return value
  }

  try {
    return structuredClone(value)
  } catch {
    return value
  }
}

const makeError = (message: string, status?: number, responseBody?: unknown): ApiError => {
  const error = new Error(message) as ApiError
  error.status = status
  error.responseBody = responseBody
  return error
}

const pushLogEntry = (
  startedAt: string,
  method: HttpMethod,
  url: string,
  started: number,
  requestBody: unknown,
  responseBody: unknown,
  errorMessage?: string,
  status?: number,
) => {
  const { pushEntry } = useRequestLog()

  pushEntry({
    id: crypto.randomUUID(),
    startedAt,
    method,
    url,
    requestBody: cloneLogPayload(requestBody),
    status,
    durationMs: Math.round(performance.now() - started),
    responseBody: cloneLogPayload(responseBody),
    errorMessage,
  })
}

const parseResponseBody = async (response: Response): Promise<unknown> => {
  const text = await response.text()

  try {
    return text ? JSON.parse(text) : null
  } catch {
    return text || null
  }
}

export async function requestJson<T>({ method, url, body }: RequestOptions): Promise<T> {
  const started = performance.now()
  const startedAt = new Date().toISOString()
  let response: Response

  try {
    const headers: Record<string, string> = {}

    if (body !== undefined) {
      headers['Content-Type'] = 'application/json'
    }

    response = await fetch(url, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    })
  } catch (error) {
    const normalized = error instanceof Error ? error : new Error('Unknown request error')
    pushLogEntry(startedAt, method, url, started, body, undefined, normalized.message)
    throw normalized
  }

  const responseBody = await parseResponseBody(response)

  if (!response.ok) {
    pushLogEntry(startedAt, method, url, started, body, responseBody, `HTTP ${response.status}`, response.status)
    throw makeError(`HTTP ${response.status}`, response.status, responseBody)
  }

  pushLogEntry(startedAt, method, url, started, body, responseBody, undefined, response.status)
  return responseBody as T
}
