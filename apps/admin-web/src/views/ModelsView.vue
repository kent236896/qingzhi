<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { DialogPlugin } from 'tdesign-vue-next'
import PageStatus from '@/components/PageStatus.vue'
import {
  clearCredential,
  copyModel,
  createModel,
  debugModel,
  deleteModel,
  downloadOllama,
  listBindings,
  listModels,
  listProviders,
  ollamaModels,
  ollamaProgress,
  ollamaStatus,
  resolveProvider,
  saveBinding,
  testModel,
  updateModel,
  type AiModel,
  type Binding,
  type BindingRole,
  type CatalogModel,
  type ModelType,
  type ModelUsage,
  type OllamaModel,
  type OllamaStatus,
  type Provider,
} from '@/api/models'
import { ApiError } from '@/utils/http'
import { feedback } from '@/utils/feedback'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const canManage = computed(() => auth.hasPermission('model:manage'))
const pageStatus = ref<'loading' | 'error' | 'ready'>('loading')
const tab = ref(route.query.tab === 'bindings' ? 'bindings' : 'list')
const type = ref<ModelType>('chat')
const keyword = ref('')
const items = ref<AiModel[]>([])
const bindings = ref<Binding[]>([])
const bindingDraft = ref<Record<string, string>>({})

const types: { value: ModelType; label: string }[] = [
  { value: 'chat', label: '对话模型' },
  { value: 'embedding', label: '向量模型' },
  { value: 'rerank', label: '重排模型' },
  { value: 'vllm', label: '视觉模型' },
  { value: 'asr', label: '语音模型' },
]
const roles: { value: BindingRole; label: string; hint: string; type: ModelType }[] = [
  { value: 'embedding', label: '向量', hint: '新建知识库的向量模型', type: 'embedding' },
  { value: 'summary', label: '摘要', hint: '新建知识库的摘要模型', type: 'chat' },
  { value: 'wiki_synthesis', label: 'Wiki 生成', hint: '勾选 Wiki 时使用', type: 'chat' },
  { value: 'vlm', label: '视觉', hint: '视觉理解场景', type: 'vllm' },
  { value: 'asr', label: '语音', hint: '语音识别场景', type: 'asr' },
  { value: 'rerank', label: '重排', hint: '检索重排场景', type: 'rerank' },
]
const roleLabel: Record<string, string> = Object.fromEntries(roles.map((item) => [item.value, item.label]))

const filtered = computed(() => {
  const text = keyword.value.trim().toLowerCase()
  return items.value.filter((item) => {
    if (item.type !== type.value) return false
    if (!text) return true
    return item.name.toLowerCase().includes(text) || (item.display_name || '').toLowerCase().includes(text)
  })
})
const counts = computed(() => {
  const result: Record<string, number> = {}
  for (const item of types) result[item.value] = items.value.filter((model) => model.type === item.value).length
  return result
})

const drawerVisible = ref(false)
const editing = ref(false)
const saving = ref(false)
const tested = ref(false)
const formError = ref('')
const testMessage = ref('')
const providers = ref<Provider[]>([])
const resolvePreview = ref<Record<string, unknown> | null>(null)
const ollama = ref<OllamaStatus | null>(null)
const ollamaEnabled = ref(false)
const installed = ref<OllamaModel[]>([])
const downloadTask = ref<{ id: string; progress: number; status: string; message: string } | null>(null)
let resolveTimer = 0
let progressTimer = 0

const form = ref(emptyForm())
const copyVisible = ref(false)
const copyName = ref('')
const copySource = ref<AiModel | null>(null)
const debugVisible = ref(false)
const debugTarget = ref<AiModel | null>(null)
const debugInput = ref('')
const debugTemperature = ref<number | undefined>()
const debugMaxTokens = ref<number | undefined>()
const debugThinking = ref(false)
const debugResult = ref<Record<string, unknown> | null>(null)
const conflictVisible = ref(false)
const conflict = ref<ModelUsage | null>(null)
const conflictMessage = ref('')

