<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  RsButton,
  RsInput,
  RsBadge,
  RsSelect,
  RsTable,
  RsDialog,
  RsConfirmDialog,
  RsCheckbox,
  RsLabel,
  RsMarkdown,
  type RsTableColumn,
} from '@/components/ui'
import OpsPager from '@/components/OpsPager.vue'
import {
  createOpsRelease,
  fetchOpsReleases,
  patchOpsRelease,
  publishOpsRelease,
  yankOpsRelease,
  type OpsRelease,
} from '@/api'
import { formatApiError, formatDateTime } from '@/utils/format'

const version = ref('')
const statusFilter = ref('all')
const platformFilter = ref('all')
const channelFilter = ref('all')
const archFilter = ref('all')
const page = ref(1)
const pageSize = 20
const items = ref<OpsRelease[]>([])
const total = ref(0)
const error = ref('')
const loading = ref(false)

const showEditor = ref(false)
const editing = ref<OpsRelease | null>(null)
const saving = ref(false)
const form = ref({
  version: '',
  platform: 'windows',
  arch: 'x64',
  channel: 'stable',
  title: '',
  notesMd: '',
  downloadUrl: '',
  sha256: '',
  fileSize: '',
  minSupportedVersion: '',
  forceUpdate: false,
})

const confirmOpen = ref(false)
const pendingConfirm = ref<{ kind: 'publish' | 'yank'; row: OpsRelease } | null>(null)

const editorTitle = computed(() =>
  editing.value ? `编辑 ${editing.value.version}` : '新建发布',
)

const confirmTitle = computed(() =>
  pendingConfirm.value?.kind === 'yank' ? '确认撤回' : '确认发布',
)

const confirmDescription = computed(() => {
  const pending = pendingConfirm.value
  if (!pending) return ''
  const { row, kind } = pending
  if (kind === 'yank') {
    return `确认撤回 ${row.version}？最新版本将回退到上一 published。`
  }
  return `确认发布 ${row.version}（${row.platform}/${row.arch}）？桌面端与官网将立即可见。`
})

const confirmTone = computed(() =>
  pendingConfirm.value?.kind === 'yank' ? 'danger' : 'info',
)

const confirmVariant = computed(() =>
  pendingConfirm.value?.kind === 'yank' ? 'danger' : 'primary',
)

const confirmText = computed(() =>
  pendingConfirm.value?.kind === 'yank' ? '撤回' : '发布',
)

const statusOptions = [
  { label: '全部状态', value: 'all' },
  { label: '草稿', value: 'draft' },
  { label: '已发布', value: 'published' },
  { label: '已撤回', value: 'yanked' },
]

const platformOptions = [
  { label: '全部平台', value: 'all' },
  { label: 'Windows', value: 'windows' },
  { label: 'Linux', value: 'linux' },
  { label: 'macOS', value: 'macos' },
]

const channelOptions = [
  { label: '全部渠道', value: 'all' },
  { label: 'stable', value: 'stable' },
  { label: 'beta', value: 'beta' },
]

const archOptions = [
  { label: '全部架构', value: 'all' },
  { label: 'x64', value: 'x64' },
  { label: 'arm64', value: 'arm64' },
]

const formPlatformOptions = [
  { label: 'Windows', value: 'windows' },
  { label: 'Linux', value: 'linux' },
  { label: 'macOS', value: 'macos' },
]

const formArchOptions = [
  { label: 'x64', value: 'x64' },
  { label: 'arm64', value: 'arm64' },
]

const columns: RsTableColumn<OpsRelease>[] = [
  { key: 'version', title: '版本', width: 110 },
  { key: 'platform', title: '平台', width: 140 },
  { key: 'title', title: '标题', minWidth: 180 },
  { key: 'status', title: '状态', width: 100 },
  { key: 'publishedAt', title: '发布时间', width: 170 },
  { key: 'actions', title: '操作', width: 220 },
]

function statusLabel(s: string): string {
  switch (s) {
    case 'draft':
      return '草稿'
    case 'published':
      return '已发布'
    case 'yanked':
      return '已撤回'
    default:
      return s
  }
}

function statusVariant(s: string): 'default' | 'success' | 'danger' | 'info' {
  switch (s) {
    case 'published':
      return 'success'
    case 'yanked':
      return 'danger'
    case 'draft':
      return 'info'
    default:
      return 'default'
  }
}

