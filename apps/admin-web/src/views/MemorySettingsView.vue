<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { onBeforeRouteLeave } from 'vue-router'
import PageStatus from '@/components/PageStatus.vue'
import { getMemoryConfig, getPersonalMemoryStatus, updateMemoryConfig, type MemoryConfig, type PersonalMemoryStatus } from '@/api/memory'
import { listModels, type AiModel } from '@/api/models'
import { ApiError } from '@/utils/http'
import { feedback } from '@/utils/feedback'
import { useAuthStore } from '@/stores/auth'
import { DialogPlugin } from 'tdesign-vue-next'

const auth = useAuthStore()
const canManage = computed(() => auth.hasPermission('model:manage'))
const pageStatus = ref<'loading' | 'error' | 'ready'>('loading')
const errorMessage = ref('')
const saving = ref(false)
const dirty = ref(false)
const personal = ref<PersonalMemoryStatus | null>(null)
const chatModels = ref<AiModel[]>([])
const embeddingModels = ref<AiModel[]>([])

const form = reactive<MemoryConfig>({
  enabled: false,
  write_mode: 'explicit_only',
  extract_model_id: '',
  max_items: 200,
  extract_delay_seconds: 30,
  extract_min_interval_seconds: 60,
  extract_instructions: '',
  interest_threshold: 3,
  embedding_model_id: '',
  vector_recall: true,
  retrieval_conditioning: true,
})

const snapshot = ref('')

const writeModeOptions = [
  { label: '仅显式写入 (explicit_only)', value: 'explicit_only' },
  { label: '自动提取 (auto)', value: 'auto' },
]

function serialize(cfg: MemoryConfig) {
  return JSON.stringify(cfg)
}

function apply(cfg: MemoryConfig) {
  Object.assign(form, {
    enabled: Boolean(cfg.enabled),
    write_mode: cfg.write_mode || 'explicit_only',
    extract_model_id: cfg.extract_model_id || '',
    max_items: cfg.max_items || 200,
    extract_delay_seconds: cfg.extract_delay_seconds ?? 30,
    extract_min_interval_seconds: cfg.extract_min_interval_seconds ?? 60,
    extract_instructions: cfg.extract_instructions || '',
    interest_threshold: cfg.interest_threshold || 3,
    embedding_model_id: cfg.embedding_model_id || '',
    vector_recall: Boolean(cfg.vector_recall),
    retrieval_conditioning: Boolean(cfg.retrieval_conditioning),
  })
  snapshot.value = serialize(form)
  dirty.value = false
}

function validate(): string | null {
  if (form.write_mode !== 'explicit_only' && form.write_mode !== 'auto') return '请选择合法的写入模式'
  if (form.max_items < 1 || form.max_items > 10000) return '有效记忆上限须在 1–10000'
  if (form.extract_delay_seconds < 0 || form.extract_delay_seconds > 3600) return '提取延迟须在 0–3600 秒'
  if (form.extract_min_interval_seconds < 0 || form.extract_min_interval_seconds > 86400) return '最短间隔须在 0–86400 秒'
  if (form.interest_threshold < 1 || form.interest_threshold > 100) return '关注主题阈值须在 1–100'
  return null
}

async function load() {
  pageStatus.value = 'loading'
  errorMessage.value = ''
  try {
    const [cfgRes, statusRes, modelsRes] = await Promise.all([
      getMemoryConfig(),
      getPersonalMemoryStatus(),
      listModels().catch(() => ({ data: [] as AiModel[] })),
    ])
    apply(cfgRes.data)
    personal.value = statusRes.data
    const models = modelsRes.data ?? []
    chatModels.value = models.filter((m) => m.type === 'chat')
    embeddingModels.value = models.filter((m) => m.type === 'embedding')
    if (!chatModels.value.length) chatModels.value = models
    if (!embeddingModels.value.length) embeddingModels.value = models
    pageStatus.value = 'ready'
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '无法加载长期记忆配置'
    pageStatus.value = 'error'
  }
}

async function save() {
  const problem = validate()
  if (problem) {
    feedback.warning(problem)
    return
  }
  if (!canManage.value) {
    feedback.warning('没有管理权限')
    return
  }
  const dialog = DialogPlugin.confirm({
    header: '确认保存空间级长期记忆配置？',
    body: '该配置影响当前 WeKnora 空间内全部用户的记忆策略，请确认后继续。',
    confirmBtn: '保存',
    onConfirm: async () => {
      dialog.hide()
      saving.value = true
      try {
        const response = await updateMemoryConfig({ ...form })
        apply(response.data)
        feedback.success('长期记忆配置已保存')
      } catch (error) {
        if (!(error instanceof ApiError)) {
          feedback.error('保存失败')
        }
      } finally {
        saving.value = false
      }
    },
  })
}

