import { apiConfig } from './config.ts'
import { requestJson } from './http.ts'

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

/**
 * Update a user by backend-generated identifier.
 */
export function updateUserById(id: string | number, email: string) {
  return requestJson<Record<string, unknown>>({
    method: 'PUT',
    url: apiConfig.users.updateById(id),
    body: { email },
  })
}

/**
 * Fetch all users.
 */
export function listUsers() {
  return requestJson<Record<string, unknown>[]>({
    method: 'GET',
    url: apiConfig.users.list(),
  })
}

/**
 * Delete a user by backend-generated identifier.
 */
export function deleteUserById(id: string | number) {
  return requestJson<null>({
    method: 'DELETE',
    url: apiConfig.users.deleteById(id),
  })
}
