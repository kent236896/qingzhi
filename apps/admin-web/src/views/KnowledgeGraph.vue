<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Graph } from '@antv/g6'
import type { WikiGraph } from '@/api/knowledge'

const props = defineProps<{ graph: WikiGraph }>()
const emit = defineEmits<{ open: [slug: string]; expand: [slug: string] }>()

const host = ref<HTMLElement | null>(null)
const hideArrow = ref(false)
let chart: Graph | null = null

const pageTypeColor: Record<string, string> = {
  summary: '#4F46E5',
  entity: '#16A34A',
  concept: '#D97706',
  synthesis: '#7C3AED',
  comparison: '#0284C7',
  index: '#64748B',
}

const legends = [
  ['summary', '摘要'],
  ['entity', '实体'],
  ['concept', '概念'],
  ['synthesis', '综合'],
  ['comparison', '对比'],
  ['index', '索引'],
]

function colorOf(type: string) {
  return pageTypeColor[type] ?? '#94A3B8'
}

function render() {
  if (!host.value) {
    return
  }
  chart?.destroy()
  chart = new Graph({
    container: host.value,
    autoFit: 'view',
    data: {
      nodes: props.graph.nodes.map((node) => ({
        id: node.slug,
        data: { title: node.title, page_type: node.page_type },
        style: { fill: colorOf(node.page_type) },
      })),
      edges: props.graph.edges.map((edge, index) => ({
        id: `${edge.source}-${edge.target}-${index}`,
        source: edge.source,
        target: edge.target,
      })),
    },
    layout: { type: 'd3-force', manyBody: { strength: -80 } },
    node: {
      style: {
        size: 28,
        labelText: (datum) => String(datum.data?.title ?? ''),
        labelFill: '#0F172A',
      },
    },
    edge: {
      style: {
        endArrow: !hideArrow.value,
        stroke: '#94A3B8',
      },
    },
    behaviors: ['drag-canvas', 'zoom-canvas', 'drag-element'],
  })
  chart.on('node:click', (event) => {
    const id = nodeID(event)
    if (id) {
      emit('open', id)
    }
  })
  chart.on('node:dblclick', (event) => {
    const id = nodeID(event)
    if (id) {
      emit('expand', id)
    }
  })
  void chart.render()
}

function nodeID(event: unknown) {
  const target = (event as { target?: { id?: string } }).target
  return target?.id
}

function fit() {
  void chart?.fitView()
}

watch(() => [props.graph, hideArrow.value], render)
onMounted(render)
onBeforeUnmount(() => chart?.destroy())
</script>

<template>
  <div class="wiki-graph">
    <div class="wiki-graph__tools">
      <span v-for="[type, label] in legends" :key="type" class="legend">
        <i :style="{ background: colorOf(type) }"></i>{{ label }}
      </span>
      <span class="legend"><i style="background: #94a3b8"></i>其他</span>
      <t-checkbox v-model="hideArrow">隐藏箭头</t-checkbox>
      <t-button size="small" variant="outline" @click="fit">适应屏幕</t-button>
      <span>已展示 {{ graph.meta.returned }} / {{ graph.meta.total }} 个节点</span>
    </div>
    <p v-if="graph.meta.truncated" class="hint">仅展示部分节点，可搜索或双击节点展开邻居。</p>
    <div ref="host" class="wiki-graph__canvas"></div>
  </div>
</template>

<style scoped>
.wiki-graph__tools {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  margin-bottom: 8px;
}
.legend {
  display: inline-flex;
  gap: 6px;
  align-items: center;
}
.legend i {
  width: 10px;
  height: 10px;
  border-radius: 50%;
}
.wiki-graph__canvas {
  height: 520px;
  border: 1px solid var(--qz-border);
  border-radius: var(--qz-radius);
}
.hint {
  color: var(--qz-text-secondary);
}
</style>
