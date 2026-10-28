<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { listAuthEvents, protocolLabel, resultLabel, type AuthEvent } from '@/api/sso'
import { formatTime } from '@/api/security'
import PageStatus from '@/components/PageStatus.vue'
import { ApiError } from '@/utils/http'

const pageStatus = ref<'loading' | 'error' | 'ready'>('loading')
const errorMessage = ref('')
const items = ref<AuthEvent[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const keyword = ref('')
const result = ref('')
const protocol = ref('')

const columns = [
  { colKey: 'create_time', title: '时间', width: 180 },
  { colKey: 'result', title: '结果', width: 90 },
  { colKey: 'protocol', title: '协议', width: 100 },
  { colKey: 'event_type', title: '事件', width: 180 },
  { colKey: 'username', title: '操作者', width: 120 },
  { colKey: 'client_id', title: '客户端', width: 140 },
  { colKey: 'message', title: '说明', ellipsis: true },
  { colKey: 'request_id', title: 'Request ID', width: 160 },
]

const resultOptions = [
  { label: '全部结果', value: '' },
  { label: '成功', value: 'success' },
  { label: '失败', value: 'failure' },
]

const protocolOptions = [
  { label: '全部协议', value: '' },
  { label: 'OIDC', value: 'oidc' },
  { label: 'SAML 2.0', value: 'saml' },
]

async function load() {
  pageStatus.value = 'loading'
  errorMessage.value = ''
  try {
    const response = await listAuthEvents({
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value.trim() || undefined,
      result: result.value || undefined,
      protocol: protocol.value || undefined,
    })
    items.value = response.data.items
    total.value = response.data.total
    pageStatus.value = 'ready'
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '无法加载认证审计'
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
        <t-input v-model="keyword" clearable placeholder="搜索事件、说明或操作者" @enter="search" />
        <t-select v-model="result" :options="resultOptions" />
        <t-select v-model="protocol" :options="protocolOptions" />
        <t-button theme="primary" variant="base" @click="search">查询</t-button>
      </div>
      <PageStatus :status="pageStatus" :error-message="errorMessage" description="还没有认证审计事件" @retry="load">
        <t-table row-key="id" :data="items" :columns="columns" hover>
          <template #create_time="{ row }">{{ formatTime(row.create_time) }}</template>
          <template #result="{ row }">
            <t-tag :theme="row.result === 'success' ? 'success' : 'danger'" variant="light" size="small">
              {{ resultLabel(row.result) }}
            </t-tag>
          </template>
          <template #protocol="{ row }">{{ protocolLabel(row.protocol) }}</template>
        </t-table>
        <div class="pager">
          <t-pagination v-model="page" v-model:page-size="pageSize" :total="total" show-page-size @change="load" @page-size-change="load" />
        </div>
      </PageStatus>
    </t-card>
  </div>
</template>

<style scoped>
.toolbar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}
.pager {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
