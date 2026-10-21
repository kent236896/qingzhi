<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Graph, GraphEvent } from '@antv/g6'
import { HelpCircleIcon, FullscreenIcon, RollbackIcon } from 'tdesign-icons-vue-next'
import WikiPageBody from '@/components/WikiPageBody.vue'
import {
  getWikiPage,
  getWikiTitles,
  searchWiki,
  type WikiGraph,
  type WikiNode,
  type WikiPage,
  type WikiTitle,
} from '@/api/knowledge'
import {
  legendTypesFromGraph,
  resolveCssVar,
  resolveWikiPageTypeColor,
  truncateWikiLabel,
  wikiNodeRadius,
  wikiPageTypeLabel,
} from '@/utils/wikiGraphTheme'
import {
  canBloomNeighbors,
  computeNeighborStatus,
  shouldShowExpansionRing,
  visibleNeighborCount,
} from '@/utils/wikiNeighborStatus'
import { ApiError } from '@/utils/http'
import { feedback } from '@/utils/feedback'

const props = defineProps<{
  graph: WikiGraph
  knowledgeBaseId: string
  focusSlug?: string
}>()

const emit = defineEmits<{
  openWiki: [slug: string]
  expand: [slug: string]
  bloom: [slug: string]
  backOverview: []
  select: [slug: string | null]
}>()

const host = ref<HTMLElement | null>(null)
const hideArrow = ref(false)
const selectedSlug = ref<string | null>(null)
const drawerOpen = ref(false)
const zoomScale = ref(1)
const searchKeyword = ref('')
const searchHits = ref<WikiPage[]>([])
const searchOpen = ref(false)
const pageData = ref<WikiPage | null>(null)
const pageLoading = ref(false)
const pageError = ref('')
const titleMap = ref<Map<string, WikiTitle>>(new Map())
let chart: Graph | null = null
let resizeObserver: ResizeObserver | null = null
let searchTimer = 0
let focusSeq = 0

const degreeMap = computed(() => {
  const degrees = new Map<string, number>()
  for (const node of props.graph.nodes) {
    degrees.set(node.slug, 0)
  }
  for (const edge of props.graph.edges) {
    degrees.set(edge.source, (degrees.get(edge.source) ?? 0) + 1)
    degrees.set(edge.target, (degrees.get(edge.target) ?? 0) + 1)
  }
  return degrees
})

const linkedSlugs = computed(() => {
  const set = new Set<string>()
  for (const edge of props.graph.edges) {
    set.add(edge.source)
    set.add(edge.target)
  }
  return set
})

const selectedNode = computed(() => props.graph.nodes.find((node) => node.slug === selectedSlug.value) ?? null)

const presentLegends = computed(() => legendTypesFromGraph(props.graph.nodes.map((node) => node.page_type)))

const isEgo = computed(() => props.graph.meta.mode === 'ego')

const overviewCaption = computed(() => {
  if (props.graph.meta.truncated) {
    return '仅展示部分节点，可搜索或展开邻居查看其余'
  }
  if (isEgo.value) {
    return '当前为节点子图，可返回全库概览'
  }
  return '已展示知识库全部节点'
})

const neighborStatus = computed(() => {
  if (!selectedNode.value) {
    return null
  }
  return computeNeighborStatus(props.graph, selectedNode.value)
})

const bloomEnabled = computed(() => canBloomNeighbors(neighborStatus.value))

function linkResolver(slug: string) {
  const hit = titleMap.value.get(slug)
  if (hit) {
    return { exists: true, title: hit.title }
  }
  const onGraph = props.graph.nodes.find((node) => node.slug === slug)
  if (onGraph) {
    return { exists: true, title: onGraph.title }
  }
  return { exists: false }
}

function forceParams(count: number) {
  if (count <= 30) {
    return { strength: -260, distance: 160, collidePad: 28, center: 0.04 }
  }
  if (count <= 100) {
    return { strength: -160, distance: 100, collidePad: 20, center: 0.06 }
  }
  return { strength: -100, distance: 70, collidePad: 14, center: 0.08 }
}

function topDegreeSlugs(limit: number) {
  return [...props.graph.nodes]
    .sort((a, b) => (degreeMap.value.get(b.slug) ?? 0) - (degreeMap.value.get(a.slug) ?? 0))
    .slice(0, limit)
    .map((node) => node.slug)
}

function neighborsOf(slug: string) {
  const set = new Set<string>([slug])
  for (const edge of props.graph.edges) {
    if (edge.source === slug) {
      set.add(edge.target)
    }
    if (edge.target === slug) {
      set.add(edge.source)
    }
  }
  return set
}

