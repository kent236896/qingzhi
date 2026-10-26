<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import {
  createSsoProvider,
  listSsoProviders,
  protocolLabel,
  setSsoProviderStatus,
  statusLabel,
  type SsoProvider,
} from '@/api/sso'
import { formatTime } from '@/api/security'
import PageStatus from '@/components/PageStatus.vue'
import { feedback } from '@/utils/feedback'
import { ApiError } from '@/utils/http'

const pageStatus = ref<'loading' | 'error' | 'ready'>('loading')
const errorMessage = ref('')
const items = ref<SsoProvider[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const keyword = ref('')
const protocol = ref('')
const dialogVisible = ref(false)
const saving = ref(false)
const form = reactive({
  name: '',
  protocol: 'oidc',
  status: 'disabled',
  issuer: '',
  client_id: '',
  client_secret: '',
  redirect_uri: '',
  scopes: 'openid profile email',
  authorization_url: '',
  token_url: '',
  userinfo_url: '',
  entity_id: '',
  sso_url: '',
  metadata_url: '',
  auto_create_user: false,
  display_on_login: true,
})

const columns = [
  { colKey: 'name', title: '名称', width: 160 },
  { colKey: 'protocol', title: '协议', width: 100 },
  { colKey: 'issuer', title: 'Issuer / Entity', ellipsis: true },
  { colKey: 'status', title: '状态', width: 90 },
  { colKey: 'priority', title: '优先级', width: 90 },
  { colKey: 'last_success_at', title: '最近成功', width: 180 },
  { colKey: 'ops', title: '操作', width: 160 },
]

const protocolOptions = [
  { label: '全部协议', value: '' },
  { label: 'OIDC', value: 'oidc' },
  { label: 'OAuth 2.0', value: 'oauth2' },
  { label: 'SAML 2.0', value: 'saml' },
]

async function load() {
  pageStatus.value = 'loading'
  errorMessage.value = ''
  try {
    const response = await listSsoProviders({
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value.trim() || undefined,
      protocol: protocol.value || undefined,
    })
    items.value = response.data.items
    total.value = response.data.total
    pageStatus.value = 'ready'
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '无法加载身份提供方'
    pageStatus.value = 'error'
  }
}

function openCreate() {
  Object.assign(form, {
    name: '',
    protocol: 'oidc',
    status: 'disabled',
    issuer: '',
    client_id: '',
    client_secret: '',
    redirect_uri: '',
    scopes: 'openid profile email',
    authorization_url: '',
    token_url: '',
    userinfo_url: '',
    entity_id: '',
    sso_url: '',
    metadata_url: '',
    auto_create_user: false,
    display_on_login: true,
  })
  dialogVisible.value = true
}

async function save() {
  saving.value = true
  try {
    await createSsoProvider({ ...form })
    feedback.success('已创建身份提供方')
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

async function toggleStatus(row: SsoProvider) {
  const next = row.status === 'enabled' ? 'disabled' : 'enabled'
  try {
    await setSsoProviderStatus(row.id, next)
    feedback.success(next === 'enabled' ? '已启用' : '已停用')
    await load()
  } catch {
    /* http client already feedbacks */
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
        <t-input v-model="keyword" clearable placeholder="搜索名称、Issuer 或 Client ID" @enter="search" />
        <t-select v-model="protocol" :options="protocolOptions" />
        <t-button theme="primary" variant="base" @click="search">查询</t-button>
        <t-button theme="primary" variant="outline" @click="openCreate">新建</t-button>
      </div>
      <PageStatus :status="pageStatus" :error-message="errorMessage" description="还没有外部身份提供方" @retry="load">
        <t-table row-key="id" :data="items" :columns="columns" hover>
          <template #protocol="{ row }">{{ protocolLabel(row.protocol) }}</template>
          <template #issuer="{ row }">{{ row.issuer || row.entity_id || '—' }}</template>
          <template #status="{ row }">
            <t-tag :theme="row.status === 'enabled' ? 'success' : 'default'" variant="light" size="small">
              {{ statusLabel(row.status) }}
            </t-tag>
          </template>
          <template #last_success_at="{ row }">
            {{ row.last_success_at ? formatTime(row.last_success_at) : '—' }}
          </template>
          <template #ops="{ row }">
            <t-button variant="text" @click="toggleStatus(row)">
              {{ row.status === 'enabled' ? '停用' : '启用' }}
            </t-button>
          </template>
        </t-table>
        <div class="pager">
          <t-pagination
            v-model="page"
            v-model:page-size="pageSize"
            :total="total"
            show-page-size
            @change="load"
            @page-size-change="load"
          />
        </div>
      </PageStatus>
    </t-card>

    <t-dialog v-model:visible="dialogVisible" header="新建外部身份提供方" width="720px" :confirm-btn="{ loading: saving }" @confirm="save">
      <t-form label-width="140px">
        <t-form-item label="名称" required>
          <t-input v-model="form.name" />
        </t-form-item>
        <t-form-item label="协议" required>
          <t-radio-group v-model="form.protocol">
            <t-radio value="oidc">OIDC</t-radio>
            <t-radio value="oauth2">OAuth 2.0</t-radio>
            <t-radio value="saml">SAML 2.0</t-radio>
          </t-radio-group>
        </t-form-item>

        <template v-if="form.protocol === 'oidc'">
          <t-form-item label="Issuer" required><t-input v-model="form.issuer" placeholder="https://idp.example.com" /></t-form-item>
          <t-form-item label="Client ID" required><t-input v-model="form.client_id" /></t-form-item>
          <t-form-item label="Client Secret"><t-input v-model="form.client_secret" type="password" /></t-form-item>
          <t-form-item label="Redirect URI"><t-input v-model="form.redirect_uri" /></t-form-item>
          <t-form-item label="Scopes"><t-input v-model="form.scopes" /></t-form-item>
        </template>

        <template v-else-if="form.protocol === 'oauth2'">
          <t-form-item label="Authorization URL" required><t-input v-model="form.authorization_url" /></t-form-item>
          <t-form-item label="Token URL" required><t-input v-model="form.token_url" /></t-form-item>
          <t-form-item label="UserInfo URL" required><t-input v-model="form.userinfo_url" /></t-form-item>
          <t-form-item label="Client ID" required><t-input v-model="form.client_id" /></t-form-item>
          <t-form-item label="Client Secret"><t-input v-model="form.client_secret" type="password" /></t-form-item>
          <t-form-item label="Redirect URI"><t-input v-model="form.redirect_uri" /></t-form-item>
        </template>

        <template v-else>
          <t-form-item label="IdP Entity ID" required><t-input v-model="form.entity_id" /></t-form-item>
          <t-form-item label="SSO URL"><t-input v-model="form.sso_url" /></t-form-item>
          <t-form-item label="Metadata URL"><t-input v-model="form.metadata_url" /></t-form-item>
        </template>

        <t-form-item label="登录页显示">
          <t-switch v-model="form.display_on_login" />
        </t-form-item>
        <t-form-item label="自动开户">
          <t-switch v-model="form.auto_create_user" />
          <span class="hint">默认关闭；开启后新用户仅最小权限</span>
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
