<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next'
import { RefreshIcon } from 'tdesign-icons-vue-next'
import {
  confirmMemoryItem,
  createMemoryItem,
  deleteMemoryItem,
  fetchMemorySettings,
  fetchMemoryStatus,
  listMemoryItems,
  rejectMemoryItem,
  updateMemoryItem,
  updateMemorySettings,
} from '@/api/memory'
import PageStatus from '@/components/common/PageStatus.vue'
import type { MemoryItem, MemoryKind, MemorySettings, MemoryStatus } from '@/types/memory'
import { ApiError } from '@/utils/http'

const loading = ref(true)
const loadError = ref('')
const status = ref<MemoryStatus | null>(null)
const settings = ref<MemorySettings | null>(null)
const items = ref<MemoryItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const listLoading = ref(false)
const savingSwitch = ref(false)
const dialogVisible = ref(false)
const savingItem = ref(false)
const editing = ref<MemoryItem | null>(null)

const form = reactive({
  type: 'fact' as MemoryKind,
  content: '',
  importance: 3,
})

const kindOptions = [
  { label: '事实', value: 'fact' },
  { label: '偏好', value: 'preference' },
  { label: '画像', value: 'profile' },
  { label: '任务', value: 'task' },
  { label: '兴趣', value: 'interest' },
]

const workspaceLabel = computed(() => {
  if (!status.value || status.value.workspace_enabled === null || status.value.workspace_enabled === undefined) {
    return '未知'
  }
  return status.value.workspace_enabled ? '已启用' : '未启用'
})

const personalSwitchLabel = computed(() => {
  if (!settings.value) return '—'
  return settings.value.user_enabled ? '已开启' : '已关闭'
})

const effectiveLabel = computed(() => {
  if (!settings.value) return '—'
  return settings.value.effective ? '生效中' : '未生效'
})

function kindLabel(kind: string) {
  return kindOptions.find((item) => item.value === kind)?.label || kind
}

function formatTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value || '—'
  return date.toLocaleString('zh-CN', { hour12: false })
}

async function loadList() {
  if (!status.value?.ready) return
  listLoading.value = true
  try {
    const response = await listMemoryItems({ page: page.value, page_size: pageSize })
    items.value = response.data.items ?? []
    total.value = response.data.total ?? items.value.length
  } catch (error) {
    void MessagePlugin.error(error instanceof ApiError ? error.message : '加载记忆列表失败')
  } finally {
    listLoading.value = false
  }
}

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    const statusRes = await fetchMemoryStatus()
    status.value = statusRes.data
    settings.value = null
    items.value = []
    total.value = 0
    if (statusRes.data.ready) {
      const settingsRes = await fetchMemorySettings()
      settings.value = settingsRes.data
      await loadList()
    }
  } catch (error) {
    loadError.value = error instanceof ApiError ? error.message : '加载记忆状态失败'
    status.value = null
  } finally {
    loading.value = false
  }
}

async function onToggle(value: boolean) {
  savingSwitch.value = true
  try {
    const response = await updateMemorySettings(value)
    settings.value = response.data
    void MessagePlugin.success(value ? '已开启个人记忆' : '已关闭个人记忆')
    await loadList()
  } catch (error) {
    void MessagePlugin.error(error instanceof ApiError ? error.message : '更新开关失败')
  } finally {
    savingSwitch.value = false
  }
}

function openCreate() {
  editing.value = null
  form.type = 'fact'
  form.content = ''
  form.importance = 3
  dialogVisible.value = true
}

function openEdit(row: MemoryItem) {
  editing.value = row
  form.type = (row.kind || 'fact') as MemoryKind
  form.content = row.content
  form.importance = row.importance || 3
  dialogVisible.value = true
}

async function saveItem() {
  const content = form.content.trim()
  if (!content) {
    void MessagePlugin.warning('请填写记忆内容')
    return
  }
  savingItem.value = true
  try {
    const body = { type: form.type, content, importance: form.importance }
    if (editing.value) {
      await updateMemoryItem(editing.value.id, body)
      void MessagePlugin.success('已更新')
    } else {
      await createMemoryItem(body)
      void MessagePlugin.success('已创建')
    }
    dialogVisible.value = false
    await Promise.all([refreshSettings(), loadList()])
  } catch (error) {
    void MessagePlugin.error(error instanceof ApiError ? error.message : '保存失败')
  } finally {
    savingItem.value = false
  }
}

