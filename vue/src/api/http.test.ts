import test from 'node:test'
import assert from 'node:assert/strict'

import { requestJson } from './http.ts'
import { createMessage, updateMessageById } from './messages.ts'
import { updateUserById } from './users.ts'

test('requestJson sends YAML for write requests', async () => {
  let init: RequestInit | undefined

  globalThis.fetch = async (_input, requestInit) => {
    init = requestInit
    return new Response(JSON.stringify({ id: 42 }), {
      status: 201,
      headers: { 'Content-Type': 'application/json' },
    })
  }

  await requestJson<{ id: number }>({
    method: 'POST',
    url: 'http://localhost:8080/users',
    body: { email: 'user@example.com' },
  })

  assert.equal(init?.headers instanceof Headers, false)
  assert.deepEqual(init?.headers, { 'Content-Type': 'application/x-yaml' })
  assert.equal(init?.body, 'email: user@example.com\n')
})

test('requestJson normalizes fetch failures into a generic message', async () => {
  globalThis.fetch = async () => {
    throw new TypeError('Failed to fetch')
  }

  await assert.rejects(
    () =>
      requestJson({
        method: 'GET',
        url: 'http://localhost:8080/users',
      }),
    (error: unknown) => {
      const normalized = error as Error
      assert.equal(normalized.message, 'Request failed')
      return true
    },
  )
})

test('requestJson preserves HTTP error status and response body', async () => {
  globalThis.fetch = async () =>
    new Response(JSON.stringify({ error: 'user not found' }), {
      status: 404,
      headers: { 'Content-Type': 'application/json' },
    })

  await assert.rejects(
    () =>
      requestJson({
        method: 'GET',
        url: 'http://localhost:8080/users/404',
      }),
    (error: unknown) => {
      const apiError = error as Error & { status?: number; responseBody?: unknown }
      assert.equal(apiError.status, 404)
      assert.deepEqual(apiError.responseBody, { error: 'user not found' })
      return true
    },
  )
})

test('requestJson sends delete requests without body or content type', async () => {
  let init: RequestInit | undefined

  globalThis.fetch = async (_input, requestInit) => {
    init = requestInit
    return new Response(null, {
      status: 204,
    })
  }

  await requestJson<null>({
    method: 'DELETE',
    url: 'http://localhost:8080/users/42',
  })

  assert.deepEqual(init?.headers, {})
  assert.equal(init?.body, undefined)
})

test('createMessage injects date_sent automatically', async () => {
  const originalDate = globalThis.Date

  globalThis.Date = class extends Date {
    constructor(...args: any[]) {
      if (args.length === 0) {
        super('2026-05-04T12:00:00.000Z')
        return
      }

      super(args[0] as string | number | Date)
    }

    static override now() {
      return originalDate.parse('2026-05-04T12:00:00.000Z')
    }
  } as DateConstructor

  let init: RequestInit | undefined

  globalThis.fetch = async (_input, requestInit) => {
    init = requestInit
    return new Response(JSON.stringify({ id: 1 }), {
      status: 201,
      headers: { 'Content-Type': 'application/json' },
    })
  }

  try {
    await createMessage({
      external_id: 'ext-1',
      sender_id: 1,
      receiver_id: 2,
      subject: 'Hi',
      text: 'Body',
    })
  } finally {
    globalThis.Date = originalDate
  }

  assert.match(String(init?.body), /date_sent: 2026-05-04T12:00:00.000Z/)
})

test('updateUserById sends PUT request to user endpoint', async () => {
  let url: RequestInfo | URL | undefined
  let init: RequestInit | undefined

  globalThis.fetch = async (input, requestInit) => {
    url = input
    init = requestInit
    return new Response(JSON.stringify({ id: 7, email: 'updated@example.com' }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    })
  }

  await updateUserById(7, 'updated@example.com')

  assert.equal(url, 'http://localhost:8080/users/7')
  assert.equal(init?.method, 'PUT')
  assert.equal(init?.body, 'email: updated@example.com\n')
})

test('updateMessageById sends PUT request and refreshes date_sent', async () => {
  const originalDate = globalThis.Date

  globalThis.Date = class extends Date {
    constructor(...args: any[]) {
      if (args.length === 0) {
        super('2026-05-04T13:30:00.000Z')
        return
      }

      super(args[0] as string | number | Date)
    }

    static override now() {
      return originalDate.parse('2026-05-04T13:30:00.000Z')
    }
  } as DateConstructor

  let url: RequestInfo | URL | undefined
  let init: RequestInit | undefined

  globalThis.fetch = async (input, requestInit) => {
    url = input
    init = requestInit
    return new Response(JSON.stringify({ id: 9 }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    })
  }

  try {
    await updateMessageById({
      id: 9,
      external_id: 'ext-9',
      sender_id: 3,
      receiver_id: 4,
      subject: 'Updated',
      text: 'Edited body',
    })
  } finally {
    globalThis.Date = originalDate
  }

  assert.equal(url, 'http://localhost:8080/messages/9')
  assert.equal(init?.method, 'PUT')
  assert.match(String(init?.body), /date_sent: 2026-05-04T13:30:00.000Z/)
})
