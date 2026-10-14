import { createRouter, createWebHistory } from 'vue-router'
import AdminLayout from '@/layouts/AdminLayout.vue'
import HomeView from '@/views/HomeView.vue'
import NotFoundView from '@/views/NotFoundView.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      component: AdminLayout,
      children: [
        {
          path: '',
          name: 'home',
          component: HomeView,
          meta: { title: '首页', breadcrumb: ['首页'] },
        },
        {
          path: ':pathMatch(.*)*',
          name: 'not-found',
          component: NotFoundView,
          meta: { title: '页面不存在', breadcrumb: ['页面不存在'] },
        },
      ],
    },
  ],
})

router.afterEach((to) => {
  const pageTitle = typeof to.meta.title === 'string' ? to.meta.title : ''
  const appTitle = import.meta.env.VITE_APP_TITLE || '擎知管理后台'
  document.title = pageTitle ? `${pageTitle} - ${appTitle}` : appTitle
})
