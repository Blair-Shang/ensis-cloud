<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { RsBadge, RsButton, RsInput, RsTable, type RsTableColumn } from '@/components/ui'
import OpsPager from '@/components/OpsPager.vue'
import {
  fetchOpsAIUsage,
  type OpsAIUsageRow,
  type OpsAIUsageSummary,
} from '@/api'
import {
  accountStatusLabel,
  accountStatusVariant,
  formatApiError,
  formatDateTime,
  initialOf,
} from '@/utils/format'
import AiUsageChart from './AiUsageChart.vue'
import { shortUsageLabel, type UsageChartSpec } from './usage-chart'

const pageSize = 20
const day = ref(utcToday())
const email = ref('')
const page = ref(1)
const items = ref<OpsAIUsageRow[]>([])
const top = ref<OpsAIUsageRow[]>([])
const trend = ref<OpsAIUsageSummary[]>([])
const total = ref(0)
const summary = ref<OpsAIUsageSummary>({
  usageDay: '',
  userCount: 0,
  requestCount: 0,
  tokenCount: 0,
})
const requestLimit = ref(0)
const error = ref('')
const loading = ref(false)

const columns: RsTableColumn<OpsAIUsageRow>[] = [
  { key: 'user', title: '用户', minWidth: 260 },
  { key: 'status', title: '状态', width: 88 },
  { key: 'requestCount', title: '请求', width: 88 },
  { key: 'quota', title: '日配额', width: 160 },
  { key: 'tokenCount', title: 'Token', width: 88 },
  { key: 'updatedAt', title: '最近调用', width: 180 },
]

const quotaHint = computed(() => {
  if (requestLimit.value > 0) return `每用户 ${requestLimit.value} 次 / UTC 日`
  return '当前接入点未设日请求上限'
})

const avgPerUser = computed(() => {
  if (!summary.value.userCount) return 0
  return Math.round((summary.value.requestCount / summary.value.userCount) * 10) / 10
})

const lineSpec = computed<UsageChartSpec>(() => ({
  type: 'line',
  title: '近 7 日调用',
  labels: trend.value.map((d) => (d.usageDay.length >= 10 ? d.usageDay.slice(5) : d.usageDay)),
  values: trend.value.map((d) => d.requestCount),
}))

const barSpec = computed<UsageChartSpec>(() => ({
  type: 'bar',
  title: '当日用户排行',
  labels: top.value.map((row) => shortUsageLabel(row)),
  values: top.value.map((row) => row.requestCount),
}))

function utcToday(): string {
  return new Date().toISOString().slice(0, 10)
}

function quotaVariant(count: number): 'success' | 'warning' | 'danger' | 'default' {
  const limit = requestLimit.value
  if (limit <= 0) return 'default'
  if (count >= limit) return 'danger'
  if (count >= Math.ceil(limit * 0.8)) return 'warning'
  return 'success'
}

function quotaText(count: number): string {
  if (requestLimit.value <= 0) return '不限'
  return `${count} / ${requestLimit.value}`
}

function quotaPct(count: number): number {
  if (requestLimit.value <= 0) return 0
  return Math.min(100, Math.round((count / requestLimit.value) * 100))
}

