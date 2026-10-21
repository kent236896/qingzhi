/**
 * Wiki 图谱邻居状态判定与提示文案。
 *
 * 逻辑移植自 Tencent/WeKnora frontend WikiBrowser.vue（MIT License）。
 * Copyright (c) Tencent. Adapted for qinzhi without WeKnora store/i18n.
 */

export interface WikiNeighborGraphLike {
  edges: { source: string; target: string }[]
  meta: { mode?: string; center?: string }
}

export interface WikiNeighborNodeLike {
  slug: string
  link_count: number
}

export type NeighborHintKind =
  | 'none'
  | 'all_shown'
  | 'overview_hidden'
  | 'ego_progress'
  | 'center_unreachable'

export interface NeighborStatus {
  visible: number
  total: number
  hidden: number
  isEgoCenter: boolean
  isOverview: boolean
  /** 展开/叠加按钮在「已探完」时应禁用 */
  fullyExplored: boolean
  kind: NeighborHintKind
  message: string
}

function undirectedNeighbors(edges: WikiNeighborGraphLike['edges'], slug: string): Set<string> {
  const neighbors = new Set<string>()
  for (const edge of edges) {
    if (edge.source === slug) {
      neighbors.add(edge.target)
    } else if (edge.target === slug) {
      neighbors.add(edge.source)
    }
  }
  return neighbors
}

export function visibleNeighborCount(edges: WikiNeighborGraphLike['edges'], slug: string): number {
  return undirectedNeighbors(edges, slug).size
}

/**
 * 虚线扩展环：仅 ego 且非中心，且还有未加载邻居时显示。
 * overview 几乎人人有缺口，画环只会噪音。
 */
export function shouldShowExpansionRing(
  graph: WikiNeighborGraphLike,
  node: WikiNeighborNodeLike,
  visibleDegree?: number,
): boolean {
  if (graph.meta?.mode !== 'ego') {
    return false
  }
  if (graph.meta.center === node.slug) {
    return false
  }
  const visible = visibleDegree ?? visibleNeighborCount(graph.edges, node.slug)
  const hidden = Math.max(0, (node.link_count || 0) - visible)
  return hidden > 0
}

export function computeNeighborStatus(graph: WikiNeighborGraphLike, node: WikiNeighborNodeLike): NeighborStatus {
  const visible = visibleNeighborCount(graph.edges, node.slug)
  const total = node.link_count || 0
  const hidden = Math.max(0, total - visible)
  const isEgoCenter = graph.meta?.mode === 'ego' && graph.meta.center === node.slug
  const isOverview = graph.meta?.mode === 'overview' || !graph.meta?.mode
  const fullyExplored = total === 0 || visible >= total || isEgoCenter

  let kind: NeighborHintKind
  let message: string
  if (total === 0) {
    kind = 'none'
    message = '该节点没有链接邻居'
  } else if (visible >= total) {
    kind = 'all_shown'
    message = `全部 ${total} 个邻居已在图中（可点「展开邻居」聚焦到该节点子图）`
  } else if (isEgoCenter) {
    kind = 'center_unreachable'
    message = `已显示邻居 ${visible}/${total}（还有 ${hidden} 个无法连通：可能是死链、被过滤或已删除）`
  } else if (isOverview) {
    kind = 'overview_hidden'
    message = `已显示邻居 ${visible}/${total}（${hidden} 个不在概览范围，点「展开邻居」以此为中心查看）`
  } else {
    kind = 'ego_progress'
    message = `已显示邻居 ${visible}/${total}（还有 ${hidden} 个未加载）`
  }

  return {
    visible,
    total,
    hidden,
    isEgoCenter,
    isOverview,
    fullyExplored,
    kind,
    message,
  }
}

/** 叠加邻居是否有意义：ego 非中心且仍有缺口；overview 应先展开（由调用方改走 ego） */
export function canBloomNeighbors(status: NeighborStatus | null): boolean {
  if (!status) {
    return true
  }
  if (status.isEgoCenter) {
    return false
  }
  return status.hidden > 0
}
