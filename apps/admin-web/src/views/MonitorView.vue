<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import {
  alertStatusLabel,
  errorTypeLabel,
  formatBytes,
  getMonitorOverview,
  listMonitorAlerts,
  type MonitorAlert,
  type MonitorOverview,
} from '@/api/monitor'
import { formatTime } from '@/api/security'
import PageStatus from '@/components/PageStatus.vue'
import { ApiError } from '@/utils/http'

const pageStatus = ref<'loading' | 'error' | 'ready'>('loading')
const errorMessage = ref('')
const overview = ref<MonitorOverview | null>(null)
const history = ref<MonitorAlert[]>([])
const historyTotal = ref(0)
let timer = 0

const errorColumns = [
  { colKey: 'type', title: '类型', width: 120 },
  { colKey: 'count', title: '次数', width: 100 },
  { colKey: 'last_time', title: '最近发生', width: 200 },
]
const recentColumns = [
  { colKey: 'time', title: '时间', width: 180 },
  { colKey: 'method', title: '方法', width: 90 },
  { colKey: 'path', title: '路径' },
  { colKey: 'status', title: '状态', width: 90 },
  { colKey: 'duration_ms', title: '耗时', width: 100 },
]
const alertColumns = [
  { colKey: 'status', title: '状态', width: 100 },
  { colKey: 'level', title: '级别', width: 100 },
  { colKey: 'title', title: '告警', width: 180 },
  { colKey: 'message', title: '说明' },
  { colKey: 'first_time', title: '开始', width: 180 },
  { colKey: 'last_time', title: '最近', width: 180 },
]

function stateTheme(state: string) {
  return state === 'up' ? 'success' : 'danger'
}

function stateLabel(state: string) {
  return state === 'up' ? '正常' : '不可用'
}

function levelTheme(level: string) {
  return level === 'critical' ? 'danger' : 'warning'
}

async function load(initial = false) {
  if (initial) {
    pageStatus.value = 'loading'
  }
  errorMessage.value = ''
  try {
    const [current, alerts] = await Promise.all([
      getMonitorOverview(),
      listMonitorAlerts({ page: 1, page_size: 20 }),
    ])
    overview.value = current.data
    history.value = alerts.data.items
    historyTotal.value = alerts.data.total
    pageStatus.value = 'ready'
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '无法获取监控数据'
    if (initial || !overview.value) {
      pageStatus.value = 'error'
    }
  }
}

onMounted(() => {
  void load(true)
  timer = window.setInterval(() => {
    void load(false)
  }, 10000)
})

onUnmounted(() => {
  window.clearInterval(timer)
})
</script>