const provider = computed(() => providers.value.find((item) => item.value === form.value.provider))
const localMode = computed(() => form.value.provider === 'ollama' || form.value.provider === 'local' || form.value.source === 'local')
const extraFields = computed(() => (provider.value?.extra_fields ?? []).filter((field) => !field.model_types?.length || field.model_types.includes(form.value.type)))
const catalogOptions = computed(() => (provider.value?.models ?? []).filter((item) => !item.type || item.type === form.value.type))
const apiKeyLabel = computed(() => credentialText('api_key', 'API Key'))
const secretLabel = computed(() => credentialText('app_secret', 'App Secret'))
const previewLines = computed(() => describePreview(resolvePreview.value))

function emptyForm() {
  return {
    id: '',
    type: 'chat' as ModelType,
    provider: '',
    name: '',
    display_name: '',
    description: '',
    source: 'remote',
    base_url: '',
    api_key: '',
    app_secret: '',
    dimension: undefined as number | undefined,
    extra: {} as Record<string, string | number>,
    headers: [] as { key: string; value: string }[],
    context_window: undefined as number | undefined,
    max_output_tokens: undefined as number | undefined,
    configured: false,
  }
}

function countOf(value: ModelType) {
  return counts.value[value] ?? 0
}

async function load() {
  pageStatus.value = items.value.length ? 'ready' : 'loading'
  try {
    const [modelResponse, bindingResponse] = await Promise.all([listModels(), listBindings()])
    items.value = modelResponse.data ?? []
    bindings.value = bindingResponse.data ?? []
    const draft: Record<string, string> = {}
    for (const item of bindings.value) draft[item.role] = item.model_id
    bindingDraft.value = draft
    pageStatus.value = 'ready'
  } catch {
    if (!items.value.length) pageStatus.value = 'error'
  }
}

async function openCreate() {
  editing.value = false
  tested.value = false
  formError.value = ''
  testMessage.value = ''
  resolvePreview.value = null
  form.value = { ...emptyForm(), type: type.value }
  drawerVisible.value = true
  await loadProviders()
}

async function openEdit(row: AiModel) {
  if (row.is_builtin) {
    feedback.warning('内置模型不可修改')
    return
  }
  editing.value = true
  tested.value = false
  formError.value = ''
  testMessage.value = ''
  const params = row.parameters ?? {}
  const embedding = (params.embedding_parameters ?? {}) as { dimension?: number }
  const extra = (params.extra_config ?? {}) as Record<string, string>
  const headers = Object.entries((params.custom_headers ?? {}) as Record<string, string>).map(([key, value]) => ({ key, value }))
  form.value = {
    ...emptyForm(),
    id: row.id,
    type: row.type,
    provider: String(params.provider ?? ''),
    name: row.name,
    display_name: row.display_name,
    description: row.description,
    source: row.source,
    base_url: String(params.base_url ?? ''),
    dimension: embedding.dimension,
    extra: { ...extra },
    headers,
    context_window: numberOrUndefined(params.context_window),
    max_output_tokens: numberOrUndefined(params.max_output_tokens),
    configured: Boolean(row.credentials?.api_key?.configured),
  }
  drawerVisible.value = true
  await loadProviders()
}

async function loadProviders() {
  try {
    providers.value = await listProviders(form.value.type, true)
  } catch (error) {
    formError.value = error instanceof ApiError ? error.message : '厂商目录加载失败'
  }
  await refreshOllama()
}

async function refreshOllama() {
  if (!localMode.value) {
    ollama.value = null
    ollamaEnabled.value = false
    return
  }
  try {
    const response = await ollamaStatus()
    ollama.value = response.data
    ollamaEnabled.value = true
    const models = await ollamaModels()
    installed.value = models.data ?? []
  } catch (error) {
    ollamaEnabled.value = false
    ollama.value = null
    if (!(error instanceof ApiError) || error.httpStatus !== 404) {
      formError.value = error instanceof ApiError ? error.message : '本地模型状态获取失败'
    }
  }
}

function onProviderChange() {
  const current = provider.value
  form.value.source = current?.value === 'ollama' || current?.value === 'local' ? 'local' : form.value.source || 'remote'
  if (!editing.value && current?.default_urls?.[form.value.type]) {
    form.value.base_url = current.default_urls[form.value.type]
  }
  void refreshOllama()
  scheduleResolve()
}

function onCatalogChange(value: string) {
  const selected = catalogOptions.value.find((item) => item.id === value || item.name === value)
  if (!selected) {
    scheduleResolve()
    return
  }
  applyCatalog(selected)
}

