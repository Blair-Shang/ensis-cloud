<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  RsBadge,
  RsButton,
  RsCheckbox,
  RsConfirmDialog,
  RsDialog,
  RsInput,
  RsLabel,
  RsSelect,
  RsTable,
  RsTooltip,
  RsTooltipProvider,
  type RsTableColumn,
} from '@/components/ui'
import {
  createOpsAIEndpoint,
  createOpsAIModel,
  deleteOpsAIEndpoint,
  deleteOpsAIModel,
  fetchOpsAIEndpoints,
  fetchOpsAIModels,
  patchOpsAIEndpoint,
  patchOpsAIModel,
  testOpsAIEndpoint,
  type OpsAIEndpoint,
  type OpsAIModel,
} from '@/api'
import { formatApiError, formatDateTime } from '@/utils/format'
import {
  AI_PROVIDER_PRESET_CUSTOM,
  AI_PROVIDER_PRESETS,
  findAIProviderPreset,
  matchAIProviderPreset,
} from './provider-presets'
import AiUsagePanel from './AiUsagePanel.vue'

const route = useRoute()
const router = useRouter()
const tab = computed(() => (route.query.tab === 'usage' ? 'usage' : 'endpoints'))

function setTab(next: 'endpoints' | 'usage') {
  const query = { ...route.query }
  if (next === 'usage') query.tab = 'usage'
  else delete query.tab
  void router.replace({ query })
}

const items = ref<OpsAIEndpoint[]>([])
const models = ref<OpsAIModel[]>([])
const selectedId = ref('')
const error = ref('')
const loading = ref(false)

const showEditor = ref(false)
const isCreate = ref(false)
const formVendor = ref(AI_PROVIDER_PRESET_CUSTOM)
const formName = ref('')
const formBase = ref('')
const formSecret = ref('')
const formEnabled = ref(false)
const formDefault = ref('')
const formReqLimit = ref('80')
const formTokenLimit = ref('0')
const formSort = ref('0')
const saving = ref(false)
const formLoading = ref(false)
const testing = ref(false)
const testOk = ref<boolean | null>(null)
const testMessage = ref('')
const seedingModels = ref(false)

const modelCode = ref('')
const modelLabel = ref('')
const addingModel = ref(false)

const confirmOpen = ref(false)
const pendingConfirm = ref<
  { kind: 'endpoint'; row: OpsAIEndpoint } | { kind: 'model'; row: OpsAIModel } | null
>(null)

const vendorOptions = AI_PROVIDER_PRESETS.map((p) => ({ label: p.label, value: p.id }))

const activePreset = computed(() => findAIProviderPreset(formVendor.value))

const defaultModelOptions = computed(() => {
  const ids = new Set<string>()
  for (const code of activePreset.value?.models ?? []) {
    if (code) ids.add(code)
  }
  for (const row of models.value) {
    if (row.code) ids.add(row.code)
  }
  if (formDefault.value.trim()) ids.add(formDefault.value.trim())
  return [...ids].map((code) => ({ label: code, value: code }))
})

const canTest = computed(
  () =>
    Boolean(formBase.value.trim()) &&
    Boolean(formDefault.value.trim()) &&
    Boolean(formSecret.value.trim() || (!isCreate.value && selectedId.value)),
)

const editorTitle = computed(() => (isCreate.value ? '新增接入点' : '编辑接入点'))
const editorDescription = computed(() =>
  isCreate.value
    ? '密钥密文入库，列表不回显明文。保存后可继续添加模型目录。'
    : '修改后立即对已开放的桌面登录用户生效。密钥留空则保留原值。',
)

const confirmTitle = computed(() =>
  pendingConfirm.value?.kind === 'model' ? '确认删除模型' : '确认删除接入点',
)

const confirmDescription = computed(() => {
  const pending = pendingConfirm.value
  if (!pending) return ''
  if (pending.kind === 'model') {
    return `删除模型「${pending.row.code}」后，桌面端目录将不再展示该编码。`
  }
  return `删除接入点「${pending.row.providerName}」会同时删除其模型目录，且不可恢复。`
})