function shouldShowLabel(slug: string) {
  const active = selectedSlug.value
  if (active) {
    if (neighborsOf(active).has(slug)) {
      return true
    }
  }
  if (zoomScale.value >= 0.75 && props.graph.nodes.length <= 80) {
    return true
  }
  const n =
    zoomScale.value >= 0.45
      ? Math.max(1, Math.min(12, Math.ceil(props.graph.nodes.length / 30)))
      : Math.max(1, Math.min(8, Math.ceil(props.graph.nodes.length / 50)))
  return topDegreeSlugs(n).includes(slug)
}

function selectionColor() {
  return resolveCssVar('--qz-wiki-selection', '#0052d9')
}

function edgeBaseColor() {
  return resolveCssVar('--qz-wiki-edge', '#aeb5c0')
}

function labelFill() {
  return resolveCssVar('--qz-text-secondary', '#64748b')
}

function surfaceFill() {
  return resolveCssVar('--qz-surface', '#ffffff')
}

function applyHighlight(slug: string | null) {
  if (!chart) {
    return
  }
  const active = slug ? neighborsOf(slug) : null
  const blue = selectionColor()
  const edgeColor = edgeBaseColor()
  for (const node of props.graph.nodes) {
    const dimmed = Boolean(active && !active.has(node.slug))
    const selected = slug === node.slug
    chart.setElementState(node.slug, dimmed ? ['dimmed'] : selected ? ['selected'] : [])
    chart.updateNodeData([
      {
        id: node.slug,
        style: {
          opacity: dimmed ? 0.18 : 1,
          stroke: selected ? blue : surfaceFill(),
          lineWidth: selected ? 2.5 : 1.5,
          labelText: shouldShowLabel(node.slug) ? truncateWikiLabel(node.title) : '',
        },
      },
    ])
  }
  for (const [index, edge] of props.graph.edges.entries()) {
    const id = `${edge.source}-${edge.target}-${index}`
    const keep = !active || (active.has(edge.source) && active.has(edge.target) && (edge.source === slug || edge.target === slug))
    const connected = Boolean(slug && (edge.source === slug || edge.target === slug))
    chart.setElementState(id, keep ? [] : ['dimmed'])
    chart.updateEdgeData([
      {
        id,
        style: {
          stroke: connected ? blue : edgeColor,
          lineWidth: connected ? 1.4 : 0.8,
          opacity: keep ? (connected ? 0.85 : 0.42) : 0.08,
          endArrow: !hideArrow.value,
        },
      },
    ])
  }
  void chart.draw()
}

function placeIsolated(width: number, height: number) {
  if (!chart) {
    return
  }
  const isolated = props.graph.nodes.filter((node) => !linkedSlugs.value.has(node.slug))
  if (isolated.length === 0) {
    return
  }
  const gap = Math.min(120, Math.max(64, width / (isolated.length + 1)))
  const y = height - 48
  const startX = (width - gap * (isolated.length - 1)) / 2
  isolated.forEach((node, index) => {
    chart?.translateElementTo(node.slug, [startX + index * gap, y], false)
  })
}

async function fit() {
  if (!chart) {
    return
  }
  await chart.fitView()
}

function canvasSize() {
  const el = host.value
  if (!el) {
    return { width: 800, height: 600 }
  }
  const parent = el.parentElement
  const width = Math.max(parent?.clientWidth || el.clientWidth, 320)
  const height = Math.max(parent?.clientHeight || el.clientHeight, 400)
  return { width, height }
}

function nodeStyle(node: WikiNode) {
  const size = wikiNodeRadius(degreeMap.value.get(node.slug) ?? node.link_count ?? 0)
  const fill = resolveWikiPageTypeColor(node.page_type)
  const visible = visibleNeighborCount(props.graph.edges, node.slug)
  const showRing = shouldShowExpansionRing(props.graph, node, visible)
  return {
    fill,
    size,
    stroke: surfaceFill(),
    lineWidth: 1.5,
    labelText: shouldShowLabel(node.slug) ? truncateWikiLabel(node.title) : '',
    labelFill: labelFill(),
    labelFontSize: 11,
    labelPlacement: 'bottom' as const,
    labelOffsetY: 4,
    labelBackground: false,
    cursor: 'pointer' as const,
    halo: showRing,
    haloStroke: fill,
    haloLineWidth: 1.5,
    haloLineDash: [3, 3],
    haloStrokeOpacity: 0.55,
  }
}

