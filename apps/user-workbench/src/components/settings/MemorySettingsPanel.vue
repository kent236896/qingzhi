<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { fetchMemoryStatus } from '@/api/memory'
import PageStatus from '@/components/common/PageStatus.vue'
import SettingsPageHeader from '@/components/settings/SettingsPageHeader.vue'
import { useSettingsStore } from '@/stores/settings'
import type { MemoryStatus } from '@/types/memory'
import { ApiError } from '@/utils/http'

const router = useRouter()
const settings = useSettingsStore()

const loading = ref(true)
const loadError = ref('')
const status = ref<MemoryStatus | null>(null)

const summary = computed(() => {
  if (!status.value) return ''
  if (status.value.ready) return '个人记忆能力已开放，可前往完整页面开关与管理条目。'
  return '个人记忆因 WeKnora /memory/* 跨用户隔离复测未通过暂不开放；不会使用模拟数据，也不会代理 settings/items。'
})

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    const response = await fetchMemoryStatus()
    status.value = response.data
  } catch (error) {
    loadError.value = error instanceof ApiError ? error.message : '加载记忆状态失败'
    status.value = null
  } finally {
    loading.value = false
  }
}

async function goMemoryPage() {
  await settings.closeSettings()
  await router.push({ name: 'memory' })
}

onMounted(() => {
  void load()
})
</script>

<template>
  <div>
    <SettingsPageHeader title="我的记忆" description="查看个人长期记忆能力状态；完整管理在独立页面。" />

    <PageStatus v-if="loading" title="正在加载" kind="loading" />
    <PageStatus v-else-if="loadError" title="无法加载" :description="loadError" kind="error">
      <t-button theme="primary" variant="outline" @click="load">重试</t-button>
    </PageStatus>

    <template v-else-if="status">
      <p class="memory-summary">{{ summary }}</p>
      <dl class="memory-meta">
        <div>
          <dt>能力状态</dt>
          <dd>{{ status.ready ? '已开放' : '未开放' }}</dd>
        </div>
        <div>
          <dt>隔离复测</dt>
          <dd>
            {{ status.verified ? '已复测' : '未复测' }}
            <span v-if="status.retested_at">（{{ status.retested_at }}）</span>
          </dd>
        </div>
        <div>
          <dt>空间级开关</dt>
          <dd>
            <template v-if="status.workspace_enabled === null || status.workspace_enabled === undefined">
              未知
            </template>
            <template v-else>{{ status.workspace_enabled ? '已启用' : '未启用' }}</template>
          </dd>
        </div>
      </dl>
      <p v-if="!status.ready" class="memory-reason">{{ status.reason }}</p>
      <t-button theme="primary" variant="outline" @click="goMemoryPage">前往「我的长期记忆」</t-button>
    </template>
  </div>
</template>

<style scoped>
.memory-summary {
  margin: 0 0 16px;
  font-size: 14px;
  line-height: 1.5;
  color: var(--wb-text);
}

.memory-meta {
  margin: 0 0 16px;
  display: grid;
  gap: 12px;
}

.memory-meta > div {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--wb-border);
}

.memory-meta dt {
  font-size: 13px;
  color: var(--wb-text-secondary);
}

.memory-meta dd {
  margin: 0;
  font-size: 14px;
  color: var(--wb-text);
  text-align: right;
}

.memory-reason {
  margin: 0 0 16px;
  padding: 12px;
  border-radius: 8px;
  background: var(--wb-sidebar-bg);
  border: 1px solid var(--wb-border);
  font-size: 13px;
  line-height: 1.55;
  color: var(--wb-text-secondary);
}
</style>