async function refreshSettings() {
  try {
    const response = await fetchMemorySettings()
    settings.value = response.data
  } catch {
    // ignore; list already reflects state
  }
}

function confirmDelete(row: MemoryItem) {
  const dialog = DialogPlugin.confirm({
    header: '删除记忆',
    body: '确定删除这条记忆吗？删除后不可恢复。',
    confirmBtn: { content: '删除', theme: 'danger' },
    onConfirm: async () => {
      try {
        await deleteMemoryItem(row.id)
        void MessagePlugin.success('已删除')
        dialog.hide()
        await Promise.all([refreshSettings(), loadList()])
      } catch (error) {
        void MessagePlugin.error(error instanceof ApiError ? error.message : '删除失败')
      }
    },
  })
}

async function onConfirm(row: MemoryItem) {
  try {
    await confirmMemoryItem(row.id)
    void MessagePlugin.success('已确认')
    await loadList()
  } catch (error) {
    void MessagePlugin.error(error instanceof ApiError ? error.message : '确认失败')
  }
}

async function onReject(row: MemoryItem) {
  try {
    await rejectMemoryItem(row.id)
    void MessagePlugin.success('已拒绝')
    await loadList()
  } catch (error) {
    void MessagePlugin.error(error instanceof ApiError ? error.message : '拒绝失败')
  }
}

function onPageChange(next: { current: number }) {
  page.value = next.current
  void loadList()
}

onMounted(() => {
  void load()
})
</script>

<template>
  <div class="memory-page">
    <header class="memory-page__header">
      <div>
        <h1>我的长期记忆</h1>
        <p>个人记忆绑定当前登录身份，由服务端注入 WeKnora 终端用户令牌；前端不传 user_id。</p>
      </div>
      <t-button theme="default" variant="outline" :loading="loading" @click="load">
        <template #icon><RefreshIcon /></template>
        重新加载
      </t-button>
    </header>

    <PageStatus v-if="loading" title="正在加载记忆状态" kind="loading" />
    <PageStatus v-else-if="loadError" title="无法加载记忆状态" :description="loadError" kind="error">
      <t-button theme="primary" variant="outline" @click="load">重试</t-button>
    </PageStatus>

    <template v-else-if="status">
      <section class="memory-status" aria-label="记忆状态">
        <div class="memory-status__item">
          <span class="memory-status__label">空间级记忆开关</span>
          <strong>{{ workspaceLabel }}</strong>
        </div>
        <div class="memory-status__item">
          <span class="memory-status__label">个人记忆开关</span>
          <strong :class="settings?.user_enabled ? 'ok' : ''">{{ personalSwitchLabel }}</strong>
        </div>
        <div class="memory-status__item">
          <span class="memory-status__label">实际是否生效</span>
          <strong>{{ effectiveLabel }}</strong>
        </div>
        <div class="memory-status__item">
          <span class="memory-status__label">个人记忆能力</span>
          <strong :class="status.ready ? 'ok' : 'blocked'">
            {{ status.ready ? '已开放' : '未开放' }}
          </strong>
        </div>
        <div class="memory-status__item">
          <span class="memory-status__label">隔离复测</span>
          <strong>{{ status.verified ? '已复测' : '未复测' }}</strong>
          <span v-if="status.retested_at" class="memory-status__meta">（{{ status.retested_at }}）</span>
        </div>
        <div v-if="settings" class="memory-status__item">
          <span class="memory-status__label">当前有效记忆数</span>
          <strong>{{ settings.item_count }}{{ settings.max_items ? ` / ${settings.max_items}` : '' }}</strong>
        </div>
      </section>

      <p v-if="status.workspace_enabled === false" class="memory-banner memory-banner--info">
        当前企业尚未启用长期记忆，个人设置不会改变企业级配置。
      </p>

      <section v-if="!status.ready" class="memory-unavailable">
        <h2>个人记忆管理暂不可用</h2>
        <p class="memory-unavailable__reason">{{ status.reason }}</p>
      </section>

      <template v-else>
        <section class="memory-toolbar">
          <div class="memory-toolbar__switch">
            <span>个人记忆</span>
            <t-switch
              :value="Boolean(settings?.user_enabled)"
              :loading="savingSwitch"
              :disabled="!settings"
              @change="(v: boolean) => onToggle(v)"
            />
          </div>
          <t-button theme="primary" :disabled="!settings?.user_enabled" @click="openCreate">新增记忆</t-button>
        </section>

        <t-table
          row-key="id"
          :data="items"
          :loading="listLoading"
          :pagination="{ current: page, pageSize, total, onChange: onPageChange }"
          empty="暂无记忆"
        >
          <t-table-column title="类型" col-key="kind" width="100">
            <template #default="{ row }">{{ kindLabel(row.kind) }}</template>
          </t-table-column>
          <t-table-column title="内容" col-key="content" ellipsis />
          <t-table-column title="状态" col-key="status" width="100" />
          <t-table-column title="重要度" col-key="importance" width="80" />
          <t-table-column title="更新时间" col-key="updated_at" width="170">
            <template #default="{ row }">{{ formatTime(row.updated_at) }}</template>
          </t-table-column>
          <t-table-column title="操作" col-key="ops" width="220">
            <template #default="{ row }">
              <t-space break-line>
                <t-button size="small" variant="text" @click="openEdit(row)">编辑</t-button>
                <t-button
                  v-if="row.status === 'pending'"
                  size="small"
                  variant="text"
                  theme="primary"
                  @click="onConfirm(row)"
                >
                  确认
                </t-button>
                <t-button
                  v-if="row.status === 'pending'"
                  size="small"
                  variant="text"
                  @click="onReject(row)"
                >
                  拒绝
                </t-button>
                <t-button size="small" variant="text" theme="danger" @click="confirmDelete(row)">删除</t-button>
              </t-space>
            </template>
          </t-table-column>
        </t-table>
      </template>
    </template>

    <t-dialog
      v-model:visible="dialogVisible"
      :header="editing ? '编辑记忆' : '新增记忆'"
      :confirm-btn="{ content: '保存', loading: savingItem }"
      @confirm="saveItem"
    >
      <t-form label-align="top">
        <t-form-item label="类型">
          <t-select v-model="form.type" :options="kindOptions" />
        </t-form-item>
        <t-form-item label="内容">
          <t-textarea v-model="form.content" :maxlength="300" :autosize="{ minRows: 3, maxRows: 6 }" />
        </t-form-item>
        <t-form-item label="重要度（1–5）">
          <t-input-number v-model="form.importance" :min="1" :max="5" />
        </t-form-item>
      </t-form>
    </t-dialog>
  </div>
