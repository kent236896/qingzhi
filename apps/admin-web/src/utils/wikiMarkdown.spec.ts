import { describe, expect, it, vi } from 'vitest'

vi.mock('dompurify', () => ({
  default: {
    sanitize: (html: string) => html,
  },
}))

import { extractWikiSlugs, renderWikiMarkdown, rewriteWikiLinks } from './wikiMarkdown'

describe('wikiMarkdown', () => {
  it('rewrites titled and bare wiki links', () => {
    const { text, anchors } = rewriteWikiLinks('见[[entity/lo-ra|LoRA]]与[[concept/x]]', (slug) => ({
      exists: slug === 'entity/lo-ra',
      title: slug === 'entity/lo-ra' ? 'LoRA' : undefined,
    }))
    expect(text).toContain('§§WIKI0§§')
    expect(text).toContain('§§WIKI1§§')
    expect(anchors[0]).toContain('wiki-link')
    expect(anchors[0]).not.toContain('missing')
    expect(anchors[1]).toContain('wiki-link--missing')
  })

  it('renders sanitized html with wiki anchors', () => {
    const html = renderWikiMarkdown('Hello [[entity/a100|A100]]', () => ({
      exists: true,
      title: 'A100',
    }))
    expect(html).toContain('data-wiki-slug="entity/a100"')
    expect(html).toContain('A100')
    expect(html).not.toContain('<script')
  })

  it('extracts slugs', () => {
    expect(extractWikiSlugs('[[a|b]] [[c]]')).toEqual(['a', 'c'])
  })
})
