import { apiConfig } from './config'
import { requestJson } from './http'

export interface CreateUserInput {
  email: string
}

/**
 * Create a user from the provided email address.
 */
export function createUser(input: CreateUserInput) {
  return requestJson<Record<string, unknown>>({
    method: 'POST',
    url: apiConfig.users.create(),
    body: input,
  })
}

/**
 * Fetch a single user by backend-generated identifier.
 */
export function getUserById(id: string) {
  return requestJson<Record<string, unknown>>({
    method: 'GET',
    url: apiConfig.users.getById(id),
  })
}
