<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import PageStatus from '@/components/PageStatus.vue'
import {
  createTenant,
  listTenants,
  statusLabel,
  updateTenant,
  type RecordStatus,
  type Tenant,
} from '@/api/identity'
import { ApiError } from '@/utils/http'
import { feedback } from '@/utils/feedback'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const canCreate = computed(() => auth.hasPermission('system:tenant:create'))
const canUpdate = computed(() => auth.hasPermission('system:tenant:update'))

const pageStatus = ref<'loading' | 'error' | 'ready'>('loading')
const errorMessage = ref('')
const items = ref<Tenant[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const keyword = ref('')

const editorVisible = ref(false)
const saving = ref(false)
const editing = ref<Tenant | null>(null)
const form = ref({
  name: '',
  code: '',
  contactName: '',
  contactMobile: '',
  accountCount: 0,
  status: 0 as RecordStatus,
})

const columns = [
  { colKey: 'name', title: '名称' },
  { colKey: 'code', title: '编码' },
  { colKey: 'contact_name', title: '联系人' },
  { colKey: 'contact_mobile', title: '联系手机' },
  { colKey: 'account_count', title: '账号数量' },
  { colKey: 'status', title: '状态' },
  { colKey: 'update_time', title: '更新时间' },
  { colKey: 'actions', title: '操作', width: 120 },
]

async function load() {
  pageStatus.value = 'loading'
  errorMessage.value = ''
  try {
    const response = await listTenants({
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value.trim() || undefined,
    })
    items.value = response.data.items
    total.value = response.data.total
    pageStatus.value = 'ready'
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '无法加载租户'
    pageStatus.value = 'error'
  }
}

function search() {
  page.value = 1
  void load()
}

function openCreate() {
  editing.value = null
  form.value = { name: '', code: '', contactName: '', contactMobile: '', accountCount: 0, status: 0 }
  editorVisible.value = true
}

function openEdit(row: Tenant) {
  editing.value = row
  form.value = {
    name: row.name,
    code: row.code,
    contactName: row.contact_name,
    contactMobile: row.contact_mobile,
    accountCount: row.account_count,
    status: row.status,
  }
  editorVisible.value = true
}

async function save() {
  const name = form.value.name.trim()
  const code = form.value.code.trim()
  const contactName = form.value.contactName.trim()
  if (!name || !contactName || (!editing.value && !code)) {
    feedback.warning('请填写名称、编码和联系人')
    return
  }
  saving.value = true
  try {
    if (editing.value) {
      await updateTenant(editing.value.id, {
        name,
        contact_name: contactName,
        contact_mobile: form.value.contactMobile.trim(),
        account_count: Number(form.value.accountCount) || 0,
        status: form.value.status,
      })
      feedback.success('租户已更新')
    } else {
      await createTenant({
        name,
        code,
        contact_name: contactName,
        contact_mobile: form.value.contactMobile.trim(),
        account_count: Number(form.value.accountCount) || 0,
      })
      feedback.success('租户已创建')
    }
    editorVisible.value = false
    await load()
  } catch {
    // The HTTP client already shows the server message.
  } finally {
    saving.value = false
  }
}

function onPageChange() {
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
        <t-input v-model="keyword" clearable placeholder="搜索名称或编码" @enter="search" />
        <t-button theme="default" variant="outline" @click="search">查询</t-button>
        <t-button v-if="canCreate" theme="primary" variant="base" @click="openCreate">新建租户</t-button>
      </div>
      <PageStatus :status="pageStatus" :error-message="errorMessage" @retry="load">
        <t-table row-key="id" :data="items" :columns="columns" hover>
          <template #status="{ row }">
            <t-tag :theme="row.status === 0 ? 'success' : 'warning'" variant="light">
              {{ statusLabel(row.status) }}
            </t-tag>
          </template>
          <template #actions="{ row }">
            <t-button v-if="canUpdate" variant="text" @click="openEdit(row)">编辑</t-button>
          </template>
        </t-table>
        <t-pagination
          v-model="page"
          v-model:page-size="pageSize"
          class="pager"
          :total="total"
          @change="onPageChange"
        />
      </PageStatus>
    </t-card>

    <t-dialog
      v-model:visible="editorVisible"
      :header="editing ? '编辑租户' : '新建租户'"
      :confirm-btn="{ content: '保存', loading: saving }"
      @confirm="save"
    >
      <t-form label-align="top">
        <t-form-item label="名称">
          <t-input v-model="form.name" maxlength="30" />
        </t-form-item>
        <t-form-item label="编码">
          <t-input v-model="form.code" :disabled="Boolean(editing)" maxlength="64" />
        </t-form-item>
        <t-form-item label="联系人">
          <t-input v-model="form.contactName" maxlength="30" />
        </t-form-item>
        <t-form-item label="联系手机">
          <t-input v-model="form.contactMobile" maxlength="500" />
        </t-form-item>
        <t-form-item label="账号数量">
          <t-input-number v-model="form.accountCount" :min="0" />
        </t-form-item>
        <t-form-item v-if="editing" label="状态">
          <t-radio-group v-model="form.status">
            <t-radio :value="0">正常</t-radio>
            <t-radio :value="1">停用</t-radio>
          </t-radio-group>
        </t-form-item>
      </t-form>
    </t-dialog>
  </div>
</template>
