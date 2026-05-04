import { apiConfig } from './config'
import { requestJson } from './http'

export interface CreateContactInput {
  user_id: number
  contact_id?: number
  value: string
  category: number
  importance: number
}

export function createContact(input: CreateContactInput) {
  const { user_id, ...payload } = input
  return requestJson<Record<string, unknown>>({
    method: 'POST',
    url: apiConfig.contacts.create(user_id),
    body: payload,
  })
}

export function getContactsByUserId(userId: string) {
  return requestJson<Record<string, unknown>[]>({
    method: 'GET',
    url: apiConfig.contacts.getByUserId(userId),
  })
}

export function listContacts() {
  return requestJson<Record<string, unknown>[]>({
    method: 'GET',
    url: apiConfig.contacts.list(),
  })
}

export function deleteContactByIds(userId: string | number, contactId: string | number) {
  return requestJson<null>({
    method: 'DELETE',
    url: apiConfig.contacts.deleteByIds(userId, contactId),
  })
}
