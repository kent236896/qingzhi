<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getHealthLive, getHealthReady } from '@/api/health'
import PageStatus from '@/components/PageStatus.vue'
import { useAppStore } from '@/stores/app'
import type { HealthReady } from '@/types/api'
import { ApiError } from '@/utils/http'

const appStore = useAppStore()
const buildMode = import.meta.env.MODE
const errorMessage = ref('')
const liveStatus = ref('')
const requestId = ref('')
const ready = ref<HealthReady | null>(null)

const pageStatus = ref<'loading' | 'error' | 'ready'>('loading')

async function loadHealth() {
  pageStatus.value = 'loading'
  errorMessage.value = ''
  ready.value = null
  try {
    const live = await getHealthLive()
    liveStatus.value = live.data.status
    requestId.value = live.requestId
    try {
      const readyResponse = await getHealthReady()
      ready.value = readyResponse.data
    } catch (error) {
      if (error instanceof ApiError && error.httpStatus === 503 && isHealthReady(error.data)) {
        ready.value = error.data
      } else {
        throw error
      }
    }
    pageStatus.value = 'ready'
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '无法连接后端服务'
    pageStatus.value = 'error'
  }
}

function isHealthReady(data: unknown): data is HealthReady {
  if (!data || typeof data !== 'object') {
    return false
  }
  const value = data as Partial<HealthReady>
  return value.status === 'ready' || value.status === 'not_ready'
}

onMounted(() => {
  void loadHealth()
})
</script>

<template>
  <div class="page-grid">
    <t-card title="工程首页" :bordered="false">
      <t-descriptions :column="2" item-layout="horizontal">
        <t-descriptions-item label="系统名称">{{ appStore.title }}</t-descriptions-item>
        <t-descriptions-item label="运行环境">{{ appStore.environment }}</t-descriptions-item>
        <t-descriptions-item label="页面">首页</t-descriptions-item>
        <t-descriptions-item label="构建模式">{{ buildMode }}</t-descriptions-item>
      </t-descriptions>
    </t-card>

    <t-card title="后端健康检查" :bordered="false">
      <PageStatus
        :status="pageStatus"
        :error-message="errorMessage || '无法获取后端健康状态'"
        @retry="loadHealth"
      >
        <t-descriptions :column="2" item-layout="horizontal">
          <t-descriptions-item label="存活">
            <t-tag theme="success" variant="light">{{ liveStatus || '未知' }}</t-tag>
          </t-descriptions-item>
          <t-descriptions-item label="就绪">
            <t-tag :theme="ready?.status === 'ready' ? 'success' : 'warning'" variant="light">
              {{ ready?.status || '未知' }}
            </t-tag>
          </t-descriptions-item>
          <t-descriptions-item label="PostgreSQL">
            {{ ready?.checks.database || '未知' }}
          </t-descriptions-item>
          <t-descriptions-item label="Redis">
            {{ ready?.checks.redis || '未知' }}
          </t-descriptions-item>
          <t-descriptions-item label="Request ID">{{ requestId || '-' }}</t-descriptions-item>
        </t-descriptions>
      </PageStatus>
    </t-card>

    <t-card title="业务模块" :bordered="false">
      <PageStatus status="empty" description="当前阶段尚未接入业务模块" />
    </t-card>
  </div>
</template>
