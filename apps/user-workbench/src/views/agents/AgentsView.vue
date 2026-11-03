<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { DialogPlugin } from 'tdesign-vue-next'
import PageStatus from '@/components/common/PageStatus.vue'
import {
  copyAgent,
  createAgent,
  deleteAgent,
  getCatalogOptions,
  listAgents,
  listMCP,
  updateAgent,
  type CatalogItem,
  type CatalogOptions,
  type CatalogScope,
} from '@/api/catalog'
import { listKnowledgeBases } from '@/api/knowledge'
import { feedback } from '@/utils/feedback'
import { useAuthStore } from '@/stores/auth'

type AgentConfigForm = {
  agent_mode: 'quick-answer' | 'smart-reasoning'
  agent_type: '' | 'custom' | 'rag-qa' | 'wiki-qa' | 'hybrid-rag-wiki'
  system_prompt: string
  kb_selection_mode: 'none' | 'selected'
  knowledge_base_ids: string[]
  mcp_selection_mode: 'none' | 'all' | 'selected'
  mcp_ids: string[]
  web_search_enabled: boolean
  multi_turn_enabled: boolean
  memory_enabled: boolean
  retrieve_kb_only_when_mentioned: boolean
  temperature: number
  max_iterations: number
}

const auth = useAuthStore()
const canManage = computed(() => auth.hasPermission('agent:manage'))
const pageStatus = ref<'loading' | 'error' | 'ready'>('loading')
const items = ref<CatalogItem[]>([])
const editorVisible = ref(false)
const editingID = ref('')
const saving = ref(false)
const kbOptions = ref<{ label: string; value: string }[]>([])
const mcpOptions = ref<{ label: string; value: string }[]>([])
const options = ref<CatalogOptions>({
  tenants: [],
  organizations: [],
  roles: [],
  users: [],
  can_manage_agents: false,
  can_manage_toolbox: false,
})

function defaultConfig(): AgentConfigForm {
  return {
    agent_mode: 'quick-answer',
    agent_type: '',
    system_prompt: '',
    kb_selection_mode: 'none',
    knowledge_base_ids: [],
    mcp_selection_mode: 'none',
    mcp_ids: [],
    web_search_enabled: false,
    multi_turn_enabled: true,
    memory_enabled: true,
    retrieve_kb_only_when_mentioned: false,
    temperature: 0.7,
    max_iterations: 10,
  }
}

function parseConfig(raw: CatalogItem['config']): AgentConfigForm {
  const base = defaultConfig()
  if (!raw || typeof raw !== 'object') return base
  const cfg = raw as Record<string, unknown>
  const mode = cfg.agent_mode === 'smart-reasoning' ? 'smart-reasoning' : 'quick-answer'
  const agentType =
    cfg.agent_type === 'rag-qa' ||
    cfg.agent_type === 'wiki-qa' ||
    cfg.agent_type === 'hybrid-rag-wiki' ||
    cfg.agent_type === 'custom'
      ? cfg.agent_type
      : mode === 'smart-reasoning'
        ? 'custom'
        : ''
  return {
    agent_mode: mode,
    agent_type: agentType,
    system_prompt: typeof cfg.system_prompt === 'string' ? cfg.system_prompt : '',
    kb_selection_mode: cfg.kb_selection_mode === 'selected' ? 'selected' : 'none',
    knowledge_base_ids: Array.isArray(cfg.knowledge_base_ids)
      ? cfg.knowledge_base_ids.filter((item): item is string => typeof item === 'string')
      : [],
    mcp_selection_mode:
      cfg.mcp_selection_mode === 'selected'
        ? 'selected'
        : cfg.mcp_selection_mode === 'all'
          ? 'all'
          : 'none',
    mcp_ids: Array.isArray(cfg.mcp_ids)
      ? cfg.mcp_ids.filter((item): item is string => typeof item === 'string')
      : [],
    web_search_enabled: Boolean(cfg.web_search_enabled),
    multi_turn_enabled: cfg.multi_turn_enabled !== false,
    memory_enabled: cfg.memory_enabled !== false,
    retrieve_kb_only_when_mentioned: Boolean(cfg.retrieve_kb_only_when_mentioned),
    temperature: typeof cfg.temperature === 'number' ? cfg.temperature : 0.7,
    max_iterations: typeof cfg.max_iterations === 'number' ? cfg.max_iterations : 10,
  }
}

