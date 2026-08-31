<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RsBadge, RsTable, type RsTableColumn } from '@/components/ui'
import {
  fetchOpsFeedback,
  fetchOpsStats,
  type OpsFeedback,
  type OpsStats,
} from '@/api'
import { feedbackStatusLabel, feedbackStatusVariant, formatApiError, formatDateTime } from '@/utils/format'

const stats = ref<OpsStats>({
  users: 0,
  feedback: 0,
  feedbackLast7d: 0,
  downloadHits: 0,
  aiUsersToday: 0,
  aiRequestsToday: 0,
})
const recent = ref<OpsFeedback[]>([])
const error = ref('')
const loading = ref(true)

const todayLabel = computed(() =>
  new Date().toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    weekday: 'short',
  }),
)

const openCount = computed(() => recent.value.filter((f) => f.status === 'open').length)

const siteUrl = import.meta.env.VITE_SITE_URL || 'https://www.niuma007.com'

const recentColumns: RsTableColumn<OpsFeedback>[] = [
  { key: 'title', title: '标题', minWidth: 200 },
  { key: 'status', title: '状态', width: 96 },
  { key: 'createdAt', title: '时间', width: 170 },
]

onMounted(async () => {
  try {
    const [s, fb] = await Promise.all([fetchOpsStats(), fetchOpsFeedback({ limit: 8 })])
    stats.value = s
    recent.value = fb.items
  } catch (e) {
    const msg = formatApiError(e, '加载失败')
    if (msg) error.value = msg
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="dash">
    <div v-if="error" class="ops-alert">{{ error }}</div>

    <section class="ops-panel-banner">
      <div>
        <h2 class="ops-panel-banner__title">运营控制面板</h2>
        <p class="ops-panel-banner__desc">
          汇总注册用户、反馈工单、更新下载与近 7 日动态，快速进入账户、员工与审计管理。
        </p>
      </div>
      <div class="ops-panel-banner__meta">
        <span class="ops-chip">
          <span class="ops-chip__dot" aria-hidden="true" />
          服务正常
        </span>
        <span class="ops-chip">{{ todayLabel }}</span>
        <span v-if="!loading" class="ops-chip">待处理反馈 {{ openCount }}</span>
      </div>
    </section>

    <div class="ops-stats">
      <div class="ops-card ops-stat ops-stat--users">
        <div class="ops-stat__meta">
          <h3>注册用户</h3>
          <p class="ops-stat__value">{{ loading ? '—' : stats.users }}</p>
          <p class="ops-stat__hint">终端产品账户总数</p>
        </div>
        <div class="ops-stat__icon ops-stat__icon--users" aria-hidden="true">
          <svg viewBox="0 0 24 24" fill="none">
            <path
              d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2M9 11a4 4 0 100-8 4 4 0 000 8zM23 21v-2a4 4 0 00-3-3.87M16 3.13a4 4 0 010 7.75"
              stroke="currentColor"
              stroke-width="1.75"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
        </div>
      </div>

      <div class="ops-card ops-stat ops-stat--feedback">
        <div class="ops-stat__meta">
          <h3>反馈总数</h3>
          <p class="ops-stat__value">{{ loading ? '—' : stats.feedback }}</p>
          <p class="ops-stat__hint">历史累计工单</p>
        </div>
        <div class="ops-stat__icon ops-stat__icon--feedback" aria-hidden="true">
          <svg viewBox="0 0 24 24" fill="none">
            <path
              d="M21 15a2 2 0 01-2 2H7l-4 4V5a2 2 0 012-2h14a2 2 0 012 2z"
              stroke="currentColor"
              stroke-width="1.75"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
        </div>
      </div>

      <div class="ops-card ops-stat ops-stat--week">
        <div class="ops-stat__meta">
          <h3>近 7 日反馈</h3>
          <p class="ops-stat__value">{{ loading ? '—' : stats.feedbackLast7d }}</p>
          <p class="ops-stat__hint">近一周新增</p>
        </div>
        <div class="ops-stat__icon ops-stat__icon--week" aria-hidden="true">
          <svg viewBox="0 0 24 24" fill="none">
            <path
              d="M3 3v18h18M7 14l4-4 4 4 5-6"
              stroke="currentColor"
              stroke-width="1.75"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
        </div>
      </div>

      <div class="ops-card ops-stat ops-stat--downloads">
        <div class="ops-stat__meta">
          <h3>更新下载</h3>
          <p class="ops-stat__value">{{ loading ? '—' : stats.downloadHits ?? 0 }}</p>
          <p class="ops-stat__hint">客户端拉起安装包次数</p>
        </div>
        <div class="ops-stat__icon ops-stat__icon--downloads" aria-hidden="true">
          <svg viewBox="0 0 24 24" fill="none">
            <path
              d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
              stroke="currentColor"
              stroke-width="1.75"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
        </div>
      </div>

      <router-link class="ops-card ops-stat ops-stat--ai" to="/ai?tab=usage">
        <div class="ops-stat__meta">
          <h3>系统模型</h3>
          <p class="ops-stat__value">{{ loading ? '—' : stats.aiRequestsToday ?? 0 }}</p>
          <p class="ops-stat__hint">今日调用 · {{ loading ? '—' : stats.aiUsersToday ?? 0 }} 人</p>
        </div>
        <div class="ops-stat__icon ops-stat__icon--ai" aria-hidden="true">
          <svg viewBox="0 0 24 24" fill="none">
            <path
              d="M12 3v2m0 14v2m9-9h-2M5 12H3m13.657-6.657l-1.414 1.414M6.757 17.243l-1.414 1.414m12.728 0l-1.414-1.414M6.757 6.757L5.343 5.343M12 8a4 4 0 100 8 4 4 0 000-8z"
              stroke="currentColor"
              stroke-width="1.75"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
        </div>
      </router-link>
    </div>

    <div class="ops-grid-2">
      <section class="ops-card ops-card--flush">
        <div class="ops-card__head">
          <h2 class="ops-card__title">最近反馈</h2>
          <router-link class="ops-card__extra" to="/feedback">查看全部</router-link>
        </div>

        <RsTable :columns="recentColumns" :data="recent" row-key="id" :loading="loading" :bordered="false">
          <template #title="{ row }">
            <router-link to="/feedback" class="ops-table__cell-title">{{ row.title }}</router-link>
            <div class="ops-table__cell-sub">{{ row.category || '未分类' }}</div>
          </template>
          <template #status="{ row }">
            <RsBadge :variant="feedbackStatusVariant(row.status)">
              {{ feedbackStatusLabel(row.status) }}
            </RsBadge>
          </template>
          <template #createdAt="{ row }">
            <span class="ops-muted" style="white-space: nowrap">{{ formatDateTime(row.createdAt) }}</span>
          </template>
          <template #empty>
            <div class="ops-empty">
              <div class="ops-empty__icon" aria-hidden="true">
                <svg viewBox="0 0 24 24" fill="none">
                  <path
                    d="M21 15a2 2 0 01-2 2H7l-4 4V5a2 2 0 012-2h14a2 2 0 012 2z"
                    stroke="currentColor"
                    stroke-width="1.5"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                </svg>
              </div>
              <p class="ops-empty__title">暂无反馈</p>
              <p class="ops-empty__desc">用户提交后将显示在这里</p>
            </div>
          </template>
        </RsTable>
      </section>

      <section class="ops-card">
        <div class="ops-card__head">
          <h2 class="ops-card__title">快捷入口</h2>
        </div>
        <div class="ops-shortcuts">
          <router-link class="ops-shortcut" to="/users">
            <span class="ops-shortcut__icon" aria-hidden="true">
              <svg viewBox="0 0 24 24" fill="none">
                <path
                  d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2M9 11a4 4 0 100-8 4 4 0 000 8z"
                  stroke="currentColor"
                  stroke-width="1.75"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </span>
            <span class="ops-shortcut__text">
              <p class="ops-shortcut__title">账户管理</p>
              <p class="ops-shortcut__desc">创建 / 启停 / 重置密码</p>
            </span>
          </router-link>

          <router-link class="ops-shortcut" to="/feedback">
            <span class="ops-shortcut__icon" aria-hidden="true">
              <svg viewBox="0 0 24 24" fill="none">
                <path
                  d="M21 15a2 2 0 01-2 2H7l-4 4V5a2 2 0 012-2h14a2 2 0 012 2z"
                  stroke="currentColor"
                  stroke-width="1.75"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </span>
            <span class="ops-shortcut__text">
              <p class="ops-shortcut__title">反馈处理</p>
              <p class="ops-shortcut__desc">分拣工单与用户回复</p>
            </span>
          </router-link>

          <router-link class="ops-shortcut" to="/staff">
            <span class="ops-shortcut__icon" aria-hidden="true">
              <svg viewBox="0 0 24 24" fill="none">
                <path
                  d="M16 21v-2a4 4 0 00-4-4H6a4 4 0 00-4 4v2M9 11a4 4 0 100-8 4 4 0 000 8zM19 8v6M22 11h-6"
                  stroke="currentColor"
                  stroke-width="1.75"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </span>
            <span class="ops-shortcut__text">
              <p class="ops-shortcut__title">员工管理</p>
              <p class="ops-shortcut__desc">运营账号与权限启停</p>
            </span>
          </router-link>

          <router-link class="ops-shortcut" to="/ai?tab=usage">
            <span class="ops-shortcut__icon" aria-hidden="true">
              <svg viewBox="0 0 24 24" fill="none">
                <path
                  d="M12 3v2m0 14v2m9-9h-2M5 12H3m13.657-6.657l-1.414 1.414M6.757 17.243l-1.414 1.414m12.728 0l-1.414-1.414M6.757 6.757L5.343 5.343M12 8a4 4 0 100 8 4 4 0 000-8z"
                  stroke="currentColor"
                  stroke-width="1.75"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </span>
            <span class="ops-shortcut__text">
              <p class="ops-shortcut__title">模型用量</p>
              <p class="ops-shortcut__desc">按日查看谁调用了多少次</p>
            </span>
          </router-link>

          <router-link class="ops-shortcut" to="/audit">
            <span class="ops-shortcut__icon" aria-hidden="true">
              <svg viewBox="0 0 24 24" fill="none">
                <path
                  d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8zM14 2v6h6M16 13H8M16 17H8M10 9H8"
                  stroke="currentColor"
                  stroke-width="1.75"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </span>
            <span class="ops-shortcut__text">
              <p class="ops-shortcut__title">审计日志</p>
              <p class="ops-shortcut__desc">关键操作留痕查询</p>
            </span>
          </router-link>

          <a
            class="ops-shortcut"
            :href="siteUrl"
            target="_blank"
            rel="noopener noreferrer"
          >
            <span class="ops-shortcut__icon" aria-hidden="true">
              <svg viewBox="0 0 24 24" fill="none">
                <path
                  d="M12 21a9 9 0 100-18 9 9 0 000 18zM3.6 9h16.8M3.6 15h16.8M12 3c2.5 2.4 4 5.6 4 9s-1.5 6.6-4 9M12 3c-2.5 2.4-4 5.6-4 9s1.5 6.6 4 9"
                  stroke="currentColor"
                  stroke-width="1.75"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </span>
            <span class="ops-shortcut__text">
              <p class="ops-shortcut__title">官方网站</p>
              <p class="ops-shortcut__desc">下载页、FAQ 与用户入口</p>
            </span>
          </a>
        </div>
      </section>
    </div>
  </div>
</template>
