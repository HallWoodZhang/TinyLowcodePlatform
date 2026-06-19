import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/LoginView.vue'),
    meta: { showHeader: false },
  },
  {
    path: '/',
    name: 'Portal',
    component: () => import('../views/PortalView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/scripts',
    name: 'Scripts',
    component: () => import('../views/EditorView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/sql',
    name: 'SqlRunner',
    component: () => import('../views/SqlView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/admin',
    name: 'Admin',
    component: () => import('../views/AdminView.vue'),
    meta: { requiresAuth: true },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach(async (to, from, next) => {
  const auth = useAuthStore()

  if (!auth.initialized) {
    await auth.init()
  }

  if (to.meta.requiresAuth && !auth.token) {
    next({ name: 'Login', query: { redirect: to.fullPath } })
  } else if (to.name === 'Login' && auth.token) {
    next({ name: 'Portal' })
  } else {
    next()
  }
})

export default router