const form = ref({
  name: '',
  description: '',
  scope_type: 'personal' as CatalogScope,
  scope_id: '',
  tenant_ids: [] as string[],
  share_user_ids: [] as string[],
  config: defaultConfig(),
})

const scopeLabel: Record<string, string> = {
  personal: '个人',
  tenant: '租户',
  org_node: '组织',
  role: '角色',
}

const modeOptions = [
  { label: '快速问答（RAG）', value: 'quick-answer' },
  { label: '智能推理（ReAct / 工具）', value: 'smart-reasoning' },
]

const typeOptions = [
  { label: '自定义', value: 'custom' },
  { label: '文档问答', value: 'rag-qa' },
  { label: 'Wiki 检索', value: 'wiki-qa' },
  { label: 'Wiki + RAG 混合', value: 'hybrid-rag-wiki' },
]

const kbSelectionOptions = [
  { label: '不使用', value: 'none' },
  { label: '指定知识库', value: 'selected' },
]

const mcpSelectionOptions = [
  { label: '不使用', value: 'none' },
  { label: '全部可见工具', value: 'all' },
  { label: '指定工具', value: 'selected' },
]

const columns = [
  { colKey: 'name', title: '名称' },
  { colKey: 'scope_type', title: '归属' },
  { colKey: 'mode', title: '模式', width: 120 },
  { colKey: 'description', title: '描述' },
  { colKey: 'creator', title: '创建人' },
  { colKey: 'actions', title: '操作', width: 220 },
]

const scopeChoices = computed(() => {
  if (form.value.scope_type === 'org_node') return options.value.organizations
  if (form.value.scope_type === 'role') return options.value.roles
  if (form.value.scope_type === 'tenant') return options.value.tenants
  return []
})

const dialogHeader = computed(() => (editingID.value ? '编辑智能体' : '新建智能体'))
const isSmart = computed(() => form.value.config.agent_mode === 'smart-reasoning')

watch(
  () => form.value.config.agent_mode,
  (mode) => {
    if (mode === 'quick-answer') {
      form.value.config.agent_type = ''
      form.value.config.mcp_selection_mode = 'none'
      form.value.config.mcp_ids = []
    } else if (!form.value.config.agent_type) {
      form.value.config.agent_type = 'custom'
    }
  },
)

watch(
  () => form.value.config.kb_selection_mode,
  (mode) => {
    if (mode === 'none') form.value.config.knowledge_base_ids = []
  },
)

watch(
  () => form.value.config.mcp_selection_mode,
  (mode) => {
    if (mode === 'none') form.value.config.mcp_ids = []
  },
)

function modeLabel(row: CatalogItem) {
  const cfg = row.config as Record<string, unknown> | undefined
  return cfg?.agent_mode === 'smart-reasoning' ? '智能推理' : '快速问答'
}

async function load() {
  pageStatus.value = items.value.length ? 'ready' : 'loading'
  try {
    const response = await listAgents()
    items.value = response.data ?? []
    pageStatus.value = 'ready'
  } catch {
    if (!items.value.length) pageStatus.value = 'error'
  }
}

async function ensureOptions() {
  const [catalogOpts, kbPage, mcpList] = await Promise.all([
    getCatalogOptions(),
    listKnowledgeBases({ page: 1, page_size: 100 }),
    listMCP(),
  ])
  options.value = catalogOpts.data
  kbOptions.value = (kbPage.data.items ?? []).map((item) => ({ label: item.name, value: item.id }))
  mcpOptions.value = (mcpList.data ?? []).map((item) => ({ label: item.name, value: item.id }))
}

async function openCreate() {
  await ensureOptions()
  editingID.value = ''
  form.value = {
    name: '',
    description: '',
    scope_type: 'personal',
    scope_id: '',
    tenant_ids: options.value.tenants.some((item) => item.id === auth.tenantId)
      ? [auth.tenantId]
      : options.value.tenants[0]
        ? [options.value.tenants[0].id]
        : [],
    share_user_ids: [],
    config: defaultConfig(),
  }
  editorVisible.value = true
}

async function openEdit(row: CatalogItem) {
  await ensureOptions()
  editingID.value = row.id
  form.value = {
    name: row.name,
    description: row.description,
    scope_type: row.scope_type,
    scope_id: row.scope_id,
    tenant_ids: [...(row.tenant_ids ?? [])],
    share_user_ids: [...(row.share_user_ids ?? [])],
    config: parseConfig(row.config),
  }
  editorVisible.value = true
}