function applyCatalog(selected: CatalogModel) {
  form.value.name = selected.id || selected.name
  if (!form.value.display_name) form.value.display_name = selected.name
  if (selected.dimension) form.value.dimension = selected.dimension
  if (selected.context_window) form.value.context_window = selected.context_window
  if (selected.max_output_tokens) form.value.max_output_tokens = selected.max_output_tokens
  scheduleResolve()
}

function scheduleResolve() {
  window.clearTimeout(resolveTimer)
  resolveTimer = window.setTimeout(() => void runResolve(), 500)
}

async function runResolve() {
  if (!form.value.provider || !form.value.name) {
    resolvePreview.value = null
    return
  }
  try {
    const response = await resolveProvider({
      provider: form.value.provider,
      model: form.value.name,
      base_url: form.value.base_url,
      type: form.value.type,
    })
    resolvePreview.value = response.data
  } catch (error) {
    resolvePreview.value = { error: error instanceof ApiError ? error.message : '无法解析接入配置' }
  }
}

function payload() {
  const extra: Record<string, string> = {}
  for (const [key, value] of Object.entries(form.value.extra)) {
    if (value !== '' && value != null) extra[key] = String(value)
  }
  const headers: Record<string, string> = {}
  for (const item of form.value.headers) {
    if (item.key.trim()) headers[item.key.trim()] = item.value
  }
  const parameters: Record<string, unknown> = {
    base_url: form.value.base_url,
    provider: form.value.provider,
    extra_config: extra,
    custom_headers: headers,
  }
  if (form.value.type === 'embedding') {
    parameters.embedding_parameters = { dimension: Number(form.value.dimension || 0) }
  }
  if (form.value.context_window) parameters.context_window = Number(form.value.context_window)
  if (form.value.max_output_tokens) parameters.max_output_tokens = Number(form.value.max_output_tokens)
  return {
    name: form.value.name,
    display_name: form.value.display_name,
    description: form.value.description,
    type: form.value.type,
    source: form.value.source || 'remote',
    parameters,
    api_key: form.value.api_key,
    app_secret: form.value.app_secret,
  }
}

async function runTestRow(row: AiModel) {
  const params = row.parameters ?? {}
  try {
    const response = await testModel({
      type: row.type,
      model_name: row.name,
      source: row.source,
      base_url: String(params.base_url ?? ''),
      provider: String(params.provider ?? ''),
      model_id: row.id,
    })
    const result = response.data
    feedback.success(`${result.available ? '连接成功' : '连接失败'}：${result.message}`)
  } catch (error) {
    feedback.error(error instanceof ApiError ? error.message : '测试失败')
  }
}

async function runTest() {
  testMessage.value = ''
  try {
    const response = await testModel({
      type: form.value.type,
      model_name: form.value.name,
      source: form.value.source,
      base_url: form.value.base_url,
      api_key: form.value.api_key,
      app_secret: form.value.app_secret,
      provider: form.value.provider,
      dimension: form.value.dimension,
      extra_config: form.value.extra,
      model_id: editing.value ? form.value.id : '',
    })
    tested.value = true
    const result = response.data
    if (result.dimension) form.value.dimension = result.dimension
    testMessage.value = `${result.available ? '连接成功' : '连接失败'}：${result.message}（${result.elapsed_ms} ms${result.dimension ? `，维度 ${result.dimension}` : ''}）`
  } catch (error) {
    testMessage.value = error instanceof ApiError ? error.message : '测试失败'
  }
}

async function save() {
  if (!form.value.base_url.startsWith('http://') && !form.value.base_url.startsWith('https://') && form.value.base_url) {
    formError.value = '接口地址必须以 http 或 https 开头'
    return
  }
  if (!tested.value) {
    const dialog = DialogPlugin.confirm({
      header: '尚未测试连接',
      body: '尚未测试连接，是否仍然保存？',
      onConfirm: async () => {
        dialog.hide()
        await persist()
      },
    })
    return
  }
  await persist()
}

async function persist() {
  saving.value = true
  formError.value = ''
  try {
    if (editing.value) await updateModel(form.value.id, payload())
    else await createModel(payload())
    feedback.success('模型已保存')
    drawerVisible.value = false
    await load()
  } catch (error) {
    formError.value = error instanceof ApiError ? error.message : '保存失败'
  } finally {
    saving.value = false
  }
}

