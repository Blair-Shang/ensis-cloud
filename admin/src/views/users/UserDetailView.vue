<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { RsBadge, RsButton, RsTable, type RsTableColumn } from '@/components/ui'
import { fetchOpsUserDetail, type OpsUserDetail } from '@/api'
import {
  accountStatusLabel,
  accountStatusVariant,
  feedbackStatusLabel,
  feedbackStatusVariant,
  formatApiError,
  formatDateTime,
  initialOf,
} from '@/utils/format'

const route = useRoute()
const data = ref<OpsUserDetail | null>(null)

const feedbackColumns: RsTableColumn<OpsUserDetail['feedback'][number]>[] = [
  { key: 'title', title: '标题', minWidth: 220 },
  { key: 'status', title: '状态', width: 96 },
  { key: 'createdAt', title: '时间', width: 170 },
]
const error = ref('')
const loading = ref(true)

onMounted(async () => {
  try {
    data.value = await fetchOpsUserDetail(String(route.params.id))
  } catch (e) {
    const msg = formatApiError(e, '加载失败')
    if (msg) error.value = msg
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div>
    <div v-if="loading" class="ops-empty">加载中…</div>
    <div v-else-if="error" class="ops-alert">{{ error }}</div>
    <template v-else-if="data">
      <div class="ops-profile">
        <span class="ops-profile__avatar">{{ initialOf(data.user.displayName || data.user.email) }}</span>
        <div style="min-width: 0; flex: 1">
          <h2 class="ops-profile__title">{{ data.user.email }}</h2>
          <p class="ops-profile__sub">
            {{ data.user.displayName || '未设置显示名' }}
            ·
            <RsBadge :variant="accountStatusVariant(data.user.status)" style="vertical-align: middle">
              {{ accountStatusLabel(data.user.status) }}
            </RsBadge>
          </p>
        </div>
        <RsButton variant="secondary" size="sm" @click="$router.push('/users')">返回列表</RsButton>
      </div>

      <div class="ops-desc" style="margin-bottom: 16px">
        <div class="ops-desc__item">
          <div class="ops-desc__label">用户 ID</div>
          <div class="ops-desc__value ops-mono">{{ data.user.id }}</div>
        </div>
        <div class="ops-desc__item">
          <div class="ops-desc__label">活跃会话</div>
          <div class="ops-desc__value">{{ data.sessionCount }}</div>
        </div>
        <div class="ops-desc__item">
          <div class="ops-desc__label">注册时间</div>
          <div class="ops-desc__value">{{ formatDateTime(data.user.createdAt) }}</div>
        </div>
        <div class="ops-desc__item">
          <div class="ops-desc__label">邮箱验证</div>
          <div class="ops-desc__value">
            {{ data.user.emailVerifiedAt ? formatDateTime(data.user.emailVerifiedAt) : '未验证' }}
          </div>
        </div>
      </div>

      <section class="ops-card ops-card--flush">
        <div class="ops-card__head">
          <h2 class="ops-card__title">关联反馈</h2>
          <span class="ops-muted">{{ data.feedback.length }} 条</span>
        </div>
        <RsTable :columns="feedbackColumns" :data="data.feedback" row-key="id" :bordered="false">
          <template #title="{ row }">
            <router-link to="/feedback" class="ops-table__cell-title">{{ row.title }}</router-link>
          </template>
          <template #status="{ row }">
            <RsBadge :variant="feedbackStatusVariant(row.status)">
              {{ feedbackStatusLabel(row.status) }}
            </RsBadge>
          </template>
          <template #createdAt="{ row }">
            <span class="ops-muted">{{ formatDateTime(row.createdAt) }}</span>
          </template>
          <template #empty>
            <div class="ops-empty">
              <p class="ops-empty__title">该用户暂无反馈</p>
            </div>
          </template>
        </RsTable>
      </section>
    </template>
  </div>
</template>
