const rawBaseUrl = import.meta.env?.VITE_API_BASE_URL?.trim()
const baseUrl = (rawBaseUrl && rawBaseUrl.length > 0 ? rawBaseUrl : 'http://localhost:8080').replace(/\/$/, '')

export const apiConfig = {
  baseUrl,
  users: {
    list: () => `${baseUrl}/users`,
    create: () => `${baseUrl}/users`,
    getById: (id: string | number) => `${baseUrl}/users/${id}`,
    updateById: (id: string | number) => `${baseUrl}/users/${id}`,
    deleteById: (id: string | number) => `${baseUrl}/users/${id}`,
  },
  messages: {
    list: () => `${baseUrl}/messages`,
    create: () => `${baseUrl}/messages`,
    updateById: (id: string | number) => `${baseUrl}/messages/${id}`,
    deleteById: (id: string | number) => `${baseUrl}/messages/${id}`,
  },
  contacts: {
    list: () => `${baseUrl}/contacts`,
    create: (userId: string | number) => `${baseUrl}/users/${userId}/contacts`,
    getByUserId: (userId: string | number) => `${baseUrl}/users/${userId}/contacts`,
    deleteByIds: (userId: string | number, contactId: string | number) =>
      `${baseUrl}/users/${userId}/contacts/${contactId}`,
  },
}
