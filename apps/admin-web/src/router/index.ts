import { createRouter, createWebHistory } from 'vue-router'
import AdminLayout from '@/layouts/AdminLayout.vue'
import DictsView from '@/views/DictsView.vue'
import LoginLogsView from '@/views/LoginLogsView.vue'
import OperateLogsView from '@/views/OperateLogsView.vue'
import PermissionLogsView from '@/views/PermissionLogsView.vue'
import SecurityPolicyView from '@/views/SecurityPolicyView.vue'
import HomeView from '@/views/HomeView.vue'
import KnowledgeDetailView from '@/views/KnowledgeDetailView.vue'
import KnowledgeView from '@/views/KnowledgeView.vue'
import SettingsView from '@/views/SettingsView.vue'
import LoginView from '@/views/LoginView.vue'
import MonitorView from '@/views/MonitorView.vue'
import NotFoundView from '@/views/NotFoundView.vue'
import MenusView from '@/views/MenusView.vue'
import OrganizationsView from '@/views/OrganizationsView.vue'
import PostsView from '@/views/PostsView.vue'
import RolesView from '@/views/RolesView.vue'
import TenantsView from '@/views/TenantsView.vue'
import UsersView from '@/views/UsersView.vue'
import { useAuthStore } from '@/stores/auth'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: LoginView,
      meta: { title: '登录', public: true },
    },
    {
      path: '/',
      component: AdminLayout,
      meta: { requiresAuth: true },
      children: [
        {
          path: '',
          name: 'home',
          component: HomeView,
          meta: { title: '首页', breadcrumb: ['首页'] },
        },
        {
          path: 'tenants',
          name: 'tenants',
          component: TenantsView,
          meta: { title: '租户管理', breadcrumb: ['系统管理', '租户管理'] },
        },
        {
          path: 'organizations',
          name: 'organizations',
          component: OrganizationsView,
          meta: { title: '组织管理', breadcrumb: ['系统管理', '组织管理'] },
        },
        {
          path: 'posts',
          name: 'posts',
          component: PostsView,
          meta: { title: '岗位管理', breadcrumb: ['系统管理', '岗位管理'] },
        },
        {
          path: 'roles',
          name: 'roles',
          component: RolesView,
          meta: { title: '角色管理', breadcrumb: ['系统管理', '角色管理'] },
        },
        {
          path: 'menus',
          name: 'menus',
          component: MenusView,
          meta: { title: '菜单管理', breadcrumb: ['系统管理', '菜单管理'] },
        },
        {
          path: 'users',
          name: 'users',
          component: UsersView,
          meta: { title: '用户管理', breadcrumb: ['系统管理', '用户管理'] },
        },
        {
          path: 'knowledge',
          name: 'knowledge',
          component: KnowledgeView,
          meta: { title: '知识管理', breadcrumb: ['知识管理'] },
        },
        {
          path: 'models',
          redirect: { name: 'settings' },
        },
        {
          path: 'settings',
          name: 'settings',
          component: SettingsView,
          meta: { title: '系统配置', breadcrumb: ['系统管理', '系统配置'] },
        },
        {
          path: 'knowledge/:id',
          name: 'knowledge-detail',
          component: KnowledgeDetailView,
          meta: { title: '知识库', breadcrumb: ['知识管理', '知识库'] },
        },
        {
          path: 'dicts',
          name: 'dicts',
          component: DictsView,
          meta: { title: '字典管理', breadcrumb: ['系统管理', '字典管理'] },
        },
        {
          path: 'audit/operates',
          name: 'operate-logs',
          component: OperateLogsView,
          meta: { title: '操作审计', breadcrumb: ['安全审计', '操作审计'] },
        },
        {
          path: 'audit/logins',
          name: 'login-logs',
          component: LoginLogsView,
          meta: { title: '登录日志', breadcrumb: ['安全审计', '登录日志'] },
        },
        {
          path: 'audit/permissions',
          name: 'permission-logs',
          component: PermissionLogsView,
          meta: { title: '权限变更', breadcrumb: ['安全审计', '权限变更'] },
        },
        {
          path: 'monitor',
          name: 'monitor',
          component: MonitorView,
          meta: { title: '监控概览', breadcrumb: ['系统监控', '监控概览'] },
        },
        {
          path: 'audit/policy',
          name: 'security-policy',
          component: SecurityPolicyView,
          meta: { title: '安全策略', breadcrumb: ['安全审计', '安全策略'] },
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

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (!auth.bootstrapped) {
    await auth.restore()
  }
  if (to.meta.public) {
    if (auth.isAuthenticated && to.name === 'login') {
      return { path: '/' }
    }
    return true
  }
  if (!auth.isAuthenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  return true
})

router.afterEach((to) => {
  const pageTitle = typeof to.meta.title === 'string' ? to.meta.title : ''
  const appTitle = import.meta.env.VITE_APP_TITLE || '擎知管理后台'
  document.title = pageTitle ? `${pageTitle} - ${appTitle}` : appTitle
})
