<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import PageStatus from '@/components/PageStatus.vue'
import ModelsView from '@/views/ModelsView.vue'
import { getSystemSettings, updateSystemSettings, type SystemSettings } from '@/api/settings'
import { listBindings, listModels, saveBinding, type AiModel, type BindingRole, type ModelType } from '@/api/models'
import { ApiError } from '@/utils/http'
import { feedback } from '@/utils/feedback'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const canUpdate = computed(() => auth.hasPermission('system:settings:update'))
const canManageModels = computed(() => auth.hasPermission('model:manage'))
const connectionReady = computed(() => Boolean(settings.value?.base_url && settings.value.api_key_configured))
const pageStatus = ref<'loading' | 'error' | 'ready'>('loading')
const errorMessage = ref('')
const saving = ref(false)
const settings = ref<SystemSettings | null>(null)
const baseURL = ref('')
const apiKey = ref('')
const models = ref<AiModel[]>([])
const modelHint = ref('')
const bindingDraft = ref<Record<string, string>>({})
const popupOpen = reactive<Record<string, boolean>>({})
const editor = ref<{
  openCreate: (modelType: ModelType) => void
  openEdit: (row: AiModel) => void
  remove: (row: AiModel) => void
} | null>(null)

const roles: { value: BindingRole; label: string; hint: string; type: ModelType }[] = [
  { value: 'embedding', label: '向量', hint: '新建知识库的向量模型', type: 'embedding' },
  { value: 'summary', label: '摘要', hint: '新建知识库的摘要模型', type: 'chat' },
  { value: 'wiki_synthesis', label: 'Wiki 生成', hint: '勾选 Wiki 时使用', type: 'chat' },
  { value: 'vlm', label: '视觉', hint: '视觉理解场景', type: 'vllm' },
  { value: 'asr', label: '语音', hint: '语音识别场景', type: 'asr' },
  { value: 'rerank', label: '重排', hint: '检索重排场景', type: 'rerank' },
]

function formatTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value || '—'
  return date.toLocaleString('zh-CN', { hour12: false })
}

function modelsFor(modelType: ModelType) {
  return models.value.filter((item) => item.type === modelType)
}

function apply(item: SystemSettings) {
  settings.value = item
  baseURL.value = item.base_url
  apiKey.value = ''
}

async function loadModels() {
  modelHint.value = ''
  try {
    const [modelResponse, bindingResponse] = await Promise.all([listModels(), listBindings()])
    models.value = modelResponse.data ?? []
    const draft: Record<string, string> = {}
    for (const item of bindingResponse.data ?? []) draft[item.role] = item.model_id
    bindingDraft.value = draft
  } catch (error) {
    models.value = []
    modelHint.value = error instanceof ApiError ? error.message : '暂时无法读取模型，请先保存可用的连接'
  }
}

async function load() {
  pageStatus.value = 'loading'
  errorMessage.value = ''
  try {
    const response = await getSystemSettings()
    apply(response.data)
    pageStatus.value = 'ready'
    if (connectionReady.value) await loadModels()
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '无法加载系统配置'
    pageStatus.value = 'error'
  }
}

async function saveConnection() {
  saving.value = true
  try {
    const response = await updateSystemSettings({ base_url: baseURL.value.trim(), api_key: apiKey.value })
    apply(response.data)
    feedback.success(response.data.reachable ? '连接已保存' : '已保存，但暂时无法连通 WeKnora')
    await loadModels()
  } catch {
    // The HTTP client already shows the server message.
  } finally {
    saving.value = false
  }
}

function addModel(role: BindingRole, modelType: ModelType) {
  popupOpen[role] = false
  editor.value?.openCreate(modelType)
}

function editModel(role: BindingRole, model: AiModel) {
  popupOpen[role] = false
  editor.value?.openEdit(model)
}

function deleteModelRow(role: BindingRole, model: AiModel) {
  popupOpen[role] = false
  editor.value?.remove(model)
}

async function saveRole(role: BindingRole) {
  try {
    const response = await saveBinding(role, bindingDraft.value[role])
    if (response.data.warning) feedback.warning(response.data.warning)
    else feedback.success('默认模型已保存')
    await loadModels()
  } catch (error) {
    feedback.error(error instanceof ApiError ? error.message : '保存失败')
  }
}

