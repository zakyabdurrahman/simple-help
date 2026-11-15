import { defineStore } from 'pinia'
import type { User } from '@/types'
import client from '@/api/client'

const TOKEN_KEY = 'simplehelp_token'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem(TOKEN_KEY) || '',
    user: null as User | null,
  }),
  getters: {
    isLoggedIn: (state) => !!state.token,
    role: (state) => state.user?.role || null,
    isAdmin: (state) => state.user?.role === 'admin',
    isAgent: (state) => state.user?.role === 'agent' || state.user?.role === 'admin',
  },
  actions: {
    setToken(token: string) {
      this.token = token
      localStorage.setItem(TOKEN_KEY, token)
    },
    setUser(user: User) {
      this.user = user
    },
    async fetchMe() {
      const { data } = await client.get<User>('/api/me')
      this.setUser(data)
      return data
    },
    async login(email: string, password: string) {
      const { data } = await client.post('/api/auth/login', { email, password })
      this.setToken(data.token)
      this.setUser(data.user)
      return data
    },
    async register(name: string, email: string, password: string) {
      const { data } = await client.post('/api/auth/register', { name, email, password })
      return data
    },
    logout() {
      this.clear()
    },
    clear() {
      this.token = ''
      this.user = null
      localStorage.removeItem(TOKEN_KEY)
    },
  },
})
