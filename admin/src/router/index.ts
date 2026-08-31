import { createRouter, createWebHistory } from 'vue-router'
import { getToken, tryRestoreSession } from '@/api'

export const router = createRouter({
  history: createWebHistory('/niuma/ops/'),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/login/LoginView.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      component: () => import('@/layouts/OpsLayout.vue'),
      children: [
        {
          path: '',
          name: 'dashboard',
          component: () => import('@/views/dashboard/DashboardView.vue'),
          meta: { title: '概览', subtitle: '运营数据总览与快捷入口' },
        },
        {
          path: 'users',
          name: 'users',
          component: () => import('@/views/users/UsersView.vue'),
          meta: { title: '账户', subtitle: '终端用户：检索、创建、重置密码与启停' },
        },
        {
          path: 'users/:id',
          name: 'user-detail',
          component: () => import('@/views/users/UserDetailView.vue'),
          meta: { title: '账户详情', subtitle: '用户资料与关联反馈' },
        },
        {
          path: 'staff',
          name: 'staff',
          component: () => import('@/views/staff/StaffView.vue'),
          meta: { title: '员工', subtitle: '运营后台账号：创建、重置密码与启停' },
        },
        {
          path: 'feedback',
          name: 'feedback',
          component: () => import('@/views/feedback/FeedbackView.vue'),
          meta: { title: '反馈', subtitle: '分拣、备注与回复用户反馈' },
        },
        {
          path: 'releases',
          name: 'releases',
          component: () => import('@/views/releases/ReleasesView.vue'),
          meta: { title: '发布', subtitle: '桌面端版本与更新说明（官网与客户端共用）' },
        },
        {
          path: 'ai',
          name: 'ai',
          component: () => import('@/views/ai/AiView.vue'),
          meta: { title: 'AI', subtitle: '系统模型接入点、目录与按日用量' },
        },
        {
          path: 'audit',
          name: 'audit',
          component: () => import('@/views/audit/AuditView.vue'),
          meta: { title: '审计', subtitle: '关键操作日志查询' },
        },
      ],
    },
  ],
})

let bootstrapped = false

router.beforeEach(async (to) => {
  if (to.meta.public) return true
  if (!getToken() && !bootstrapped) {
    bootstrapped = true
    await tryRestoreSession()
  }
  if (!getToken()) return { name: 'login', query: { redirect: to.fullPath } }
  return true
})
