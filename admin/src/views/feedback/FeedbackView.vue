<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { RsButton, RsInput, RsBadge, RsSelect, RsTable, type RsTableColumn } from '@/components/ui'
import OpsPager from '@/components/OpsPager.vue'
import {
  fetchOpsFeedback,
  fetchOpsFeedbackAttachmentURL,
  updateOpsFeedback,
  type OpsFeedback,
} from '@/api'
import {
  feedbackCategoryLabel,
  feedbackStatusLabel,
  feedbackStatusVariant,
  formatApiError,
  formatDateTime,
  formatFileSize,
} from '@/utils/format'

const ticketId = ref('')
const contact = ref('')
const status = ref('all')
const category = ref('all')
const page = ref(1)
const pageSize = 20
const items = ref<OpsFeedback[]>([])
const total = ref(0)
const error = ref('')
const loading = ref(false)
const editing = ref<OpsFeedback | null>(null)
const note = ref('')
const reply = ref('')
const nextStatus = ref('triaged')
const saving = ref(false)
const attUrls = ref<Record<string, string>>({})
const lightbox = ref<string | null>(null)

const editingMeta = computed(() => {
  const f = editing.value
  if (!f) return []
  return [
    { label: '工单号', value: f.id },
    { label: '类型', value: feedbackCategoryLabel(f.category) },
    { label: '联系', value: f.contact || '—' },
    { label: '版本', value: f.clientVersion || '未上报' },
    { label: '产品', value: f.product || 'niuma' },
    { label: '提交时间', value: formatDateTime(f.createdAt) },
    { label: '回复时间', value: f.staffReplyAt ? formatDateTime(f.staffReplyAt) : '—' },
  ]
})

function revokeAttUrls() {
  for (const url of Object.values(attUrls.value)) URL.revokeObjectURL(url)
  attUrls.value = {}
}

async function loadAttachments(f: OpsFeedback) {
  revokeAttUrls()
  const next: Record<string, string> = {}
  for (const att of f.attachments || []) {
    try {
      next[att.id] = await fetchOpsFeedbackAttachmentURL(f.id, att.id)
    } catch {
      /* ignore */
    }
  }
  attUrls.value = next
}

watch(editing, (f) => {
  if (f) void loadAttachments(f)
  else {
    revokeAttUrls()
    lightbox.value = null
  }
})

onUnmounted(revokeAttUrls)

const statusFilterOptions = [
  { label: '全部状态', value: 'all' },
  { label: '待处理', value: 'open' },
  { label: '处理中', value: 'triaged' },
  { label: '已关闭', value: 'closed' },
]

const categoryFilterOptions = [
  { label: '全部分类', value: 'all' },
  { label: '缺陷', value: 'bug' },
  { label: '建议', value: 'feature' },
  { label: '其他', value: 'other' },
]

const statusEditOptions = [
  { label: '待处理', value: 'open' },
  { label: '处理中', value: 'triaged' },
  { label: '已关闭', value: 'closed' },
]

const columns: RsTableColumn<OpsFeedback>[] = [
  { key: 'id', title: '工单号', width: 168, ellipsis: true },
  { key: 'title', title: '标题', minWidth: 220, ellipsis: true },
  { key: 'attachments', title: '截图', width: 72 },
  { key: 'staffReply', title: '回复', minWidth: 180 },
  { key: 'category', title: '分类', width: 100 },
  { key: 'status', title: '状态', width: 100 },
  { key: 'contact', title: '联系', width: 200, formatter: (v) => String(v || '—'), ellipsis: true },
  { key: 'clientVersion', title: '版本', width: 120 },
  { key: 'createdAt', title: '提交时间', width: 176 },
  { key: 'actions', title: '操作', width: 88 },
]

