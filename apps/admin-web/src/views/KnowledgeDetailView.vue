<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PageStatus from '@/components/PageStatus.vue'
import KnowledgeGraph from '@/views/KnowledgeGraph.vue'
import KnowledgeWikiBrowser from '@/views/KnowledgeWikiBrowser.vue'
import {
  deleteKnowledgeFiles,
  getKnowledgeBase,
  getKnowledgeLimits,
  getWikiGraph,
  getWikiStatus,
  listKnowledgeFiles,
  reparseKnowledgeFiles,
  uploadKnowledgeFiles,
  uploadKnowledgeZip,
  type KnowledgeBase,
  type KnowledgeFile,
  type KnowledgeLimits,
  type UploadResult,
  type WikiGraph,
  type WikiStatus,
} from '@/api/knowledge'
import { formatTime } from '@/api/security'
import { feedback } from '@/utils/feedback'

type DetailTab = 'files' | 'wiki' | 'graph'

const route = useRoute()
const router = useRouter()
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
const graphFocusSlug = ref<string | undefined>()
const wikiBrowser = ref<{ openSlug: (slug: string) => Promise<void> } | null>(null)
let timer = 0
let wikiTimer = 0
const BLOOM_MAX_NODES = 1500

const terminal = new Set(['completed', 'failed', 'duplicate', 'skipped'])
const polling = computed(() => files.value.some((file) => !terminal.has(file.parse_status)))

const activeTab = computed<DetailTab>(() => {
  const tab = String(route.query.tab || 'files')
  if (tab === 'wiki' || tab === 'graph' || tab === 'files') {
    if ((tab === 'wiki' || tab === 'graph') && !base.value?.wiki_enabled) {
      return 'files'
    }
    return tab
  }
  return 'files'
})

const wikiSlug = computed(() => {
  const slug = route.query.slug
  return typeof slug === 'string' && slug ? slug : undefined
})

const wikiBanner = computed(() => {
  switch (wiki.value?.status) {
    case 'generating':
      return { theme: 'warning' as const, text: 'Wiki 生成中，完成后页面与图谱会自动更新。' }
    case 'timeout':
      return { theme: 'error' as const, text: 'Wiki 生成超时。可以选中文件后重新解析。' }
    case 'empty':
      return { theme: 'default' as const, text: '还没有 Wiki 页面，文档生成完成后会出现在这里。' }
    default:
      return null
  }
})

const wikiReadyTag = computed(() => {
  if (wiki.value?.status !== 'ready') {
    return ''
  }
  return `已完成 · ${wiki.value.page_count} 个页面`
})

async function setTab(tab: DetailTab) {
  const query: Record<string, string> = {}
  for (const [key, value] of Object.entries(route.query)) {
    if (typeof value === 'string') {
      query[key] = value
    }
  }
  query.tab = tab
  if (tab === 'files') {
    delete query.slug
  }
  await router.replace({ query })
}

async function setSlug(slug: string | undefined) {
  const query: Record<string, string> = {}
  for (const [key, value] of Object.entries(route.query)) {
    if (typeof value === 'string') {
      query[key] = value
    }
  }
  query.tab = 'wiki'
  if (slug) {
    query.slug = slug
  } else {
    delete query.slug
  }
  await router.replace({ query })
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
  window.clearTimeout(wikiTimer)
  const response = await getWikiStatus(id.value)
  wiki.value = response.data
  if (response.data.status === 'ready') {
    if (!graph.value || graph.value.meta.mode !== 'ego') {
      const graphResponse = await getWikiGraph(id.value, { mode: 'overview', limit: 500 })
      graph.value = graphResponse.data
    }
  } else {
    graph.value = null
  }
  if (response.data.status === 'generating') {
    wikiTimer = window.setTimeout(() => void refreshWiki(), 5000)
  }
}

function mergeGraph(current: WikiGraph | null, next: WikiGraph): WikiGraph {
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
  const mergedNodes = [...nodes.values()]
  return {
    nodes: mergedNodes,
    edges,
    meta: {
      ...next.meta,
      returned: mergedNodes.length,
      total: Math.max(next.meta.total, current.meta.total),
      truncated: next.meta.truncated || current.meta.truncated || mergedNodes.length < Math.max(next.meta.total, current.meta.total),
    },
  }
}