function askClearKey() {
  const dialog = DialogPlugin.confirm({
    header: '清除密钥',
    body: '清除后，调用该模型将不再携带已保存的密钥。',
    onConfirm: async () => {
      await clearCredential(form.value.id, 'api_key')
      form.value.configured = false
      feedback.success('密钥已清除')
      dialog.hide()
    },
  })
}

function openCopy(row: AiModel) {
  if (row.is_builtin) {
    feedback.warning('内置模型不可修改')
    return
  }
  copySource.value = row
  copyName.value = `${row.display_name || row.name} 副本`
  copyVisible.value = true
}

async function confirmCopy() {
  if (!copySource.value) return
  try {
    const response = await copyModel(copySource.value.id, copyName.value)
    feedback.success('模型已复制')
    copyVisible.value = false
    await load()
    const created = items.value.find((item) => item.id === response.data.id)
    if (created) type.value = created.type
  } catch (error) {
    feedback.error(error instanceof ApiError ? error.message : '复制失败')
  }
}

function openDebug(row: AiModel) {
  const dialog = DialogPlugin.confirm({
    header: '调试会产生费用',
    body: '调试会向模型服务发起真实调用，并可能产生费用。',
    onConfirm: () => {
      dialog.hide()
      debugTarget.value = row
      debugInput.value = ''
      debugResult.value = null
      debugVisible.value = true
    },
  })
}

async function submitDebug() {
  if (!debugTarget.value) return
  const dialog = DialogPlugin.confirm({
    header: '确认发起真实调用',
    body: '提交后会向模型服务发起真实调用，并可能产生费用。',
    onConfirm: async () => {
      dialog.hide()
      const data = new FormData()
      data.set('input', debugInput.value)
      data.set('options', JSON.stringify({
        temperature: debugTemperature.value,
        max_tokens: debugMaxTokens.value,
        thinking: debugThinking.value,
      }))
      try {
        const response = await debugModel(debugTarget.value!.id, data)
        debugResult.value = response.data
      } catch (error) {
        debugResult.value = { error: error instanceof ApiError ? error.message : '调试失败' }
      }
    },
  })
}

function remove(row: AiModel) {
  if (row.is_builtin) {
    feedback.warning('内置模型不可修改')
    return
  }
  const dialog = DialogPlugin.confirm({
    header: '删除模型',
    body: `删除「${row.display_name || row.name}」后无法恢复。`,
    onConfirm: async () => {
      try {
        await deleteModel(row.id)
        feedback.success('模型已删除')
        dialog.hide()
        await load()
      } catch (error) {
        dialog.hide()
        conflictMessage.value = error instanceof ApiError ? error.message : '删除失败'
        conflict.value = (error instanceof ApiError ? error.data : null) as ModelUsage | null
        conflictVisible.value = true
      }
    },
  })
}

async function saveRole(role: BindingRole) {
  try {
    const response = await saveBinding(role, bindingDraft.value[role])
    if (response.data.warning) feedback.warning(response.data.warning)
    else feedback.success('默认模型已保存')
    await load()
  } catch (error) {
    feedback.error(error instanceof ApiError ? error.message : '保存失败')
  }
}

async function startDownload() {
  const name = form.value.name.trim()
  if (!name) return
  const response = await downloadOllama(name)
  downloadTask.value = { id: response.data.id, progress: response.data.progress, status: response.data.status, message: response.data.message }
  window.clearInterval(progressTimer)
  progressTimer = window.setInterval(async () => {
    if (!downloadTask.value) return
    try {
      const progress = await ollamaProgress(downloadTask.value.id)
      downloadTask.value = { id: progress.data.id, progress: progress.data.progress, status: progress.data.status, message: progress.data.message }
      if (['completed', 'failed', 'success', 'error'].includes(progress.data.status)) {
        window.clearInterval(progressTimer)
        await refreshOllama()
      }
    } catch {
      window.clearInterval(progressTimer)
    }
  }, 2000)
}

function credentialText(key: string, fallback: string) {
  const labels = provider.value?.credential_labels ?? []
  for (const item of labels) {
    const name = String(item.key ?? item.field ?? '')
    if (name === key || name.includes(key)) {
      return String(item.label ?? item.labels && (item.labels as Record<string, string>)['zh-CN'] ?? fallback)
    }
  }
  return fallback
}

