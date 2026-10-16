<script setup lang="ts">
import { onMounted, ref } from 'vue'
import PageStatus from '@/components/PageStatus.vue'
import { formatTime, listOperateLogs, type OperateLog } from '@/api/security'
import { ApiError } from '@/utils/http'

const pageStatus = ref<'loading' | 'error' | 'ready'>('loading')
const errorMessage = ref('')
const items = ref<OperateLog[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const keyword = ref('')

const columns = [
  { colKey: 'create_time', title: '时间', width: 180 },
  { colKey: 'username', title: '操作人', width: 120 },
  { colKey: 'module', title: '模块', width: 100 },
  { colKey: 'action', title: '动作', width: 100 },
  { colKey: 'request_method', title: '方法', width: 80 },
  { colKey: 'request_url', title: '路径', ellipsis: true },
  { colKey: 'status_code', title: '状态', width: 80 },
  { colKey: 'duration_ms', title: '耗时', width: 90 },
  { colKey: 'ip', title: 'IP', width: 140 },
]

async function load() {
  pageStatus.value = 'loading'
  errorMessage.value = ''
  try {
    const response = await listOperateLogs({
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value.trim() || undefined,
    })
    items.value = response.data.items
    total.value = response.data.total
    pageStatus.value = 'ready'
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '无法加载操作审计'
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
        <t-input v-model="keyword" clearable placeholder="搜索操作人、模块、动作或路径" @enter="search" />
        <t-button theme="default" @click="search">查询</t-button>
      </div>
      <PageStatus :status="pageStatus" :error-message="errorMessage" description="还没有操作记录" @retry="load">
        <t-table row-key="id" :data="items" :columns="columns" hover>
          <template #create_time="{ row }">{{ formatTime(row.create_time) }}</template>
          <template #duration_ms="{ row }">{{ row.duration_ms }} ms</template>
          <template #status_code="{ row }">
            <t-tag :theme="row.status_code < 400 ? 'success' : 'danger'" variant="light" size="small">
              {{ row.status_code }}
            </t-tag>
          </template>
        </t-table>
        <t-pagination v-model="page" v-model:page-size="pageSize" class="pager" :total="total" @change="load" />
      </PageStatus>
    </t-card>
  </div>
</template>
