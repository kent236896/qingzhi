import purify from 'dompurify'
import { marked } from 'marked'

const DOMPurify = (purify as unknown as { default?: typeof purify }).default ?? purify

export function renderSafeMarkdown(markdown: string): string {
  const html = marked.parse(markdown || '', { async: false, gfm: true, breaks: true }) as string
  return DOMPurify.sanitize(html, {
    USE_PROFILES: { html: true },
    ADD_ATTR: ['target', 'rel', 'class'],
  })
}
