<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { RsButton, RsInput } from '@/components/ui'
import { changeOpsPassword, getStaff, logoutOps, type OpsStaff } from '@/api'
import { formatApiError } from '@/utils/format'

const route = useRoute()
const router = useRouter()

const staff = ref<OpsStaff | null>(getStaff())
const menuOpen = ref(false)
const collapsed = ref(false)
const showPwd = ref(false)
const currentPassword = ref('')
const newPassword = ref('')
const pwdError = ref('')
const pwdOk = ref('')
const pwdLoading = ref(false)

const markSrc = `${import.meta.env.BASE_URL}favicon.svg`

const pageTitle = computed(() => route.meta.title || '控制台')

const pageSubtitle = computed(() => {
  if (route.name === 'dashboard') {
    const hour = new Date().getHours()
    const greet = hour < 12 ? '上午好' : hour < 18 ? '下午好' : '晚上好'
    const name = staff.value?.displayName?.trim() || staff.value?.email || '同事'
    return `${greet}，${name} · 运营数据总览与快捷入口`
  }
  return typeof route.meta.subtitle === 'string' ? route.meta.subtitle : ''
})

const pageIcon = computed(() => {
  switch (route.name) {
    case 'dashboard':
      return 'M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-4 0h4'
    case 'users':
    case 'user-detail':
      return 'M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2M9 11a4 4 0 100-8 4 4 0 000 8z'
    case 'staff':
      return 'M16 21v-2a4 4 0 00-4-4H6a4 4 0 00-4 4v2M9 11a4 4 0 100-8 4 4 0 000 8zM19 8v6M22 11h-6'
    case 'feedback':
      return 'M21 15a2 2 0 01-2 2H7l-4 4V5a2 2 0 012-2h14a2 2 0 012 2z'
    case 'releases':
      return 'M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4'
    case 'audit':
      return 'M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8zM14 2v6h6M16 13H8M16 17H8M10 9H8'
    default:
      return 'M4 6h16M4 12h16M4 18h10'
  }
})

const pageIconTone = computed(() => {
  switch (route.name) {
    case 'dashboard':
      return 'blue'
    case 'users':
    case 'user-detail':
      return 'cyan'
    case 'staff':
      return 'amber'
    case 'feedback':
      return 'orange'
    case 'releases':
      return 'cyan'
    case 'audit':
      return 'green'
    default:
      return 'blue'
  }
})

const breadcrumbs = computed(() => {
  const crumbs: Array<{ label: string; to?: string }> = [{ label: '首页', to: '/' }]
  const name = route.name
  if (name === 'dashboard') {
    crumbs.push({ label: '概览' })
  } else if (name === 'users') {
    crumbs.push({ label: '账户' })
  } else if (name === 'user-detail') {
    crumbs.push({ label: '账户', to: '/users' })
    crumbs.push({ label: '详情' })
  } else if (name === 'staff') {
    crumbs.push({ label: '员工' })
  } else if (name === 'feedback') {
    crumbs.push({ label: '反馈' })
  } else if (name === 'releases') {
    crumbs.push({ label: '发布' })
  } else if (name === 'audit') {
    crumbs.push({ label: '审计' })
  } else {
    crumbs.push({ label: String(pageTitle.value) })
  }
  return crumbs
})

const staffLabel = computed(() => {
  const s = staff.value
  if (!s) return '员工'
  return s.displayName?.trim() || s.email
})

const staffInitial = computed(() => staffLabel.value.slice(0, 1).toUpperCase())