const columns: RsTableColumn<OpsAIEndpoint>[] = [
  { key: 'name', title: '接入点', minWidth: 220 },
  { key: 'enabled', title: '开放', width: 88 },
  { key: 'quota', title: '日配额', minWidth: 160 },
  { key: 'defaultModel', title: '默认模型', minWidth: 140 },
  { key: 'updatedAt', title: '编辑时间', width: 180 },
  { key: 'actions', title: '操作', width: 140 },
]

const modelColumns: RsTableColumn<OpsAIModel>[] = [
  { key: 'code', title: '模型编码', minWidth: 160 },
  { key: 'label', title: '展示名', minWidth: 140 },
  { key: 'status', title: '状态', width: 88 },
  { key: 'actions', title: '操作', width: 120 },
]

function mapErr(e: unknown): string {
  return formatApiError(e)
}

function quotaText(row: OpsAIEndpoint): string {
  const req = row.dailyRequestLimit > 0 ? `${row.dailyRequestLimit} 次` : '请求不限'
  const tok = row.dailyTokenLimit > 0 ? `${row.dailyTokenLimit} token` : 'token 不限'
  return `${req} · ${tok}`
}

async function load() {
  error.value = ''
  loading.value = true
  try {
    const res = await fetchOpsAIEndpoints()
    items.value = res.items ?? []
    if (selectedId.value && !items.value.some((x) => x.id === selectedId.value)) {
      selectedId.value = ''
      models.value = []
    }
  } catch (e) {
    const msg = mapErr(e)
    if (msg) error.value = msg
  } finally {
    loading.value = false
  }
}

async function loadModels(id: string) {
  const res = await fetchOpsAIModels(id)
  models.value = res.items ?? []
}

function clearTest() {
  testOk.value = null
  testMessage.value = ''
}

function resetForm() {
  formVendor.value = AI_PROVIDER_PRESET_CUSTOM
  formName.value = 'NiuMa'
  formBase.value = ''
  formSecret.value = ''
  formEnabled.value = false
  formDefault.value = ''
  formReqLimit.value = '80'
  formTokenLimit.value = '0'
  formSort.value = '0'
  modelCode.value = ''
  modelLabel.value = ''
  models.value = []
  clearTest()
}

function applyPreset(id: string, fillName: boolean) {
  const preset = findAIProviderPreset(id)
  if (!preset) return
  formVendor.value = preset.id
  formBase.value = preset.baseUrl
  if (preset.defaultModel) formDefault.value = preset.defaultModel
  if (fillName && preset.defaultName) formName.value = preset.defaultName
  clearTest()
}

function onVendorChange(value: string) {
  const id = value || AI_PROVIDER_PRESET_CUSTOM
  if (id === AI_PROVIDER_PRESET_CUSTOM) {
    formVendor.value = AI_PROVIDER_PRESET_CUSTOM
    clearTest()
    return
  }
  const fillName =
    isCreate.value || !formName.value.trim() || formName.value.trim() === 'NiuMa'
  applyPreset(id, fillName)
}

function openCreate() {
  isCreate.value = true
  selectedId.value = ''
  resetForm()
  error.value = ''
  showEditor.value = true
}

async function openEdit(row: OpsAIEndpoint) {
  isCreate.value = false
  selectedId.value = row.id
  formName.value = row.providerName
  formBase.value = row.baseUrl
  formSecret.value = ''
  formEnabled.value = row.enabled
  formDefault.value = row.defaultModel
  formReqLimit.value = String(row.dailyRequestLimit ?? 0)
  formTokenLimit.value = String(row.dailyTokenLimit ?? 0)
  formSort.value = String(row.sortSeq ?? 0)
  formVendor.value = matchAIProviderPreset({
    baseUrl: row.baseUrl,
    providerName: row.providerName,
  })
  modelCode.value = ''
  modelLabel.value = ''
  error.value = ''
  clearTest()
  showEditor.value = true
  formLoading.value = true
  try {
    await loadModels(row.id)
  } catch (e) {
    const msg = mapErr(e)
    if (msg) error.value = msg
  } finally {
    formLoading.value = false
  }
}

function closeEditor() {
  showEditor.value = false
}