onMounted(() => {
  void load()
})
</script>

<template>
  <div class="page-grid">
    <t-card :bordered="false" title="WeKnora 连接">
      <PageStatus :status="pageStatus" :error-message="errorMessage" @retry="load">
        <t-form class="settings-form" label-align="top">
          <t-form-item label="服务地址" help="填写主机和端口，不要带 /api/v1">
            <t-input v-model="baseURL" placeholder="http://127.0.0.1:8083" :disabled="!canUpdate" />
          </t-form-item>
          <t-form-item label="API Key">
            <t-input
              v-model="apiKey"
              type="password"
              autocomplete="new-password"
              :disabled="!canUpdate"
              :placeholder="settings?.api_key_configured ? '已配置，留空保持不变' : '请填写 API Key'"
            />
          </t-form-item>
          <p class="hint">
            {{ settings?.api_key_configured ? '密钥已配置' : '尚未配置密钥' }}
            · {{ settings?.reachable ? `已连通 ${settings.version || ''}` : '当前未连通' }}
          </p>
          <p class="hint">最近更新：{{ settings?.updater || '系统' }} · {{ formatTime(settings?.updated_at || '') }}</p>
          <t-button v-if="canUpdate" theme="primary" variant="base" :loading="saving" @click="saveConnection">保存连接</t-button>
        </t-form>
      </PageStatus>
    </t-card>

    <t-card v-if="pageStatus === 'ready' && !connectionReady" :bordered="false" title="模型">
      <t-alert theme="warning" title="请先完成 WeKnora 连接">
        填写并保存上方的服务地址和 API Key 后，才能在下拉框里添加、编辑和删除模型。
      </t-alert>
    </t-card>

    <template v-else-if="connectionReady">
      <ModelsView ref="editor" hide-list @changed="loadModels" />
      <t-card :bordered="false" title="默认模型">
        <p v-if="modelHint" class="hint">{{ modelHint }}</p>
        <div v-for="role in roles" :key="role.value" class="binding">
          <div>
            <strong>{{ role.label }}</strong>
            <p>{{ role.hint }}</p>
          </div>
          <t-select
            v-model="bindingDraft[role.value]"
            v-model:popup-visible="popupOpen[role.value]"
            clearable
            placeholder="选择模型"
            class="model-select"
            :disabled="!canManageModels"
            :popup-props="{ overlayInnerStyle: { width: '420px' } }"
          >
            <t-option v-for="model in modelsFor(role.type)" :key="model.id" :value="model.id" :label="model.display_name || model.name">
              <div class="model-option">
                <span class="model-option__name">{{ model.display_name || model.name }}</span>
                <span v-if="canManageModels" class="model-option__actions" @mousedown.stop @click.stop>
                  <t-button variant="text" :disabled="model.is_builtin" @mousedown.stop @click.stop="editModel(role.value, model)">编辑</t-button>
                  <t-button variant="text" theme="danger" :disabled="model.is_builtin" @mousedown.stop @click.stop="deleteModelRow(role.value, model)">删除</t-button>
                </span>
              </div>
            </t-option>
            <template #empty>
              <p class="hint">还没有这类模型</p>
            </template>
            <template v-if="canManageModels" #panelBottomContent>
              <div class="model-select-add">
                <t-button variant="text" @click="addModel(role.value, role.type)">添加模型</t-button>
              </div>
            </template>
          </t-select>
          <t-button v-if="canManageModels" theme="primary" variant="base" @click="saveRole(role.value)">保存</t-button>
        </div>
        <p class="hint">更换向量模型只影响之后新建的知识库，已有文件的知识库不能更换向量模型。</p>
      </t-card>
    </template>
  </div>
</template>

<style scoped>
.settings-form {
  max-width: 560px;
}

.binding {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  padding: 12px 0;
  border-bottom: 1px solid var(--td-component-border);
}

.binding p,
.hint {
  color: var(--qz-text-secondary);
}

.model-select {
  width: 420px;
}

.model-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  width: 100%;
}

.model-option__name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-option__actions {
  display: flex;
  flex-shrink: 0;
}

.model-select-add {
  padding: 4px 8px;
  border-top: 1px solid var(--td-component-border);
}
</style>
