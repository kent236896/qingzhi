<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import DOMPurify from 'dompurify'
import PageStatus from '@/components/PageStatus.vue'
import KnowledgeGraph from '@/views/KnowledgeGraph.vue'
import {
  deleteKnowledgeFiles,
  getKnowledgeBase,
  getKnowledgeLimits,
  getWikiGraph,
  getWikiPage,
  getWikiStatus,
  listKnowledgeFiles,
  reparseKnowledgeFiles,
  searchWiki,
  uploadKnowledgeFiles,
  uploadKnowledgeZip,
  type KnowledgeBase,
  type KnowledgeFile,
  type KnowledgeLimits,
  type UploadResult,
  type WikiGraph,
  type WikiPage,
  type WikiStatus,
} from '@/api/knowledge'
import { feedback } from '@/utils/feedback'

const route = useRoute()
const id = computed(() => String(route.params.id))
const pageStatus = ref<'loading' | 'error' | 'ready'>('loading')
const base = ref<KnowledgeBase | null>(null)
const files = ref<KnowledgeFile[]>([])
const selected = ref<string[]>([])
const limits = ref<KnowledgeLimits | null>(null)
const results = ref<UploadResult[]>([])
const failedFiles = ref<File[]>([])
const wiki = ref<WikiStatus | null>(null)
const graph = ref<WikiGraph | null>(null)
const page = ref<WikiPage | null>(null)
const drawerVisible = ref(false)
const keyword = ref('')
let timer = 0

const terminal = new Set(['completed', 'failed', 'duplicate', 'skipped'])
const polling = computed(() => files.value.some((file) => !terminal.has(file.parse_status)))

function safeHtml(content: string) {
  const escaped = content.replace(/[&<>"']/g, (char) => {
    return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[char] ?? char
  })
  return DOMPurify.sanitize(escaped.replace(/\n/g, '<br>'))
}

async function load() {
  try {
    const [baseResponse, fileResponse, limitResponse] = await Promise.all([
      getKnowledgeBase(id.value),
      listKnowledgeFiles(id.value, true),
      getKnowledgeLimits(),
    ])
    base.value = baseResponse.data
    files.value = fileResponse.data ?? []
    limits.value = limitResponse.data
    pageStatus.value = 'ready'
    if (base.value.wiki_enabled) {
      await refreshWiki()
    }
    schedule()
  } catch {
    pageStatus.value = 'error'
  }
}

async function refreshFiles() {
  const response = await listKnowledgeFiles(id.value, true)
  files.value = response.data ?? []
  schedule()
}

function schedule() {
  window.clearTimeout(timer)
  if (polling.value) {
    timer = window.setTimeout(() => void refreshFiles(), 5000)
  }
}

async function refreshWiki() {
  const response = await getWikiStatus(id.value)
  wiki.value = response.data
  if (response.data.status === 'ready') {
    const graphResponse = await getWikiGraph(id.value, { mode: 'overview', limit: 500 })
    graph.value = mergeGraph(graph.value, graphResponse.data)
  } else {
    graph.value = null
  }
  if (response.data.status === 'generating') {
    window.setTimeout(() => void refreshWiki(), 5000)
  }
}

function mergeGraph(current: WikiGraph | null, next: WikiGraph) {
  if (!current) {
    return next
  }
  const nodes = new Map(current.nodes.map((node) => [node.slug, node]))
  for (const node of next.nodes) {
    nodes.set(node.slug, node)
  }
  const edges = [...current.edges]
  for (const edge of next.edges) {
    if (!edges.some((item) => item.source === edge.source && item.target === edge.target)) {
      edges.push(edge)
    }
  }
  return { nodes: [...nodes.values()], edges, meta: next.meta }
}

async function onSelect(event: Event) {
  const input = event.target as HTMLInputElement
  const list = [...(input.files ?? [])]
  input.value = ''
  if (!limits.value || list.length === 0) {
    return
  }
  const zip = list.find((file) => file.name.toLowerCase().endsWith('.zip'))
  if (zip) {
    results.value = (await uploadKnowledgeZip(id.value, zip)).data
  } else {
    const accepted = list.filter((file) => accept(file))
    failedFiles.value = list.filter((file) => !accept(file))
    results.value = accepted.length ? (await uploadKnowledgeFiles(id.value, accepted)).data : []
  }
  await refreshFiles()
}

function accept(file: File) {
  if (!limits.value) {
    return false
  }
  const ext = file.name.slice(file.name.lastIndexOf('.')).toLowerCase()
  return limits.value.allowed_exts.includes(ext) && file.size <= limits.value.max_file_mb * 1024 * 1024
}

async function retryFailed() {
  if (failedFiles.value.length === 0) {
    return
  }
  results.value = (await uploadKnowledgeFiles(id.value, failedFiles.value)).data
  failedFiles.value = []
  await refreshFiles()
}

