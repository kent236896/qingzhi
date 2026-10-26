<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  getSsoOverview,
  protocolLabel,
  resultLabel,
  type SsoOverview,
} from '@/api/sso'
import { formatTime } from '@/api/security'
import PageStatus from '@/components/PageStatus.vue'
import { ApiError } from '@/utils/http'

const router = useRouter()
const pageStatus = ref<'loading' | 'error' | 'ready'>('loading')
const errorMessage = ref('')
const overview = ref<SsoOverview | null>(null)

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

async function load() {
  pageStatus.value = 'loading'
  errorMessage.value = ''
  try {
    const response = await getSsoOverview()
    overview.value = response.data
    pageStatus.value = 'ready'
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '无法加载认证概览'
    pageStatus.value = 'error'
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
    <PageStatus :status="pageStatus" :error-message="errorMessage" @retry="load">
      <template v-if="overview">
        <div class="monitor-toolbar">
          <span>数据来自后端实时统计，不含模拟值</span>
          <span>采集时间 {{ formatTime(overview.collected_at) }}</span>
        </div>

        <div class="metric-grid">
          <t-card :bordered="false" title="本地认证">
            <t-tag :theme="overview.local_login_enabled ? 'success' : 'warning'" variant="light">
              {{ overview.local_login_enabled ? '已启用（应急访问）' : '已关闭' }}
            </t-tag>
            <p class="muted">Issuer：{{ overview.issuer || '未配置' }}</p>
          </t-card>
          <t-card :bordered="false" title="外部 OIDC" class="clickable" @click="go('/security/sso/providers')">
            <strong class="metric-value">{{ overview.oidc_providers.enabled }}</strong>
            <span class="muted"> / {{ overview.oidc_providers.total }} 已配置</span>
          </t-card>
          <t-card :bordered="false" title="外部 OAuth 2.0" class="clickable" @click="go('/security/sso/providers')">
            <strong class="metric-value">{{ overview.oauth2_providers.enabled }}</strong>
            <span class="muted"> / {{ overview.oauth2_providers.total }} 已配置</span>
          </t-card>
          <t-card :bordered="false" title="外部 SAML IdP" class="clickable" @click="go('/security/sso/providers')">
            <strong class="metric-value">{{ overview.saml_providers.enabled }}</strong>
            <span class="muted"> / {{ overview.saml_providers.total }} 已配置</span>
          </t-card>
        </div>

        <div class="metric-grid">
          <t-card :bordered="false" title="OAuth / OIDC 客户端" class="clickable" @click="go('/security/sso/oauth-clients')">
            <strong class="metric-value">{{ overview.oauth_clients.enabled }}</strong>
            <span class="muted"> / {{ overview.oauth_clients.total }} 已注册</span>
          </t-card>
          <t-card :bordered="false" title="SAML 服务提供方" class="clickable" @click="go('/security/sso/saml-sps')">
            <strong class="metric-value">{{ overview.saml_sps.enabled }}</strong>
            <span class="muted"> / {{ overview.saml_sps.total }} 已注册</span>
          </t-card>
          <t-card :bordered="false" title="身份映射" class="clickable" @click="go('/security/sso/identity-links')">
            <strong class="metric-value">{{ overview.identity_links }}</strong>
          </t-card>
          <t-card :bordered="false" title="近 24 小时登录" class="clickable" @click="go('/security/sso/auth-events')">
            <div class="login-stats">
              <span>成功 {{ overview.recent_login_success }}</span>
              <span>失败 {{ overview.recent_login_failure }}</span>
            </div>
          </t-card>
        </div>

        <t-card :bordered="false" title="最近协议错误">
          <t-table
            row-key="id"
            :data="overview.recent_protocol_failures"
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
            :data="overview.certificate_warnings"
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
      </template>
    </PageStatus>
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
.metric-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
}
.metric-value {
  font-size: 28px;
  line-height: 1.2;
}
.muted {
  color: var(--td-text-color-secondary);
  font-size: 13px;
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
  .metric-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
@media (max-width: 640px) {
  .metric-grid {
    grid-template-columns: 1fr;
  }
}
</style>
