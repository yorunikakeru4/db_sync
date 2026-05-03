import { apiConfig } from './config'
import { requestJson } from './http'

export interface CreateEmailInput {
  user_id: string
  address: string
  category: string
  importance: number
}

export function createEmail(input: CreateEmailInput) {
  return requestJson<Record<string, unknown>>({
    method: 'POST',
    url: apiConfig.emails.create(),
    body: input,
  })
}

export function getEmailsByUserId(userId: string) {
  return requestJson<Record<string, unknown>[]>({
    method: 'GET',
    url: apiConfig.emails.getByUserId(userId),
  })
}
