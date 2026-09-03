<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RsButton, RsInput, RsBadge, RsSelect, RsTable, type RsTableColumn } from '@/components/ui'
import OpsPager from '@/components/OpsPager.vue'
import {
  createOpsStaff,
  fetchOpsStaff,
  getStaff,
  isOpsAdmin,
  patchOpsStaff,
  resetOpsStaffPassword,
  type OpsStaffAccount,
} from '@/api'
import {
  accountStatusLabel,
  accountStatusVariant,
  formatApiError,
  formatDateTime,
  initialOf,
} from '@/utils/format'

const email = ref('')
const status = ref('all')
const role = ref('all')
const page = ref(1)
const pageSize = 20
const items = ref<OpsStaffAccount[]>([])
const total = ref(0)
const error = ref('')
const loading = ref(false)
const me = getStaff()
const canAdmin = isOpsAdmin(me)

const statusOptions = [
  { label: '全部状态', value: 'all' },
  { label: '正常', value: 'active' },
  { label: '停用', value: 'disabled' },
]

const roleOptions = [
  { label: '全部角色', value: 'all' },
  { label: '超管', value: 'admin' },
  { label: '运营', value: 'operator' },
]

const showCreate = ref(false)
const createEmail = ref('')
const createName = ref('')
const createPassword = ref('')
const creating = ref(false)

const columns: RsTableColumn<OpsStaffAccount>[] = [
  { key: 'staff', title: '员工', minWidth: 260, ellipsis: true },
  { key: 'role', title: '角色', width: 88 },
  { key: 'status', title: '状态', width: 100 },
  { key: 'createdAt', title: '创建时间', width: 180 },
  { key: 'actions', title: '操作', width: 150 },
]

const pwdTarget = ref<OpsStaffAccount | null>(null)
const newPassword = ref('')
const resetting = ref(false)

function mapErr(e: unknown): string {
  return formatApiError(e)
}

async function load() {
  error.value = ''
  loading.value = true
  try {
    const res = await fetchOpsStaff({
      email: email.value.trim() || undefined,
      status: status.value === 'all' ? undefined : status.value,
      role: role.value === 'all' ? undefined : role.value,
      limit: pageSize,
      offset: (page.value - 1) * pageSize,
    })
    items.value = res.items
    total.value = res.total
  } catch (e) {
    const msg = mapErr(e)
    if (msg) error.value = msg
  } finally {
    loading.value = false
  }
}

function openCreate() {
  showCreate.value = true
  createEmail.value = ''
  createName.value = ''
  createPassword.value = ''
  error.value = ''
}

async function create() {
  creating.value = true
  error.value = ''
  try {
    await createOpsStaff({
      email: createEmail.value.trim(),
      password: createPassword.value,
      displayName: createName.value.trim() || undefined,
    })
    showCreate.value = false
    await load()
  } catch (e) {
    const msg = mapErr(e)
    if (msg) error.value = msg
  } finally {
    creating.value = false
  }
}

async function toggleStatus(row: OpsStaffAccount) {
  const next = row.status === 'active' ? 'disabled' : 'active'
  error.value = ''
  try {
    await patchOpsStaff(row.id, { status: next })
    await load()
  } catch (e) {
    const msg = mapErr(e)
    if (msg) error.value = msg
  }
}

function openReset(row: OpsStaffAccount) {
  pwdTarget.value = row
  newPassword.value = ''
  error.value = ''
}

async function confirmReset() {
  if (!pwdTarget.value) return
  resetting.value = true
  error.value = ''
  try {
    await resetOpsStaffPassword(pwdTarget.value.id, newPassword.value)
    pwdTarget.value = null
  } catch (e) {
    const msg = mapErr(e)
    if (msg) error.value = msg
  } finally {
    resetting.value = false
  }
}

function search() {
  page.value = 1
  void load()
}

onMounted(() => {
  void load()
})
</script>

