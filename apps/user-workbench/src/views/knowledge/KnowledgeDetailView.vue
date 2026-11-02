<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { DialogPlugin } from 'tdesign-vue-next'
import PageStatus from '@/components/common/PageStatus.vue'
import {
  deleteKnowledgeBase,
  deleteKnowledgeFiles,
  getKnowledgeBase,
  getKnowledgeLimits,
  getKnowledgeOptions,
  listKnowledgeFiles,
  searchWiki,
  shareKnowledgeBase,
  updateKnowledgeBase,
  uploadKnowledgeFiles,
} from '@/api/knowledge'
import { useAuthStore } from '@/stores/auth'
import type {
  KnowledgeBase,
  KnowledgeFile,
  KnowledgeFormOptions,
  KnowledgeLimits,
  UploadResult,
  WikiPage,
} from '@/types/knowledge'
import { feedback } from '@/utils/feedback'
import {
  canManagePersonalKnowledge,
  formatFileSize,
  parseStatusLabel,
  SCOPE_LABEL,
} from '@/utils/knowledge'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const id = computed(() => String(route.params.id || ''))
const pageStatus = ref<'loading' | 'error' | 'ready'>('loading')
const errorMessage = ref('')
const base = ref<KnowledgeBase | null>(null)
const files = ref<KnowledgeFile[]>([])
const limits = ref<KnowledgeLimits | null>(null)
const options = ref<KnowledgeFormOptions | null>(null)
const uploadResults = ref<UploadResult[]>([])
const selected = ref<string[]>([])
const uploading = ref(false)
const saving = ref(false)

const editVisible = ref(false)
const shareVisible = ref(false)
const editForm = ref({ name: '', description: '' })
const shareForm = ref({ share_user_ids: [] as string[] })

const searchQuery = ref('')
const searchLoading = ref(false)
const searchHits = ref<WikiPage[]>([])

let pollTimer = 0
const terminal = new Set(['completed', 'failed', 'duplicate', 'skipped'])

const writable = computed(() =>
  base.value ? canManagePersonalKnowledge(base.value, auth.user?.id) : false,
)
const userOptions = computed(() =>
  (options.value?.users ?? [])
    .filter((item) => item.id !== auth.user?.id)
    .map((item) => ({ label: item.name, value: item.id })),
)
const polling = computed(() => files.value.some((file) => !terminal.has(file.parse_status)))

async function load() {
  pageStatus.value = 'loading'
  errorMessage.value = ''
  try {
    const [baseRes, fileRes, limitRes] = await Promise.all([
      getKnowledgeBase(id.value),
      listKnowledgeFiles(id.value, true),
      getKnowledgeLimits(),
    ])
    base.value = baseRes.data
    files.value = fileRes.data ?? []
    limits.value = limitRes.data
    pageStatus.value = 'ready'
    schedulePoll()
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '加载失败'
    pageStatus.value = 'error'
  }
}

async function refreshFiles() {
  const response = await listKnowledgeFiles(id.value, true)
  files.value = response.data ?? []
  schedulePoll()
}

function schedulePoll() {
  window.clearTimeout(pollTimer)
  if (polling.value) {
    pollTimer = window.setTimeout(() => void refreshFiles(), 5000)
  }
}

async function ensureOptions() {
  if (options.value) return
  const response = await getKnowledgeOptions()
  options.value = response.data
}

function openEdit() {
  if (!base.value) return
  editForm.value = { name: base.value.name, description: base.value.description || '' }
  editVisible.value = true
}

async function openShare() {
  if (!base.value) return
  try {
    await ensureOptions()
    shareForm.value = { share_user_ids: [...(base.value.share_user_ids ?? [])] }
    shareVisible.value = true
  } catch (error) {
    feedback.error(error instanceof Error ? error.message : '加载选项失败')
  }
}

async function saveEdit() {
  if (!base.value) return
  const name = editForm.value.name.trim()
  if (!name) {
    feedback.warning('请填写名称')
    return
  }
  saving.value = true
  try {
    const response = await updateKnowledgeBase(base.value.id, {
      name,
      description: editForm.value.description.trim(),
    })
    base.value = response.data
    feedback.success('已保存')
    editVisible.value = false
  } catch {
    /* toast */
  } finally {
    saving.value = false
  }
}

async function saveShare() {
  if (!base.value) return
  saving.value = true
  try {
    const response = await shareKnowledgeBase(base.value.id, {
      share_user_ids: shareForm.value.share_user_ids,
    })
    base.value = response.data
    feedback.success('分享已更新')
    shareVisible.value = false
  } catch {
    /* toast */
  } finally {
    saving.value = false
  }
}

function confirmDelete() {
  if (!base.value) return
  const name = base.value.name
  const dialog = DialogPlugin.confirm({
    header: '删除个人知识库',
    body: `确认删除「${name}」？`,
    theme: 'danger',
    confirmBtn: '删除',
    onConfirm: async () => {
      try {
        await deleteKnowledgeBase(id.value)
        feedback.success('已删除')
        dialog.destroy()
        await router.replace({ name: 'knowledge' })
      } catch {
        /* toast */
      }
    },
  })
}

