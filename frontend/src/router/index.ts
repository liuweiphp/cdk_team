import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/user/LoginView.vue'),
    meta: { guest: true }
  },
  {
    path: '/',
    redirect: '/exchange'
  },
  {
    path: '/exchange',
    component: () => import('@/layouts/UserLayout.vue'),
    children: [
      { path: '', name: 'Exchange', component: () => import('@/views/user/ExchangeView.vue') },
    ]
  },
  {
    path: '/admin',
    component: () => import('@/layouts/AdminLayout.vue'),
    meta: { auth: true },
    children: [
      { path: '', name: 'Dashboard', component: () => import('@/views/admin/DashboardView.vue') },
      { path: 'cdk', name: 'CdkManage', component: () => import('@/views/admin/CdkManageView.vue') },
      { path: 'redeem-items', name: 'RedeemItems', component: () => import('@/views/admin/RedeemItemManageView.vue') },
      { path: 'templates', name: 'Templates', component: () => import('@/views/admin/TemplateManageView.vue') },
      { path: 'purchase-tasks', name: 'PurchaseTasks', component: () => import('@/views/admin/PurchaseTaskManageView.vue') },
      { path: 'teams', name: 'Teams', component: () => import('@/views/admin/TeamManageView.vue') },
      { path: 'users', name: 'UserManage', component: () => import('@/views/admin/UserManageView.vue') },
      { path: 'announcements', name: 'AnnounceManage', component: () => import('@/views/admin/AnnounceManageView.vue') },
    ]
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/exchange'
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to, _from, next) => {
  const token = localStorage.getItem('token')
  const userStr = localStorage.getItem('user')

  if (to.meta.guest) {
    if (token) {
      const user = userStr ? JSON.parse(userStr) : null
      if (user?.role === 'admin') next('/admin')
      else next('/exchange')
    } else {
      next()
    }
    return
  }

  if (to.meta.auth && !token) {
    next('/login')
    return
  }

  const user = userStr ? JSON.parse(userStr) : null
  if (to.path === '/admin' && user?.role !== 'admin') {
    next('/admin/cdk')
    return
  }

  next()
})

export default router
