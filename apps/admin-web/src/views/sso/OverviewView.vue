<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  browserFrontendBase,
  getSsoOverview,
  getSsoSettings,
  protocolLabel,
  resultLabel,
  updateSsoSettings,
  type SsoOverview,
  type SsoSettings,
} from '@/api/sso'
import { formatTime } from '@/api/security'
import PageStatus from '@/components/PageStatus.vue'
import { feedback } from '@/utils/feedback'
import { ApiError } from '@/utils/http'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const auth = useAuthStore()
const canManage = computed(() => auth.hasPermission('system:sso:provider:manage'))
const pageStatus = ref<'loading' | 'error' | 'ready'>('loading')
const errorMessage = ref('')
const overview = ref<SsoOverview | null>(null)
const settings = ref<SsoSettings | null>(null)
const saving = ref(false)
const readyOverview = computed(() => (pageStatus.value === 'ready' ? overview.value : null))
const form = reactive({
  local_login_enabled: true,
  frontend_base_url: '',
  default_tenant_id: '',
})

const failureColumns = [
  { colKey: 'create_time', title: '时间', width: 180 },
  { colKey: 'protocol', title: '协议', width: 100 },
  { colKey: 'event_type', title: '事件', width: 160 },
  { colKey: 'message', title: '说明', ellipsis: true },
  { colKey: 'request_id', title: 'Request ID', width: 160 },
]

const certColumns = [
  { colKey: 'name', title: '名称' },
  { colKey: 'kind', title: '类型', width: 120 },
  { colKey: 'not_after', title: '到期时间', width: 180 },
  { colKey: 'days_left', title: '剩余天数', width: 100 },
  { colKey: 'expired', title: '状态', width: 100 },
]

function applySettings(item: SsoSettings) {
  settings.value = item
  form.local_login_enabled = item.local_login_enabled
  form.frontend_base_url = item.frontend_base_url || browserFrontendBase()
  form.default_tenant_id = item.default_tenant_id || ''
}

async function load() {
  pageStatus.value = 'loading'
  errorMessage.value = ''
  try {
    const [overviewRes, settingsRes] = await Promise.all([getSsoOverview(), getSsoSettings()])
    // TDesign Table crashes on null data (Go nil slices JSON-encode as null) and can stick the router-view.
    overview.value = {
      ...overviewRes.data,
      recent_protocol_failures: overviewRes.data.recent_protocol_failures ?? [],
      certificate_warnings: overviewRes.data.certificate_warnings ?? [],
    }
    applySettings(settingsRes.data)
    pageStatus.value = 'ready'
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '无法加载认证概览'
    pageStatus.value = 'error'
  }
}

function useBrowserOrigin() {
  form.frontend_base_url = browserFrontendBase()
}

async function saveSettings() {
  if (!canManage.value || saving.value) {
    return
  }
  saving.value = true
  try {
    const response = await updateSsoSettings({
      local_login_enabled: form.local_login_enabled,
      frontend_base_url: form.frontend_base_url.trim(),
      default_tenant_id: form.default_tenant_id.trim(),
    })
    applySettings(response.data)
    if (overview.value) {
      overview.value.local_login_enabled = response.data.local_login_enabled
    }
    feedback.success('认证策略已保存')
  } catch {
    // HTTP client already shows the server message.
  } finally {
    saving.value = false
  }
}

function go(path: string) {
  void router.push(path)
}

onMounted(() => {
  void load()
})
</script>