function render() {
  if (!host.value) {
    return
  }
  const { width, height } = canvasSize()
  const linkedCount = props.graph.nodes.filter((node) => linkedSlugs.value.has(node.slug)).length
  const params = forceParams(Math.max(linkedCount, 1))
  const edgeColor = edgeBaseColor()
  const blue = selectionColor()
  chart?.destroy()
  chart = new Graph({
    container: host.value,
    width,
    height,
    padding: 48,
    animation: false,
    data: {
      nodes: props.graph.nodes.map((node) => {
        const isolated = !linkedSlugs.value.has(node.slug)
        const size = wikiNodeRadius(degreeMap.value.get(node.slug) ?? node.link_count ?? 0)
        return {
          id: node.slug,
          data: {
            title: node.title,
            page_type: node.page_type,
            isolated,
            size,
            link_count: node.link_count,
          },
          style: nodeStyle(node),
        }
      }),
      edges: props.graph.edges.map((edge, index) => ({
        id: `${edge.source}-${edge.target}-${index}`,
        source: edge.source,
        target: edge.target,
        style: {
          stroke: edgeColor,
          lineWidth: 0.8,
          opacity: 0.42,
          endArrow: !hideArrow.value,
        },
      })),
    },
    layout: {
      type: 'd3-force',
      animation: false,
      nodeFilter: (node: { data?: { isolated?: boolean } }) => !node.data?.isolated,
      manyBody: { strength: params.strength },
      link: { distance: params.distance, strength: 0.4 },
      collide: {
        radius: (node: { data?: { size?: number; title?: string } }) => {
          const size = Number(node.data?.size ?? 12)
          const label = String(node.data?.title ?? '')
          const labelW = Math.min(label.length, 12) * 5.5
          return size / 2 + params.collidePad + labelW / 8
        },
        strength: 0.85,
      },
      center: { strength: params.center },
    },
    node: {
      state: {
        dimmed: { opacity: 0.18 },
        selected: {
          stroke: blue,
          lineWidth: 2.5,
        },
      },
    },
    edge: {
      state: {
        dimmed: { opacity: 0.08 },
      },
    },
    behaviors: ['drag-canvas', 'zoom-canvas', 'drag-element'],
  })

  chart.on(GraphEvent.AFTER_LAYOUT, () => {
    placeIsolated(width, height)
    void fit()
  })
  chart.on('node:click', (event: unknown) => {
    const id = nodeID(event)
    if (!id) {
      return
    }
    const native = (event as { nativeEvent?: MouseEvent }).nativeEvent
    if (native?.shiftKey) {
      emit('bloom', id)
      return
    }
    void selectNode(id)
  })
  chart.on('node:dblclick', (event: unknown) => {
    const id = nodeID(event)
    if (id) {
      emit('expand', id)
    }
  })
  chart.on('canvas:click', () => {
    closeDrawer()
  })
  chart.on(GraphEvent.AFTER_TRANSFORM, () => {
    const zoom = chart?.getZoom?.() ?? 1
    zoomScale.value = zoom
    refreshLabels()
  })
  void chart.render()
  if (selectedSlug.value) {
    applyHighlight(selectedSlug.value)
  }
}

function refreshLabels() {
  if (!chart) {
    return
  }
  for (const node of props.graph.nodes) {
    chart.updateNodeData([
      {
        id: node.slug,
        style: {
          labelText: shouldShowLabel(node.slug) ? truncateWikiLabel(node.title) : '',
        },
      },
    ])
  }
  void chart.draw()
}

function nodeID(event: unknown) {
  const target = (event as { target?: { id?: string } }).target
  return target?.id
}

async function ensureTitles() {
  if (titleMap.value.size > 0) {
    return
  }
  try {
    const response = await getWikiTitles(props.knowledgeBaseId)
    titleMap.value = new Map((response.data ?? []).map((item) => [item.slug, item]))
  } catch {
    /* 标题地图失败不阻断正文 */
  }
}

async function loadPage(slug: string) {
  pageLoading.value = true
  pageError.value = ''
  pageData.value = null
  await ensureTitles()
  try {
    const response = await getWikiPage(props.knowledgeBaseId, slug)
    pageData.value = response.data
    if (response.data.title) {
      titleMap.value.set(slug, {
        slug,
        title: response.data.title,
        page_type: response.data.page_type,
      })
    }
  } catch (error) {
    pageError.value = error instanceof ApiError ? error.message : '页面不存在'
  } finally {
    pageLoading.value = false
  }
}

