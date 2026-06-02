import axios from 'axios'
import { ElMessage } from 'element-plus'

const api = axios.create({
  baseURL: '/api',
  timeout: 10000,
})

api.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  res => {
    const data = res.data
    if (data.code !== 0) {
      ElMessage.error(data.message || '请求失败')
      if (data.code === 40101) {
        localStorage.removeItem('token')
        localStorage.removeItem('user')
        window.location.href = '/login'
      }
      return Promise.reject(new Error(data.message))
    }
    return data.data
  },
  err => {
    ElMessage.error('网络错误')
    return Promise.reject(err)
  }
)

// Auth
export const login = (username: string, password: string) =>
  api.post('/auth/login', { username, password })

// User
export const getMe = () => api.get('/user/me')
export const updateProfile = (data: Record<string, any>) => api.put('/user/profile', data)
export const getMyOrders = (params: Record<string, any>) => api.get('/user/orders', { params })
export const changePassword = (old_password: string, new_password: string) =>
  api.put('/user/password', { old_password, new_password })

// Redeem
export const redeemCode = (code: string) => api.post('/redeem', { code })
export const getAmounts = () => api.get('/amounts')
export const exchange = (amount: number, quantity: number) =>
  api.post('/exchange', { amount, quantity })

// Announcements
export const getAnnouncements = (params: Record<string, any>) =>
  api.get('/announcements', { params })

// Admin: Users
export const getUsers = (params: Record<string, any>) => api.get('/admin/users', { params })
export const createUser = (data: Record<string, any>) => api.post('/admin/users', data)
export const updateUser = (id: number, data: Record<string, any>) => api.patch(`/admin/users/${id}`, data)

// Admin: CDK
export const importCdk = (formData: FormData) => api.post('/admin/cdk/import', formData, {
  headers: { 'Content-Type': 'multipart/form-data' },
})
export const getCdkList = (params: Record<string, any>) => api.get('/admin/cdk/list', { params })
export const getImportHistory = (params: Record<string, any>) => api.get('/admin/cdk/imports', { params })

// Admin: Redeem Items
export const getRedeemItems = (params: Record<string, any>) => api.get('/admin/redeem-items', { params })
export const createRedeemItem = (data: Record<string, any>) => api.post('/admin/redeem-items', data)
export const importRedeemItemFiles = (formData: FormData) => api.post('/admin/redeem-items/import', formData, {
  headers: { 'Content-Type': 'multipart/form-data' },
})
export const updateRedeemItem = (id: number, data: Record<string, any>) => api.put(`/admin/redeem-items/${id}`, data)
export const deleteRedeemItem = (id: number) => api.delete(`/admin/redeem-items/${id}`)

// Admin: Templates
export const getTemplates = (params: Record<string, any>) => api.get('/admin/templates', { params })
export const createTemplate = (data: Record<string, any>) => api.post('/admin/templates', data)
export const updateTemplate = (id: number, data: Record<string, any>) => api.put(`/admin/templates/${id}`, data)
export const deleteTemplate = (id: number) => api.delete(`/admin/templates/${id}`)

// Admin: Teams
export const getMyTeam = () => api.get('/admin/teams/my')
export const getJoinedTeams = () => api.get('/admin/teams/joined')
export const joinTeam = (owner_username: string) => api.post('/admin/teams/join', { owner_username })
export const removeTeamMember = (memberId: number) => api.delete(`/admin/teams/members/${memberId}`)

// Admin: Purchase Tasks
export const getPurchaseTasks = (params: Record<string, any>) => api.get('/admin/purchase-tasks', { params })
export const createPurchaseTask = (template_id: number) => api.post('/admin/purchase-tasks', { template_id })
export const processPurchaseTask = (id: number) => api.post(`/admin/purchase-tasks/${id}/process`)
export const fetchPurchaseTaskSubscribe = (id: number) => api.post(`/admin/purchase-tasks/${id}/fetch-subscribe`)
export const manualCompletePurchaseTask = (id: number, subscribe_url: string) =>
  api.post(`/admin/purchase-tasks/${id}/manual-complete`, { subscribe_url })

// Admin: Announcements
export const createAnnouncement = (data: Record<string, any>) => api.post('/admin/announcements', data)
export const updateAnnouncement = (id: number, data: Record<string, any>) => api.put(`/admin/announcements/${id}`, data)
export const deleteAnnouncement = (id: number) => api.delete(`/admin/announcements/${id}`)

// Admin: Stats
export const getStatsOverview = () => api.get('/admin/stats/overview')
export const getStatsByAmount = () => api.get('/admin/stats/by-amount')
export const getStatsByItem = () => api.get('/admin/stats/by-item')
export const getStatsDaily = (start: string, end: string) => api.get('/admin/stats/daily', { params: { start, end } })
export const getTopUsers = (limit: number = 10) => api.get('/admin/stats/top-users', { params: { limit } })
export const getStatsByUserAmount = (params: Record<string, any>) => api.get('/admin/stats/by-user-amount', { params })
