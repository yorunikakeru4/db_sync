import { apiConfig } from './config'
import { requestJson } from './http'

export interface CreateContactInput {
  user_id: string
  value: string
  category: string
  importance: number
}

export function createContact(input: CreateContactInput) {
  return requestJson<Record<string, unknown>>({
    method: 'POST',
    url: apiConfig.contacts.create(),
    body: input,
  })
}

export function getContactsByUserId(userId: string) {
  return requestJson<Record<string, unknown>[]>({
    method: 'GET',
    url: apiConfig.contacts.getByUserId(userId),
  })
}