<template>
  <div class="page-grid sso-overview">
    <PageStatus
      v-if="!readyOverview"
      :status="pageStatus === 'ready' ? 'loading' : pageStatus"
      :error-message="errorMessage"
      @retry="load"
    />
    <div v-else class="sso-overview__ready">
        <div class="monitor-toolbar">
          <span>数据来自后端实时统计，不含模拟值</span>
          <span>采集时间 {{ formatTime(readyOverview?.collected_at) }}</span>
        </div>

        <t-card :bordered="false" title="认证策略">
          <div class="settings-form">
            <div class="settings-grid">
              <div class="settings-field">
                <label>本地账号登录</label>
                <t-switch v-model="form.local_login_enabled" :disabled="!canManage" />
                <p class="muted">关闭后仅保留外部身份登录（请先确认外部登录可用）</p>
              </div>
              <div class="settings-field">
                <label>前端基址（OIDC 回调回跳）</label>
                <div class="inline-field">
                  <t-input
                    v-model="form.frontend_base_url"
                    :disabled="!canManage"
                    placeholder="例如 http://localhost:5173"
                  />
                  <t-button variant="outline" :disabled="!canManage" @click="useBrowserOrigin">
                    使用当前浏览器地址
                  </t-button>
                </div>
                <p class="muted">
                  默认读取当前浏览器地址。登录页也会自动携带该地址；须与 CORS 白名单一致。
                </p>
              </div>
              <div class="settings-field">
                <label>外部自动开户默认租户 ID</label>
                <t-input
                  v-model="form.default_tenant_id"
                  :disabled="!canManage"
                  placeholder="可留空，使用最早可用租户"
                />
              </div>
            </div>
            <t-button
              v-if="canManage"
              theme="primary"
              :loading="saving"
              @click="saveSettings"
            >
              保存策略
            </t-button>
            <p v-else class="muted">当前账号无权限修改认证策略</p>
            <p v-if="settings?.updated_at" class="muted">
              最近更新 {{ formatTime(settings.updated_at) }}
              <template v-if="settings.updater"> · {{ settings.updater }}</template>
            </p>
          </div>
        </t-card>

        <div class="metric-grid">
          <t-card :bordered="false" title="本地认证">
            <t-tag :theme="readyOverview?.local_login_enabled ? 'success' : 'warning'" variant="light">
              {{ readyOverview?.local_login_enabled ? '已启用（应急访问）' : '已关闭' }}
            </t-tag>
            <p class="muted">本平台仅作为 OIDC / SAML 客户端登录</p>
          </t-card>
          <t-card :bordered="false" title="外部 OIDC" class="clickable" @click="go('/security/sso/providers')">
            <strong class="metric-value">{{ readyOverview?.oidc_providers.enabled }}</strong>
            <span class="muted"> / {{ readyOverview?.oidc_providers.total }} 已配置</span>
          </t-card>
          <t-card :bordered="false" title="外部 SAML IdP" class="clickable" @click="go('/security/sso/providers')">
            <strong class="metric-value">{{ readyOverview?.saml_providers.enabled }}</strong>
            <span class="muted"> / {{ readyOverview?.saml_providers.total }} 已配置</span>
          </t-card>
          <t-card :bordered="false" title="身份映射" class="clickable" @click="go('/security/sso/identity-links')">
            <strong class="metric-value">{{ readyOverview?.identity_links }}</strong>
          </t-card>
        </div>

        <div class="metric-grid metric-grid-2">
          <t-card :bordered="false" title="近 24 小时登录" class="clickable" @click="go('/security/sso/auth-events')">
            <div class="login-stats">
              <span>成功 {{ readyOverview?.recent_login_success }}</span>
              <span>失败 {{ readyOverview?.recent_login_failure }}</span>
            </div>
          </t-card>
        </div>

        <t-card :bordered="false" title="最近协议错误">
          <t-table
            row-key="id"
            :data="readyOverview?.recent_protocol_failures ?? []"
            :columns="failureColumns"
            :empty="'暂无协议失败事件'"
            hover
          >
            <template #create_time="{ row }">{{ formatTime(row.create_time) }}</template>
            <template #protocol="{ row }">{{ protocolLabel(row.protocol) }}</template>
            <template #message="{ row }">
              <t-tag theme="danger" variant="light" size="small">{{ resultLabel(row.result) }}</t-tag>
              {{ row.message }}
            </template>
          </t-table>
        </t-card>

        <t-card :bordered="false" title="证书到期提醒（30 天内）">
          <t-table
            row-key="ref_id"
            :data="readyOverview?.certificate_warnings ?? []"
            :columns="certColumns"
            :empty="'暂无证书到期提醒'"
            hover
          >
            <template #not_after="{ row }">
              {{ row.not_after ? formatTime(row.not_after) : '—' }}
            </template>
            <template #days_left="{ row }">
              {{ row.days_left ?? '—' }}
            </template>
            <template #expired="{ row }">
              <t-tag :theme="row.expired ? 'danger' : 'warning'" variant="light" size="small">
                {{ row.expired ? '已过期' : '即将到期' }}
              </t-tag>
            </template>
          </t-table>
        </t-card>
    </div>
  </div>
</template>

<style scoped>
.monitor-toolbar {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  color: var(--td-text-color-secondary);
  font-size: 13px;
}
.settings-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.settings-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px 24px;
}
.settings-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}
.settings-field > label {
  font-size: 14px;
  color: var(--td-text-color-primary);
}
.inline-field {
  display: flex;
  gap: 8px;
  width: 100%;
}
.inline-field :deep(.t-input) {
  flex: 1;
}
.metric-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
}
.metric-grid-2 {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}
.metric-value {
  font-size: 28px;
  line-height: 1.2;
}
.muted {
  color: var(--td-text-color-secondary);
  font-size: 13px;
  margin: 4px 0 0;
}
.clickable {
  cursor: pointer;
}
.login-stats {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 15px;
}
@media (max-width: 1100px) {
  .metric-grid,
  .settings-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
@media (max-width: 640px) {
  .metric-grid,
  .settings-grid {
    grid-template-columns: 1fr;
  }
  .inline-field {
    flex-direction: column;
  }
}
</style>
