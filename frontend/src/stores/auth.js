import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '../utils/api'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const user = ref(null)
  const betamap = ref(null)
  const initialized = ref(false)

  const isAdmin = computed(() => user.value?.role === 'admin')
  const tenantName = computed(() => user.value?.tenantName || '')

  async function init() {
    if (!token.value) {
      initialized.value = true
      return
    }
    try {
      api.defaults.headers.common['Authorization'] = `Bearer ${token.value}`
      const res = await fetch('/api/auth/me', {
        headers: { Authorization: `Bearer ${token.value}` },
        credentials: 'include',
      })
      if (!res.ok) throw new Error('invalid token')
      user.value = await res.json()
    } catch {
      token.value = ''
      localStorage.removeItem('token')
    } finally {
      initialized.value = true
    }
  }

  async function login(tenant, username, password) {
    const res = await api.post('/api/auth/login', { tenant, username, password })
    token.value = res.data.token
    user.value = res.data.user
    api.defaults.headers.common['Authorization'] = `Bearer ${token.value}`
    localStorage.setItem('token', token.value)
    return res.data
  }

  async function logout() {
    try {
      await api.post('/api/auth/logout')
    } catch {}
    token.value = ''
    user.value = null
    localStorage.removeItem('token')
    delete api.defaults.headers.common['Authorization']
  }

  async function fetchBetamap() {
    if (!token.value) return
    try {
      const res = await api.get('/api/auth/betamap')
      betamap.value = res.data
    } catch {}
  }

  return { token, user, betamap, initialized, isAdmin, tenantName, init, login, logout, fetchBetamap }
})