async function onUpload(fileList: File[]) {
  if (!writable.value || !fileList.length) return
  uploading.value = true
  uploadResults.value = []
  try {
    const response = await uploadKnowledgeFiles(id.value, fileList)
    uploadResults.value = response.data ?? []
    feedback.success('上传完成')
    await refreshFiles()
  } catch {
    /* toast */
  } finally {
    uploading.value = false
  }
}

async function onDeleteFiles() {
  if (!writable.value || !selected.value.length) return
  const dialog = DialogPlugin.confirm({
    header: '删除文件',
    body: `确认删除选中的 ${selected.value.length} 个文件？`,
    theme: 'danger',
    confirmBtn: '删除',
    onConfirm: async () => {
      try {
        await deleteKnowledgeFiles(id.value, selected.value)
        selected.value = []
        feedback.success('已删除')
        dialog.destroy()
        await refreshFiles()
      } catch {
        /* toast */
      }
    },
  })
}

async function runSearch() {
  const q = searchQuery.value.trim()
  if (!q) {
    feedback.warning('请输入检索词')
    return
  }
  if (!base.value?.wiki_enabled) {
    feedback.warning('该知识库未启用 Wiki，请使用「对话检索」')
    return
  }
  searchLoading.value = true
  try {
    const response = await searchWiki(id.value, q.slice(0, 100), 20)
    searchHits.value = response.data.pages ?? []
    if (!searchHits.value.length) {
      feedback.warning('没有匹配的 Wiki 页面')
    }
  } catch {
    searchHits.value = []
  } finally {
    searchLoading.value = false
  }
}

function openChat() {
  void router.push({ name: 'chat', query: { kb: id.value } })
}

watch(id, () => {
  selected.value = []
  searchHits.value = []
  searchQuery.value = ''
  void load()
})

onMounted(() => {
  void load()
})

onBeforeUnmount(() => {
  window.clearTimeout(pollTimer)
})
</script>

<template>
  <div class="kb-detail">
    <div class="kb-detail__nav">
      <t-button variant="text" theme="primary" @click="router.push({ name: 'knowledge' })">
        ← 返回列表
      </t-button>
    </div>

    <PageStatus v-if="pageStatus === 'loading'" kind="loading" title="加载中…" />
    <PageStatus
      v-else-if="pageStatus === 'error'"
      kind="error"
      title="无法打开知识库"
      :description="errorMessage || '不存在或无权访问'"
    >
      <t-button size="small" @click="load">重试</t-button>
    </PageStatus>

    <template v-else-if="base">
      <header class="kb-detail__header">
        <div>
          <div class="kb-detail__title-row">
            <h2>{{ base.name }}</h2>
            <span class="kb-detail__tag">{{ SCOPE_LABEL[base.scope_type] }}</span>
            <span v-if="!writable" class="kb-detail__tag kb-detail__tag--muted">只读</span>
          </div>
          <p class="kb-detail__desc">{{ base.description || '暂无描述' }}</p>
          <p class="kb-detail__meta">
            <span v-if="base.scope_name">归属 {{ base.scope_name }}</span>
            <span v-if="base.creator"> · 创建人 {{ base.creator }}</span>
            <span v-if="base.wiki_enabled"> · Wiki 已启用</span>
          </p>
        </div>
        <div class="kb-detail__actions">
          <t-button theme="primary" variant="outline" @click="openChat">对话检索</t-button>
          <template v-if="writable">
            <t-button variant="outline" @click="openEdit">修改</t-button>
            <t-button variant="outline" @click="openShare">分享</t-button>
            <t-button theme="danger" variant="outline" @click="confirmDelete">删除</t-button>
          </template>
        </div>
      </header>

      <section class="kb-detail__panel">
        <div class="kb-detail__panel-head">
          <h3>文档</h3>
          <div v-if="writable" class="kb-detail__panel-actions">
            <label class="kb-detail__upload" :class="{ 'kb-detail__upload--busy': uploading }">
              <input
                type="file"
                multiple
                class="kb-detail__file-input"
                :disabled="uploading"
                @change="
                  (event) => {
                    const input = event.target as HTMLInputElement
                    const list = [...(input.files ?? [])]
                    input.value = ''
                    void onUpload(list)
                  }
                "
              />
              {{ uploading ? '上传中…' : '上传文件' }}
            </label>
            <t-button
              size="small"
              theme="danger"
              variant="outline"
              :disabled="!selected.length"
              @click="onDeleteFiles"
            >
              删除选中
            </t-button>
          </div>
        </div>
        <p v-if="limits" class="kb-detail__hint">
          单文件 ≤ {{ limits.max_file_mb }}MB，扩展名：{{ limits.allowed_exts.join(', ') || '未配置' }}
        </p>
        <t-table
          row-key="id"
          :data="files"
          :columns="[
            ...(writable
              ? [{ colKey: 'row-select', type: 'multiple' as const, width: 46 }]
              : []),
            { colKey: 'file_name', title: '文件名', ellipsis: true },
            { colKey: 'file_size', title: '大小', width: 100 },
            { colKey: 'parse_status', title: '状态', width: 100 },
          ]"
          :selected-row-keys="selected"
          hover
          size="small"
          @select-change="(keys: string[]) => (selected = keys)"
        >
          <template #file_size="{ row }">{{ formatFileSize(row.file_size) }}</template>
          <template #parse_status="{ row }">
            <span :title="row.error_msg || ''">{{ parseStatusLabel(row.parse_status) }}</span>
          </template>
        </t-table>
        <ul v-if="uploadResults.length" class="kb-detail__upload-log">
          <li v-for="(item, index) in uploadResults" :key="`${item.file_name}-${index}`">
            {{ item.file_name }} · {{ item.status }}
            <span v-if="item.error">（{{ item.error }}）</span>
          </li>
        </ul>
      </section>

      <section class="kb-detail__panel">
        <div class="kb-detail__panel-head">
          <h3>检索</h3>
        </div>
        <p class="kb-detail__hint">
          企业知识库与共享库可在此查看 Wiki 检索结果；完整问答请使用「对话检索」。
        </p>
        <div class="kb-detail__search">
          <t-input
            v-model="searchQuery"
            clearable
            placeholder="输入关键词检索 Wiki"
            @enter="runSearch"
          />
          <t-button :loading="searchLoading" theme="primary" @click="runSearch">检索</t-button>
        </div>
        <div v-if="searchHits.length" class="kb-detail__hits">
          <article v-for="hit in searchHits" :key="hit.id || hit.slug" class="kb-hit">
            <h4>{{ hit.title || hit.slug }}</h4>
            <p>{{ hit.summary || '无摘要' }}</p>
          </article>
        </div>
      </section>
    </template>

    <t-dialog
      v-model:visible="editVisible"
      header="修改知识库"
      :confirm-btn="{ content: '保存', loading: saving }"
      @confirm="saveEdit"
    >
      <t-form label-width="72px">
        <t-form-item label="名称" required>
          <t-input v-model="editForm.name" maxlength="100" />
        </t-form-item>
        <t-form-item label="描述">
          <t-textarea v-model="editForm.description" maxlength="500" :autosize="{ minRows: 2, maxRows: 4 }" />
        </t-form-item>
      </t-form>
    </t-dialog>

    <t-dialog
      v-model:visible="shareVisible"
      header="分享个人知识库"
      :confirm-btn="{ content: '保存', loading: saving }"
      @confirm="saveShare"
    >
      <t-select
        v-model="shareForm.share_user_ids"
        multiple
        filterable
        clearable
        placeholder="选择用户"
        :options="userOptions"
      />
    </t-dialog>
  </div>
