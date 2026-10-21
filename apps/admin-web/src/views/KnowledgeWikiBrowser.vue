<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { SearchIcon } from 'tdesign-icons-vue-next'
import {
  getWikiIndex,
  getWikiPage,
  getWikiStats,
  getWikiTitles,
  listWikiFolders,
  listWikiPages,
  searchWiki,
  wikiGroupCounts,
  WIKI_KNOWLEDGE_TYPES,
  WIKI_SUMMARY_TYPES,
  type WikiFolder,
  type WikiIndex,
  type WikiIndexItem,
  type WikiPage,
  type WikiStats,
  type WikiTitle,
} from '@/api/knowledge'
import WikiPageBody from '@/components/WikiPageBody.vue'
import { ApiError } from '@/utils/http'
import { feedback } from '@/utils/feedback'

const props = defineProps<{
  knowledgeBaseId: string
  slug?: string
}>()

const emit = defineEmits<{
  'update:slug': [slug: string | undefined]
  openGraph: [slug: string]
}>()

type GroupKey = 'knowledge' | 'summary'
type ViewMode = 'tree' | 'list'
type RightMode = 'index' | 'page' | 'empty'

const leftWidth = ref(320)
const dragging = ref(false)
const mobileDrawer = ref(false)
const group = ref<GroupKey>('knowledge')
const viewMode = ref<ViewMode>('tree')
const searchKeyword = ref('')
const searchHits = ref<WikiPage[]>([])
const searchOpen = ref(false)
let searchTimer = 0

const stats = ref<WikiStats | null>(null)
const titleMap = ref<Map<string, WikiTitle>>(new Map())
const folders = ref<WikiFolder[]>([])
const folderPages = ref<Record<string, WikiPage[]>>({})
const expanded = ref<Set<string>>(new Set())
const loadingTree = ref(false)

const listItems = ref<WikiPage[]>([])
const listPage = ref(1)
const listTotal = ref(0)
const listSort = ref<'title' | 'updated_at'>('updated_at')
const listLoading = ref(false)

const rightMode = ref<RightMode>('index')
const indexData = ref<WikiIndex | null>(null)
const indexLoading = ref(false)
const pageData = ref<WikiPage | null>(null)
const pageLoading = ref(false)
const pageError = ref('')

const counts = computed(() => wikiGroupCounts(stats.value))
const pageTypesParam = computed(() =>
  group.value === 'knowledge' ? WIKI_KNOWLEDGE_TYPES.join(',') : WIKI_SUMMARY_TYPES.join(','),
)

function resolveTitle(slug: string) {
  return titleMap.value.get(slug)
}

function linkResolver(slug: string) {
  const hit = titleMap.value.get(slug)
  return { exists: Boolean(hit), title: hit?.title }
}

const backlinks = computed(() => {
  const links = pageData.value?.in_links ?? []
  return links.map((slug) => ({
    slug,
    title: resolveTitle(slug)?.title || slug,
  }))
})

async function bootstrap() {
  loadingTree.value = true
  try {
    const [statsRes, titlesRes] = await Promise.all([
      getWikiStats(props.knowledgeBaseId),
      getWikiTitles(props.knowledgeBaseId),
    ])
    stats.value = statsRes.data
    titleMap.value = new Map((titlesRes.data ?? []).map((item) => [item.slug, item]))
    await loadRootFolders()
    if (props.slug) {
      await openSlug(props.slug, false)
    } else {
      await openIndex(false)
    }
  } catch (error) {
    feedback.error(error instanceof ApiError ? error.message : '无法加载 Wiki')
  } finally {
    loadingTree.value = false
  }
}

async function loadRootFolders() {
  const response = await listWikiFolders(props.knowledgeBaseId, { page_types: pageTypesParam.value })
  folders.value = response.data.folders ?? []
  if (group.value === 'summary') {
    const pages: WikiPage[] = []
    for (const pageType of WIKI_SUMMARY_TYPES) {
      const orphan = await listWikiPages(props.knowledgeBaseId, {
        page: 1,
        page_size: 50,
        page_type: pageType,
        sort_by: 'updated_at',
        sort_order: 'desc',
      })
      pages.push(...(orphan.data.pages ?? []))
    }
    folderPages.value = { ...folderPages.value, root: pages }
  } else {
    folderPages.value = { ...folderPages.value, root: [] }
  }
}