<template>
  <div class="page-grid monitor-page">
    <PageStatus :status="pageStatus" :error-message="errorMessage || '无法获取监控数据'" @retry="load(true)">
      <template v-if="overview">
        <div class="monitor-toolbar">
          <span>统计窗口 {{ overview.window_seconds }} 秒，每 10 秒刷新</span>
          <span>采集时间 {{ formatTime(overview.collected_at) }}</span>
        </div>

        <div class="metric-grid">
          <t-card :bordered="false" title="API 服务">
            <t-tag :theme="stateTheme(overview.service.api)" variant="light">{{ stateLabel(overview.service.api) }}</t-tag>
          </t-card>
          <t-card :bordered="false" title="数据库">
            <t-tag :theme="stateTheme(overview.service.database)" variant="light">{{ stateLabel(overview.service.database) }}</t-tag>
          </t-card>
          <t-card :bordered="false" title="Redis">
            <t-tag :theme="stateTheme(overview.service.redis)" variant="light">{{ stateLabel(overview.service.redis) }}</t-tag>
          </t-card>
          <t-card :bordered="false" title="未恢复告警">
            <strong class="metric-value">{{ overview.alerts.length }}</strong>
          </t-card>
        </div>

        <div class="metric-grid">
          <t-card :bordered="false" title="请求量">
            <strong class="metric-value">{{ overview.traffic.requests }}</strong>
          </t-card>
          <t-card :bordered="false" title="服务端错误">
            <strong class="metric-value">{{ overview.traffic.server_errors }}</strong>
          </t-card>
          <t-card :bordered="false" title="超时">
            <strong class="metric-value">{{ overview.traffic.timeouts }}</strong>
          </t-card>
          <t-card :bordered="false" title="P95 响应">
            <strong class="metric-value">{{ overview.traffic.p95_ms }} ms</strong>
            <p class="metric-note">平均 {{ overview.traffic.avg_ms }} ms，拦截 {{ overview.traffic.rejected }}，客户端错误 {{ overview.traffic.client_errors }}</p>
          </t-card>
        </div>

        <div class="metric-grid metric-grid-3">
          <t-card :bordered="false" title="CPU">
            <t-progress theme="plump" :percentage="overview.resources.cpu_percent" />
          </t-card>
          <t-card :bordered="false" title="内存">
            <t-progress theme="plump" :percentage="overview.resources.memory_percent" />
            <p class="metric-note">{{ formatBytes(overview.resources.memory_used) }} / {{ formatBytes(overview.resources.memory_total) }}</p>
          </t-card>
          <t-card :bordered="false" :title="`磁盘 ${overview.resources.disk_mount || ''}`">
            <t-progress v-if="overview.resources.available" theme="plump" :percentage="overview.resources.disk_percent" />
            <t-tag v-else theme="warning" variant="light">采集失败</t-tag>
            <p v-if="overview.resources.available" class="metric-note">{{ formatBytes(overview.resources.disk_used) }} / {{ formatBytes(overview.resources.disk_total) }}</p>
          </t-card>
        </div>

        <t-card title="错误统计" :bordered="false">
          <t-table row-key="type" :data="overview.errors ?? []" :columns="errorColumns" size="small" hover>
            <template #type="{ row }">{{ errorTypeLabel(row.type) }}</template>
            <template #last_time="{ row }">{{ formatTime(row.last_time) }}</template>
            <template #empty>当前窗口没有错误</template>
          </t-table>
        </t-card>

        <t-card title="最近错误" :bordered="false">
          <t-table row-key="request_id" :data="overview.recent_errors ?? []" :columns="recentColumns" size="small" hover>
            <template #time="{ row }">{{ formatTime(row.time) }}</template>
            <template #duration_ms="{ row }">{{ row.duration_ms }} ms</template>
            <template #empty>当前窗口没有错误请求</template>
          </t-table>
        </t-card>

        <t-card title="未恢复告警" :bordered="false">
          <t-table row-key="id" :data="overview.alerts ?? []" :columns="alertColumns" size="small" hover>
            <template #status="{ row }">
              <t-tag :theme="row.status === 'firing' ? 'danger' : 'success'" variant="light">{{ alertStatusLabel(row.status) }}</t-tag>
            </template>
            <template #level="{ row }">
              <t-tag :theme="levelTheme(row.level)" variant="light">{{ row.level === 'critical' ? '严重' : '警告' }}</t-tag>
            </template>
            <template #first_time="{ row }">{{ formatTime(row.first_time) }}</template>
            <template #last_time="{ row }">{{ formatTime(row.last_time) }}</template>
            <template #empty>当前没有未恢复告警</template>
          </t-table>
        </t-card>

        <t-card :title="`告警历史（${historyTotal}）`" :bordered="false">
          <t-table row-key="id" :data="history" :columns="alertColumns" size="small" hover>
            <template #status="{ row }">
              <t-tag :theme="row.status === 'firing' ? 'danger' : 'success'" variant="light">{{ alertStatusLabel(row.status) }}</t-tag>
            </template>
            <template #level="{ row }">
              <t-tag :theme="levelTheme(row.level)" variant="light">{{ row.level === 'critical' ? '严重' : '警告' }}</t-tag>
            </template>
            <template #first_time="{ row }">{{ formatTime(row.first_time) }}</template>
            <template #last_time="{ row }">{{ formatTime(row.last_time) }}</template>
            <template #empty>还没有告警记录</template>
          </t-table>
        </t-card>

        <t-card title="请求防护" :bordered="false">
          <t-descriptions :column="2" item-layout="horizontal">
            <t-descriptions-item label="限流">{{ overview.protection.rate_limit_rps }} 次/秒，突发 {{ overview.protection.rate_limit_burst }}</t-descriptions-item>
            <t-descriptions-item label="请求超时">{{ overview.protection.request_timeout_ms }} ms</t-descriptions-item>
            <t-descriptions-item label="请求体上限">{{ formatBytes(overview.protection.max_body_bytes) }}</t-descriptions-item>
            <t-descriptions-item label="作用范围">当前 API 进程，按客户端 IP 限流</t-descriptions-item>
          </t-descriptions>
        </t-card>
      </template>
    </PageStatus>
  </div>
</template>

<style scoped>
.monitor-toolbar {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  color: var(--td-text-color-secondary);
  font-size: 13px;
}
.metric-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
}
.metric-grid-3 {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}
.metric-value {
  font-size: 28px;
  line-height: 1.2;
}
.metric-note {
  margin: 8px 0 0;
  color: var(--td-text-color-secondary);
  font-size: 12px;
}
@media (max-width: 960px) {
  .metric-grid,
  .metric-grid-3 {
    grid-template-columns: 1fr;
  }
}
</style>
