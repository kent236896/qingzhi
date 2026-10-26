<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { createSamlSp, listSamlSps, setSamlSpStatus, statusLabel, type SamlSP } from '@/api/sso'
import { formatTime } from '@/api/security'
import PageStatus from '@/components/PageStatus.vue'
import { feedback } from '@/utils/feedback'
import { ApiError } from '@/utils/http'

const pageStatus = ref<'loading' | 'error' | 'ready'>('loading')
const errorMessage = ref('')
const items = ref<SamlSP[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const keyword = ref('')
const dialogVisible = ref(false)
const saving = ref(false)
const form = reactive({
  name: '',
  entity_id: '',
  status: 'disabled',
  acs_urls_text: '',
  allow_idp_initiated: false,
})

const columns = [
  { colKey: 'name', title: '名称', width: 160 },
  { colKey: 'entity_id', title: 'Entity ID', ellipsis: true },
  { colKey: 'status', title: '状态', width: 90 },
  { colKey: 'allow_idp_initiated', title: 'IdP-initiated', width: 120 },
  { colKey: 'updated_at', title: '更新时间', width: 180 },
  { colKey: 'ops', title: '操作', width: 120 },
]

async function load() {
  pageStatus.value = 'loading'
  errorMessage.value = ''
  try {
    const response = await listSamlSps({
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value.trim() || undefined,
    })
    items.value = response.data.items
    total.value = response.data.total
    pageStatus.value = 'ready'
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '无法加载 SAML 服务提供方'
    pageStatus.value = 'error'
  }
}

function openCreate() {
  Object.assign(form, {
    name: '',
    entity_id: '',
    status: 'disabled',
    acs_urls_text: '',
    allow_idp_initiated: false,
  })
  dialogVisible.value = true
}

async function save() {
  saving.value = true
  try {
    await createSamlSp({
      name: form.name,
      entity_id: form.entity_id,
      status: form.status,
      acs_urls: form.acs_urls_text
        .split('\n')
        .map((v) => v.trim())
        .filter(Boolean),
      allow_idp_initiated: form.allow_idp_initiated,
    })
    feedback.success('已注册 SAML 服务提供方')
    dialogVisible.value = false
    await load()
  } catch (error) {
    if (!(error instanceof ApiError)) {
      feedback.error('创建失败')
    }
  } finally {
    saving.value = false
  }
}

async function toggleStatus(row: SamlSP) {
  const next = row.status === 'enabled' ? 'disabled' : 'enabled'
  await setSamlSpStatus(row.id, next)
  feedback.success(next === 'enabled' ? '已启用' : '已停用')
  await load()
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
        <t-input v-model="keyword" clearable placeholder="搜索名称或 Entity ID" @enter="search" />
        <t-button theme="primary" variant="base" @click="search">查询</t-button>
        <t-button theme="primary" variant="outline" @click="openCreate">注册 SP</t-button>
      </div>
      <PageStatus :status="pageStatus" :error-message="errorMessage" description="还没有登记 SAML 服务提供方" @retry="load">
        <t-table row-key="id" :data="items" :columns="columns" hover>
          <template #status="{ row }">
            <t-tag :theme="row.status === 'enabled' ? 'success' : 'default'" variant="light" size="small">
              {{ statusLabel(row.status) }}
            </t-tag>
          </template>
          <template #allow_idp_initiated="{ row }">
            <t-tag :theme="row.allow_idp_initiated ? 'warning' : 'default'" variant="light" size="small">
              {{ row.allow_idp_initiated ? '已开启' : '关闭（默认）' }}
            </t-tag>
          </template>
          <template #updated_at="{ row }">{{ formatTime(row.updated_at) }}</template>
          <template #ops="{ row }">
            <t-button variant="text" @click="toggleStatus(row)">
              {{ row.status === 'enabled' ? '停用' : '启用' }}
            </t-button>
          </template>
        </t-table>
        <div class="pager">
          <t-pagination v-model="page" v-model:page-size="pageSize" :total="total" show-page-size @change="load" @page-size-change="load" />
        </div>
      </PageStatus>
    </t-card>

    <t-dialog v-model:visible="dialogVisible" header="注册 SAML 服务提供方" width="640px" :confirm-btn="{ loading: saving }" @confirm="save">
      <t-form label-width="140px">
        <t-form-item label="名称" required><t-input v-model="form.name" /></t-form-item>
        <t-form-item label="Entity ID" required><t-input v-model="form.entity_id" /></t-form-item>
        <t-form-item label="ACS URL" required>
          <t-textarea v-model="form.acs_urls_text" placeholder="每行一个精确 ACS URL" />
        </t-form-item>
        <t-form-item label="IdP-initiated">
          <t-switch v-model="form.allow_idp_initiated" />
          <span class="hint">默认关闭，需管理员明确开启</span>
        </t-form-item>
      </t-form>
    </t-dialog>
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
.hint {
  margin-left: 8px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
}
</style>
