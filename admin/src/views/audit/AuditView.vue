<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RsButton, RsInput, RsBadge, RsSelect, RsTable, type RsTableColumn } from '@/components/ui'
import OpsPager from '@/components/OpsPager.vue'
import { fetchOpsAuditLogs, type OpsAuditLog } from '@/api'
import { actorKindLabel, detailText, formatApiError, formatDateTime } from '@/utils/format'

const actorId = ref('')
const resourceId = ref('')
const action = ref('all')
const actorKind = ref('all')
const page = ref(1)
const pageSize = 20
const items = ref<OpsAuditLog[]>([])
const total = ref(0)
const error = ref('')
const loading = ref(false)

const actionOptions = [
  { label: '全部动作', value: 'all' },
  { label: '登录', value: 'ops.login' },
  { label: '创建员工', value: 'staff.create' },
  { label: '更新员工', value: 'staff.update' },
  { label: '重置员工密码', value: 'staff.password_reset' },
  { label: '修改密码', value: 'staff.password_change' },
  { label: '创建用户', value: 'user.create' },
  { label: '用户状态变更', value: 'user.status_update' },
  { label: '重置用户密码', value: 'user.password_reset' },
  { label: '更新反馈', value: 'feedback.update' },
  { label: '回复反馈', value: 'feedback.reply' },
  { label: '创建发布', value: 'release.create' },
  { label: '更新发布', value: 'release.update' },
  { label: '发布版本', value: 'release.publish' },
  { label: '撤回发布', value: 'release.yank' },
]

const actorKindOptions = [
  { label: '全部操作者', value: 'all' },
  { label: '员工', value: 'staff' },
  { label: '用户', value: 'user' },
  { label: '系统', value: 'system' },
]

const columns: RsTableColumn<OpsAuditLog>[] = [
  { key: 'createdAt', title: '时间', width: 180 },
  { key: 'actionCode', title: '动作', width: 180 },
  { key: 'actor', title: '操作者', width: 200 },
  { key: 'resource', title: '资源', width: 200 },
  { key: 'detail', title: '详情', minWidth: 240 },
]

async function load() {
  error.value = ''
  loading.value = true
  try {
    const res = await fetchOpsAuditLogs({
      actorId: actorId.value.trim() || undefined,
      resourceId: resourceId.value.trim() || undefined,
      action: action.value === 'all' ? undefined : action.value,
      actorKind: actorKind.value === 'all' ? undefined : actorKind.value,
      limit: pageSize,
      offset: (page.value - 1) * pageSize,
    })
    items.value = res.items
    total.value = res.total
  } catch (e) {
    const msg = formatApiError(e, '加载失败')
    if (msg) error.value = msg
  } finally {
    loading.value = false
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
      <RsInput v-model="actorId" placeholder="操作者 ID（精确）" style="min-width: 180px" @keydown.enter="search" />
      <RsInput v-model="resourceId" placeholder="资源 ID（精确）" style="min-width: 180px" @keydown.enter="search" />
      <RsSelect
        v-model="action"
        class="ops-select-ctl ops-select-ctl--wide"
        :options="actionOptions"
        placeholder="全部动作"
      />
      <RsSelect
        v-model="actorKind"
        class="ops-select-ctl"
        :options="actorKindOptions"
        placeholder="全部操作者"
      />
      <RsButton variant="secondary" :loading="loading" @click="search">筛选</RsButton>
    </div>

    <div v-if="error" class="ops-alert">{{ error }}</div>

    <div class="ops-card ops-card--flush">
      <RsTable :columns="columns" :data="items" row-key="id" :loading="loading" :bordered="false">
        <template #createdAt="{ row }">
          <span class="ops-muted" style="white-space: nowrap">{{ formatDateTime(row.createdAt) }}</span>
        </template>
        <template #actionCode="{ row }">
          <RsBadge variant="info">{{ row.actionCode }}</RsBadge>
        </template>
        <template #actor="{ row }">
          <div class="ops-table__cell-title">{{ actorKindLabel(row.actorKind) }}</div>
          <div class="ops-table__cell-sub ops-mono">{{ row.actorID || '—' }}</div>
        </template>
        <template #resource="{ row }">
          <div class="ops-table__cell-title">{{ row.resourceKind || '—' }}</div>
          <div class="ops-table__cell-sub ops-mono">{{ row.resourceID || '—' }}</div>
        </template>
        <template #detail="{ row }">
          <code class="ops-mono" style="color: rgba(0, 0, 0, 0.65); word-break: break-all">
            {{ detailText(row.detail) }}
          </code>
        </template>
        <template #empty>
          <div class="ops-empty">
            <div class="ops-empty__icon" aria-hidden="true">
              <svg viewBox="0 0 24 24" fill="none">
                <path
                  d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8zM14 2v6h6"
                  stroke="currentColor"
                  stroke-width="1.5"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </div>
            <p class="ops-empty__title">暂无审计记录</p>
            <p class="ops-empty__desc">登录、账号变更与反馈处理会自动留痕</p>
          </div>
        </template>
      </RsTable>
      <div class="ops-card__footer">
        <OpsPager :total="total" :page="page" :page-size="pageSize" @change="(p) => { page = p; void load() }" />
      </div>
    </div>
  </div>
</template>