const navItems: Array<{ to: string; label: string; icon: string; exact?: boolean }> = [
  {
    to: '/',
    label: '概览',
    exact: true,
    icon: 'M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-4 0h4',
  },
  {
    to: '/users',
    label: '账户',
    icon: 'M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2M9 11a4 4 0 100-8 4 4 0 000 8zM23 21v-2a4 4 0 00-3-3.87M16 3.13a4 4 0 010 7.75',
  },
  {
    to: '/staff',
    label: '员工',
    icon: 'M16 21v-2a4 4 0 00-4-4H6a4 4 0 00-4 4v2M9 11a4 4 0 100-8 4 4 0 000 8zM19 8v6M22 11h-6',
  },
  {
    to: '/feedback',
    label: '反馈',
    icon: 'M21 15a2 2 0 01-2 2H7l-4 4V5a2 2 0 012-2h14a2 2 0 012 2z',
  },
  {
    to: '/releases',
    label: '发布',
    icon: 'M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4',
  },
  {
    to: '/audit',
    label: '审计',
    icon: 'M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8zM14 2v6h6M16 13H8M16 17H8M10 9H8',
  },
]

function isActive(to: string, exact?: boolean): boolean {
  if (exact) return route.path === '/' || route.path === ''
  return route.path === to || route.path.startsWith(`${to}/`)
}

function toggleCollapsed() {
  collapsed.value = !collapsed.value
}

function toggleMenu() {
  menuOpen.value = !menuOpen.value
}

function closeMenu() {
  menuOpen.value = false
}

function openChangePassword() {
  closeMenu()
  showPwd.value = true
  currentPassword.value = ''
  newPassword.value = ''
  pwdError.value = ''
  pwdOk.value = ''
}

async function submitPassword() {
  pwdError.value = ''
  pwdOk.value = ''
  pwdLoading.value = true
  try {
    await changeOpsPassword({
      currentPassword: currentPassword.value,
      newPassword: newPassword.value,
    })
    pwdOk.value = '密码已更新'
    currentPassword.value = ''
    newPassword.value = ''
  } catch (e) {
    const msg = formatApiError(e, '修改失败')
    if (msg) pwdError.value = msg
  } finally {
    pwdLoading.value = false
  }
}

async function logout() {
  closeMenu()
  await logoutOps()
  await router.replace({ name: 'login' })
}

function onDocClick(e: MouseEvent) {
  const t = e.target as HTMLElement | null
  if (!t?.closest?.('.ops-user')) closeMenu()
}

onMounted(() => {
  staff.value = getStaff()
  document.addEventListener('click', onDocClick)
})

onUnmounted(() => {
  document.removeEventListener('click', onDocClick)
})
</script>