async function toggleFolder(folder: WikiFolder) {
  const next = new Set(expanded.value)
  if (next.has(folder.id)) {
    next.delete(folder.id)
    expanded.value = next
    return
  }
  next.add(folder.id)
  expanded.value = next
  if (folder.has_children) {
    const children = await listWikiFolders(props.knowledgeBaseId, {
      parent_id: folder.id,
      page_types: pageTypesParam.value,
    })
    const existing = new Set(folders.value.map((item) => item.id))
    folders.value = [...folders.value, ...(children.data.folders ?? []).filter((item) => !existing.has(item.id))]
  }
  if (!folderPages.value[folder.id]) {
    const pages = await listWikiPages(props.knowledgeBaseId, {
      folder_id: folder.id,
      page: 1,
      page_size: 50,
    })
    folderPages.value = { ...folderPages.value, [folder.id]: pages.data.pages ?? [] }
  }
}

function childFolders(parentId: string) {
  return folders.value.filter((folder) => folder.parent_id === parentId)
}

function pagesOf(folderId: string) {
  return folderPages.value[folderId] ?? []
}

async function loadList() {
  listLoading.value = true
  try {
    const types = group.value === 'knowledge' ? WIKI_KNOWLEDGE_TYPES : WIKI_SUMMARY_TYPES
    const pages: WikiPage[] = []
    let total = 0
    for (const pageType of types) {
      const response = await listWikiPages(props.knowledgeBaseId, {
        page: listPage.value,
        page_size: 50,
        page_type: pageType,
        sort_by: listSort.value,
        sort_order: listSort.value === 'title' ? 'asc' : 'desc',
      })
      pages.push(...(response.data.pages ?? []))
      total += response.data.total
    }
    listItems.value = pages
    listTotal.value = total
  } finally {
    listLoading.value = false
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

async function openIndex(syncUrl = true) {
  rightMode.value = 'index'
  pageData.value = null
  pageError.value = ''
  indexLoading.value = true
  if (syncUrl) {
    emit('update:slug', undefined)
  }
  try {
    const response = await getWikiIndex(props.knowledgeBaseId, { limit: 50 })
    indexData.value = response.data
  } catch (error) {
    feedback.error(error instanceof ApiError ? error.message : '无法加载索引')
  } finally {
    indexLoading.value = false
  }
}

async function loadMoreIndex(groupType: string, cursor: string) {
  if (!indexData.value) {
    return
  }
  const response = await getWikiIndex(props.knowledgeBaseId, { types: groupType, limit: 50, cursor })
  const incoming = response.data.groups.find((item) => item.type === groupType)
  if (!incoming) {
    return
  }
  indexData.value = {
    ...indexData.value,
    groups: indexData.value.groups.map((groupItem) => {
      if (groupItem.type !== groupType) {
        return groupItem
      }
      return {
        ...groupItem,
        items: [...groupItem.items, ...incoming.items],
        next_cursor: incoming.next_cursor,
      }
    }),
  }
}

async function openSlug(slug: string, syncUrl = true) {
  if (slug === 'index') {
    await openIndex(syncUrl)
    return
  }
  rightMode.value = 'page'
  pageLoading.value = true
  pageError.value = ''
  pageData.value = null
  if (syncUrl) {
    emit('update:slug', slug)
  }
  mobileDrawer.value = false
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
    await expandToPage(response.data)
  } catch (error) {
    pageError.value = error instanceof ApiError ? error.message : '页面不存在'
  } finally {
    pageLoading.value = false
  }
}

async function expandToPage(page: WikiPage) {
  if (!page.folder_id) {
    return
  }
  const path: string[] = []
  let current = folders.value.find((folder) => folder.id === page.folder_id)
  while (current) {
    path.unshift(current.id)
    current = folders.value.find((folder) => folder.id === current?.parent_id)
  }
  for (const folderId of path) {
    const folder = folders.value.find((item) => item.id === folderId)
    if (folder && !expanded.value.has(folder.id)) {
      await toggleFolder(folder)
    }
  }
}

function warnMissingPage() {
  void feedback.warning('页面不存在')
}

function onResizeStart(event: MouseEvent) {
  dragging.value = true
  const startX = event.clientX
  const startWidth = leftWidth.value
  function onMove(move: MouseEvent) {
    leftWidth.value = Math.min(480, Math.max(240, startWidth + move.clientX - startX))
  }
  function onUp() {
    dragging.value = false
    window.removeEventListener('mousemove', onMove)
    window.removeEventListener('mouseup', onUp)
  }
  window.addEventListener('mousemove', onMove)
  window.addEventListener('mouseup', onUp)
}

function indexGroupTitle(type: string) {
  const map: Record<string, string> = {
    summary: '摘要',
    entity: '实体',
    concept: '概念',
    synthesis: '综合',
    comparison: '对比',
  }
  return map[type] || type
}

function groupItemsByCategory(items: WikiIndexItem[]) {
  const groups = new Map<string, WikiIndexItem[]>()
  for (const item of items) {
    const key = item.category_path?.[0] || '未分类'
    const list = groups.get(key) ?? []
    list.push(item)
    groups.set(key, list)
  }
  return [...groups.entries()]
}

watch(group, async () => {
  folders.value = []
  folderPages.value = {}
  expanded.value = new Set()
  await loadRootFolders()
  if (viewMode.value === 'list') {
    listPage.value = 1
    await loadList()
  }
})

watch(viewMode, async (mode) => {
  if (mode === 'list') {
    await loadList()
  }
})

watch(
  () => props.slug,
  (slug) => {
    if (!slug) {
      if (rightMode.value !== 'index') {
        void openIndex(false)
      }
      return
    }
    if (pageData.value?.slug !== slug) {
      void openSlug(slug, false)
    }
  },
)

onMounted(() => {
  void bootstrap()
})

defineExpose({ openSlug })
</script>

<template>
  <div class="wiki-browser" :class="{ 'is-dragging': dragging }">
    <button class="wiki-browser__mobile-toggle" type="button" @click="mobileDrawer = !mobileDrawer">
      {{ mobileDrawer ? '收起目录' : '打开目录' }}
    </button>
    <aside class="wiki-browser__left" :class="{ 'is-open': mobileDrawer }" :style="{ width: `${leftWidth}px` }">
      <div class="wiki-search">
        <t-input
          v-model="searchKeyword"
          clearable
          placeholder="搜索 Wiki 页面"
          @enter="runSearch"
          @change="onSearchInput"
        >
          <template #prefix-icon><SearchIcon /></template>
        </t-input>
        <div v-if="searchOpen && searchHits.length" class="wiki-search__dropdown">
          <button
            v-for="hit in searchHits"
            :key="hit.slug"
            type="button"
            class="wiki-search__item"
            @click="openSlug(hit.slug)"
          >
            <strong>{{ hit.title }}</strong>
            <t-tag size="small" variant="light">{{ hit.page_type }}</t-tag>
            <span>{{ hit.summary }}</span>
          </button>
        </div>
      </div>

      <button
        type="button"
        class="wiki-index-entry"
        :class="{ 'is-active': rightMode === 'index' }"
        @click="openIndex()"
      >
        索引
      </button>

      <div class="wiki-groups">
        <button type="button" :class="{ 'is-active': group === 'knowledge' }" @click="group = 'knowledge'">
          知识 {{ counts.knowledge }}
        </button>
        <button type="button" :class="{ 'is-active': group === 'summary' }" @click="group = 'summary'">
          摘要 {{ counts.summary }}
        </button>
      </div>

      <div class="wiki-view-switch">
        <t-radio-group v-model="viewMode" variant="default-filled" size="small">
          <t-radio-button value="tree">树形</t-radio-button>
          <t-radio-button value="list">列表</t-radio-button>
        </t-radio-group>
      </div>

      <div v-if="loadingTree" class="wiki-hint">正在加载目录</div>
      <template v-else-if="viewMode === 'tree'">
        <div class="wiki-tree">
          <div v-for="folder in childFolders('')" :key="folder.id" class="wiki-tree__folder">
            <button type="button" class="wiki-tree__folder-btn" @click="toggleFolder(folder)">
              <span>{{ expanded.has(folder.id) ? '▾' : '▸' }}</span>
              <span>{{ folder.name }}</span>
              <em>{{ folder.page_count }}</em>
            </button>
            <div v-if="expanded.has(folder.id)" class="wiki-tree__children">
              <div v-for="child in childFolders(folder.id)" :key="child.id" class="wiki-tree__folder">
                <button type="button" class="wiki-tree__folder-btn" @click="toggleFolder(child)">
                  <span>{{ expanded.has(child.id) ? '▾' : '▸' }}</span>
                  <span>{{ child.name }}</span>
                  <em>{{ child.page_count }}</em>
                </button>
                <div v-if="expanded.has(child.id)" class="wiki-tree__pages">
                  <button
                    v-for="page in pagesOf(child.id)"
                    :key="page.slug"
                    type="button"
                    class="wiki-tree__page"
                    :class="{ 'is-active': pageData?.slug === page.slug }"
                    @click="openSlug(page.slug)"
                  >
                    {{ page.title }}
                  </button>
                </div>
              </div>
              <div class="wiki-tree__pages">
                <button
                  v-for="page in pagesOf(folder.id)"
                  :key="page.slug"
                  type="button"
                  class="wiki-tree__page"
                  :class="{ 'is-active': pageData?.slug === page.slug }"
                  @click="openSlug(page.slug)"
                >
                  {{ page.title }}
                </button>
              </div>
            </div>
          </div>
          <div v-if="pagesOf('root').length" class="wiki-tree__pages wiki-tree__pages--root">
            <p class="wiki-hint">未归入目录</p>
            <button
              v-for="page in pagesOf('root')"
              :key="page.slug"
              type="button"
              class="wiki-tree__page"
              :class="{ 'is-active': pageData?.slug === page.slug }"
              @click="openSlug(page.slug)"
            >
              {{ page.title }}
            </button>
          </div>
        </div>
      </template>
      <template v-else>
        <div class="wiki-list-toolbar">
          <t-select
            v-model="listSort"
            class="wiki-list-sort"
            :options="[
              { label: '按更新时间', value: 'updated_at' },
              { label: '按标题', value: 'title' },
            ]"
            @change="loadList"
          />
        </div>
        <div v-if="listLoading" class="wiki-hint">正在加载列表</div>
        <button
          v-for="page in listItems"
          :key="page.slug"
          type="button"
          class="wiki-tree__page"
          :class="{ 'is-active': pageData?.slug === page.slug }"
          @click="openSlug(page.slug)"
        >
          {{ page.title }}
        </button>
        <p class="wiki-hint">共 {{ listTotal }} 条</p>
      </template>
    </aside>
    <div class="wiki-browser__resizer" @mousedown="onResizeStart"></div>
    <section class="wiki-browser__right">
      <template v-if="rightMode === 'index'">
        <div v-if="indexLoading" class="wiki-hint">正在加载索引</div>
        <template v-else-if="indexData">
          <article class="wiki-index">
            <WikiPageBody :content="indexData.intro || ''" :link-resolver="linkResolver" @navigate="openSlug" @missing="warnMissingPage" />
            <section v-for="groupItem in indexData.groups" :key="groupItem.type" class="wiki-index__group">
              <h3>{{ indexGroupTitle(groupItem.type) }} {{ groupItem.total }}</h3>
              <template v-if="groupItem.type === 'entity' || groupItem.type === 'concept'">
                <div v-for="[category, items] in groupItemsByCategory(groupItem.items)" :key="category">
                  <h4>{{ category }}</h4>
                  <button
                    v-for="item in items"
                    :key="item.slug"
                    type="button"
                    class="wiki-index__row"
                    @click="openSlug(item.slug)"
                  >
                    <strong>{{ item.title }}</strong>
                    <span>{{ item.summary }}</span>
                  </button>
                </div>
              </template>
              <template v-else>
                <button
                  v-for="item in groupItem.items"
                  :key="item.slug"
                  type="button"
                  class="wiki-index__row"
                  @click="openSlug(item.slug)"
                >
                  <strong>{{ item.title }}</strong>
                  <span>{{ item.summary }}</span>
                </button>
              </template>
              <t-button
                v-if="groupItem.next_cursor"
                theme="default"
                variant="outline"
                size="small"
                @click="loadMoreIndex(groupItem.type, groupItem.next_cursor)"
              >
                加载更多
              </t-button>
            </section>
          </article>
        </template>
        <div v-else class="wiki-empty">还没有 Wiki 页面，文档生成完成后会出现在这里</div>
      </template>

      <template v-else-if="rightMode === 'page'">
        <article class="wiki-page">
          <header v-if="pageData && !pageLoading && !pageError" class="wiki-page__head">
            <div>
              <h2>{{ pageData.title }}</h2>
              <div class="wiki-page__meta">
                <t-tag variant="light" size="small">{{ pageData.page_type }}</t-tag>
                <span v-if="pageData.aliases?.length">别名：{{ pageData.aliases.join('、') }}</span>
                <span v-if="pageData.updated_at">更新 {{ pageData.updated_at }}</span>
                <span v-if="pageData.version">v{{ pageData.version }}</span>
                <span v-if="pageData.last_edit_source">{{ pageData.last_edit_source }}</span>
              </div>
              <p v-if="pageData.summary" class="wiki-page__summary">{{ pageData.summary }}</p>
            </div>
            <t-button theme="default" variant="outline" @click="emit('openGraph', pageData.slug)">在图谱中查看</t-button>
          </header>
          <WikiPageBody
            :content="pageData?.content"
            :loading="pageLoading"
            :error="pageError"
            :link-resolver="linkResolver"
            @navigate="openSlug"
            @missing="warnMissingPage"
          />
          <section v-if="backlinks.length && pageData && !pageLoading && !pageError" class="wiki-backlinks">
            <h3>引用此页的页面</h3>
            <button
              v-for="item in backlinks"
              :key="item.slug"
              type="button"
              class="wiki-index__row"
              @click="openSlug(item.slug)"
            >
              {{ item.title }}
            </button>
          </section>
        </article>
      </template>
    </section>
  </div>