function describePreview(value: Record<string, unknown> | null) {
  if (!value) return []
  if (typeof value.error === 'string') return [value.error]
  const caps = (value.capabilities ?? {}) as Record<string, unknown>
  const model = (value.model ?? {}) as Record<string, unknown>
  const lines = [
    `协议：${String(value.api ?? '—')}`,
    `目录匹配：${value.cataloged ? '是' : '否'}`,
    `支持推理：${caps.reasoning || model.reasoning ? '是' : '否'}`,
  ]
  const levels = (caps.thinking_levels ?? model.thinking_levels) as unknown
  if (Array.isArray(levels) && levels.length) lines.push(`思考等级：${levels.join('、')}`)
  if (model.context_window) lines.push(`上下文窗口：${model.context_window}`)
  if (model.max_output_tokens) lines.push(`最大输出：${model.max_output_tokens}`)
  return lines
}

function numberOrUndefined(value: unknown) {
  const number = Number(value)
  return Number.isFinite(number) && number > 0 ? number : undefined
}

function modelsFor(modelType: ModelType) {
  return items.value.filter((item) => item.type === modelType)
}

function sourceText(source: string) {
  if (source === 'local') return '本地'
  if (source === 'remote') return '远程'
  return source
}

watch(() => form.value.type, () => {
  if (drawerVisible.value) void loadProviders()
})
watch(() => [form.value.provider, form.value.name, form.value.base_url], scheduleResolve)
watch(tab, (value) => {
  void router.replace({ query: value === 'bindings' ? { tab: 'bindings' } : {} })
})

onMounted(load)
onUnmounted(() => {
  window.clearTimeout(resolveTimer)
  window.clearInterval(progressTimer)
})
</script>