async function save() {
  saving.value = true
  error.value = ''
  try {
    const body = {
      providerName: formName.value.trim(),
      baseUrl: formBase.value.trim(),
      apiSecret: formSecret.value.trim() || undefined,
      enabled: formEnabled.value,
      defaultModel: formDefault.value.trim(),
      dailyRequestLimit: Number(formReqLimit.value) || 0,
      dailyTokenLimit: Number(formTokenLimit.value) || 0,
      sortSeq: Number(formSort.value) || 0,
    }
    const createdNow = isCreate.value
    if (isCreate.value) {
      const created = await createOpsAIEndpoint(body)
      selectedId.value = created.id
      isCreate.value = false
    } else if (selectedId.value) {
      await patchOpsAIEndpoint(selectedId.value, body)
    }
    await load()
    if (selectedId.value) await loadModels(selectedId.value)
    if (createdNow && selectedId.value) await seedRecommendedModels(selectedId.value)
  } catch (e) {
    const msg = mapErr(e)
    if (msg) error.value = msg
  } finally {
    saving.value = false
  }
}

function askRemoveEndpoint(row: OpsAIEndpoint) {
  pendingConfirm.value = { kind: 'endpoint', row }
  confirmOpen.value = true
}

function askRemoveModel(row: OpsAIModel) {
  pendingConfirm.value = { kind: 'model', row }
  confirmOpen.value = true
}

async function onConfirmDelete() {
  const pending = pendingConfirm.value
  if (!pending) return
  error.value = ''
  try {
    if (pending.kind === 'endpoint') {
      await deleteOpsAIEndpoint(pending.row.id)
      if (selectedId.value === pending.row.id) {
        selectedId.value = ''
        models.value = []
        showEditor.value = false
      }
      await load()
    } else {
      await deleteOpsAIModel(pending.row.id)
      if (selectedId.value) await loadModels(selectedId.value)
    }
  } catch (e) {
    const msg = mapErr(e)
    if (msg) error.value = msg
  } finally {
    pendingConfirm.value = null
  }
}

async function testConnection() {
  if (!canTest.value) return
  testing.value = true
  error.value = ''
  clearTest()
  try {
    const result = await testOpsAIEndpoint({
      endpointId: !isCreate.value && selectedId.value ? selectedId.value : undefined,
      baseUrl: formBase.value.trim(),
      apiSecret: formSecret.value.trim() || undefined,
      model: formDefault.value.trim(),
    })
    testOk.value = result.ok
    testMessage.value = result.ok
      ? `连通正常，${result.latencyMs} ms（${result.model}）`
      : result.message || `上游返回 ${result.status || '错误'}`
  } catch (e) {
    const msg = mapErr(e)
    if (msg) error.value = msg
  } finally {
    testing.value = false
  }
}

async function seedRecommendedModels(endpointId: string) {
  const codes = activePreset.value?.models ?? []
  if (!codes.length) return
  seedingModels.value = true
  try {
    const existing = new Set(models.value.map((m) => m.code))
    for (const code of codes) {
      if (!code || existing.has(code)) continue
      await createOpsAIModel(endpointId, { code, label: code })
      existing.add(code)
    }
    await loadModels(endpointId)
  } catch (e) {
    const msg = mapErr(e)
    if (msg) error.value = msg
  } finally {
    seedingModels.value = false
  }
}

function pickRecommended(code: string) {
  formDefault.value = code
  modelCode.value = code
  modelLabel.value = code
}

async function addModel() {
  if (!selectedId.value || !modelCode.value.trim()) return
  addingModel.value = true
  error.value = ''
  try {
    await createOpsAIModel(selectedId.value, {
      code: modelCode.value.trim(),
      label: modelLabel.value.trim() || modelCode.value.trim(),
    })
    modelCode.value = ''
    modelLabel.value = ''
    await loadModels(selectedId.value)
  } catch (e) {
    const msg = mapErr(e)
    if (msg) error.value = msg
  } finally {
    addingModel.value = false
  }
}