async function load() {
  error.value = ''
  loading.value = true
  try {
    const res = await fetchOpsAIUsage({
      day: day.value,
      email: email.value.trim() || undefined,
      limit: pageSize,
      offset: (page.value - 1) * pageSize,
    })
    items.value = res.items ?? []
    top.value = res.top ?? res.items?.slice(0, 8) ?? []
    total.value = res.total
    summary.value = res.summary
    trend.value = res.trend ?? []
    requestLimit.value = res.dailyRequestLimit
    day.value = res.day
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

let timer: number | undefined

onMounted(() => {
  void load()
  timer = window.setInterval(() => {
    void load()
  }, 30000)
})

onUnmounted(() => {
  if (timer) window.clearInterval(timer)
})
</script>

<template>
  <div class="ai-usage">
    <section class="ops-panel-banner">
      <div>
        <h2 class="ops-panel-banner__title">系统模型调用监控</h2>
        <p class="ops-panel-banner__desc">
          按 UTC 日看登录用户走云端代理的次数。图表与对话围栏同一套 SVG，不拉上游明细。
        </p>
      </div>
      <div class="ops-panel-banner__meta">
        <span class="ops-chip">
          <span class="ops-chip__dot" aria-hidden="true" />
          每 30 秒刷新
        </span>
        <span class="ops-chip">{{ quotaHint }}</span>
      </div>
    </section>

    <div class="ops-toolbar">
      <RsInput v-model="day" type="date" style="width: 160px" @change="search" />
      <RsInput
        v-model="email"
        placeholder="邮箱（精确）"
        style="min-width: 200px"
        @keydown.enter="search"
      />
      <RsButton variant="secondary" :loading="loading" @click="search">查询</RsButton>
    </div>

    <div v-if="error" class="ops-alert">{{ error }}</div>

    <div class="ops-stats ai-usage__stats">
      <div class="ops-card ops-stat ops-stat--ai">
        <div class="ops-stat__meta">
          <h3>当日调用</h3>
          <p class="ops-stat__value">{{ loading && !summary.requestCount ? '—' : summary.requestCount }}</p>
          <p class="ops-stat__hint">系统模型对话次数</p>
        </div>
      </div>
      <div class="ops-card ops-stat ops-stat--users">
        <div class="ops-stat__meta">
          <h3>调用用户</h3>
          <p class="ops-stat__value">{{ loading && !summary.userCount ? '—' : summary.userCount }}</p>
          <p class="ops-stat__hint">当日至少请求 1 次</p>
        </div>
      </div>
      <div class="ops-card ops-stat ops-stat--week">
        <div class="ops-stat__meta">
          <h3>人均</h3>
          <p class="ops-stat__value">{{ loading && !avgPerUser ? '—' : avgPerUser }}</p>
          <p class="ops-stat__hint">当日请求 / 调用用户</p>
        </div>
      </div>
    </div>

    <div class="ai-usage__charts">
      <section class="ops-card ai-usage__chart-card">
        <AiUsageChart :spec="lineSpec" empty="近 7 日还没有调用" />
      </section>
      <section class="ops-card ai-usage__chart-card">
        <AiUsageChart :spec="barSpec" empty="这一天还没有用户排行" />
      </section>
    </div>

    <div class="ops-card ops-card--flush">
      <RsTable :columns="columns" :data="items" row-key="userId" :loading="loading" :bordered="false">
        <template #user="{ row }">
          <div class="ops-person">
            <span class="ops-avatar">{{ initialOf(row.displayName || row.email || row.userId) }}</span>
            <div class="ops-person__meta">
              <router-link
                v-if="row.userId"
                :to="`/users/${row.userId}`"
                class="ops-person__name"
              >
                {{ row.email || row.userId }}
              </router-link>
              <p class="ops-person__sub">{{ row.displayName || '未设置显示名' }}</p>
            </div>
          </div>
        </template>
        <template #status="{ row }">
          <RsBadge v-if="row.status" :variant="accountStatusVariant(row.status)">
            {{ accountStatusLabel(row.status) }}
          </RsBadge>
          <span v-else class="ops-muted">—</span>
        </template>
        <template #requestCount="{ row }">
          <span class="ai-usage__num">{{ row.requestCount }}</span>
        </template>
        <template #quota="{ row }">
          <div class="ai-usage__quota">
            <RsBadge :variant="quotaVariant(row.requestCount)">{{ quotaText(row.requestCount) }}</RsBadge>
            <span
              v-if="requestLimit > 0"
              class="ai-usage__bar"
              :class="{
                'ai-usage__bar--warn': quotaVariant(row.requestCount) === 'warning',
                'ai-usage__bar--full': quotaVariant(row.requestCount) === 'danger',
              }"
            >
              <i :style="{ width: `${quotaPct(row.requestCount)}%` }" />
            </span>
          </div>
        </template>
        <template #tokenCount="{ row }">
          <span class="ops-muted">{{ row.tokenCount || '—' }}</span>
        </template>
        <template #updatedAt="{ row }">
          <span class="ops-muted" style="white-space: nowrap">{{ formatDateTime(row.updatedAt) }}</span>
        </template>
        <template #empty>
          <div class="ops-empty">
            <p class="ops-empty__title">这一天还没有系统模型调用</p>
            <p class="ops-empty__desc">登录用户对话成功后会出现在这里</p>
          </div>
        </template>
      </RsTable>
      <div class="ops-card__footer">
        <OpsPager
          :total="total"
          :page="page"
          :page-size="pageSize"
          @change="
            (p) => {
              page = p
              void load()
            }
          "
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.ai-usage__stats {
  grid-template-columns: repeat(3, minmax(0, 1fr));
  margin-bottom: 16px;
}

.ai-usage__charts {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  margin-bottom: 16px;
}

.ai-usage__chart-card {
  padding: 8px 12px 4px;
}

.ai-usage__num {
  font-variant-numeric: tabular-nums;
  font-weight: 600;
}

.ai-usage__quota {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 110px;
}

.ai-usage__bar {
  display: block;
  height: 4px;
  border-radius: 999px;
  background: #f0f0f0;
  overflow: hidden;
}

.ai-usage__bar i {
  display: block;
  height: 100%;
  background: #1677ff;
  border-radius: inherit;
}

.ai-usage__bar--warn i {
  background: #faad14;
}

.ai-usage__bar--full i {
  background: #ff4d4f;
}

@media (max-width: 1100px) {
  .ai-usage__stats,
  .ai-usage__charts {
    grid-template-columns: 1fr;
  }
}
</style>