</template>

<style scoped>
.memory-page {
  max-width: 960px;
  margin: 0 auto;
}

.memory-page__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 24px;
}

.memory-page__header h1 {
  margin: 0 0 8px;
  font-size: 24px;
  font-weight: 700;
}

.memory-page__header p {
  margin: 0;
  font-size: 14px;
  line-height: 1.5;
  color: var(--wb-text-secondary);
}

.memory-status {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 12px;
  padding: 16px;
  border: 1px solid var(--wb-border);
  border-radius: 12px;
  background: var(--wb-surface);
  margin-bottom: 16px;
}

.memory-status__item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.memory-status__label {
  font-size: 12px;
  color: var(--wb-text-secondary);
}

.memory-status__item strong {
  font-size: 15px;
  font-weight: 600;
  color: var(--wb-text);
}

.memory-status__item strong.blocked {
  color: var(--td-warning-color, #e37318);
}

.memory-status__item strong.ok {
  color: var(--wb-primary);
}

.memory-status__meta {
  font-size: 12px;
  color: var(--wb-text-secondary);
}

.memory-banner {
  margin: 0 0 16px;
  padding: 12px 14px;
  border-radius: 8px;
  font-size: 14px;
  line-height: 1.5;
}

.memory-banner--info {
  border: 1px solid var(--wb-border);
  background: var(--wb-sidebar-bg);
  color: var(--wb-text-secondary);
}

.memory-unavailable {
  padding: 20px;
  border: 1px solid var(--wb-border);
  border-radius: 12px;
  background: var(--wb-surface);
}

.memory-unavailable h2 {
  margin: 0 0 10px;
  font-size: 18px;
  font-weight: 600;
}

.memory-unavailable__reason {
  margin: 0;
  font-size: 14px;
  line-height: 1.6;
  color: var(--wb-text);
}

.memory-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.memory-toolbar__switch {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 14px;
}

@media (max-width: 640px) {
  .memory-page__header,
  .memory-toolbar {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