async function save() {
  if (!form.value.name.trim()) {
    feedback.warning('请填写名称')
    return
  }
  if (
    form.value.config.kb_selection_mode === 'selected' &&
    form.value.config.knowledge_base_ids.length === 0
  ) {
    feedback.warning('请至少选择一个知识库')
    return
  }
  if (
    form.value.config.mcp_selection_mode === 'selected' &&
    form.value.config.mcp_ids.length === 0
  ) {
    feedback.warning('请至少选择一个 MCP 工具')
    return
  }
  const configPayload = {
    ...form.value.config,
    agent_type: form.value.config.agent_mode === 'smart-reasoning' ? form.value.config.agent_type || 'custom' : undefined,
    knowledge_base_ids:
      form.value.config.kb_selection_mode === 'selected' ? form.value.config.knowledge_base_ids : [],
    mcp_ids:
      form.value.config.mcp_selection_mode === 'selected' ? form.value.config.mcp_ids : [],
  }
  saving.value = true
  try {
    if (editingID.value) {
      await updateAgent(editingID.value, {
        name: form.value.name,
        description: form.value.description,
        share_user_ids: form.value.scope_type === 'personal' ? form.value.share_user_ids : undefined,
        config: configPayload,
      })
      feedback.success('智能体已更新')
    } else {
      await createAgent({
        name: form.value.name,
        description: form.value.description,
        scope_type: form.value.scope_type,
        scope_id: form.value.scope_type === 'personal' ? undefined : form.value.scope_id,
        tenant_ids: form.value.tenant_ids,
        share_user_ids: form.value.scope_type === 'personal' ? form.value.share_user_ids : undefined,
        config: configPayload,
      })
      feedback.success('智能体已创建')
    }
    editorVisible.value = false
    await load()
  } finally {
    saving.value = false
  }
}

async function onCopy(row: CatalogItem) {
  await copyAgent(row.id, {})
  feedback.success('已复制为个人智能体')
  await load()
}

function remove(row: CatalogItem) {
  const dialog = DialogPlugin.confirm({
    header: '删除智能体',
    body: `确定删除「${row.name}」吗？`,
    onConfirm: async () => {
      await deleteAgent(row.id)
      feedback.success('已删除')
      dialog.hide()
      await load()
    },
  })
}

onMounted(load)
</script>