async function selectNode(slug: string, options: { openDrawer?: boolean; focus?: boolean } = {}) {
  const { openDrawer = true, focus = true } = options
  selectedSlug.value = slug
  drawerOpen.value = openDrawer
  applyHighlight(slug)
  emit('select', slug)
  if (focus) {
    await focusNode(slug)
  }
  await loadPage(slug)
}

function closeDrawer() {
  selectedSlug.value = null
  drawerOpen.value = false
  pageData.value = null
  pageError.value = ''
  applyHighlight(null)
  emit('select', null)
}

async function focusNode(slug: string) {
  if (!chart) {
    return
  }
  const seq = ++focusSeq
  try {
    await chart.focusElement(slug, { duration: 280 })
    if (seq !== focusSeq) {
      return
    }
    const zoom = chart.getZoom?.() ?? 1
    if (zoom < 0.6) {
      await chart.zoomTo(0.85, { duration: 280 })
    }
  } catch {
    /* 节点可能尚未完成布局 */
  }
}

function onSearchInput() {
  window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(() => void runSearch(), 300)
}

async function runSearch() {
  const q = searchKeyword.value.trim()
  if (!q) {
    searchHits.value = []
    searchOpen.value = false
    return
  }
  try {
    const response = await searchWiki(props.knowledgeBaseId, q.slice(0, 100), 10)
    searchHits.value = response.data.pages ?? []
    searchOpen.value = true
  } catch (error) {
    feedback.error(error instanceof ApiError ? error.message : '搜索失败')
  }
}

async function onSearchSelect(slug: string) {
  searchOpen.value = false
  searchKeyword.value = ''
  if (props.graph.nodes.some((node) => node.slug === slug)) {
    await selectNode(slug)
    return
  }
  emit('expand', slug)
  // 父组件替换图后，通过 focusSlug / watch 再选中
}

async function onContentNavigate(slug: string) {
  if (props.graph.nodes.some((node) => node.slug === slug)) {
    await selectNode(slug)
    return
  }
  emit('expand', slug)
}

function onExpand() {
  if (selectedSlug.value) {
    emit('expand', selectedSlug.value)
  }
}

function onBloom() {
  if (selectedSlug.value) {
    emit('bloom', selectedSlug.value)
  }
}

function openWiki() {
  if (selectedSlug.value) {
    emit('openWiki', selectedSlug.value)
  }
}

function warnMissingPage() {
  void feedback.warning('页面不存在')
}

async function focusSelected() {
  if (selectedSlug.value) {
    await focusNode(selectedSlug.value)
  }
}

watch(
  () => [props.graph, hideArrow.value] as const,
  async () => {
    await nextTick(render)
    if (selectedSlug.value && props.graph.nodes.some((node) => node.slug === selectedSlug.value)) {
      await nextTick()
      applyHighlight(selectedSlug.value)
      if (drawerOpen.value) {
        void loadPage(selectedSlug.value)
      }
    } else if (selectedSlug.value) {
      closeDrawer()
    }
  },
)

watch(
  () => props.focusSlug,
  async (slug) => {
    if (!slug) {
      return
    }
    if (props.graph.nodes.some((node) => node.slug === slug)) {
      await nextTick()
      await selectNode(slug)
    }
  },
)

onMounted(async () => {
  await ensureTitles()
  render()
  if (host.value?.parentElement) {
    resizeObserver = new ResizeObserver(() => {
      if (!chart || !host.value) {
        return
      }
      const { width, height } = canvasSize()
      chart.resize(width, height)
      void fit()
    })
    resizeObserver.observe(host.value.parentElement)
  }
  if (props.focusSlug && props.graph.nodes.some((node) => node.slug === props.focusSlug)) {
    await nextTick()
    await selectNode(props.focusSlug)
  }
})

onBeforeUnmount(() => {
  window.clearTimeout(searchTimer)
  resizeObserver?.disconnect()
  chart?.destroy()
})

defineExpose({ selectNode, focusNode })
</script>

