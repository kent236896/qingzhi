<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { deleteIdentityLink, listIdentityLinks, protocolLabel, type IdentityLink } from '@/api/sso'
import { formatTime } from '@/api/security'
import PageStatus from '@/components/PageStatus.vue'
import { feedback } from '@/utils/feedback'
import { ApiError } from '@/utils/http'
import { DialogPlugin } from 'tdesign-vue-next'

const pageStatus = ref<'loading' | 'error' | 'ready'>('loading')
const errorMessage = ref('')
const items = ref<IdentityLink[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const keyword = ref('')

const columns = [
  { colKey: 'provider_name', title: '提供方', width: 160 },
  { colKey: 'protocol', title: '协议', width: 100 },
  { colKey: 'external_subject', title: '外部 Subject / NameID', ellipsis: true },
  { colKey: 'user_id', title: '本平台用户 ID', width: 280 },
  { colKey: 'created_at', title: '绑定时间', width: 180 },
  { colKey: 'ops', title: '操作', width: 100 },
]

async function load() {
  pageStatus.value = 'loading'
  errorMessage.value = ''
  try {
    const response = await listIdentityLinks({
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value.trim() || undefined,
    })
    items.value = response.data.items
    total.value = response.data.total
    pageStatus.value = 'ready'
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '无法加载身份映射'
    pageStatus.value = 'error'
  }
}

function unbind(row: IdentityLink) {
  const dialog = DialogPlugin.confirm({
    header: '解绑外部身份',
    body: '解绑后该外部身份将无法继续登录，除非重新映射。',
    theme: 'warning',
    onConfirm: async () => {
      try {
        await deleteIdentityLink(row.id)
        feedback.success('已解绑')
        await load()
      } finally {
        dialog.destroy()
      }
    },
  })
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
        <t-input v-model="keyword" clearable placeholder="搜索提供方、Subject 或用户 ID" @enter="search" />
        <t-button theme="primary" variant="base" @click="search">查询</t-button>
      </div>
      <PageStatus :status="pageStatus" :error-message="errorMessage" description="还没有外部身份映射" @retry="load">
        <t-table row-key="id" :data="items" :columns="columns" hover>
          <template #protocol="{ row }">{{ protocolLabel(row.protocol) }}</template>
          <template #created_at="{ row }">{{ formatTime(row.created_at) }}</template>
          <template #ops="{ row }">
            <t-button theme="danger" variant="text" @click="unbind(row)">解绑</t-button>
          </template>
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