</template>

<style scoped>
.wiki-browser {
  display: flex;
  min-height: 640px;
  border: 1px solid var(--qz-border);
  border-radius: 12px;
  background: var(--qz-surface);
  overflow: hidden;
}
.wiki-browser__mobile-toggle {
  display: none;
}
.wiki-browser__left {
  flex: 0 0 auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 240px;
  max-width: 480px;
  padding: 16px;
  border-right: 1px solid var(--qz-border);
  overflow: auto;
}
.wiki-browser__resizer {
  width: 6px;
  cursor: col-resize;
  background: transparent;
}
.wiki-browser__resizer:hover,
.is-dragging .wiki-browser__resizer {
  background: var(--qz-brand-light);
}
.wiki-browser__right {
  flex: 1;
  min-width: 0;
  padding: 20px 24px;
  overflow: auto;
}
.wiki-search {
  position: relative;
}
.wiki-search__dropdown {
  position: absolute;
  z-index: 5;
  top: 100%;
  right: 0;
  left: 0;
  max-height: 280px;
  overflow: auto;
  border: 1px solid var(--qz-border);
  border-radius: 8px;
  background: var(--qz-surface);
  box-shadow: none;
}
.wiki-search__item,
.wiki-index-entry,
.wiki-groups button,
.wiki-tree__folder-btn,
.wiki-tree__page,
.wiki-index__row {
  display: flex;
  gap: 6px;
  width: 100%;
  padding: 8px 10px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--qz-text);
  text-align: left;
  cursor: pointer;
}
.wiki-search__item {
  flex-direction: column;
  align-items: flex-start;
}
.wiki-search__item span,
.wiki-index__row span,
.wiki-page__summary,
.wiki-hint {
  color: var(--qz-text-secondary);
  font-size: 12px;
}
.wiki-index-entry.is-active,
.wiki-groups button.is-active,
.wiki-tree__page.is-active {
  background: var(--qz-brand-light);
  color: var(--qz-brand);
  font-weight: 500;
}
.wiki-groups {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}
.wiki-groups button {
  justify-content: center;
  border: 1px solid var(--qz-border);
}
.wiki-tree__folder-btn {
  justify-content: flex-start;
  align-items: center;
}
.wiki-tree__folder-btn em {
  margin-left: auto;
  color: var(--qz-text-placeholder);
  font-style: normal;
  font-size: 12px;
}
.wiki-tree__children,
.wiki-tree__pages {
  padding-left: 16px;
}
.wiki-page__head {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 16px;
}
.wiki-page__head h2 {
  margin: 0 0 8px;
  font-size: 22px;
  font-weight: 500;
}
.wiki-page__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  color: var(--qz-text-secondary);
  font-size: 12px;
}
.wiki-index__group {
  margin-top: 24px;
}
.wiki-index__group h3 {
  margin: 0 0 12px;
  font-size: 16px;
  font-weight: 500;
}
.wiki-index__group h4 {
  margin: 12px 0 8px;
  color: var(--qz-text-secondary);
  font-size: 13px;
  font-weight: 500;
}
.wiki-index__row {
  flex-direction: column;
  align-items: flex-start;
  margin-bottom: 4px;
}
.wiki-index__row:hover,
.wiki-tree__page:hover,
.wiki-search__item:hover {
  background: var(--qz-hover);
}
.wiki-backlinks {
  margin-top: 28px;
  padding-top: 16px;
  border-top: 1px solid var(--qz-border);
}
.wiki-empty {
  display: grid;
  place-content: center;
  min-height: 240px;
  color: var(--qz-text-secondary);
}
.wiki-list-sort {
  width: 140px;
}
@media (max-width: 900px) {
  .wiki-browser {
    flex-direction: column;
  }
  .wiki-browser__mobile-toggle {
    display: inline-flex;
    margin: 12px 12px 0;
    padding: 8px 12px;
    border: 1px solid var(--qz-border);
    border-radius: 8px;
    background: var(--qz-surface);
  }
  .wiki-browser__left {
    display: none;
    width: 100% !important;
    max-width: none;
    border-right: 0;
    border-bottom: 1px solid var(--qz-border);
  }
  .wiki-browser__left.is-open {
    display: flex;
  }
  .wiki-browser__resizer {
    display: none;
  }
}
</style>