async function toggleModel(row: OpsAIModel) {
  error.value = ''
  try {
    await patchOpsAIModel(row.id, {
      recordStatus: row.recordStatus === 'active' ? 'disabled' : 'active',
    })
    if (selectedId.value) await loadModels(selectedId.value)
  } catch (e) {
    const msg = mapErr(e)
    if (msg) error.value = msg
  }
}

onMounted(() => {
  void load()
})
</script>

<template>
  <RsTooltipProvider>
  <div>
    <div class="ai-tabs" role="tablist">
      <button
        type="button"
        role="tab"
        class="ai-tabs__btn"
        :class="{ 'ai-tabs__btn--on': tab === 'endpoints' }"
        :aria-selected="tab === 'endpoints'"
        @click="setTab('endpoints')"
      >
        接入点
      </button>
      <button
        type="button"
        role="tab"
        class="ai-tabs__btn"
        :class="{ 'ai-tabs__btn--on': tab === 'usage' }"
        :aria-selected="tab === 'usage'"
        @click="setTab('usage')"
      >
        用量
      </button>
    </div>

    <AiUsagePanel v-if="tab === 'usage'" />

    <div v-else>
    <div class="ops-toolbar">
      <p class="ops-muted">
        接入点与模型目录存在数据库。密钥密文入库，列表不回显明文。开放后桌面登录即可使用。
      </p>
      <div class="ops-toolbar__grow" />
      <RsButton variant="primary" @click="openCreate">新增接入点</RsButton>
    </div>

    <div v-if="error && !showEditor" class="ops-alert">{{ error }}</div>

    <div class="ops-card ops-card--flush">
      <RsTable :columns="columns" :data="items" row-key="id" :loading="loading" :bordered="false">
        <template #name="{ row }">
          <div>
            <p class="ops-person__name">{{ row.providerName }}</p>
            <p class="ops-person__sub">{{ row.baseUrl }}</p>
          </div>
        </template>
        <template #enabled="{ row }">
          <RsBadge :variant="row.enabled ? 'success' : 'default'">
            {{ row.enabled ? '已开放' : '已关闭' }}
          </RsBadge>
        </template>
        <template #quota="{ row }">
          <span>{{ quotaText(row) }}</span>
        </template>
        <template #defaultModel="{ row }">
          <span>{{ row.defaultModel || '—' }}</span>
        </template>
        <template #updatedAt="{ row }">
          <span class="ops-muted" style="white-space: nowrap">{{ formatDateTime(row.updatedAt) }}</span>
        </template>
        <template #actions="{ row }">
          <div class="ops-row">
            <button type="button" class="ops-link-btn" @click="openEdit(row)">编辑</button>
            <span class="ops-link-sep">|</span>
            <button type="button" class="ops-link-btn ops-link-btn--danger" @click="askRemoveEndpoint(row)">
              删除
            </button>
          </div>
        </template>
        <template #empty>
          <div class="ops-empty">
            <p class="ops-empty__title">尚未配置系统模型接入点</p>
            <p class="ops-empty__desc">点击右上角「新增接入点」填写上游地址与密钥</p>
          </div>
        </template>
      </RsTable>
    </div>

    <RsDialog
      v-model:open="showEditor"
      :title="editorTitle"
      :description="editorDescription"
      layout="form"
      width="lg"
      show-overlay
      close-on-overlay-click
      :fullscreenable="false"
    >
      <template #body>
        <div class="ai-form">
          <div v-if="error" class="ops-alert">{{ error }}</div>
          <p v-if="formLoading" class="ops-muted" style="margin: 0">正在读取模型目录…</p>

          <section class="ai-form__section">
            <h4 class="ai-form__section-title">接入配置</h4>
            <div class="ai-form__grid">
              <div class="ops-field">
                <RsLabel for-id="ai-vendor">厂商</RsLabel>
                <RsSelect
                  id="ai-vendor"
                  v-model="formVendor"
                  :options="vendorOptions"
                  searchable
                  placeholder="选择厂商以填充地址与模型"
                  @update:model-value="onVendorChange"
                />
              </div>
              <div class="ops-field">
                <RsLabel for-id="ai-name" required>展示名</RsLabel>
                <RsInput id="ai-name" v-model="formName" placeholder="如 NiuMa / DeepSeek" />
              </div>
            </div>
            <div class="ops-field">
              <RsLabel for-id="ai-base" required>
                <RsTooltip
                  v-if="activePreset?.hint"
                  icon
                  side="top"
                  align="start"
                  :content="activePreset.hint"
                >
                  上游 Base URL
                </RsTooltip>
                <template v-else>上游 Base URL</template>
              </RsLabel>
              <RsInput
                id="ai-base"
                v-model="formBase"
                placeholder="https://api.deepseek.com/v1"
                @update:model-value="clearTest"
              />
            </div>
            <div class="ops-field">
              <RsLabel for-id="ai-secret">
                <RsTooltip
                  icon
                  side="top"
                  align="start"
                  content="密钥密文入库，编辑不再回显。留空表示保留原值。"
                >
                  上游 API 密钥
                </RsTooltip>
              </RsLabel>
              <RsInput
                id="ai-secret"
                v-model="formSecret"
                type="password"
                :placeholder="isCreate ? 'sk-…' : '留空保留原值'"
                @update:model-value="clearTest"
              />
            </div>
            <div class="ai-form__grid">
              <div class="ops-field">
                <RsLabel for-id="ai-default">默认模型编码</RsLabel>
                <RsSelect
                  id="ai-default"
                  v-model="formDefault"
                  :options="defaultModelOptions"
                  searchable
                  creatable
                  placeholder="选择或输入模型编码"
                  @update:model-value="clearTest"
                />
              </div>
              <div class="ops-field">
                <RsLabel for-id="ai-sort">
                  <RsTooltip
                    icon
                    side="top"
                    align="start"
                    content="数值越小越优先。桌面端只使用已开放且排序最前的一条。"
                  >
                    排序
                  </RsTooltip>
                </RsLabel>
                <RsInput id="ai-sort" v-model="formSort" placeholder="0" />
              </div>
            </div>
            <div v-if="activePreset?.models.length" class="ai-form__rec">
              <span class="ai-form__rec-label">推荐模型</span>
              <button
                v-for="code in activePreset.models"
                :key="code"
                type="button"
                class="ai-form__chip"
                :class="{ 'ai-form__chip--on': formDefault === code }"
                @click="pickRecommended(code)"
              >
                {{ code }}
              </button>
              <RsButton
                v-if="!isCreate && selectedId"
                size="sm"
                variant="secondary"
                :loading="seedingModels"
                @click="seedRecommendedModels(selectedId)"
              >
                写入目录
              </RsButton>
            </div>
            <RsCheckbox v-model="formEnabled">对终端用户开放</RsCheckbox>
          </section>

          <section class="ai-form__section">
            <h4 class="ai-form__section-title">日配额（按登录用户）</h4>
            <div class="ai-form__grid">
              <div class="ops-field">
                <RsLabel for-id="ai-req">
                  <RsTooltip
                    icon
                    side="top"
                    align="start"
                    content="每个用户每个 UTC 日最多发起的对话次数。0 表示不限制。默认 80。"
                  >
                    日请求上限
                  </RsTooltip>
                </RsLabel>
                <RsInput id="ai-req" v-model="formReqLimit" placeholder="80" />
              </div>
              <div class="ops-field">
                <RsLabel for-id="ai-token">
                  <RsTooltip
                    icon
                    side="top"
                    align="start"
                    content="0 表示不限制。当前代理只计请求次数，token 尚未从上游回写。"
                  >
                    日 token 上限
                  </RsTooltip>
                </RsLabel>
                <RsInput id="ai-token" v-model="formTokenLimit" placeholder="0" />
              </div>
            </div>
          </section>

          <section v-if="!isCreate && selectedId" class="ai-form__section">
            <h4 class="ai-form__section-title">模型目录</h4>
            <div class="ai-form__model-add">
              <RsInput v-model="modelCode" placeholder="模型编码" />
              <RsInput v-model="modelLabel" placeholder="展示名（可空）" />
              <RsButton variant="secondary" :loading="addingModel" @click="addModel">添加</RsButton>
            </div>
            <RsTable :columns="modelColumns" :data="models" row-key="id" :bordered="false">
              <template #code="{ row }">{{ row.code }}</template>
              <template #label="{ row }">{{ row.label }}</template>
              <template #status="{ row }">
                <RsBadge :variant="row.recordStatus === 'active' ? 'success' : 'default'">
                  {{ row.recordStatus === 'active' ? '启用' : '停用' }}
                </RsBadge>
              </template>
              <template #actions="{ row }">
                <div class="ops-row">
                  <button type="button" class="ops-link-btn" @click="toggleModel(row)">
                    {{ row.recordStatus === 'active' ? '停用' : '启用' }}
                  </button>
                  <span class="ops-link-sep">|</span>
                  <button type="button" class="ops-link-btn ops-link-btn--danger" @click="askRemoveModel(row)">
                    删除
                  </button>
                </div>
              </template>
              <template #empty>
                <div class="ops-empty" style="padding: 24px 8px">
                  该接入点还没有模型。至少添加一条，或填写默认模型编码。
                </div>
              </template>
            </RsTable>
          </section>
        </div>
      </template>
      <template #footer>
        <div class="ai-form__footer">
          <div class="ai-form__footer-test">
            <RsButton
              variant="secondary"
              :loading="testing"
              :disabled="!canTest"
              @click="testConnection"
            >
              测试连接
            </RsButton>
            <span
              v-if="testMessage"
              class="ai-form__test"
              :class="testOk ? 'ai-form__test--ok' : 'ai-form__test--fail'"
            >
              {{ testMessage }}
            </span>
          </div>
          <div class="ai-form__footer-right">
            <RsButton variant="secondary" @click="closeEditor">取消</RsButton>
            <RsButton variant="primary" :loading="saving" @click="save">保存</RsButton>
          </div>
        </div>
      </template>
    </RsDialog>

    <RsConfirmDialog
      v-model:open="confirmOpen"
      :title="confirmTitle"
      :description="confirmDescription"
      tone="danger"
      confirm-variant="danger"
      confirm-text="删除"
      show-overlay
      @confirm="onConfirmDelete"
    />
    </div>
  </div>
  </RsTooltipProvider>