<template>
  <div class="ops-shell" :class="{ 'ops-shell--collapsed': collapsed }">
    <aside class="ops-sider" aria-label="主导航">
      <router-link class="ops-sider__brand" to="/" :title="collapsed ? 'NiuMa Ops' : undefined">
        <img class="ops-sider__mark" :src="markSrc" width="28" height="28" alt="" />
        <span v-if="!collapsed" class="ops-sider__brand-text">NiuMa Ops</span>
      </router-link>

      <div class="ops-sider__group">
        <p v-if="!collapsed" class="ops-sider__group-label">运营</p>
        <nav class="ops-sider__menu">
          <router-link
            v-for="item in navItems"
            :key="item.to"
            :to="item.to"
            class="ops-sider__link"
            active-class=""
            exact-active-class=""
            :class="{ 'ops-sider__link--active': isActive(item.to, item.exact) }"
            :title="collapsed ? item.label : undefined"
          >
            <svg class="ops-sider__icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <path
                :d="item.icon"
                stroke="currentColor"
                stroke-width="1.75"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
            <span v-if="!collapsed">{{ item.label }}</span>
          </router-link>
        </nav>
      </div>
    </aside>

    <div class="ops-workspace">
      <header class="ops-header">
        <div class="ops-header__left">
          <button
            type="button"
            class="ops-header__trigger"
            :aria-label="collapsed ? '展开侧栏' : '收起侧栏'"
            @click="toggleCollapsed"
          >
            <svg viewBox="0 0 24 24" width="18" height="18" fill="none" aria-hidden="true">
              <path
                d="M4 7h16M4 12h16M4 17h16"
                stroke="currentColor"
                stroke-width="1.75"
                stroke-linecap="round"
              />
            </svg>
          </button>

          <nav class="ops-breadcrumb" aria-label="面包屑">
            <template v-for="(c, i) in breadcrumbs" :key="`${c.label}-${i}`">
              <span v-if="i > 0" class="ops-breadcrumb__sep">/</span>
              <router-link v-if="c.to" :to="c.to" class="ops-breadcrumb__link">{{ c.label }}</router-link>
              <span v-else class="ops-breadcrumb__current">{{ c.label }}</span>
            </template>
          </nav>
        </div>

        <div class="ops-header__right">
          <span class="ops-header__env">内部</span>

          <div class="ops-user">
            <button type="button" class="ops-user__btn" aria-haspopup="menu" :aria-expanded="menuOpen" @click.stop="toggleMenu">
              <span class="ops-user__avatar">{{ staffInitial }}</span>
              <span class="ops-user__name">{{ staffLabel }}</span>
              <svg class="ops-user__caret" viewBox="0 0 24 24" width="14" height="14" fill="none" aria-hidden="true">
                <path d="M6 9l6 6 6-6" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" />
              </svg>
            </button>

            <div v-if="menuOpen" class="ops-user__menu" role="menu">
              <div class="ops-user__menu-head">
                <p class="ops-user__menu-name">{{ staffLabel }}</p>
                <p v-if="staff?.email" class="ops-user__menu-email">
                  {{ staff.email }}
                  <span v-if="staff.role"> · {{ staff.role === 'admin' ? '超管' : '运营' }}</span>
                </p>
              </div>
              <button type="button" class="ops-user__menu-item" role="menuitem" @click="openChangePassword">修改密码</button>
              <button type="button" class="ops-user__menu-item" role="menuitem" @click="logout">退出登录</button>
            </div>
          </div>
        </div>
      </header>

      <main class="ops-main">
        <div class="ops-page-head">
          <div class="ops-page-head__inner">
            <span class="ops-page-head__icon" :class="`ops-page-head__icon--${pageIconTone}`" aria-hidden="true">
              <svg viewBox="0 0 24 24" fill="none">
                <path
                  :d="pageIcon"
                  stroke="currentColor"
                  stroke-width="1.75"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </span>
            <div class="ops-page-head__text">
              <div class="ops-page-head__title-row">
                <h1 class="ops-page-head__title">{{ pageTitle }}</h1>
                <span v-if="route.name === 'dashboard'" class="ops-page-head__tag">控制台</span>
              </div>
              <p v-if="pageSubtitle" class="ops-page-head__desc">{{ pageSubtitle }}</p>
            </div>
          </div>
        </div>
        <div class="ops-page-body">
          <router-view />
        </div>
      </main>
    </div>

    <div v-if="showPwd" class="ops-overlay" role="dialog" aria-modal="true" aria-labelledby="pwd-title" @click.self="showPwd = false">
      <div class="ops-modal">
        <div class="ops-modal__head">
          <h2 id="pwd-title" class="ops-modal__title">修改密码</h2>
          <button type="button" class="ops-modal__close" aria-label="关闭" @click="showPwd = false">×</button>
        </div>
        <div class="ops-modal__body">
          <label class="ops-field">
            <span>当前密码</span>
            <RsInput v-model="currentPassword" type="password" autocomplete="current-password" />
          </label>
          <label class="ops-field">
            <span>新密码</span>
            <RsInput v-model="newPassword" type="password" autocomplete="new-password" placeholder="至少 10 位，含字母和数字" />
          </label>
          <p v-if="pwdError" class="ops-error">{{ pwdError }}</p>
          <p v-if="pwdOk" style="margin: 0; color: #52c41a; font-size: 14px">{{ pwdOk }}</p>
        </div>
        <div class="ops-modal__foot">
          <RsButton variant="secondary" @click="showPwd = false">关闭</RsButton>
          <RsButton variant="primary" :loading="pwdLoading" @click="submitPassword">保存</RsButton>
        </div>
      </div>
    </div>
  </div>
</template>