<template>
  <div>
    <div class="ops-toolbar">
      <RsInput v-model="email" placeholder="邮箱（精确）" style="min-width: 200px" @keydown.enter="search" />
      <RsSelect v-model="status" class="ops-select-ctl" :options="statusOptions" />
      <RsSelect v-model="role" class="ops-select-ctl" :options="roleOptions" />
      <RsButton variant="secondary" :loading="loading" @click="search">查询</RsButton>
      <div class="ops-toolbar__grow" />
      <div class="ops-toolbar__right">
        <RsButton v-if="canAdmin" variant="primary" @click="openCreate">新增员工</RsButton>
      </div>
    </div>

    <div v-if="error && !showCreate && !pwdTarget" class="ops-alert">{{ error }}</div>

    <div class="ops-card ops-card--flush">
      <RsTable :columns="columns" :data="items" row-key="id" :loading="loading" :bordered="false">
        <template #staff="{ row }">
          <div class="ops-person">
            <span class="ops-avatar ops-avatar--amber">{{ initialOf(row.displayName || row.email) }}</span>
            <div class="ops-person__meta">
              <p class="ops-person__name">
                {{ row.email }}
                <RsBadge v-if="me?.id === row.id" variant="info" style="margin-left: 6px">我</RsBadge>
              </p>
              <p class="ops-person__sub">{{ row.displayName || '未设置显示名' }}</p>
            </div>
          </div>
        </template>
        <template #role="{ row }">
          <RsBadge :variant="row.role === 'admin' ? 'info' : 'default'">
            {{ row.role === 'admin' ? '超管' : '运营' }}
          </RsBadge>
        </template>
        <template #status="{ row }">
          <RsBadge :variant="accountStatusVariant(row.status)">
            {{ accountStatusLabel(row.status) }}
          </RsBadge>
        </template>
        <template #createdAt="{ row }">
          <span class="ops-muted" style="white-space: nowrap">{{ formatDateTime(row.createdAt) }}</span>
        </template>
        <template #actions="{ row }">
          <div v-if="canAdmin" class="ops-row">
            <button type="button" class="ops-link-btn" @click="openReset(row)">重置密码</button>
            <span class="ops-link-sep">|</span>
            <button
              type="button"
              class="ops-link-btn"
              :class="{ 'ops-link-btn--danger': row.status === 'active' }"
              :disabled="me?.id === row.id && row.status === 'active'"
              @click="toggleStatus(row)"
            >
              {{ row.status === 'active' ? '停用' : '启用' }}
            </button>
          </div>
          <span v-else class="ops-muted">—</span>
        </template>
        <template #empty>
          <div class="ops-empty">
            <div class="ops-empty__icon" aria-hidden="true">
              <svg viewBox="0 0 24 24" fill="none">
                <path
                  d="M16 21v-2a4 4 0 00-4-4H6a4 4 0 00-4 4v2M9 11a4 4 0 100-8 4 4 0 000 8z"
                  stroke="currentColor"
                  stroke-width="1.5"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </div>
            <p class="ops-empty__title">暂无员工</p>
          </div>
        </template>
      </RsTable>
      <div class="ops-card__footer">
        <OpsPager :total="total" :page="page" :page-size="pageSize" @change="(p) => { page = p; void load() }" />
      </div>
    </div>

    <div v-if="showCreate" class="ops-overlay" @click.self="showCreate = false">
      <div class="ops-modal" role="dialog" aria-modal="true">
        <div class="ops-modal__head">
          <h2 class="ops-modal__title">新增员工账号</h2>
          <button type="button" class="ops-modal__close" aria-label="关闭" @click="showCreate = false">×</button>
        </div>
        <div class="ops-modal__body">
          <p v-if="error" class="ops-error">{{ error }}</p>
          <label class="ops-field">
            <span>邮箱</span>
            <RsInput v-model="createEmail" type="email" placeholder="name@niuma.com" />
          </label>
          <label class="ops-field">
            <span>显示名</span>
            <RsInput v-model="createName" placeholder="可选" />
          </label>
          <label class="ops-field">
            <span>初始密码</span>
            <RsInput v-model="createPassword" type="password" placeholder="至少 10 位，含字母和数字" />
          </label>
        </div>
        <div class="ops-modal__foot">
          <RsButton variant="secondary" @click="showCreate = false">取消</RsButton>
          <RsButton variant="primary" :loading="creating" @click="create">创建</RsButton>
        </div>
      </div>
    </div>

    <div v-if="pwdTarget" class="ops-overlay" @click.self="pwdTarget = null">
      <div class="ops-modal" role="dialog" aria-modal="true">
        <div class="ops-modal__head">
          <h2 class="ops-modal__title">重置密码</h2>
          <button type="button" class="ops-modal__close" aria-label="关闭" @click="pwdTarget = null">×</button>
        </div>
        <div class="ops-modal__body">
          <p class="ops-muted" style="margin: 0">员工：{{ pwdTarget.email }}</p>
          <p v-if="error" class="ops-error">{{ error }}</p>
          <label class="ops-field">
            <span>新密码</span>
            <RsInput v-model="newPassword" type="password" placeholder="至少 10 位，含字母和数字" />
          </label>
        </div>
        <div class="ops-modal__foot">
          <RsButton variant="secondary" @click="pwdTarget = null">取消</RsButton>
          <RsButton variant="primary" :loading="resetting" @click="confirmReset">确认重置</RsButton>
        </div>
      </div>
    </div>
  </div>
</template>
