export type HttpMethod = 'GET' | 'POST'

export interface RequestLogEntry {
  id: string
  startedAt: string
  method: HttpMethod
  url: string
  requestBody?: unknown
  status?: number
  durationMs?: number
  responseBody?: unknown
  errorMessage?: string
}

export interface ApiError extends Error {
  status?: number
  responseBody?: unknown
}
