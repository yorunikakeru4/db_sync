const baseUrl = import.meta.env.VITE_API_BASE_URL?.replace(/\/$/, '') ?? ''

export const apiConfig = {
  baseUrl,
  users: {
    create: () => `${baseUrl}/users`,
    getById: (id: string | number) => `${baseUrl}/users/${id}`,
  },
  messages: {
    create: () => `${baseUrl}/messages`,
    get: (value: string | number) => `${baseUrl}/messages/${value}`,
  },
  emails: {
    create: () => `${baseUrl}/emails`,
    getByUserId: (userId: string | number) => `${baseUrl}/users/${userId}/emails`,
  },
}