function trimBloom(data: WikiGraph, protect: Set<string>): WikiGraph {
  if (data.nodes.length <= BLOOM_MAX_NODES) {
    return data
  }
  const keep = new Set<string>(protect)
  for (const node of data.nodes) {
    if (keep.size >= BLOOM_MAX_NODES) {
      break
    }
    keep.add(node.slug)
  }
  const nodes = data.nodes.filter((node) => keep.has(node.slug))
  const edges = data.edges.filter((edge) => keep.has(edge.source) && keep.has(edge.target))
  return {
    nodes,
    edges,
    meta: {
      ...data.meta,
      returned: nodes.length,
      truncated: true,
    },
  }
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

/** 展开邻居：替换为 ego 子图（对齐 WeKnora） */
async function expandNode(slug: string) {
  const response = await getWikiGraph(id.value, { mode: 'ego', center: slug, depth: 1, limit: 500 })
  graph.value = response.data
  graphFocusSlug.value = undefined
  await Promise.resolve()
  graphFocusSlug.value = slug
}

/** 叠加邻居：ego 下合并；overview 下先 pivot 到 ego */
async function bloomNode(slug: string) {
  if (!graph.value || graph.value.meta.mode !== 'ego') {
    await expandNode(slug)
    return
  }
  const response = await getWikiGraph(id.value, { mode: 'ego', center: slug, depth: 1, limit: 500 })
  const merged = mergeGraph(graph.value, response.data)
  const protect = new Set<string>([graph.value.meta.center || '', slug].filter(Boolean))
  graph.value = trimBloom(merged, protect)
  graphFocusSlug.value = undefined
  await Promise.resolve()
  graphFocusSlug.value = slug
}

async function backToOverview() {
  const response = await getWikiGraph(id.value, { mode: 'overview', limit: 500 })
  graph.value = response.data
  graphFocusSlug.value = undefined
}

async function openWikiFromGraph(slug: string) {
  await setSlug(slug)
  await wikiBrowser.value?.openSlug(slug)
}

async function openGraphFromWiki(slug: string) {
  await expandNode(slug)
  const query: Record<string, string> = {}
  for (const [key, value] of Object.entries(route.query)) {
    if (typeof value === 'string') {
      query[key] = value
    }
  }
  query.tab = 'graph'
  query.slug = slug
  await router.replace({ query })
}

watch(
  () => route.query.tab,
  (tab) => {
    if ((tab === 'wiki' || tab === 'graph') && base.value && !base.value.wiki_enabled) {
      void setTab('files')
    }
  },
)

watch(
  () => [activeTab.value, wikiSlug.value] as const,
  ([tab, slug]) => {
    if (tab === 'graph' && slug) {
      graphFocusSlug.value = slug
    }
  },
  { immediate: true },
)

onMounted(load)
onBeforeUnmount(() => {
  window.clearTimeout(timer)
  window.clearTimeout(wikiTimer)
})
</script>

<template>
  <PageStatus v-if="pageStatus !== 'ready' || !base || !limits" :status="pageStatus === 'ready' ? 'loading' : pageStatus" @retry="load" />
  <div v-else class="kb-detail" :class="{ 'kb-detail--graph': activeTab === 'graph' }">
    <div class="kb-detail__tabs">
      <t-tabs :value="activeTab" @change="(value: string | number) => setTab(String(value) as DetailTab)">
        <t-tab-panel value="files" label="文档" />
        <t-tab-panel v-if="base.wiki_enabled" value="wiki" label="Wiki" />
        <t-tab-panel v-if="base.wiki_enabled" value="graph" label="图谱" />
      </t-tabs>
      <t-tag v-if="base.wiki_enabled && wikiReadyTag && (activeTab === 'wiki' || activeTab === 'graph')" size="small" variant="light" theme="success">
        {{ wikiReadyTag }}
      </t-tag>
    </div>

    <t-alert
      v-if="base.wiki_enabled && wikiBanner && activeTab !== 'files'"
      class="kb-detail__banner"
      :theme="wikiBanner.theme"
      :message="wikiBanner.text"
    />

    <section v-show="activeTab === 'files'" class="kb-detail__panel">
      <div class="toolbar">
        <label class="upload">
          上传文件或 ZIP
          <input type="file" multiple :accept="limits.allowed_exts.concat(['.zip']).join(',')" @change="onSelect" />
        </label>
        <span class="hint">单个文件不超过 {{ limits.max_file_mb }} MB。选择 ZIP 时将自动解压并保留目录结构。</span>
        <t-button :disabled="selected.length === 0" theme="default" variant="outline" @click="reparse">重新解析</t-button>
        <t-divider layout="vertical" />
        <t-button :disabled="selected.length === 0" theme="danger" variant="outline" @click="removeFiles">删除</t-button>
      </div>
      <t-table
        row-key="id"
        :data="files"
        :selected-row-keys="selected"
        :columns="[
          { colKey: 'row-select', type: 'multiple', width: 48 },
          { colKey: 'file_name', title: '名称' },
          { colKey: 'folder_path', title: '文件夹' },
          { colKey: 'file_size', title: '大小' },
          { colKey: 'parse_status', title: '解析状态' },
          { colKey: 'create_time', title: '时间' },
        ]"
        @select-change="(keys: string[]) => (selected = keys)"
      >
        <template #parse_status="{ row }">
          <t-tag variant="light">{{ row.parse_status }}</t-tag>
          <span v-if="row.error_msg">{{ row.error_msg }}</span>
        </template>
        <template #create_time="{ row }">{{ formatTime(row.create_time) }}</template>
      </t-table>
      <t-table
        v-if="results.length"
        row-key="file_name"
        :data="results"
        :columns="[
          { colKey: 'file_name', title: '文件' },
          { colKey: 'status', title: '结果' },
          { colKey: 'error', title: '说明' },
        ]"
      />
      <t-button v-if="failedFiles.length" theme="default" variant="outline" @click="retryFailed">重试失败项</t-button>
    </section>

    <section v-if="base.wiki_enabled" v-show="activeTab === 'wiki'" class="kb-detail__panel">
      <div v-if="wiki?.status === 'generating' || wiki?.status === 'empty'" class="hint">
        Wiki 还没有可浏览的页面。
      </div>
      <KnowledgeWikiBrowser
        v-else-if="wiki?.status === 'ready' || wiki?.status === 'timeout'"
        ref="wikiBrowser"
        :knowledge-base-id="id"
        :slug="wikiSlug"
        @update:slug="setSlug"
        @open-graph="openGraphFromWiki"
      />
    </section>

    <section
      v-if="base.wiki_enabled && activeTab === 'graph'"
      class="kb-detail__panel kb-detail__panel--graph"
    >
      <div v-if="wiki?.status === 'generating' || wiki?.status === 'empty'" class="hint">Wiki 还没有可展示的图谱。</div>
      <KnowledgeGraph
        v-else-if="graph"
        :graph="graph"
        :knowledge-base-id="id"
        :focus-slug="graphFocusSlug"
        @open-wiki="openWikiFromGraph"
        @expand="expandNode"
        @bloom="bloomNode"
        @back-overview="backToOverview"
      />
    </section>
  </div>
</template>

<style scoped>
.kb-detail__tabs {
  display: flex;
  gap: 12px;
  align-items: center;
  flex: 0 0 auto;
}
.kb-detail__tabs :deep(.t-tabs) {
  flex: 1;
  min-width: 0;
}
.kb-detail__banner {
  margin: 12px 0 16px;
  flex: 0 0 auto;
}
.kb-detail__panel {
  margin-top: 8px;
}
.kb-detail--graph {
  display: flex;
  flex-direction: column;
  /* 页头 + 内容区上下 padding + 页面标题区，避免 100vh 再叠一层导致页面滚动条反馈 */
  height: calc(100dvh - var(--qz-header-height) - 2 * var(--qz-content-padding) - 56px);
  max-height: calc(100dvh - var(--qz-header-height) - 2 * var(--qz-content-padding) - 56px);
  overflow: hidden;
}
.kb-detail--graph .kb-detail__panel--graph {
  flex: 1 1 auto;
  min-height: 0;
  margin-top: 8px;
  display: flex;
  flex-direction: column;
}
.kb-detail--graph .kb-detail__panel--graph > * {
  flex: 1 1 auto;
  min-height: 0;
  width: 100%;
}
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