</template>

<style scoped>
.kb-detail {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-width: 960px;
}

.kb-detail__header {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  padding: 18px 20px;
  background: var(--wb-surface);
  border: 1px solid var(--wb-border);
  border-radius: var(--wb-radius);
}

.kb-detail__title-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.kb-detail__title-row h2 {
  margin: 0;
  font-size: 20px;
}

.kb-detail__tag {
  padding: 2px 8px;
  border-radius: 4px;
  background: #ecf2fe;
  color: var(--wb-primary);
  font-size: 12px;
  font-weight: 600;
}

.kb-detail__tag--muted {
  background: #f3f3f3;
  color: var(--wb-text-secondary);
}

.kb-detail__desc,
.kb-detail__meta,
.kb-detail__hint {
  margin: 8px 0 0;
  color: var(--wb-text-secondary);
  font-size: 13px;
}

.kb-detail__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: flex-start;
}

.kb-detail__panel {
  padding: 16px 18px;
  background: var(--wb-surface);
  border: 1px solid var(--wb-border);
  border-radius: var(--wb-radius);
}

.kb-detail__panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
}

.kb-detail__panel-head h3 {
  margin: 0;
  font-size: 15px;
}

.kb-detail__panel-actions {
  display: flex;
  gap: 8px;
}

.kb-detail__upload {
  position: relative;
  display: inline-flex;
  align-items: center;
  height: 28px;
  padding: 0 12px;
  border: 1px solid var(--wb-border);
  border-radius: 6px;
  background: #fff;
  color: var(--wb-text);
  font-size: 12px;
  cursor: pointer;
}

.kb-detail__upload--busy {
  opacity: 0.7;
  pointer-events: none;
}

.kb-detail__file-input {
  position: absolute;
  inset: 0;
  opacity: 0;
  cursor: pointer;
}

.kb-detail__upload-log {
  margin: 12px 0 0;
  padding-left: 18px;
  color: var(--wb-text-secondary);
  font-size: 12px;
}

.kb-detail__search {
  display: flex;
  gap: 8px;
  margin-top: 8px;
}

.kb-detail__hits {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-top: 14px;
}

.kb-hit {
  padding: 12px;
  border: 1px solid var(--wb-border);
  border-radius: 8px;
}

.kb-hit h4 {
  margin: 0 0 6px;
  font-size: 14px;
}

.kb-hit p {
  margin: 0;
  color: var(--wb-text-secondary);
  font-size: 13px;
  line-height: 1.45;
}

@media (max-width: 720px) {
  .kb-detail__header {
    flex-direction: column;
  }
}
</style>