<template>
  <div>
    <PageStatus
      v-if="pageStatus === 'loading' || pageStatus === 'error'"
      :kind="pageStatus === 'loading' ? 'loading' : 'error'"
      :title="pageStatus === 'loading' ? '加载中' : '加载失败'"
      :description="pageStatus === 'error' ? '无法加载智能体列表，请稍后重试。' : ''"
    >
      <t-button v-if="pageStatus === 'error'" variant="outline" @click="load">重试</t-button>
    </PageStatus>
    <t-card v-else :bordered="false">
      <div class="toolbar">
        <t-button theme="primary" @click="openCreate">新建智能体</t-button>
      </div>
      <t-table row-key="id" :data="items" :columns="columns">
        <template #scope_type="{ row }">{{ scopeLabel[row.scope_type] ?? row.scope_type }}</template>
        <template #mode="{ row }">{{ modeLabel(row) }}</template>
        <template #actions="{ row }">
          <t-button v-if="row.writable || canManage" variant="text" @click="openEdit(row)">编辑</t-button>
          <t-button variant="text" @click="onCopy(row)">复制</t-button>
          <t-button
            v-if="row.writable || canManage"
            theme="danger"
            variant="text"
            @click="remove(row)"
          >
            删除
          </t-button>
        </template>
      </t-table>
    </t-card>

    <t-dialog
      v-model:visible="editorVisible"
      :header="dialogHeader"
      width="760px"
      :confirm-btn="{ content: editingID ? '保存' : '创建', loading: saving }"
      @confirm="save"
    >
      <t-form label-align="top" class="agent-form">
        <t-form-item label="名称"><t-input v-model="form.name" maxlength="100" /></t-form-item>
        <t-form-item label="描述"><t-textarea v-model="form.description" maxlength="500" /></t-form-item>

        <template v-if="!editingID">
          <t-form-item label="租户">
            <t-select
              v-model="form.tenant_ids"
              multiple
              :options="options.tenants.map((item) => ({ label: item.name, value: item.id }))"
            />
          </t-form-item>
          <t-form-item label="归属">
            <t-radio-group v-model="form.scope_type">
              <t-radio value="personal">个人</t-radio>
              <t-radio v-if="canManage" value="tenant">租户</t-radio>
              <t-radio v-if="canManage" value="org_node">组织</t-radio>
              <t-radio v-if="canManage" value="role">角色</t-radio>
            </t-radio-group>
          </t-form-item>
          <t-form-item v-if="form.scope_type !== 'personal'" label="归属对象">
            <t-select
              v-model="form.scope_id"
              :options="scopeChoices.map((item) => ({ label: item.name, value: item.id }))"
            />
          </t-form-item>
        </template>
        <t-form-item v-if="form.scope_type === 'personal'" label="分享给">
          <t-select
            v-model="form.share_user_ids"
            multiple
            filterable
            :options="options.users.map((item) => ({ label: item.name, value: item.id }))"
          />
        </t-form-item>

        <div class="agent-form__section">运行配置</div>
        <t-form-item label="模式">
          <t-select v-model="form.config.agent_mode" :options="modeOptions" />
        </t-form-item>
        <t-form-item v-if="isSmart" label="推理类型">
          <t-select v-model="form.config.agent_type" :options="typeOptions" />
          <p class="agent-form__hint">Wiki 检索请选「Wiki 检索」或「Wiki + RAG 混合」。</p>
        </t-form-item>
        <t-form-item label="系统提示词">
          <t-textarea
            v-model="form.config.system_prompt"
            :autosize="{ minRows: 4, maxRows: 12 }"
            placeholder="定义角色、回答风格与约束…"
          />
        </t-form-item>

        <t-form-item label="知识库">
          <t-radio-group v-model="form.config.kb_selection_mode">
            <t-radio v-for="opt in kbSelectionOptions" :key="opt.value" :value="opt.value">
              {{ opt.label }}
            </t-radio>
          </t-radio-group>
        </t-form-item>
        <t-form-item v-if="form.config.kb_selection_mode === 'selected'" label="绑定知识库">
          <t-select
            v-model="form.config.knowledge_base_ids"
            multiple
            filterable
            placeholder="选择可见知识库"
            :options="kbOptions"
          />
        </t-form-item>
        <t-form-item label="仅 @ 提及时检索知识库">
          <t-switch v-model="form.config.retrieve_kb_only_when_mentioned" />
        </t-form-item>

        <template v-if="isSmart">
          <t-form-item label="工具（MCP）">
            <t-radio-group v-model="form.config.mcp_selection_mode">
              <t-radio v-for="opt in mcpSelectionOptions" :key="opt.value" :value="opt.value">
                {{ opt.label }}
              </t-radio>
            </t-radio-group>
            <p class="agent-form__hint">
              「全部可见工具」会绑定当前账号可见且已同步的 MCP；新增可见工具后请重新保存智能体。
            </p>
          </t-form-item>
          <t-form-item v-if="form.config.mcp_selection_mode === 'selected'" label="指定工具">
            <t-select
              v-model="form.config.mcp_ids"
              multiple
              filterable
              placeholder="选择可见 MCP 工具"
              :options="mcpOptions"
            />
          </t-form-item>
          <t-form-item label="最大推理轮次">
            <t-input-number v-model="form.config.max_iterations" :min="1" :max="50" />
          </t-form-item>
        </template>
        <p v-else class="agent-form__hint">切换到「智能推理」后可配置 MCP 工具（全部 / 指定）。</p>

        <t-form-item label="联网搜索">
          <t-switch v-model="form.config.web_search_enabled" />
        </t-form-item>
        <t-form-item label="多轮对话">
          <t-switch v-model="form.config.multi_turn_enabled" />
        </t-form-item>
        <t-form-item label="长期记忆">
          <t-switch v-model="form.config.memory_enabled" />
        </t-form-item>
        <t-form-item label="温度">
          <t-input-number v-model="form.config.temperature" :min="0" :max="2" :step="0.1" :decimal-places="1" />
        </t-form-item>
      </t-form>
    </t-dialog>
  </div>
</template>

<style scoped>
.toolbar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}

.agent-form__section {
  margin: 8px 0 4px;
  font-size: 13px;
  font-weight: 600;
  color: var(--td-text-color-secondary);
}

.agent-form__hint {
  margin: 6px 0 0;
  font-size: 12px;
  color: var(--td-text-color-secondary);
}
</style>
