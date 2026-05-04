import type { ApiError, HttpMethod } from '../types/api.ts'
import { useRequestLog } from '../composables/useRequestLog.ts'

interface RequestOptions {
  method: HttpMethod
  url: string
  body?: unknown
}

type YamlScalar = string | number | boolean | null

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

const isWriteMethod = (method: HttpMethod): boolean =>
  method === 'POST' || method === 'PUT' || method === 'DELETE'

const toYamlScalar = (value: YamlScalar): string => {
  if (value === null) {
    return 'null'
  }
  if (typeof value === 'string') {
    return /[\n\r]/.test(value) || /:\s/.test(value) || value.includes('#') || value.trim() !== value
      ? JSON.stringify(value)
      : value
  }
  return String(value)
}

const serializeYamlBody = (body: unknown): string => {
  if (body === null || typeof body !== 'object' || Array.isArray(body)) {
    throw new TypeError('YAML request body must be a plain object')
  }

  const lines: string[] = []
  for (const [key, value] of Object.entries(body as Record<string, YamlScalar | undefined>)) {
    if (value === undefined) {
      continue
    }
    lines.push(`${key}: ${toYamlScalar(value)}`)
  }
  return lines.length === 0 ? '' : `${lines.join('\n')}\n`
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
    const requestBody = body !== undefined && isWriteMethod(method) ? serializeYamlBody(body) : undefined

    if (requestBody !== undefined) {
      headers['Content-Type'] = 'application/x-yaml'
    }

    response = await fetch(url, {
      method,
      headers,
      body: requestBody,
    })
  } catch (error) {
    const normalized = makeError('Request failed')
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
