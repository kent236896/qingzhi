import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import WorkbenchLayout from '@/layouts/WorkbenchLayout.vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { safeRedirectPath } from '@/utils/redirect'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/auth/LoginView.vue'),
    meta: { title: '登录', public: true },
  },
  {
    path: '/login/sso/callback',
    name: 'sso-callback',
    component: () => import('@/views/auth/SsoCallbackView.vue'),
    meta: { title: '外部登录', public: true },
  },
  {
    path: '/',
    component: WorkbenchLayout,
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        redirect: { name: 'chat' },
      },
      {
        path: 'home',
        name: 'home',
        component: () => import('@/views/home/HomeView.vue'),
        meta: { title: '首页' },
      },
      {
        path: 'chat',
        name: 'chat',
        component: () => import('@/views/chat/ChatView.vue'),
        meta: { title: 'AI 对话' },
      },
      {
        path: 'agents',
        name: 'agents',
        component: () => import('@/views/agents/AgentsView.vue'),
        meta: { title: '智能体' },
      },
      {
        path: 'memory',
        name: 'memory',
        component: () => import('@/views/memory/MemoryView.vue'),
        meta: { title: '我的记忆' },
      },
      {
        path: 'profile',
        name: 'profile',
        component: () => import('@/views/profile/ProfileView.vue'),
        meta: { title: '个人中心' },
      },
    ],
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/views/NotFoundView.vue'),
    meta: { title: '页面不存在', public: true },
  },
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (!auth.bootstrapped) {
    await auth.restore()
  }
  const isPublic = Boolean(to.meta.public)
  if (!isPublic && !auth.isAuthenticated) {
    return {
      name: 'login',
      query: { redirect: safeRedirectPath(to.fullPath) },
    }
  }
  if (to.name === 'login' && auth.isAuthenticated) {
    return safeRedirectPath(to.query.redirect, '/chat')
  }
  return true
})

router.afterEach((to) => {
  const title = typeof to.meta.title === 'string' ? to.meta.title : '用户工作台'
  document.title = `${title} · 用户工作台`
  const appStore = useAppStore()
  appStore.setPageTitle(title)
})

export default router
