import purify from 'dompurify'
import { marked } from 'marked'

const DOMPurify = (purify as unknown as { default?: typeof purify }).default ?? purify

const wikiLinkPattern = /\[\[([^\]|#]+)(?:\|([^\]]+))?\]\]/g

export type WikiLinkResolver = (slug: string) => { title?: string; exists: boolean }

function escapeHtml(value: string) {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

/** Convert [[slug|title]] / [[slug]] into temporary tokens before Markdown. */
export function rewriteWikiLinks(markdown: string, resolve: WikiLinkResolver): { text: string; anchors: string[] } {
  const anchors: string[] = []
  const text = markdown.replace(wikiLinkPattern, (_full, rawSlug: string, label?: string) => {
    const slug = String(rawSlug).trim()
    const info = resolve(slug)
    const display = escapeHtml(label?.trim() || info.title || slug)
    const index = anchors.length
    if (!info.exists) {
      anchors.push(
        `<a class="wiki-link wiki-link--missing" href="#" data-wiki-slug="${escapeHtml(slug)}" title="页面不存在">${display}</a>`,
      )
    } else {
      anchors.push(`<a class="wiki-link" href="#" data-wiki-slug="${escapeHtml(slug)}">${display}</a>`)
    }
    return `§§WIKI${index}§§`
  })
  return { text, anchors }
}

export function renderWikiMarkdown(markdown: string, resolve: WikiLinkResolver): string {
  const { text, anchors } = rewriteWikiLinks(markdown || '', resolve)
  let html = marked.parse(text, { async: false, gfm: true, breaks: false }) as string
  html = html.replace(/§§WIKI(\d+)§§/g, (_full, raw: string) => anchors[Number(raw)] ?? '')
  const clean = DOMPurify.sanitize(html, {
    USE_PROFILES: { html: true },
    ADD_ATTR: ['data-wiki-slug', 'target', 'rel', 'class', 'title'],
  })
  return clean.replace(/<a\b([^>]*?)href="(https?:\/\/[^"]+)"([^>]*)>/gi, (_full, pre, href, post) => {
    if (/rel=/i.test(pre + post)) {
      return `<a${pre}href="${href}"${post}>`
    }
    return `<a${pre}href="${href}" rel="noopener noreferrer" target="_blank"${post}>`
  })
}

export function extractWikiSlugs(markdown: string): string[] {
  const slugs = new Set<string>()
  for (const match of markdown.matchAll(wikiLinkPattern)) {
    slugs.add(String(match[1]).trim())
  }
  return [...slugs]
}