</template>

<style scoped>
.ai-tabs {
  display: flex;
  gap: 4px;
  margin-bottom: 16px;
  padding: 4px;
  width: fit-content;
  border: 1px solid var(--ops-border-strong);
  border-radius: 8px;
  background: #fff;
}

.ai-tabs__btn {
  height: 32px;
  padding: 0 14px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: rgba(0, 0, 0, 0.65);
  font: inherit;
  font-size: 13px;
  cursor: pointer;
}

.ai-tabs__btn--on {
  background: #e6f4ff;
  color: #1677ff;
  font-weight: 650;
}

.ai-form {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.ai-form__section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.ai-form__section-title {
  margin: 0;
  font-size: 13px;
  font-weight: 650;
  color: rgba(0, 0, 0, 0.65);
}

.ai-form__grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.ai-form__rec {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}

.ai-form__rec-label {
  font-size: 12px;
  color: var(--ops-muted);
}

.ai-form__chip {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 8px;
  border: 1px solid var(--ops-border-strong);
  border-radius: 999px;
  background: #fff;
  color: rgba(0, 0, 0, 0.65);
  font: inherit;
  font-size: 12px;
  cursor: pointer;
}

.ai-form__chip:hover,
.ai-form__chip--on {
  border-color: #91caff;
  background: #e6f4ff;
  color: #1677ff;
}

.ai-form__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  width: 100%;
}

.ai-form__footer-test,
.ai-form__footer-right {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.ai-form__test {
  font-size: 12px;
  line-height: 1.4;
}

.ai-form__test--ok {
  color: #389e0d;
}

.ai-form__test--fail {
  color: #a8071a;
}

.ai-form__model-add {
  display: grid;
  grid-template-columns: 1fr 1fr auto;
  gap: 8px;
  align-items: center;
}

@media (max-width: 720px) {
  .ai-form__grid,
  .ai-form__model-add {
    grid-template-columns: 1fr;
  }
}
</style>