function cancel() {
  if (!snapshot.value) return
  apply(JSON.parse(snapshot.value) as MemoryConfig)
}

watch(
  form,
  () => {
    dirty.value = snapshot.value !== '' && serialize(form) !== snapshot.value
  },
  { deep: true },
)

function beforeUnload(event: BeforeUnloadEvent) {
  if (!dirty.value) return
  event.preventDefault()
  event.returnValue = ''
}

onBeforeRouteLeave((_to, _from, next) => {
  if (!dirty.value) {
    next()
    return
  }
  const dialog = DialogPlugin.confirm({
    header: '有未保存的修改',
    body: '离开页面将丢失未保存的长期记忆配置，确定离开吗？',
    onConfirm: () => {
      dialog.hide()
      next()
    },
    onClose: () => next(false),
  })
})

onMounted(() => {
  window.addEventListener('beforeunload', beforeUnload)
  void load()
})

onBeforeUnmount(() => {
  window.removeEventListener('beforeunload', beforeUnload)
})
</script>

<template>
  <div class="page-grid">
    <t-card :bordered="false" title="WeKnora 长期记忆（空间级）">
      <PageStatus :status="pageStatus" :error-message="errorMessage" description="读取 WeKnora 空间记忆配置" @retry="load">
        <t-alert
          theme="info"
          title="说明"
          message="此处管理当前 API Key 所属 WeKnora 空间的记忆策略。保存时会先读取完整配置再合并提交，避免清空未展示字段。"
          class="mb"
        />
        <t-form label-width="180px" class="form">
          <t-form-item label="启用长期记忆">
            <t-switch v-model="form.enabled" :disabled="!canManage" />
          </t-form-item>
          <t-form-item label="写入模式" required>
            <t-select v-model="form.write_mode" :options="writeModeOptions" :disabled="!canManage" />
          </t-form-item>
          <t-form-item label="记忆提取模型">
            <t-select
              v-model="form.extract_model_id"
              clearable
              filterable
              placeholder="可选，从 WeKnora 模型列表选择"
              :disabled="!canManage"
              :options="chatModels.map((m) => ({ label: m.display_name || m.name, value: m.id }))"
            />
          </t-form-item>
          <t-form-item label="向量模型">
            <t-select
              v-model="form.embedding_model_id"
              clearable
              filterable
              placeholder="记忆语义召回向量模型"
              :disabled="!canManage"
              :options="embeddingModels.map((m) => ({ label: m.display_name || m.name, value: m.id }))"
            />
          </t-form-item>
          <t-form-item label="每人有效记忆上限">
            <t-input-number v-model="form.max_items" :min="1" :max="10000" :disabled="!canManage" />
          </t-form-item>
          <t-form-item label="提取延迟（秒）">
            <t-input-number v-model="form.extract_delay_seconds" :min="0" :max="3600" :disabled="!canManage" />
          </t-form-item>
          <t-form-item label="提取最短间隔（秒）">
            <t-input-number v-model="form.extract_min_interval_seconds" :min="0" :max="86400" :disabled="!canManage" />
          </t-form-item>
          <t-form-item label="关注主题阈值">
            <t-input-number v-model="form.interest_threshold" :min="1" :max="100" :disabled="!canManage" />
          </t-form-item>
          <t-form-item label="向量召回">
            <t-switch v-model="form.vector_recall" :disabled="!canManage" />
          </t-form-item>
          <t-form-item label="检索条件化">
            <t-switch v-model="form.retrieval_conditioning" :disabled="!canManage" />
            <span class="hint">允许记忆影响问题理解与文档排序</span>
          </t-form-item>
          <t-form-item label="提取规则">
            <t-textarea v-model="form.extract_instructions" :disabled="!canManage" :autosize="{ minRows: 3, maxRows: 8 }" />
          </t-form-item>
          <t-form-item>
            <t-space>
              <t-button v-if="canManage" theme="primary" :loading="saving" :disabled="!dirty" @click="save">保存</t-button>
              <t-button variant="outline" :disabled="!dirty || saving" @click="cancel">取消修改</t-button>
              <t-tag v-if="dirty" theme="warning" variant="light">有未保存修改</t-tag>
            </t-space>
          </t-form-item>
        </t-form>
      </PageStatus>
    </t-card>

    <t-card :bordered="false" title="个人长期记忆">
      <t-alert
        v-if="personal && !personal.ready"
        theme="warning"
        title="待集成"
        :message="personal.reason"
      />
      <p v-else class="hint">个人记忆设置页将在隔离验证通过后开放。</p>
    </t-card>
  </div>
</template>

<style scoped>
.form {
  max-width: 720px;
}
.mb {
  margin-bottom: 16px;
}
.hint {
  margin-left: 8px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
}
</style>