<template>
  <div class="wiki-graph">
    <div ref="host" class="wiki-graph__canvas" title=""></div>

    <div class="wiki-graph__float wiki-graph__float--left">
      <div class="wiki-graph__search">
        <t-input
          v-model="searchKeyword"
          clearable
          size="small"
          placeholder="搜索 Wiki 页面"
          @enter="runSearch"
          @change="onSearchInput"
        />
        <t-tooltip content="单击节点查看详情；双击展开邻居；Shift+单击叠加邻居；拖拽空白处平移，滚轮缩放">
          <button type="button" class="wiki-graph__help" aria-label="操作提示">
            <HelpCircleIcon />
          </button>
        </t-tooltip>
        <div v-if="searchOpen && searchHits.length" class="wiki-graph__search-dropdown">
          <button
            v-for="hit in searchHits"
            :key="hit.slug"
            type="button"
            class="wiki-graph__search-item"
            @click="onSearchSelect(hit.slug)"
          >
            <strong>{{ hit.title }}</strong>
            <span>{{ wikiPageTypeLabel(hit.page_type) }}</span>
          </button>
        </div>
      </div>
    </div>

    <aside class="wiki-graph__float wiki-graph__float--right">
      <div class="wiki-graph__card">
        <div class="wiki-graph__legends">
          <span v-for="type in presentLegends" :key="type" class="legend">
            <i :style="{ background: resolveWikiPageTypeColor(type) }"></i>{{ wikiPageTypeLabel(type) }}
          </span>
        </div>
        <div class="wiki-graph__divider"></div>
        <div class="wiki-graph__actions">
          <button type="button" class="wiki-graph__action" @click="fit">
            <FullscreenIcon />适应屏幕
          </button>
          <button type="button" class="wiki-graph__action" :class="{ 'is-active': hideArrow }" @click="hideArrow = !hideArrow">
            {{ hideArrow ? '显示箭头' : '隐藏箭头' }}
          </button>
          <button v-if="isEgo" type="button" class="wiki-graph__action" @click="emit('backOverview')">
            <RollbackIcon />返回全库概览
          </button>
        </div>
        <div class="wiki-graph__divider"></div>
        <div class="wiki-graph__meta">
          <strong>{{ graph.meta.returned }} / {{ graph.meta.total }} 个节点</strong>
          <p>{{ overviewCaption }}</p>
        </div>
      </div>
    </aside>

    <aside v-if="drawerOpen && selectedNode" class="wiki-graph__drawer">
      <header class="wiki-graph__drawer-head">
        <h3 :title="selectedNode.title">{{ selectedNode.title }}</h3>
        <t-button theme="default" variant="text" shape="square" @click="closeDrawer">×</t-button>
      </header>
      <div class="wiki-graph__drawer-meta">
        <span class="type-tag" :style="{ borderColor: resolveWikiPageTypeColor(selectedNode.page_type), color: resolveWikiPageTypeColor(selectedNode.page_type) }">
          {{ wikiPageTypeLabel(selectedNode.page_type) }}
        </span>
        <span v-if="pageData?.version" class="version">v{{ pageData.version }}</span>
        <div class="wiki-graph__drawer-ops">
          <t-button size="small" theme="default" variant="outline" :disabled="!bloomEnabled && isEgo" @click="onBloom">
            叠加邻居
          </t-button>
          <t-button size="small" theme="primary" variant="base" @click="onExpand">展开邻居</t-button>
        </div>
      </div>
      <p v-if="neighborStatus" class="wiki-graph__neighbor-hint">{{ neighborStatus.message }}</p>
      <div class="wiki-graph__drawer-body">
        <WikiPageBody
          :content="pageData?.content"
          :loading="pageLoading"
          :error="pageError"
          :link-resolver="linkResolver"
          @navigate="onContentNavigate"
          @missing="warnMissingPage"
        />
      </div>
      <footer class="wiki-graph__drawer-foot">
        <t-button theme="default" variant="outline" @click="openWiki">在 Wiki 中打开</t-button>
        <t-button theme="default" variant="outline" @click="focusSelected">在图中聚焦</t-button>
      </footer>
    </aside>
  </div>
</template>

