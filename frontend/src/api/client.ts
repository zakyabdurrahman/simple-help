import axios, { AxiosError } from 'axios'
import { useAuthStore } from '@/stores/auth'
import type { ApiError } from '@/types'

// Single axios instance. In production, Nginx proxies /api to the backend
// (same origin), so no baseURL is needed. In dev, Vite proxies /api.
const client = axios.create({
  baseURL: '',
  headers: { 'Content-Type': 'application/json' },
})

// Attach JWT to every request.
client.interceptors.request.use((config) => {
  const auth = useAuthStore()
  if (auth.token) {
    config.headers.Authorization = `Bearer ${auth.token}`
  }
  return config
})

// On 401, clear auth and redirect to login.
client.interceptors.response.use(
  (res) => res,
  (error: AxiosError) => {
    if (error.response?.status === 401) {
      const auth = useAuthStore()
      auth.clear()
      if (window.location.pathname !== '/login') {
        window.location.href = '/login'
      }
    }
    return Promise.reject(error)
  },
)

// Helper to extract the ApiError envelope from an axios error.
export function extractApiError(err: unknown): ApiError {
  if (axios.isAxiosError(err) && err.response?.data) {
    return err.response.data as ApiError
  }
  return { error: 'network_error' }
}

export default client
