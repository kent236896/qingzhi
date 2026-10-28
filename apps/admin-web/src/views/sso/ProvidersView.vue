<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import {
  createSsoProvider,
  getSsoProvider,
  listSsoProviders,
  protocolLabel,
  setSsoProviderStatus,
  statusLabel,
  updateSsoProvider,
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
const loadingDetail = ref(false)
const editingId = ref<string | null>(null)
const secretHint = ref('')
const secretSet = ref(false)

const emptyForm = () => ({
  name: '',
  protocol: 'oidc',
  status: 'enabled',
  issuer: '',
  client_id: '',
  client_secret: '',
  redirect_uri: '',
  scopes: 'openid profile email',
  entity_id: '',
  sso_url: '',
  metadata_url: '',
  idp_certificate_pem: '',
  acs_url: 'http://127.0.0.1:8080/api/v1/sso/saml/acs',
  sp_entity_id: '',
  auto_create_user: false,
  display_on_login: true,
})

const form = reactive(emptyForm())
const isEditing = computed(() => editingId.value !== null)
const dialogHeader = computed(() => (isEditing.value ? '编辑外部身份提供方' : '新建外部身份提供方'))

const columns = [
  { colKey: 'name', title: '名称', width: 160 },
  { colKey: 'protocol', title: '协议', width: 100 },
  { colKey: 'issuer', title: 'Issuer / Entity', ellipsis: true },
  { colKey: 'status', title: '状态', width: 90 },
  { colKey: 'priority', title: '优先级', width: 90 },
  { colKey: 'last_success_at', title: '最近成功', width: 180 },
  { colKey: 'ops', title: '操作', width: 200 },
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

function fillForm(row: SsoProvider, keepSecretBlank = true) {
  Object.assign(form, {
    name: row.name,
    protocol: row.protocol || 'oidc',
    status: row.status || 'disabled',
    issuer: row.issuer || '',
    client_id: row.client_id || '',
    client_secret: keepSecretBlank ? '' : form.client_secret,
    redirect_uri: row.redirect_uri || '',
    scopes: row.scopes || 'openid profile email',
    entity_id: row.entity_id || '',
    sso_url: row.sso_url || '',
    metadata_url: row.metadata_url || '',
    idp_certificate_pem: row.idp_certificate_pem || '',
    acs_url: row.acs_url || 'http://127.0.0.1:8080/api/v1/sso/saml/acs',
    sp_entity_id: row.sp_entity_id || '',
    auto_create_user: row.auto_create_user,
    display_on_login: row.display_on_login,
  })
  secretSet.value = row.client_secret_set
  secretHint.value = row.client_secret_hint || ''
}

function openCreate() {
  editingId.value = null
  secretSet.value = false
  secretHint.value = ''
  Object.assign(form, emptyForm())
  dialogVisible.value = true
}

async function openEdit(row: SsoProvider) {
  editingId.value = row.id
  loadingDetail.value = true
  dialogVisible.value = true
  fillForm(row)
  try {
    const response = await getSsoProvider(row.id)
    fillForm(response.data)
  } catch (error) {
    if (!(error instanceof ApiError)) {
      feedback.error('无法加载提供方详情')
    }
  } finally {
    loadingDetail.value = false
  }
}

async function save() {
  if (!form.name.trim()) {
    feedback.warning('请填写名称')
    return
  }
  if (form.protocol === 'oidc') {
    if (!form.issuer.trim() || !form.client_id.trim()) {
      feedback.warning('请填写 Issuer 与 Client ID')
      return
    }
  } else if (!form.entity_id.trim()) {
    feedback.warning('请填写 IdP Entity ID')
    return
  }

  saving.value = true
  try {
    const body: Record<string, unknown> = { ...form }
    if (isEditing.value && !String(form.client_secret || '').trim()) {
      delete body.client_secret
    }
    if (isEditing.value && editingId.value) {
      await updateSsoProvider(editingId.value, body)
      feedback.success('已更新身份提供方')
    } else {
      await createSsoProvider(body)
      feedback.success('已创建身份提供方')
    }
    dialogVisible.value = false
    await load()
  } catch (error) {
    if (!(error instanceof ApiError)) {
      feedback.error(isEditing.value ? '更新失败' : '创建失败')
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
            <t-button variant="text" @click="openEdit(row)">编辑</t-button>
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

    <t-dialog
      v-model:visible="dialogVisible"
      :header="dialogHeader"
      width="720px"
      :confirm-btn="{ loading: saving || loadingDetail, disabled: loadingDetail }"
      @confirm="save"
    >
      <t-loading :loading="loadingDetail" size="small">
        <t-form label-width="140px">
          <t-form-item label="名称" required>
            <t-input v-model="form.name" />
          </t-form-item>
          <t-form-item label="协议" required>
            <t-radio-group v-model="form.protocol" :disabled="isEditing">
              <t-radio value="oidc">OIDC</t-radio>
              <t-radio value="saml">SAML 2.0</t-radio>
            </t-radio-group>
            <span v-if="isEditing" class="hint">协议创建后不可更改</span>
          </t-form-item>

          <template v-if="form.protocol === 'oidc'">
            <t-form-item label="Issuer" required>
              <t-input v-model="form.issuer" placeholder="https://idp.example.com/realms/xxx" />
            </t-form-item>
            <t-form-item label="Client ID" required><t-input v-model="form.client_id" /></t-form-item>
            <t-form-item label="Client Secret">
              <t-input
                v-model="form.client_secret"
                type="password"
                :placeholder="isEditing ? (secretSet ? `已配置 ${secretHint || '****'}，留空表示不修改` : '未配置，可在此填写') : ''"
              />
            </t-form-item>
            <t-form-item label="Redirect URI">
              <t-input v-model="form.redirect_uri" placeholder="http://127.0.0.1:8080/api/v1/sso/oidc/callback" />
            </t-form-item>
            <t-form-item label="Scopes"><t-input v-model="form.scopes" /></t-form-item>
          </template>

          <template v-else>
            <t-form-item label="IdP Entity ID" required><t-input v-model="form.entity_id" /></t-form-item>
            <t-form-item label="Metadata URL">
              <t-input v-model="form.metadata_url" placeholder="优先填写，可自动发现 SSO 与证书" />
            </t-form-item>
            <t-form-item label="SSO URL"><t-input v-model="form.sso_url" placeholder="无 Metadata 时必填" /></t-form-item>
            <t-form-item label="IdP 证书 PEM">
              <t-textarea v-model="form.idp_certificate_pem" placeholder="无 Metadata 时必填" :autosize="{ minRows: 3, maxRows: 8 }" />
            </t-form-item>
            <t-form-item label="本平台 ACS URL">
              <t-input v-model="form.acs_url" placeholder="http://127.0.0.1:8080/api/v1/sso/saml/acs" />
            </t-form-item>
            <t-form-item label="本平台 SP Entity ID">
              <t-input v-model="form.sp_entity_id" placeholder="可选，默认等于 ACS URL" />
            </t-form-item>
          </template>

          <t-form-item :label="isEditing ? '启用' : '立即启用'">
            <t-switch
              :value="form.status === 'enabled'"
              @change="(v: boolean) => (form.status = v ? 'enabled' : 'disabled')"
            />
            <span class="hint">停用时登录页不会展示，即使已开「登录页显示」</span>
          </t-form-item>
          <t-form-item label="登录页显示">
            <t-switch v-model="form.display_on_login" />
          </t-form-item>
          <t-form-item label="自动开户">
            <t-switch v-model="form.auto_create_user" />
            <span class="hint">默认关闭；开启后新用户仅最小权限</span>
          </t-form-item>
        </t-form>
      </t-loading>
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