<style scoped>
.wiki-graph {
  position: relative;
  width: 100%;
  height: calc(100vh - var(--qz-header-height) - 88px);
  min-height: 480px;
  overflow: hidden;
  border: 1px solid var(--qz-border);
  border-radius: var(--qz-radius);
  background: var(--qz-surface);
}
.wiki-graph__canvas {
  width: 100%;
  height: 100%;
}
.wiki-graph__float {
  position: absolute;
  z-index: 2;
  top: 12px;
  pointer-events: none;
}
.wiki-graph__float--left {
  left: 12px;
  right: 280px;
  max-width: 360px;
}
.wiki-graph__float--right {
  right: 12px;
  width: 240px;
}
.wiki-graph__search,
.wiki-graph__card,
.wiki-graph__help,
.wiki-graph__action,
.wiki-graph__search-item {
  pointer-events: auto;
}
.wiki-graph__search {
  position: relative;
  display: flex;
  gap: 8px;
  align-items: center;
}
.wiki-graph__search :deep(.t-input) {
  background: var(--qz-surface);
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.06);
}
.wiki-graph__help {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: 1px solid var(--qz-border);
  border-radius: var(--qz-radius-control);
  background: var(--qz-surface);
  color: var(--qz-text-secondary);
  cursor: help;
}
.wiki-graph__search-dropdown {
  position: absolute;
  z-index: 3;
  top: calc(100% + 4px);
  left: 0;
  right: 36px;
  max-height: 280px;
  overflow: auto;
  border: 1px solid var(--qz-border);
  border-radius: var(--qz-radius);
  background: var(--qz-surface);
}
.wiki-graph__search-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
  width: 100%;
  padding: 8px 10px;
  border: 0;
  background: transparent;
  color: var(--qz-text);
  text-align: left;
  cursor: pointer;
}
.wiki-graph__search-item span {
  color: var(--qz-text-secondary);
  font-size: 12px;
}
.wiki-graph__search-item:hover {
  background: var(--qz-hover);
}
.wiki-graph__card {
  padding: 12px;
  border: 1px solid var(--qz-border);
  border-radius: var(--qz-radius);
  background: color-mix(in srgb, var(--qz-surface) 92%, transparent);
  backdrop-filter: blur(6px);
}
.wiki-graph__legends {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.legend {
  display: inline-flex;
  gap: 8px;
  align-items: center;
  color: var(--qz-text-secondary);
  font-size: 12px;
}
.legend i {
  width: 10px;
  height: 10px;
  border-radius: 50%;
}
.wiki-graph__divider {
  height: 1px;
  margin: 10px 0;
  background: var(--qz-border);
}
.wiki-graph__actions {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.wiki-graph__action {
  display: inline-flex;
  gap: 6px;
  align-items: center;
  padding: 4px 2px;
  border: 0;
  background: transparent;
  color: var(--qz-text);
  font-size: 12px;
  cursor: pointer;
}
.wiki-graph__action:hover,
.wiki-graph__action.is-active {
  color: var(--qz-wiki-selection);
}
.wiki-graph__meta strong {
  display: block;
  font-size: 12px;
  font-weight: 600;
}
.wiki-graph__meta p {
  margin: 4px 0 0;
  color: var(--qz-text-secondary);
  font-size: 12px;
  line-height: 1.4;
}
.wiki-graph__drawer {
  position: absolute;
  z-index: 4;
  top: 0;
  right: 0;
  bottom: 0;
  display: flex;
  flex-direction: column;
  width: min(460px, 100%);
  border-left: 1px solid var(--qz-border);
  background: var(--qz-surface);
  box-shadow: -8px 0 24px rgba(15, 23, 42, 0.06);
}
.wiki-graph__drawer-head {
  display: flex;
  gap: 8px;
  align-items: flex-start;
  justify-content: space-between;
  padding: 16px 16px 8px;
}
.wiki-graph__drawer-head h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 500;
  line-height: 1.35;
  word-break: break-word;
}
.wiki-graph__drawer-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  padding: 0 16px 8px;
}
.wiki-graph__drawer-ops {
  display: flex;
  gap: 8px;
  margin-left: auto;
}
.type-tag {
  display: inline-flex;
  padding: 0 8px;
  border: 1px solid;
  border-radius: 999px;
  font-size: 12px;
  line-height: 22px;
  background: transparent;
}
.version {
  color: var(--qz-text-secondary);
  font-size: 12px;
}
.wiki-graph__neighbor-hint {
  margin: 0;
  padding: 0 16px 10px;
  color: var(--qz-text-secondary);
  font-size: 12px;
  line-height: 1.5;
}
.wiki-graph__drawer-body {
  flex: 1;
  min-height: 0;
  padding: 0 16px 16px;
  overflow: auto;
}
.wiki-graph__drawer-foot {
  display: flex;
  gap: 8px;
  padding: 12px 16px;
  border-top: 1px solid var(--qz-border);
}
@media (max-width: 900px) {
  .wiki-graph__float--left {
    right: 12px;
    max-width: none;
  }
  .wiki-graph__float--right {
    top: auto;
    bottom: 12px;
    width: min(240px, calc(100% - 24px));
  }
  .wiki-graph__drawer {
    width: 100%;
  }
}
</style>
