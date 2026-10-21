import { describe, expect, it } from 'vitest'
import {
  canBloomNeighbors,
  computeNeighborStatus,
  shouldShowExpansionRing,
  visibleNeighborCount,
} from './wikiNeighborStatus'
import { legendTypesFromGraph, resolveWikiPageTypeColor, wikiPageTypeCssVar } from './wikiGraphTheme'

describe('wikiNeighborStatus', () => {
  const edges = [
    { source: 'a', target: 'b' },
    { source: 'a', target: 'c' },
    { source: 'b', target: 'c' },
  ]

  it('counts undirected neighbors', () => {
    expect(visibleNeighborCount(edges, 'a')).toBe(2)
    expect(visibleNeighborCount(edges, 'b')).toBe(2)
  })

  it('reports none when link_count is 0', () => {
    const status = computeNeighborStatus({ edges, meta: { mode: 'overview' } }, { slug: 'x', link_count: 0 })
    expect(status.kind).toBe('none')
    expect(status.message).toContain('没有链接邻居')
    expect(status.fullyExplored).toBe(true)
  })

  it('reports all neighbors shown', () => {
    const status = computeNeighborStatus(
      { edges, meta: { mode: 'overview' } },
      { slug: 'a', link_count: 2 },
    )
    expect(status.kind).toBe('all_shown')
    expect(status.message).toContain('全部 2 个邻居已在图中')
    expect(status.fullyExplored).toBe(true)
  })

  it('reports overview partial neighbors', () => {
    const status = computeNeighborStatus(
      { edges, meta: { mode: 'overview' } },
      { slug: 'a', link_count: 5 },
    )
    expect(status.kind).toBe('overview_hidden')
    expect(status.hidden).toBe(3)
    expect(status.message).toContain('不在概览范围')
    expect(status.fullyExplored).toBe(false)
  })

  it('reports ego progress for non-center with gap', () => {
    const status = computeNeighborStatus(
      { edges, meta: { mode: 'ego', center: 'b' } },
      { slug: 'a', link_count: 5 },
    )
    expect(status.kind).toBe('ego_progress')
    expect(status.message).toContain('还有 3 个未加载')
    expect(canBloomNeighbors(status)).toBe(true)
  })

  it('reports center unreachable when ego center still has gap', () => {
    const status = computeNeighborStatus(
      { edges, meta: { mode: 'ego', center: 'a' } },
      { slug: 'a', link_count: 5 },
    )
    expect(status.kind).toBe('center_unreachable')
    expect(status.fullyExplored).toBe(true)
    expect(canBloomNeighbors(status)).toBe(false)
  })

  it('shows expansion ring only for ego non-center with hidden neighbors', () => {
    const graph = { edges, meta: { mode: 'ego', center: 'b' } }
    expect(shouldShowExpansionRing(graph, { slug: 'a', link_count: 5 })).toBe(true)
    expect(shouldShowExpansionRing(graph, { slug: 'b', link_count: 5 })).toBe(false)
    expect(shouldShowExpansionRing({ edges, meta: { mode: 'overview' } }, { slug: 'a', link_count: 5 })).toBe(false)
  })
})

describe('wikiGraphTheme', () => {
  it('falls unknown page_type to unknown css var', () => {
    expect(wikiPageTypeCssVar('category')).toBe('--qz-wiki-unknown')
    expect(wikiPageTypeCssVar('')).toBe('--qz-wiki-unknown')
  })

  it('orders legend by known types then extras', () => {
    expect(legendTypesFromGraph(['comparison', 'entity', 'category', 'summary'])).toEqual([
      'summary',
      'entity',
      'comparison',
      'category',
    ])
  })

  it('resolveWikiPageTypeColor returns hex-like fallback without DOM vars', () => {
    const color = resolveWikiPageTypeColor('nosuch')
    expect(color.length).toBeGreaterThan(0)
  })
})