async function removeFiles() {
  await deleteKnowledgeFiles(id.value, selected.value)
  selected.value = []
  feedback.success('文件已删除')
  await refreshFiles()
}

async function reparse() {
  await reparseKnowledgeFiles(id.value, selected.value)
  feedback.success('已提交重新解析')
  await refreshFiles()
  if (base.value?.wiki_enabled) {
    await refreshWiki()
  }
}

async function openNode(slug: string) {
  const response = await getWikiPage(id.value, slug)
  page.value = response.data
  drawerVisible.value = true
}

async function expandNode(slug: string) {
  const response = await getWikiGraph(id.value, { mode: 'ego', center: slug, depth: 1, limit: 500 })
  graph.value = mergeGraph(graph.value, response.data)
}

async function locate() {
  if (!keyword.value) {
    return
  }
  const response = await searchWiki(id.value, keyword.value)
  const first = response.data.pages?.[0]
  if (!first) {
    feedback.warning('没有找到对应页面')
    return
  }
  await openNode(first.slug)
}

onMounted(load)
onBeforeUnmount(() => window.clearTimeout(timer))
</script>

<template>
  <PageStatus v-if="pageStatus !== 'ready' || !base || !limits" :status="pageStatus === 'ready' ? 'loading' : pageStatus" @retry="load" />
  <div v-else>
    <t-tabs :default-value="base.wiki_enabled ? 'files' : 'files'">
      <t-tab-panel value="files" label="文档">
        <div class="toolbar">
          <label class="upload">
            上传文件或 ZIP
            <input type="file" multiple :accept="limits.allowed_exts.concat(['.zip']).join(',')" @change="onSelect" />
          </label>
          <span class="hint">单个文件不超过 {{ limits.max_file_mb }} MB。选择 ZIP 时将自动解压并保留目录结构。</span>
          <t-button :disabled="selected.length === 0" theme="danger" variant="outline" @click="removeFiles">删除</t-button>
          <t-button :disabled="selected.length === 0" variant="outline" @click="reparse">重新解析</t-button>
        </div>
        <t-table row-key="id" :data="files" :selected-row-keys="selected" :columns="[
          { colKey: 'row-select', type: 'multiple', width: 48 },
          { colKey: 'file_name', title: '名称' },
          { colKey: 'folder_path', title: '文件夹' },
          { colKey: 'file_size', title: '大小' },
          { colKey: 'parse_status', title: '解析状态' },
          { colKey: 'create_time', title: '时间' },
        ]" @select-change="(keys: string[]) => selected = keys">
          <template #parse_status="{ row }">
            <t-tag variant="light">{{ row.parse_status }}</t-tag>
            <span v-if="row.error_msg">{{ row.error_msg }}</span>
          </template>
        </t-table>
        <t-table v-if="results.length" row-key="file_name" :data="results" :columns="[
          { colKey: 'file_name', title: '文件' },
          { colKey: 'status', title: '结果' },
          { colKey: 'error', title: '说明' },
        ]" />
        <t-button v-if="failedFiles.length" variant="text" @click="retryFailed">重试失败项</t-button>
      </t-tab-panel>
      <t-tab-panel v-if="base.wiki_enabled" value="wiki" label="Wiki">
        <p>状态：{{ wiki?.status === 'generating' ? '生成中' : wiki?.status === 'ready' ? '已完成' : wiki?.status === 'timeout' ? '超时' : wiki?.status === 'empty' ? '还没有文件' : '未启用' }}</p>
        <div v-if="wiki?.status === 'timeout'" class="toolbar">
          <span>生成超时。可以选中文件后重新解析。</span>
        </div>
        <div v-else-if="wiki?.status === 'generating' || wiki?.status === 'empty'" class="hint">Wiki 还没有可展示的图谱。</div>
        <template v-else-if="graph">
          <div class="toolbar">
            <t-input v-model="keyword" placeholder="搜索页面" style="width: 240px" @enter="locate" />
            <t-button @click="locate">定位</t-button>
          </div>
          <KnowledgeGraph :graph="graph" @open="openNode" @expand="expandNode" />
        </template>
      </t-tab-panel>
    </t-tabs>
    <t-drawer v-model:visible="drawerVisible" :header="page?.title || 'Wiki 页面'" size="480px">
      <div v-if="page" v-html="safeHtml(page.content)"></div>
    </t-drawer>
  </div>
</template>

<style scoped>
.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  margin-bottom: 16px;
}
.upload {
  position: relative;
  overflow: hidden;
  padding: 6px 12px;
  border: 1px solid var(--qz-border);
  border-radius: var(--qz-radius-control);
  cursor: pointer;
}
.upload input {
  position: absolute;
  inset: 0;
  opacity: 0;
  cursor: pointer;
}
.hint {
  color: var(--qz-text-secondary);
}
</style>