<template>
  <div>
    <PageStatus v-if="pageStatus !== 'ready'" :status="pageStatus" @retry="load" />
    <t-card v-else :bordered="false">
      <t-tabs v-model="tab">
        <t-tab-panel value="list" label="模型列表">
          <div class="toolbar">
            <t-radio-group v-model="type" variant="default-filled">
              <t-radio-button v-for="item in types" :key="item.value" :value="item.value">{{ item.label }} {{ countOf(item.value) }}</t-radio-button>
            </t-radio-group>
            <t-input v-model="keyword" placeholder="搜索模型名或显示名" clearable style="width: 240px" />
            <t-button v-if="canManage" theme="primary" @click="openCreate">添加模型</t-button>
          </div>
          <div v-if="!filtered.length" class="empty">
            <p>还没有{{ types.find((item) => item.value === type)?.label }}</p>
            <t-button v-if="canManage" theme="primary" @click="openCreate">添加第一个模型</t-button>
          </div>
          <t-table v-else row-key="id" :data="filtered" :columns="[
            { colKey: 'display_name', title: '显示名' },
            { colKey: 'name', title: '模型名' },
            { colKey: 'source', title: '来源', width: 100 },
            { colKey: 'flags', title: '标记', width: 220 },
            { colKey: 'secret', title: '密钥', width: 110 },
            { colKey: 'actions', title: '操作', width: 280 },
          ]">
            <template #display_name="{ row }">{{ row.display_name || row.name }}</template>
            <template #source="{ row }">{{ sourceText(row.source) }}</template>
            <template #flags="{ row }">
              <t-tag v-if="row.is_builtin" size="small">内置</t-tag>
              <t-tag v-for="role in row.bound_roles" :key="role" size="small" theme="primary" variant="light">默认·{{ roleLabel[role] || role }}</t-tag>
            </template>
            <template #secret="{ row }">{{ row.credentials?.api_key?.configured ? '已配置' : '未配置' }}</template>
            <template #actions="{ row }">
              <t-button theme="primary" variant="text" :disabled="row.is_builtin || !canManage" @click="openEdit(row)">编辑</t-button>
              <t-button variant="text" :disabled="row.is_builtin || !canManage" @click="openCopy(row)">复制</t-button>
              <t-button variant="text" :disabled="!canManage" @click="runTestRow(row)">测试</t-button>
              <t-button variant="text" :disabled="!canManage" @click="openDebug(row)">调试</t-button>
              <t-button theme="danger" variant="text" :disabled="row.is_builtin || !canManage" @click="remove(row)">删除</t-button>
            </template>
          </t-table>
        </t-tab-panel>
        <t-tab-panel value="bindings" label="默认模型">
          <div v-for="role in roles" :key="role.value" class="binding">
            <div>
              <strong>{{ role.label }}</strong>
              <p>{{ role.hint }}</p>
            </div>
            <t-select v-model="bindingDraft[role.value]" clearable placeholder="选择模型" style="width: 320px" :disabled="!canManage">
              <t-option v-for="model in modelsFor(role.type)" :key="model.id" :value="model.id" :label="model.display_name || model.name" />
            </t-select>
            <t-button v-if="canManage" theme="primary" @click="saveRole(role.value)">保存</t-button>
          </div>
          <p class="hint">更换向量模型只影响之后新建的知识库，已有文件的知识库不能更换向量模型。</p>
        </t-tab-panel>
      </t-tabs>
    </t-card>

    <t-drawer v-model:visible="drawerVisible" :header="editing ? '编辑模型' : '添加模型'" size="640px" :footer="false">
      <t-alert v-if="formError" theme="error" :message="formError" class="gap" />
      <t-form label-align="top">
        <t-form-item label="类型">
          <t-select v-model="form.type" :disabled="editing">
            <t-option v-for="item in types" :key="item.value" :value="item.value" :label="item.label" />
          </t-select>
        </t-form-item>
        <t-form-item label="厂商">
          <t-select v-model="form.provider" filterable placeholder="选择厂商" @change="onProviderChange">
            <t-option v-for="item in providers" :key="item.value" :value="item.value" :label="item.label" />
          </t-select>
        </t-form-item>
        <t-form-item label="模型名">
          <t-select v-model="form.name" filterable creatable placeholder="选择或输入模型名" @change="onCatalogChange">
            <t-option v-for="item in catalogOptions" :key="item.id" :value="item.id" :label="item.name || item.id" />
          </t-select>
        </t-form-item>
        <t-form-item label="显示名"><t-input v-model="form.display_name" /></t-form-item>
        <t-form-item label="接口地址">
          <t-input v-model="form.base_url" placeholder="https://" />
          <p class="hint">内网地址需要在 WeKnora 配置 SSRF_WHITELIST_EXTRA。</p>
        </t-form-item>
        <template v-if="localMode">
          <t-alert v-if="!ollamaEnabled" theme="warning" message="本地模型下载未开启。容器内访问宿主机应使用 host.docker.internal。" />
          <t-alert v-else :theme="ollama?.available ? 'success' : 'warning'" :message="ollama?.available ? `Ollama ${ollama.version} · ${ollama.base_url}` : `Ollama 不可用。容器内请使用 host.docker.internal。${ollama?.error || ''}`" />
          <t-form-item v-if="ollamaEnabled" label="已安装模型">
            <t-select v-model="form.name" filterable creatable placeholder="选择已安装模型">
              <t-option v-for="item in installed" :key="item.name" :value="item.name" :label="item.name" />
            </t-select>
            <t-button variant="outline" @click="startDownload">下载模型</t-button>
            <t-progress v-if="downloadTask" :percentage="Math.round(downloadTask.progress)" />
          </t-form-item>
        </template>
        <t-form-item :label="apiKeyLabel">
          <t-input v-model="form.api_key" type="password" autocomplete="new-password" :placeholder="form.configured ? '已配置，留空保持不变' : ''" />
          <t-button v-if="editing && form.configured" variant="text" @click="askClearKey">清除密钥</t-button>
        </t-form-item>
        <t-form-item :label="secretLabel">
          <t-input v-model="form.app_secret" type="password" autocomplete="new-password" placeholder="留空保持不变" />
        </t-form-item>
        <t-form-item v-for="field in extraFields" :key="field.key" :label="field.label">
          <t-select v-if="field.type === 'select'" v-model="form.extra[field.key]" :options="(field.options ?? []).map((item) => ({ label: item.label, value: item.value }))" />
          <t-input-number v-else-if="field.type === 'number'" v-model="form.extra[field.key]" />
          <t-input v-else v-model="form.extra[field.key]" :type="field.secret || field.type === 'password' ? 'password' : 'text'" autocomplete="new-password" :placeholder="field.placeholder" />
        </t-form-item>
        <t-form-item v-if="form.type === 'embedding'" label="维度">
          <t-input-number v-model="form.dimension" :min="1" />
          <t-button variant="outline" :disabled="!canManage" @click="runTest">测试并检测维度</t-button>
          <p class="hint">更换已被知识库使用的向量模型前，请确认维度一致。已有文件的知识库不能更换向量模型。</p>
        </t-form-item>
        <t-collapse>
          <t-collapse-panel header="高级设置" value="advanced">
            <div v-for="(header, index) in form.headers" :key="index" class="pair">
              <t-input v-model="header.key" placeholder="请求头" />
              <t-input v-model="header.value" placeholder="值" />
            </div>
            <t-button variant="text" @click="form.headers.push({ key: '', value: '' })">添加请求头</t-button>
            <t-form-item label="上下文窗口"><t-input-number v-model="form.context_window" :min="0" /></t-form-item>
            <t-form-item label="最大输出"><t-input-number v-model="form.max_output_tokens" :min="0" /></t-form-item>
          </t-collapse-panel>
        </t-collapse>
        <div v-if="previewLines.length" class="preview">
          <p v-for="line in previewLines" :key="line">{{ line }}</p>
        </div>
        <p v-if="testMessage" class="hint">{{ testMessage }}</p>
        <div class="actions">
          <t-button v-if="canManage" variant="outline" @click="runTest">测试连接</t-button>
          <t-button v-if="canManage" theme="primary" :loading="saving" @click="save">保存</t-button>
        </div>
      </t-form>
    </t-drawer>

    <t-dialog v-model:visible="copyVisible" header="复制模型" @confirm="confirmCopy">
      <t-input v-model="copyName" placeholder="新的显示名" />
    </t-dialog>
    <t-dialog v-model:visible="debugVisible" header="调试模型" :footer="false" width="720px">
      <t-textarea v-model="debugInput" placeholder="输入调试文本" :maxlength="65536" />
      <t-collapse class="gap">
        <t-collapse-panel header="高级选项">
          <t-form-item label="温度"><t-input-number v-model="debugTemperature" :min="0" :max="2" :decimal-places="2" /></t-form-item>
          <t-form-item label="最大输出"><t-input-number v-model="debugMaxTokens" :min="1" :max="8192" /></t-form-item>
          <t-checkbox v-model="debugThinking">开启思考</t-checkbox>
        </t-collapse-panel>
      </t-collapse>
      <t-button theme="primary" @click="submitDebug">提交</t-button>
      <pre v-if="debugResult" class="preview">{{ JSON.stringify(debugResult, null, 2) }}</pre>
    </t-dialog>
    <t-dialog v-model:visible="conflictVisible" header="模型仍被占用" :footer="false">
      <p>{{ conflictMessage }}</p>
      <p v-if="conflict?.local_roles?.length">本地默认场景：{{ conflict.local_roles.map((role) => roleLabel[role] || role).join('、') }}</p>
      <ul>
        <li v-for="item in conflict?.knowledge_bases ?? []" :key="item.id">知识库 {{ item.name }}（{{ item.bindings.join('、') }}）</li>
        <li v-for="item in conflict?.agents ?? []" :key="item.id">智能体 {{ item.name }}（{{ item.bindings.join('、') }}）</li>
      </ul>
      <p v-if="conflict?.long_term_memory?.bindings?.length">长期记忆：{{ conflict.long_term_memory.bindings.join('、') }}</p>
      <p>请先在这些位置更换模型。</p>
      <t-button v-if="conflict?.local_roles?.length" theme="primary" variant="text" @click="tab = 'bindings'; conflictVisible = false">前往默认模型</t-button>
    </t-dialog>
  </div>
</template>

<style scoped>
.toolbar, .actions, .pair, .binding { display: flex; gap: 12px; align-items: center; }
.toolbar { margin: 16px 0; flex-wrap: wrap; }
.binding { justify-content: space-between; padding: 12px 0; border-bottom: 1px solid var(--td-component-border); }
.binding p, .hint, .preview { color: var(--qz-text-secondary); }
.empty { padding: 48px 0; text-align: center; }
.gap { margin: 12px 0; }
.preview { margin-top: 12px; white-space: pre-wrap; }
</style>
