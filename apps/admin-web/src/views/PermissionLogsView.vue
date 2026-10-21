<script setup lang="ts">
import { onMounted, ref } from 'vue'
import PageStatus from '@/components/PageStatus.vue'
import { changeTypeLabel, formatTime, listPermissionLogs, type PermissionLog } from '@/api/security'
import { ApiError } from '@/utils/http'

const pageStatus = ref<'loading' | 'error' | 'ready'>('loading')
const errorMessage = ref('')
const items = ref<PermissionLog[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const keyword = ref('')

const columns = [
  { colKey: 'create_time', title: '时间', width: 180 },
  { colKey: 'operator_name', title: '操作人', width: 120 },
  { colKey: 'role_name', title: '角色', width: 140 },
  { colKey: 'change_type', title: '类型', width: 110 },
  { colKey: 'summary', title: '摘要', ellipsis: true },
  { colKey: 'before_text', title: '变更前', ellipsis: true },
  { colKey: 'after_text', title: '变更后', ellipsis: true },
]

async function load() {
  pageStatus.value = 'loading'
  errorMessage.value = ''
  try {
    const response = await listPermissionLogs({
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value.trim() || undefined,
    })
    items.value = response.data.items
    total.value = response.data.total
    pageStatus.value = 'ready'
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '无法加载权限变更'
    pageStatus.value = 'error'
  }
}

function search() {
  page.value = 1
  void load()
}

onMounted(() => {
  void load()
})
</script>

<template>
  <div class="page-grid">
    <t-card :bordered="false">
      <div class="toolbar">
        <t-input v-model="keyword" clearable placeholder="搜索操作人、角色或摘要" @enter="search" />
        <t-button theme="primary" variant="base" @click="search">查询</t-button>
      </div>
      <PageStatus :status="pageStatus" :error-message="errorMessage" description="还没有权限变更" @retry="load">
        <t-table row-key="id" :data="items" :columns="columns" hover>
          <template #create_time="{ row }">{{ formatTime(row.create_time) }}</template>
          <template #change_type="{ row }">{{ changeTypeLabel(row.change_type) }}</template>
          <template #before_text="{ row }">{{ row.before_text || '—' }}</template>
          <template #after_text="{ row }">{{ row.after_text || '—' }}</template>
        </t-table>
        <t-pagination v-model="page" v-model:page-size="pageSize" class="pager" :total="total" @change="load" />
      </PageStatus>
    </t-card>
  </div>
</template>
