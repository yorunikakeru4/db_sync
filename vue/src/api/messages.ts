import { apiConfig } from './config.ts'
import { requestJson } from './http.ts'

export interface CreateMessageInput {
  external_id: string
  sender_id: number
  receiver_id: number
  subject: string
  text: string
}

export interface UpdateMessageInput extends CreateMessageInput {
  id: string | number
}

/**
 * Add the current timestamp to a message write payload.
 */
function withCurrentDateSent<T extends CreateMessageInput>(input: T) {
  return {
    ...input,
    date_sent: new Date().toISOString(),
  }
}

/**
 * Create a message from the provided form payload.
 */
export function createMessage(input: CreateMessageInput) {
  return requestJson<Record<string, unknown>>({
    method: 'POST',
    url: apiConfig.messages.create(),
    body: withCurrentDateSent(input),
  })
}

/**
 * Update a message by backend-generated identifier.
 */
export function updateMessageById(input: UpdateMessageInput) {
  const { id, ...rest } = input

  return requestJson<Record<string, unknown>>({
    method: 'PUT',
    url: apiConfig.messages.updateById(id),
    body: withCurrentDateSent(rest),
  })
}

/**
 * Fetch all messages.
 */
export function listMessages() {
  return requestJson<Record<string, unknown>[]>({
    method: 'GET',
    url: apiConfig.messages.list(),
  })
}

/**
 * Delete a message by backend-generated identifier.
 */
export function deleteMessageById(id: string | number) {
  return requestJson<null>({
    method: 'DELETE',
    url: apiConfig.messages.deleteById(id),
  })
}
