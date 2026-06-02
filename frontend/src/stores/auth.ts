import { defineStore } from 'pinia'
import { ref } from 'vue'

interface User {
  id: number
  username: string
  role: string
  status: string
  last_login_at?: string
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const user = ref<User | null>(null)

  try {
    const stored = localStorage.getItem('user')
    if (stored) user.value = JSON.parse(stored)
  } catch {}

  function setAuth(t: string, u: User) {
    token.value = t
    user.value = u
    localStorage.setItem('token', t)
    localStorage.setItem('user', JSON.stringify(u))
  }

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('user')
  }

  const isAdmin = () => user.value?.role === 'admin'

  return { token, user, setAuth, logout, isAdmin }
})