async function load() {
  error.value = ''
  loading.value = true
  try {
    const res = await fetchOpsReleases({
      version: version.value.trim() || undefined,
      status: statusFilter.value === 'all' ? undefined : statusFilter.value,
      platform: platformFilter.value === 'all' ? undefined : platformFilter.value,
      channel: channelFilter.value === 'all' ? undefined : channelFilter.value,
      arch: archFilter.value === 'all' ? undefined : archFilter.value,
      limit: pageSize,
      offset: (page.value - 1) * pageSize,
    })
    items.value = res.items
    total.value = res.total
  } catch (e) {
    const msg = formatApiError(e)
    if (msg) error.value = msg
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = null
  form.value = {
    version: '',
    platform: 'windows',
    arch: 'x64',
    channel: 'stable',
    title: '',
    notesMd: '',
    downloadUrl: '',
    sha256: '',
    fileSize: '',
    minSupportedVersion: '',
    forceUpdate: false,
  }
  error.value = ''
  showEditor.value = true
}

function openEdit(row: OpsRelease) {
  editing.value = row
  form.value = {
    version: row.version,
    platform: row.platform,
    arch: row.arch,
    channel: row.channel,
    title: row.title,
    notesMd: row.notesMd || '',
    downloadUrl: row.downloadUrl,
    sha256: row.sha256,
    fileSize: row.fileSize ? String(row.fileSize) : '',
    minSupportedVersion: row.minSupportedVersion || '',
    forceUpdate: !!row.forceUpdate,
  }
  error.value = ''
  showEditor.value = true
}

async function save() {
  saving.value = true
  error.value = ''
  const size = form.value.fileSize.trim() ? Number(form.value.fileSize) : 0
  if (Number.isNaN(size) || size < 0) {
    error.value = '文件大小无效'
    saving.value = false
    return
  }
  try {
    if (editing.value) {
      await patchOpsRelease(editing.value.id, {
        title: form.value.title.trim(),
        notesMd: form.value.notesMd,
        downloadUrl: form.value.downloadUrl.trim(),
        sha256: form.value.sha256.trim().toLowerCase(),
        fileSize: size,
        minSupportedVersion: form.value.minSupportedVersion.trim() || undefined,
        forceUpdate: form.value.forceUpdate,
      })
    } else {
      await createOpsRelease({
        version: form.value.version.trim(),
        platform: form.value.platform,
        arch: form.value.arch,
        channel: form.value.channel,
        title: form.value.title.trim(),
        notesMd: form.value.notesMd,
        downloadUrl: form.value.downloadUrl.trim(),
        sha256: form.value.sha256.trim().toLowerCase(),
        fileSize: size,
        minSupportedVersion: form.value.minSupportedVersion.trim() || undefined,
        forceUpdate: form.value.forceUpdate,
      })
    }
    showEditor.value = false
    await load()
  } catch (e) {
    const msg = formatApiError(e)
    if (msg) error.value = msg
  } finally {
    saving.value = false
  }
}

function askPublish(row: OpsRelease) {
  pendingConfirm.value = { kind: 'publish', row }
  confirmOpen.value = true
}

function askYank(row: OpsRelease) {
  pendingConfirm.value = { kind: 'yank', row }
  confirmOpen.value = true
}

async function onConfirmAction() {
  const pending = pendingConfirm.value
  if (!pending) return
  error.value = ''
  try {
    if (pending.kind === 'publish') {
      await publishOpsRelease(pending.row.id)
    } else {
      await yankOpsRelease(pending.row.id)
    }
    pendingConfirm.value = null
    await load()
  } catch (e) {
    const msg = formatApiError(e)
    if (msg) error.value = msg
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
      <RsInput v-model="version" placeholder="版本（精确，如 1.0.1）" style="min-width: 160px" @keydown.enter="search" />
      <RsSelect v-model="statusFilter" :options="statusOptions" style="width: 140px" />
      <RsSelect v-model="platformFilter" :options="platformOptions" style="width: 140px" />
      <RsSelect v-model="channelFilter" :options="channelOptions" style="width: 140px" />
      <RsSelect v-model="archFilter" :options="archOptions" style="width: 130px" />
      <RsButton variant="secondary" :loading="loading" @click="search">筛选</RsButton>
      <div class="ops-toolbar__grow" />
      <div class="ops-toolbar__right">
        <RsButton variant="primary" @click="openCreate">新建发布</RsButton>
      </div>
    </div>

    <div v-if="error && !showEditor" class="ops-alert">{{ error }}</div>

    <p class="ops-muted" style="margin: 0 0 12px">
      安装包请先上传至 HTTPS CDN，再在此填写直链与 SHA-256。Cloud
      <strong>不会</strong>代理或拉取安装包。公开接口仅返回已发布记录。
    </p>

    <div class="ops-card ops-card--flush">
      <RsTable :columns="columns" :data="items" row-key="id" :loading="loading" :bordered="false">
        <template #version="{ row }">
          <span class="ops-mono">{{ row.version }}</span>
          <span v-if="row.forceUpdate" class="ops-chip" style="margin-left: 6px">强制</span>
        </template>
        <template #platform="{ row }">
          <span class="ops-mono">{{ row.platform }}/{{ row.arch }}</span>
          <div class="ops-table__cell-sub">{{ row.channel }}</div>
        </template>
        <template #title="{ row }">
          <div class="ops-table__cell-title">{{ row.title }}</div>
          <div class="ops-table__cell-sub ops-mono" :title="row.downloadUrl">
            {{ row.downloadUrl }}
          </div>
        </template>
        <template #status="{ row }">
          <RsBadge :variant="statusVariant(row.status)">{{ statusLabel(row.status) }}</RsBadge>
        </template>
        <template #publishedAt="{ row }">
          <span class="ops-muted">{{ formatDateTime(row.publishedAt) }}</span>
        </template>
        <template #actions="{ row }">
          <div class="ops-row-actions">
            <RsButton
              v-if="row.status !== 'yanked'"
              size="sm"
              variant="secondary"
              @click="openEdit(row)"
            >
              编辑
            </RsButton>
            <RsButton
              v-if="row.status === 'draft'"
              size="sm"
              variant="primary"
              @click="askPublish(row)"
            >
              发布
            </RsButton>
            <RsButton
              v-if="row.status === 'published'"
              size="sm"
              variant="danger"
              @click="askYank(row)"
            >
              撤回
            </RsButton>
          </div>
        </template>
      </RsTable>
      <div class="ops-card__footer">
        <OpsPager :total="total" :page="page" :page-size="pageSize" @change="(p) => { page = p; void load() }" />
      </div>
    </div>

    <RsDialog
      v-model:open="showEditor"
      :title="editorTitle"
      width="lg"
      show-overlay
      close-on-overlay-click
      :fullscreenable="false"
    >
      <template #body>
        <div class="ops-release-form">
          <div v-if="error" class="ops-alert">{{ error }}</div>
          <div class="ops-field">
            <RsLabel for-id="release-version" required>版本 (X.Y.Z)</RsLabel>
            <RsInput
              id="release-version"
              v-model="form.version"
              :disabled="!!editing"
              placeholder="1.0.1"
            />
          </div>
          <div class="ops-field-row">
            <div class="ops-field">
              <RsLabel>平台</RsLabel>
              <RsSelect
                v-model="form.platform"
                :disabled="!!editing"
                :options="formPlatformOptions"
              />
            </div>
            <div class="ops-field">
              <RsLabel>架构</RsLabel>
              <RsSelect v-model="form.arch" :disabled="!!editing" :options="formArchOptions" />
            </div>
          </div>
          <div class="ops-field">
            <RsLabel for-id="release-title" required>标题</RsLabel>
            <RsInput id="release-title" v-model="form.title" placeholder="1.0.1 稳定性修复" />
          </div>
          <div class="ops-field">
            <RsLabel>更新说明 (Markdown)</RsLabel>
            <RsMarkdown
              v-model="form.notesMd"
              :height="220"
              placeholder="## 新功能&#10;- …"
            />
          </div>
          <div class="ops-field">
            <RsLabel for-id="release-url" required>下载 URL (HTTPS)</RsLabel>
            <RsInput
              id="release-url"
              v-model="form.downloadUrl"
              placeholder="https://cdn.example.com/.../Setup.exe"
            />
          </div>
          <div class="ops-field">
            <RsLabel for-id="release-sha" required>SHA-256</RsLabel>
            <RsInput id="release-sha" v-model="form.sha256" placeholder="64 位小写十六进制" />
          </div>
          <div class="ops-field-row">
            <div class="ops-field">
              <RsLabel for-id="release-size">文件大小（字节，可选）</RsLabel>
              <RsInput id="release-size" v-model="form.fileSize" placeholder="123456789" />
            </div>
            <div class="ops-field">
              <RsLabel for-id="release-min">最低可用版本（可选）</RsLabel>
              <RsInput id="release-min" v-model="form.minSupportedVersion" placeholder="1.0.0" />
            </div>
          </div>
          <RsCheckbox v-model="form.forceUpdate">
            强制更新（有新版本时客户端不可跳过）
          </RsCheckbox>
        </div>
      </template>
      <template #footer>
        <RsButton variant="secondary" @click="showEditor = false">取消</RsButton>
        <RsButton variant="primary" :loading="saving" @click="save">保存</RsButton>
      </template>
    </RsDialog>

    <RsConfirmDialog
      v-model:open="confirmOpen"
      :title="confirmTitle"
      :description="confirmDescription"
      :tone="confirmTone"
      :confirm-variant="confirmVariant"
      :confirm-text="confirmText"
      show-overlay
      @confirm="onConfirmAction"
    />
  </div>
</template>

<style scoped>
.ops-release-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.ops-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.ops-field-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}
.ops-row-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
</style>
