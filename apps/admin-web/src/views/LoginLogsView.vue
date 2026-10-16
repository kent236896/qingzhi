<script setup lang="ts">
import { onMounted, ref } from 'vue'
import PageStatus from '@/components/PageStatus.vue'
import { formatTime, listLoginLogs, loginResultLabel, type LoginLog } from '@/api/security'
import { ApiError } from '@/utils/http'

const pageStatus = ref<'loading' | 'error' | 'ready'>('loading')
const errorMessage = ref('')
const items = ref<LoginLog[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const keyword = ref('')
const result = ref<number | ''>('')

const columns = [
  { colKey: 'create_time', title: '时间', width: 180 },
  { colKey: 'username', title: '账号', width: 140 },
  { colKey: 'result', title: '结果', width: 90 },
  { colKey: 'message', title: '说明', ellipsis: true },
  { colKey: 'ip', title: 'IP', width: 140 },
  { colKey: 'user_agent', title: '客户端', ellipsis: true },
]

const resultOptions = [
  { label: '全部结果', value: '' },
  { label: '成功', value: 0 },
  { label: '失败', value: 1 },
  { label: '锁定', value: 2 },
]

function resultTheme(value: number): 'success' | 'danger' | 'warning' {
  if (value === 0) return 'success'
  if (value === 2) return 'warning'
  return 'danger'
}

async function load() {
  pageStatus.value = 'loading'
  errorMessage.value = ''
  try {
    const response = await listLoginLogs({
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value.trim() || undefined,
      result: result.value === '' ? undefined : result.value,
    })
    items.value = response.data.items
    total.value = response.data.total
    pageStatus.value = 'ready'
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '无法加载登录日志'
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
        <t-input v-model="keyword" clearable placeholder="搜索账号、说明或 IP" @enter="search" />
        <t-select v-model="result" class="result-filter" :options="resultOptions" />
        <t-button theme="default" @click="search">查询</t-button>
      </div>
      <PageStatus :status="pageStatus" :error-message="errorMessage" description="还没有登录日志" @retry="load">
        <t-table row-key="id" :data="items" :columns="columns" hover>
          <template #create_time="{ row }">{{ formatTime(row.create_time) }}</template>
          <template #result="{ row }">
            <t-tag :theme="resultTheme(row.result)" variant="light" size="small">
              {{ loginResultLabel(row.result) }}
            </t-tag>
          </template>
          <template #message="{ row }">{{ row.message || '—' }}</template>
          <template #user_agent="{ row }">{{ row.user_agent || '—' }}</template>
        </t-table>
        <t-pagination v-model="page" v-model:page-size="pageSize" class="pager" :total="total" @change="load" />
      </PageStatus>
    </t-card>
  </div>
</template>

<style scoped>
.result-filter {
  width: 140px;
}
</style>
