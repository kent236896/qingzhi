<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { DialogPlugin } from 'tdesign-vue-next'
import PageStatus from '@/components/common/PageStatus.vue'
import { deleteArtifact, listArtifacts, type ArtifactItem } from '@/api/catalog'
import { feedback } from '@/utils/feedback'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const canQuery = computed(() => auth.hasPermission('artifact:query') || auth.hasPermission('artifact:manage'))
const pageStatus = ref<'loading' | 'error' | 'ready' | 'forbidden'>('loading')
const items = ref<ArtifactItem[]>([])
const total = ref(0)
const page = ref(1)
const keyword = ref('')

function formatSize(size?: number) {
  if (!size || size <= 0) return '-'
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / (1024 * 1024)).toFixed(1)} MB`
}

async function load() {
  if (!canQuery.value) {
    pageStatus.value = 'forbidden'
    return
  }
  pageStatus.value = items.value.length ? 'ready' : 'loading'
  try {
    const response = await listArtifacts({
      page: page.value,
      page_size: 20,
      keyword: keyword.value,
    })
    items.value = response.data.items ?? []
    total.value = response.data.total
    pageStatus.value = 'ready'
  } catch {
    if (!items.value.length) pageStatus.value = 'error'
  }
}

function remove(row: ArtifactItem) {
  const dialog = DialogPlugin.confirm({
    header: '删除产物',
    body: `确定删除「${row.file_name}」吗？`,
    onConfirm: async () => {
      await deleteArtifact({
        session_id: row.session_id,
        message_id: row.message_id,
        index: row.index,
      })
      feedback.success('已删除')
      dialog.destroy()
      await load()
    },
  })
}

onMounted(load)
</script>

<template>
  <div class="page">
    <header class="page__head">
      <div>
        <h2 class="page__title">产物</h2>
        <p class="page__desc">对话与智能体生成的文件，仅显示你有权访问的会话产物。</p>
      </div>
      <div class="page__tools">
        <t-input v-model="keyword" placeholder="搜索文件名" clearable style="width: 220px" @enter="load" />
        <t-button theme="primary" variant="outline" @click="load">查询</t-button>
      </div>
    </header>

    <PageStatus
      v-if="pageStatus === 'forbidden'"
      kind="error"
      title="无权限"
      description="当前账号没有产物查询权限（artifact:query）。"
    />
    <PageStatus
      v-else-if="pageStatus === 'loading' || pageStatus === 'error'"
      :kind="pageStatus === 'loading' ? 'loading' : 'error'"
      :title="pageStatus === 'loading' ? '加载中' : '加载失败'"
      :description="pageStatus === 'error' ? '请稍后重试。' : ''"
    >
      <t-button v-if="pageStatus === 'error'" variant="outline" @click="load">重试</t-button>
    </PageStatus>
    <PageStatus
      v-else-if="!items.length"
      kind="empty"
      title="暂无产物"
      description="还没有生成文件。在对话中让智能体产出文件后会出现在这里。"
    />
    <div v-else>
      <div class="list">
        <article v-for="row in items" :key="`${row.session_id}-${row.message_id}-${row.index}`" class="card">
          <div>
            <h3>{{ row.file_name }}</h3>
            <p>
              {{ row.session_title || '未命名会话' }} · {{ row.file_type || '-' }} ·
              {{ formatSize(row.file_size) }}
            </p>
          </div>
          <t-button theme="danger" variant="text" @click="remove(row)">删除</t-button>
        </article>
      </div>
      <div v-if="total > 20" class="pager">
        <t-pagination
          v-model="page"
          :total="total"
          :page-size="20"
          @change="(p: number) => { page = p; load() }"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.page__head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}
.page__title {
  margin: 0 0 6px;
  font-size: 22px;
}
.page__desc {
  margin: 0;
  color: var(--wb-text-secondary, #667085);
}
.page__tools {
  display: flex;
  gap: 8px;
}
.list {
  display: grid;
  gap: 12px;
}
.card {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 16px;
  border: 1px solid var(--wb-border, #e5e7eb);
  border-radius: 12px;
  background: var(--wb-surface, #fff);
}
.card h3 {
  margin: 0 0 6px;
  font-size: 16px;
}
.card p {
  margin: 0;
  color: var(--wb-text-secondary, #667085);
}
.pager {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
</style>
