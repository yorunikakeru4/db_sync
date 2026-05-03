import { apiConfig } from './config'
import { requestJson } from './http'

export interface CreateMessageInput {
  user_id: string
  subject: string
  content: string
  date_sent: string
}

/**
 * Create a message from the provided form payload.
 */
export function createMessage(input: CreateMessageInput) {
  return requestJson<Record<string, unknown>>({
    method: 'POST',
    url: apiConfig.messages.create(),
    body: input,
  })
}

/**
 * Fetch messages using the backend lookup value.
 */
export function getMessages(value: string) {
  return requestJson<unknown>({
    method: 'GET',
    url: apiConfig.messages.get(value),
  })
}
