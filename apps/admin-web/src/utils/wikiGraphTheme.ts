/**
 * Wiki 图谱色板与图例标签。
 * 取值对齐 Tencent/WeKnora（MIT）Wiki 图谱；通过 CSS 变量接入主题，不写死运行时色值。
 */

export const WIKI_PAGE_TYPE_LABELS: Record<string, string> = {
  summary: '摘要',
  entity: '实体',
  concept: '概念',
  synthesis: '综合',
  comparison: '对比',
  index: '索引',
}

/** 图例优先顺序（仅展示当前图中实际存在的类型） */
export const WIKI_PAGE_TYPE_LEGEND_ORDER = [
  'summary',
  'entity',
  'concept',
  'synthesis',
  'comparison',
  'index',
] as const

const TYPE_CSS_VARS: Record<string, string> = {
  summary: '--qz-wiki-summary',
  entity: '--qz-wiki-entity',
  concept: '--qz-wiki-concept',
  synthesis: '--qz-wiki-synthesis',
  comparison: '--qz-wiki-comparison',
  index: '--qz-wiki-index',
}

const FALLBACK_HEX: Record<string, string> = {
  summary: '#0052d9',
  entity: '#2ba471',
  concept: '#e37318',
  synthesis: '#0594fa',
  comparison: '#d54941',
  index: '#8c8c8c',
}

export function wikiPageTypeLabel(pageType: string): string {
  return WIKI_PAGE_TYPE_LABELS[pageType] || pageType || '未知'
}

export function wikiPageTypeCssVar(pageType: string): string {
  return TYPE_CSS_VARS[pageType] ?? '--qz-wiki-unknown'
}

/** 解析为画布可用的具体色值（G6 不能直接吃 var()） */
export function resolveWikiPageTypeColor(pageType: string, root?: HTMLElement | Document | null): string {
  const varName = wikiPageTypeCssVar(pageType)
  if (typeof document === 'undefined' && !root) {
    return FALLBACK_HEX[pageType] ?? '#94a3b8'
  }
  const el = !root || root instanceof Document ? (root instanceof Document ? root.documentElement : document.documentElement) : root
  const value = getComputedStyle(el).getPropertyValue(varName).trim()
  if (value) {
    return value
  }
  return FALLBACK_HEX[pageType] ?? (getComputedStyle(el).getPropertyValue('--qz-wiki-unknown').trim() || '#94a3b8')
}

export function resolveCssVar(name: string, fallback: string, root?: HTMLElement | Document | null): string {
  if (typeof document === 'undefined' && !root) {
    return fallback
  }
  const el = !root || root instanceof Document ? (root instanceof Document ? root.documentElement : document.documentElement) : root
  return getComputedStyle(el).getPropertyValue(name).trim() || fallback
}

export function legendTypesFromGraph(pageTypes: Iterable<string>): string[] {
  const present = new Set(pageTypes)
  const ordered = WIKI_PAGE_TYPE_LEGEND_ORDER.filter((type) => present.has(type))
  const extras = [...present].filter((type) => !WIKI_PAGE_TYPE_LEGEND_ORDER.includes(type as (typeof WIKI_PAGE_TYPE_LEGEND_ORDER)[number]))
  extras.sort()
  return [...ordered, ...extras]
}

/** 节点半径：按连接数轻度缩放，限制在 8–20（对齐 WeKnora 5–20，略抬叶子下限） */
export function wikiNodeRadius(linkCount: number): number {
  return Math.max(8, Math.min(20, 4 + Math.sqrt(Math.max(0, linkCount)) * 1.8))
}

export function truncateWikiLabel(title: string, max = 12): string {
  const text = title || ''
  return text.length > max ? `${text.slice(0, max)}…` : text
}
