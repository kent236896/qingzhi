<script setup lang="ts">
import { computed } from 'vue'
import { renderWikiMarkdown, type WikiLinkResolver } from '@/utils/wikiMarkdown'

const props = defineProps<{
  content?: string
  loading?: boolean
  error?: string
  emptyText?: string
  linkResolver?: WikiLinkResolver
}>()

const emit = defineEmits<{
  navigate: [slug: string]
  missing: [slug: string]
}>()

const html = computed(() => {
  if (!props.content) {
    return ''
  }
  const resolve = props.linkResolver ?? (() => ({ exists: true as const }))
  return renderWikiMarkdown(props.content, resolve)
})

function onClick(event: MouseEvent) {
  const target = event.target as HTMLElement | null
  const link = target?.closest('a[data-wiki-slug]') as HTMLAnchorElement | null
  if (!link) {
    return
  }
  event.preventDefault()
  const slug = link.getAttribute('data-wiki-slug')
  if (!slug) {
    return
  }
  if (link.classList.contains('wiki-link--missing')) {
    emit('missing', slug)
    return
  }
  emit('navigate', slug)
}
</script>

<template>
  <div v-if="loading" class="wiki-page-body__hint">正在加载页面</div>
  <div v-else-if="error" class="wiki-page-body__empty">{{ error }}</div>
  <div v-else-if="!content" class="wiki-page-body__empty">{{ emptyText || '页面不存在' }}</div>
  <div v-else class="wiki-content" v-html="html" @click="onClick"></div>
</template>

<style scoped>
.wiki-page-body__hint,
.wiki-page-body__empty {
  display: grid;
  place-content: center;
  min-height: 160px;
  color: var(--qz-text-secondary);
  font-size: 13px;
}
.wiki-content :deep(.wiki-link) {
  color: var(--qz-success);
  text-decoration: underline dashed;
  text-underline-offset: 3px;
}
.wiki-content :deep(.wiki-link--missing) {
  color: var(--qz-text-placeholder);
  text-decoration: underline dashed;
}
.wiki-content :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 12px 0;
}
.wiki-content :deep(th),
.wiki-content :deep(td) {
  padding: 8px;
  border: 1px solid var(--qz-border);
}
.wiki-content :deep(h1),
.wiki-content :deep(h2),
.wiki-content :deep(h3) {
  margin: 1.2em 0 0.6em;
  font-weight: 500;
}
.wiki-content :deep(p) {
  margin: 0.6em 0;
  line-height: 1.7;
}
.wiki-content :deep(ul),
.wiki-content :deep(ol) {
  padding-left: 1.4em;
}
</style>
