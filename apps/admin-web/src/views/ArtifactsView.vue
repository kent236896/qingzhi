<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { DialogPlugin } from 'tdesign-vue-next'
import PageStatus from '@/components/PageStatus.vue'
import {
  deleteAdminArtifact,
  listAdminArtifacts,
  type ArtifactItem,
} from '@/api/catalog'
import { feedback } from '@/utils/feedback'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const pageStatus = ref<'loading' | 'error' | 'ready'>('loading')
const items = ref<ArtifactItem[]>([])
const total = ref(0)
const page = ref(1)
const keyword = ref('')
const canManage = auth.hasPermission('artifact:manage')

const columns = [
  { colKey: 'file_name', title: '文件' },
  { colKey: 'session_title', title: '会话' },
  { colKey: 'file_type', title: '类型', width: 100 },
  { colKey: 'file_size', title: '大小', width: 100 },
  { colKey: 'created_at', title: '时间', width: 180 },
  { colKey: 'actions', title: '操作', width: 100 },
]

function formatSize(size?: number) {
  if (!size || size <= 0) return '-'
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / (1024 * 1024)).toFixed(1)} MB`
}

async function load() {
  pageStatus.value = items.value.length ? 'ready' : 'loading'
  try {
    const response = await listAdminArtifacts({
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
  if (!canManage) return
  const dialog = DialogPlugin.confirm({
    header: '删除产物',
    body: `确定删除「${row.file_name}」吗？`,
    onConfirm: async () => {
      await deleteAdminArtifact({
        session_id: row.session_id,
        message_id: row.message_id,
        index: row.index,
      })
      feedback.success('已删除')
      dialog.hide()
      await load()
    },
  })
}

onMounted(load)
</script>

<template>
  <div>
    <PageStatus v-if="pageStatus !== 'ready'" :status="pageStatus" @retry="load" />
    <t-card v-else :bordered="false">
      <div class="toolbar">
        <t-input
          v-model="keyword"
          placeholder="搜索文件名"
          clearable
          style="width: 240px"
          @enter="load"
        />
        <t-button theme="primary" variant="outline" @click="load">查询</t-button>
      </div>
      <p class="hint">产物由对话/智能体生成，按会话归属管理；此处为运维全局视图。</p>
      <t-table
        row-key="file_name"
        :data="items"
        :columns="columns"
        :pagination="{ current: page, pageSize: 20, total }"
        @page-change="(info: { current: number }) => { page = info.current; load() }"
      >
        <template #file_size="{ row }">{{ formatSize(row.file_size) }}</template>
        <template #actions="{ row }">
          <t-button v-if="canManage" theme="danger" variant="text" @click="remove(row)">删除</t-button>
        </template>
      </t-table>
    </t-card>
  </div>
</template>

<style scoped>
.toolbar {
  display: flex;
  gap: 12px;
  margin-bottom: 8px;
}
.hint {
  margin: 0 0 16px;
  color: var(--qz-text-secondary);
}
</style>
