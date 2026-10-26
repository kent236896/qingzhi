<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import {
  createOAuthClient,
  listOAuthClients,
  rotateOAuthClientSecret,
  setOAuthClientStatus,
  statusLabel,
  type OAuthClient,
} from '@/api/sso'
import { formatTime } from '@/api/security'
import PageStatus from '@/components/PageStatus.vue'
import { feedback } from '@/utils/feedback'
import { ApiError } from '@/utils/http'
import { DialogPlugin } from 'tdesign-vue-next'

const pageStatus = ref<'loading' | 'error' | 'ready'>('loading')
const errorMessage = ref('')
const items = ref<OAuthClient[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const keyword = ref('')
const dialogVisible = ref(false)
const secretOnce = ref('')
const saving = ref(false)
const form = reactive({
  name: '',
  description: '',
  client_type: 'confidential',
  status: 'disabled',
  redirect_uris_text: '',
  scopes_text: 'openid',
  pkce_policy: 'required',
})

const columns = [
  { colKey: 'name', title: '名称', width: 160 },
  { colKey: 'client_id', title: 'Client ID', width: 220 },
  { colKey: 'client_type', title: '类型', width: 100 },
  { colKey: 'status', title: '状态', width: 90 },
  { colKey: 'pkce_policy', title: 'PKCE', width: 100 },
  { colKey: 'client_secret_hint', title: 'Secret', width: 120 },
  { colKey: 'updated_at', title: '更新时间', width: 180 },
  { colKey: 'ops', title: '操作', width: 200 },
]

async function load() {
  pageStatus.value = 'loading'
  errorMessage.value = ''
  try {
    const response = await listOAuthClients({
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value.trim() || undefined,
    })
    items.value = response.data.items
    total.value = response.data.total
    pageStatus.value = 'ready'
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '无法加载 OAuth 客户端'
    pageStatus.value = 'error'
  }
}

function openCreate() {
  Object.assign(form, {
    name: '',
    description: '',
    client_type: 'confidential',
    status: 'disabled',
    redirect_uris_text: '',
    scopes_text: 'openid',
    pkce_policy: 'required',
  })
  secretOnce.value = ''
  dialogVisible.value = true
}

async function save() {
  saving.value = true
  try {
    const response = await createOAuthClient({
      name: form.name,
      description: form.description,
      client_type: form.client_type,
      status: form.status,
      redirect_uris: form.redirect_uris_text
        .split('\n')
        .map((v) => v.trim())
        .filter(Boolean),
      scopes: form.scopes_text
        .split(/[,\s]+/)
        .map((v) => v.trim())
        .filter(Boolean),
      pkce_policy: form.client_type === 'public' ? 'required' : form.pkce_policy,
      grant_types: ['authorization_code'],
    })
    secretOnce.value = response.data.client_secret || ''
    feedback.success('客户端已创建')
    if (!secretOnce.value) {
      dialogVisible.value = false
    }
    await load()
  } catch (error) {
    if (!(error instanceof ApiError)) {
      feedback.error('创建失败')
    }
  } finally {
    saving.value = false
  }
}

async function rotate(row: OAuthClient) {
  const dialog = DialogPlugin.confirm({
    header: '轮换 Client Secret',
    body: '旧密钥将失效。新密钥仅显示一次，请立即保存。',
    onConfirm: async () => {
      try {
        const response = await rotateOAuthClientSecret(row.id)
        secretOnce.value = response.data.client_secret || ''
        dialogVisible.value = true
        feedback.success('密钥已轮换')
        await load()
      } finally {
        dialog.destroy()
      }
    },
  })
}

async function toggleStatus(row: OAuthClient) {
  const next = row.status === 'enabled' ? 'disabled' : 'enabled'
  await setOAuthClientStatus(row.id, next)
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
        <t-input v-model="keyword" clearable placeholder="搜索名称或 Client ID" @enter="search" />
        <t-button theme="primary" variant="base" @click="search">查询</t-button>
        <t-button theme="primary" variant="outline" @click="openCreate">新建客户端</t-button>
      </div>
      <PageStatus :status="pageStatus" :error-message="errorMessage" description="还没有注册 OAuth 客户端" @retry="load">
        <t-table row-key="id" :data="items" :columns="columns" hover>
          <template #client_type="{ row }">{{ row.client_type === 'public' ? '公开' : '机密' }}</template>
          <template #status="{ row }">
            <t-tag :theme="row.status === 'enabled' ? 'success' : 'default'" variant="light" size="small">
              {{ statusLabel(row.status) }}
            </t-tag>
          </template>
          <template #client_secret_hint="{ row }">
            {{ row.client_secret_set ? row.client_secret_hint || '已配置' : '—' }}
          </template>
          <template #updated_at="{ row }">{{ formatTime(row.updated_at) }}</template>
          <template #ops="{ row }">
            <t-button variant="text" @click="toggleStatus(row)">
              {{ row.status === 'enabled' ? '停用' : '启用' }}
            </t-button>
            <t-button v-if="row.client_type === 'confidential'" variant="text" @click="rotate(row)">轮换密钥</t-button>
          </template>
        </t-table>
        <div class="pager">
          <t-pagination v-model="page" v-model:page-size="pageSize" :total="total" show-page-size @change="load" @page-size-change="load" />
        </div>
      </PageStatus>
    </t-card>

    <t-dialog
      v-model:visible="dialogVisible"
      :header="secretOnce ? '请保存 Client Secret' : '新建 OAuth / OIDC 客户端'"
      width="640px"
      :confirm-btn="secretOnce ? null : { loading: saving }"
      :cancel-btn="secretOnce ? { content: '关闭' } : undefined"
      @confirm="save"
    >
      <t-alert v-if="secretOnce" theme="warning" message="Client Secret 仅显示一次，离开后无法再查看完整值。" />
      <t-input v-if="secretOnce" :value="secretOnce" readonly class="secret-once" />
      <t-form v-else label-width="120px">
        <t-form-item label="名称" required><t-input v-model="form.name" /></t-form-item>
        <t-form-item label="描述"><t-textarea v-model="form.description" /></t-form-item>
        <t-form-item label="类型" required>
          <t-radio-group v-model="form.client_type">
            <t-radio value="confidential">机密客户端</t-radio>
            <t-radio value="public">公开客户端</t-radio>
          </t-radio-group>
        </t-form-item>
        <t-form-item label="Redirect URI" required>
          <t-textarea v-model="form.redirect_uris_text" placeholder="每行一个精确 URL，禁止通配符" />
        </t-form-item>
        <t-form-item label="Scopes"><t-input v-model="form.scopes_text" /></t-form-item>
        <t-form-item v-if="form.client_type === 'confidential'" label="PKCE">
          <t-select
            v-model="form.pkce_policy"
            :options="[
              { label: '强制', value: 'required' },
              { label: '可选', value: 'optional' },
              { label: '禁止', value: 'forbidden' },
            ]"
          />
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
.secret-once {
  margin-top: 12px;
}
</style>