async function load() {
  error.value = ''
  loading.value = true
  try {
    const res = await fetchOpsFeedback({
      id: ticketId.value.trim() || undefined,
      contact: contact.value.trim() || undefined,
      category: category.value === 'all' ? undefined : category.value,
      status: status.value === 'all' ? undefined : status.value,
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

function openEdit(f: OpsFeedback) {
  editing.value = f
  note.value = f.adminNote || ''
  reply.value = f.staffReply || ''
  nextStatus.value = f.status === 'open' ? 'triaged' : f.status
  error.value = ''
}

async function save() {
  if (!editing.value) return
  saving.value = true
  error.value = ''
  try {
    await updateOpsFeedback(editing.value.id, {
      status: nextStatus.value,
      adminNote: note.value,
      staffReply: reply.value,
    })
    editing.value = null
    await load()
  } catch (e) {
    const msg = formatApiError(e, '保存失败')
    if (msg) error.value = msg
  } finally {
    saving.value = false
  }
}

function shortTicketId(id: string): string {
  // fb_ + 32 hex → 展示前缀与尾部，便于扫读且列宽可控
  if (id.length <= 14) return id
  return `${id.slice(0, 10)}…${id.slice(-4)}`
}

function copyId(id: string) {
  if (!id) return
  void navigator.clipboard?.writeText(id)
}

function copyTicketId() {
  const id = editing.value?.id
  if (!id) return
  copyId(id)
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
      <RsInput v-model="ticketId" placeholder="工单号（精确）" style="min-width: 200px" @keydown.enter="search" />
      <RsInput v-model="contact" placeholder="联系邮箱（精确）" style="min-width: 180px" @keydown.enter="search" />
      <RsSelect
        v-model="status"
        class="ops-select-ctl"
        :options="statusFilterOptions"
        placeholder="全部状态"
      />
      <RsSelect
        v-model="category"
        class="ops-select-ctl"
        :options="categoryFilterOptions"
        placeholder="全部分类"
      />
      <RsButton variant="secondary" :loading="loading" @click="search">筛选</RsButton>
    </div>

    <div v-if="error && !editing" class="ops-alert">{{ error }}</div>

    <div class="ops-card ops-card--flush">
      <RsTable :columns="columns" :data="items" row-key="id" :loading="loading" :bordered="false">
        <template #id="{ row }">
          <button
            type="button"
            class="ops-fb-id"
            :title="`点击复制 ${row.id}`"
            @click="copyId(row.id)"
          >
            {{ shortTicketId(row.id) }}
          </button>
        </template>
        <template #title="{ row }">
          <span class="ops-table__cell-title" :title="row.title">{{ row.title }}</span>
        </template>
        <template #attachments="{ row }">
          <span class="ops-muted">{{ row.attachments?.length || '—' }}</span>
        </template>
        <template #staffReply="{ row }">
          <div v-if="row.staffReply" class="ops-fb-reply-cell">
            <span class="ops-fb-reply" :title="row.staffReply">{{ row.staffReply }}</span>
            <span v-if="row.staffReplyAt" class="ops-muted ops-fb-reply-at">
              {{ formatDateTime(row.staffReplyAt) }}
            </span>
          </div>
          <span v-else class="ops-muted">—</span>
        </template>
        <template #category="{ row }">
          <RsBadge variant="default">{{ feedbackCategoryLabel(row.category) }}</RsBadge>
        </template>
        <template #status="{ row }">
          <RsBadge :variant="feedbackStatusVariant(row.status)">
            {{ feedbackStatusLabel(row.status) }}
          </RsBadge>
        </template>
        <template #clientVersion="{ row }">
          <span class="ops-mono" :class="{ 'ops-muted': !row.clientVersion }">
            {{ row.clientVersion || '—' }}
          </span>
        </template>
        <template #createdAt="{ row }">
          <span class="ops-muted" style="white-space: nowrap">{{ formatDateTime(row.createdAt) }}</span>
        </template>
        <template #actions="{ row }">
          <button type="button" class="ops-link-btn" @click="openEdit(row)">处理</button>
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
          </div>
        </template>
      </RsTable>
      <div class="ops-card__footer">
        <OpsPager :total="total" :page="page" :page-size="pageSize" @change="(p) => { page = p; void load() }" />
      </div>
    </div>

    <div v-if="editing" class="ops-overlay" @click.self="editing = null">
      <div class="ops-modal ops-fb-modal" role="dialog" aria-modal="true">
        <div class="ops-modal__head ops-fb-modal__head">
          <div class="ops-fb-modal__heading">
            <div class="ops-fb-modal__eyebrow">
              <RsBadge :variant="feedbackStatusVariant(editing.status)">
                {{ feedbackStatusLabel(editing.status) }}
              </RsBadge>
              <RsBadge variant="default">{{ feedbackCategoryLabel(editing.category) }}</RsBadge>
              <button type="button" class="ops-fb-ticket" title="复制工单号" @click="copyTicketId">
                {{ editing.id }}
              </button>
            </div>
            <h2 class="ops-modal__title">{{ editing.title }}</h2>
          </div>
          <button type="button" class="ops-modal__close" aria-label="关闭" @click="editing = null">
            ×
          </button>
        </div>

        <div class="ops-modal__body ops-fb-modal__body">
          <section class="ops-fb-ticket-card">
            <dl class="ops-fb-meta">
              <div v-for="item in editingMeta" :key="item.label" class="ops-fb-meta__item">
                <dt>{{ item.label }}</dt>
                <dd :class="{ 'ops-mono': item.label === '工单号' || item.label === '版本' }">
                  {{ item.value }}
                </dd>
              </div>
            </dl>

            <div class="ops-fb-body">
              <div class="ops-fb-body__label">问题描述</div>
              <p class="ops-fb-body__text">{{ editing.body }}</p>
            </div>

            <div v-if="editing.attachments?.length" class="ops-fb-shots">
              <div class="ops-fb-body__label">
                截图
                <span class="ops-muted">（{{ editing.attachments.length }}）</span>
              </div>
              <ul class="ops-fb-gallery">
                <li v-for="att in editing.attachments" :key="att.id" class="ops-fb-shot">
                  <button
                    type="button"
                    class="ops-fb-shot__preview"
                    :title="att.name"
                    @click="lightbox = attUrls[att.id] || null"
                  >
                    <img
                      v-if="attUrls[att.id]"
                      :src="attUrls[att.id]"
                      :alt="att.name"
                    />
                    <span v-else class="ops-fb-shot__ph">加载中…</span>
                  </button>
                  <div class="ops-fb-shot__meta">
                    <span class="ops-fb-shot__name" :title="att.name">{{ att.name }}</span>
                    <span class="ops-muted">{{ formatFileSize(att.size) }}</span>
                  </div>
                </li>
              </ul>
            </div>
          </section>

          <section class="ops-fb-form">
            <div class="ops-fb-form__title">处理</div>
            <p v-if="error" class="ops-error">{{ error }}</p>
            <label class="ops-field">
              <span>状态</span>
              <RsSelect v-model="nextStatus" :options="statusEditOptions" placeholder="选择状态" />
            </label>
            <label class="ops-field">
              <span>对用户可见回复</span>
              <textarea
                v-model="reply"
                class="ops-textarea"
                rows="4"
                placeholder="用户可在桌面「我的反馈」查看；匿名工单将邮件通知"
              />
            </label>
            <label class="ops-field">
              <span>内部备注</span>
              <textarea v-model="note" class="ops-textarea" rows="3" placeholder="仅运营可见，不对用户展示" />
            </label>
          </section>
        </div>

        <div class="ops-modal__foot">
          <RsButton variant="secondary" @click="editing = null">取消</RsButton>
          <RsButton variant="primary" :loading="saving" @click="save">保存处理结果</RsButton>
        </div>
      </div>
    </div>

    <div v-if="lightbox" class="ops-fb-lightbox" @click="lightbox = null">
      <img :src="lightbox" alt="" @click.stop />
    </div>
  </div>
</template>

<style scoped>
.ops-fb-reply-cell {
  display: grid;
  gap: 2px;
  min-width: 0;
}

.ops-fb-reply {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #389e0d;
  font-size: 13px;
}

.ops-fb-reply-at {
  font-size: 12px;
  white-space: nowrap;
}

.ops-fb-id {
  display: inline-block;
  max-width: 100%;
  border: none;
  background: transparent;
  padding: 0;
  margin: 0;
  color: rgba(0, 0, 0, 0.65);
  font: inherit;
  font-size: 12px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  cursor: pointer;
  text-align: left;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-fb-id:hover {
  color: var(--ops-accent);
}

.ops-fb-modal {
  width: min(760px, 100%);
  max-height: min(88vh, 900px);
  display: flex;
  flex-direction: column;
}

.ops-fb-modal__head {
  align-items: flex-start;
}

.ops-fb-modal__heading {
  display: grid;
  gap: 8px;
  min-width: 0;
  padding-right: 8px;
}

.ops-fb-modal__eyebrow {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.ops-fb-ticket {
  border: none;
  background: #f5f5f5;
  color: rgba(0, 0, 0, 0.55);
  border-radius: 999px;
  padding: 2px 10px;
  font: inherit;
  font-size: 12px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  cursor: pointer;
}

.ops-fb-ticket:hover {
  background: #ebebeb;
  color: rgba(0, 0, 0, 0.88);
}

.ops-fb-modal__body {
  gap: 16px;
  overflow: auto;
  min-height: 0;
  background: #f7f8fa;
}

.ops-fb-ticket-card,
.ops-fb-form {
  background: #fff;
  border: 1px solid var(--ops-border);
  border-radius: 10px;
  padding: 16px;
  display: grid;
  gap: 14px;
}

.ops-fb-meta {
  margin: 0;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px 12px;
}

.ops-fb-meta__item {
  display: grid;
  gap: 2px;
  min-width: 0;
}

.ops-fb-meta__item dt {
  margin: 0;
  font-size: 12px;
  color: var(--ops-muted);
}

.ops-fb-meta__item dd {
  margin: 0;
  font-size: 13px;
  color: rgba(0, 0, 0, 0.88);
  overflow-wrap: anywhere;
}

.ops-fb-body__label {
  font-size: 12px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.65);
  margin-bottom: 6px;
}

.ops-fb-body__text {
  margin: 0;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  line-height: 1.65;
  font-size: 14px;
  color: rgba(0, 0, 0, 0.88);
  background: #fafafa;
  border: 1px solid var(--ops-border);
  border-radius: 8px;
  padding: 12px 14px;
}

.ops-fb-gallery {
  list-style: none;
  margin: 0;
  padding: 0;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 12px;
}

.ops-fb-shot {
  display: grid;
  gap: 6px;
  min-width: 0;
  padding: 8px;
  border: 1px solid var(--ops-border);
  border-radius: 10px;
  background: #fafafa;
}

.ops-fb-shot__preview {
  display: grid;
  place-items: center;
  width: 100%;
  aspect-ratio: 16 / 10;
  margin: 0;
  padding: 6px;
  border: none;
  border-radius: 8px;
  background: #fff;
  cursor: zoom-in;
  overflow: hidden;
}

.ops-fb-shot__preview img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
  display: block;
}

.ops-fb-shot__ph {
  font-size: 12px;
  color: var(--ops-muted);
}

.ops-fb-shot__meta {
  display: grid;
  gap: 2px;
  min-width: 0;
}

.ops-fb-shot__name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  font-weight: 500;
}

.ops-fb-form__title {
  font-size: 13px;
  font-weight: 650;
  color: rgba(0, 0, 0, 0.88);
}

.ops-fb-lightbox {
  position: fixed;
  inset: 0;
  z-index: 1000;
  background: rgb(0 0 0 / 0.72);
  display: grid;
  place-items: center;
  padding: 1.5rem;
  cursor: zoom-out;
}

.ops-fb-lightbox img {
  max-width: min(92vw, 56rem);
  max-height: 88vh;
  border-radius: 6px;
  box-shadow: 0 18px 48px rgb(0 0 0 / 0.35);
}

@media (max-width: 720px) {
  .ops-fb-meta {
    grid-template-columns: 1fr 1fr;
  }
}
</style>
