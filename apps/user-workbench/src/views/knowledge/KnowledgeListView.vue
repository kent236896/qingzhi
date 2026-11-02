<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { DialogPlugin } from 'tdesign-vue-next'
import PageStatus from '@/components/common/PageStatus.vue'
import {
  createKnowledgeBase,
  deleteKnowledgeBase,
  getKnowledgeOptions,
  listKnowledgeBases,
  shareKnowledgeBase,
  updateKnowledgeBase,
} from '@/api/knowledge'
import { useAuthStore } from '@/stores/auth'
import type { KnowledgeBase, KnowledgeFormOptions } from '@/types/knowledge'
import { feedback } from '@/utils/feedback'
import { canManagePersonalKnowledge, SCOPE_LABEL } from '@/utils/knowledge'

type TabKey = 'personal' | 'enterprise'

const auth = useAuthStore()
const router = useRouter()

const tab = ref<TabKey>('personal')
const keyword = ref('')
const page = ref(1)
const pageSize = 20
const total = ref(0)
const items = ref<KnowledgeBase[]>([])
const loading = ref(false)
const loadError = ref('')
const options = ref<KnowledgeFormOptions | null>(null)

const createVisible = ref(false)
const editVisible = ref(false)
const shareVisible = ref(false)
const saving = ref(false)
const editing = ref<KnowledgeBase | null>(null)
const sharing = ref<KnowledgeBase | null>(null)

const createForm = ref({
  name: '',
  description: '',
  wiki_enabled: false,
  share_user_ids: [] as string[],
})
const editForm = ref({ name: '', description: '' })
const shareForm = ref({ share_user_ids: [] as string[] })

const canQuery = computed(() => auth.hasPermission('knowledge:query') || auth.hasPermission('knowledge:manage'))
const userOptions = computed(() =>
  (options.value?.users ?? [])
    .filter((item) => item.id !== auth.user?.id)
    .map((item) => ({ label: item.name, value: item.id })),
)

const emptyHint = computed(() => {
  if (tab.value === 'personal') {
    return '还没有个人知识库。可以新建一个，并分享给同事。'
  }
  return '暂无可见的企业知识库（组织 / 岗位 / 角色，或他人分享给你的个人库）。'
})

async function ensureOptions() {
  if (options.value) return
  const response = await getKnowledgeOptions()
  options.value = response.data
}

async function load() {
  if (!canQuery.value) {
    loadError.value = '当前账号没有知识库查询权限（knowledge:query）。'
    items.value = []
    total.value = 0
    return
  }
  loading.value = true
  loadError.value = ''
  try {
    const response = await listKnowledgeBases({
      page: 1,
      page_size: 100,
      keyword: keyword.value.trim() || undefined,
    })
    const all = response.data.items ?? []
    const ownedPersonal = all.filter((item) => canManagePersonalKnowledge(item, auth.user?.id))
    // Enterprise / shared: org·岗位·角色，以及别人分享给我的个人库（只读）。
    const enterprise = all.filter((item) => !canManagePersonalKnowledge(item, auth.user?.id))
    const source = tab.value === 'personal' ? ownedPersonal : enterprise
    total.value = source.length
    const start = (page.value - 1) * pageSize
    items.value = source.slice(start, start + pageSize)
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : '加载知识库失败'
    items.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

async function openCreate() {
  try {
    await ensureOptions()
    createForm.value = { name: '', description: '', wiki_enabled: false, share_user_ids: [] }
    createVisible.value = true
  } catch (error) {
    feedback.error(error instanceof Error ? error.message : '加载选项失败')
  }
}

async function openEdit(row: KnowledgeBase) {
  editing.value = row
  editForm.value = { name: row.name, description: row.description || '' }
  editVisible.value = true
}

async function openShare(row: KnowledgeBase) {
  try {
    await ensureOptions()
    sharing.value = row
    shareForm.value = { share_user_ids: [...(row.share_user_ids ?? [])] }
    shareVisible.value = true
  } catch (error) {
    feedback.error(error instanceof Error ? error.message : '加载选项失败')
  }
}

async function saveCreate() {
  const name = createForm.value.name.trim()
  if (!name) {
    feedback.warning('请填写知识库名称')
    return
  }
  const tenantId = auth.tenantId
  if (!tenantId) {
    feedback.warning('缺少当前租户，无法创建')
    return
  }
  if (createForm.value.wiki_enabled && !options.value?.models?.wiki_synthesis?.id) {
    feedback.warning('尚未绑定 Wiki 生成模型，请取消勾选或联系管理员配置')
    return
  }
  saving.value = true
  try {
    await createKnowledgeBase({
      name,
      description: createForm.value.description.trim(),
      scope_type: 'personal',
      tenant_ids: [tenantId],
      share_user_ids: createForm.value.share_user_ids,
      wiki_enabled: createForm.value.wiki_enabled,
    })
    feedback.success('个人知识库已创建')
    createVisible.value = false
    page.value = 1
    await load()
  } catch {
    /* http client toasts */
  } finally {
    saving.value = false
  }
}

async function saveEdit() {
  if (!editing.value) return
  const name = editForm.value.name.trim()
  if (!name) {
    feedback.warning('请填写知识库名称')
    return
  }
  saving.value = true
  try {
    await updateKnowledgeBase(editing.value.id, {
      name,
      description: editForm.value.description.trim(),
    })
    feedback.success('已保存')
    editVisible.value = false
    await load()
  } catch {
    /* http client toasts */
  } finally {
    saving.value = false
  }
}

async function saveShare() {
  if (!sharing.value) return
  saving.value = true
  try {
    await shareKnowledgeBase(sharing.value.id, {
      share_user_ids: shareForm.value.share_user_ids,
    })
    feedback.success('分享已更新')
    shareVisible.value = false
    await load()
  } catch {
    /* http client toasts */
  } finally {
    saving.value = false
  }
}

function confirmDelete(row: KnowledgeBase) {
  const dialog = DialogPlugin.confirm({
    header: '删除个人知识库',
    body: `确认删除「${row.name}」？删除后不可恢复。`,
    confirmBtn: '删除',
    theme: 'danger',
    onConfirm: async () => {
      try {
        await deleteKnowledgeBase(row.id)
        feedback.success('已删除')
        dialog.destroy()
        await load()
      } catch {
        /* http client toasts */
      }
    },
  })
}

function openDetail(row: KnowledgeBase) {
  void router.push({ name: 'knowledge-detail', params: { id: row.id } })
}

function openChat(row: KnowledgeBase) {
  void router.push({ name: 'chat', query: { kb: row.id } })
}

watch([tab], () => {
  page.value = 1
  void load()
})

onMounted(() => {
  void load()
})
</script>

<template>
  <div class="kb-list">
    <header class="kb-list__header">
      <div>
        <h2 class="kb-list__title">知识库</h2>
        <p class="kb-list__lead">
          个人知识库可新建、修改、删除与分享；组织 / 岗位 / 角色知识库仅可查看与检索。
        </p>
      </div>
      <t-button v-if="tab === 'personal' && canQuery" theme="primary" @click="openCreate">
        新建个人知识库
      </t-button>
    </header>

    <div class="kb-list__toolbar">
      <t-radio-group v-model="tab" variant="default-filled">
        <t-radio-button value="personal">我的知识库</t-radio-button>
        <t-radio-button value="enterprise">企业知识库</t-radio-button>
      </t-radio-group>
      <t-input
        v-model="keyword"
        clearable
        placeholder="按名称搜索"
        class="kb-list__search"
        @enter="
          () => {
            page = 1
            void load()
          }
        "
      >
        <template #suffix>
          <t-button variant="text" size="small" @click="page = 1; void load()">搜索</t-button>
        </template>
      </t-input>
    </div>

    <PageStatus v-if="!canQuery" kind="error" title="无权限" :description="loadError" />
    <PageStatus v-else-if="loading && !items.length" kind="loading" title="加载中…" />
    <PageStatus v-else-if="loadError && !items.length" kind="error" title="加载失败" :description="loadError">
      <t-button size="small" @click="load">重试</t-button>
    </PageStatus>
    <PageStatus v-else-if="!items.length" kind="empty" title="暂无知识库" :description="emptyHint">
      <t-button v-if="tab === 'personal'" theme="primary" size="small" @click="openCreate">新建</t-button>
    </PageStatus>

    <div v-else class="kb-list__grid">
      <article v-for="item in items" :key="item.id" class="kb-card">
        <div class="kb-card__top">
          <h3 class="kb-card__name">{{ item.name }}</h3>
          <span class="kb-card__tag">{{ SCOPE_LABEL[item.scope_type] }}</span>
        </div>
        <p class="kb-card__desc">{{ item.description || '暂无描述' }}</p>
        <p class="kb-card__meta">
          <span v-if="item.scope_name">归属 {{ item.scope_name }}</span>
          <span v-if="item.wiki_enabled"> · Wiki</span>
          <span v-if="tab === 'personal' && (item.share_user_ids?.length || 0) > 0">
            · 已分享 {{ item.share_user_ids?.length }} 人
          </span>
        </p>
        <div class="kb-card__actions">
          <t-button size="small" variant="outline" @click="openDetail(item)">查看</t-button>
          <t-button size="small" variant="outline" @click="openChat(item)">对话检索</t-button>
          <template v-if="tab === 'personal' && canManagePersonalKnowledge(item, auth.user?.id)">
            <t-button size="small" variant="text" @click="openEdit(item)">修改</t-button>
            <t-button size="small" variant="text" @click="openShare(item)">分享</t-button>
            <t-button size="small" variant="text" theme="danger" @click="confirmDelete(item)">
              删除
            </t-button>
          </template>
        </div>
      </article>
    </div>

    <div v-if="total > pageSize" class="kb-list__pager">
      <t-pagination
        v-model="page"
        :total="total"
        :page-size="pageSize"
        size="small"
        @change="load"
      />
    </div>

    <t-dialog
      v-model:visible="createVisible"
      header="新建个人知识库"
      :confirm-btn="{ content: '创建', loading: saving }"
      @confirm="saveCreate"
    >
      <t-form label-width="88px">
        <t-form-item label="名称" required>
          <t-input v-model="createForm.name" maxlength="100" placeholder="知识库名称" />
        </t-form-item>
        <t-form-item label="描述">
          <t-textarea v-model="createForm.description" maxlength="500" :autosize="{ minRows: 2, maxRows: 4 }" />
        </t-form-item>
        <t-form-item label="Wiki">
          <t-checkbox v-model="createForm.wiki_enabled">启用 Wiki（创建后不可关闭）</t-checkbox>
        </t-form-item>
        <t-form-item label="分享给">
          <t-select
            v-model="createForm.share_user_ids"
            multiple
            filterable
            clearable
            placeholder="可选，分享给其他用户只读"
            :options="userOptions"
          />
        </t-form-item>
      </t-form>
    </t-dialog>

    <t-dialog
      v-model:visible="editVisible"
      header="修改知识库"
      :confirm-btn="{ content: '保存', loading: saving }"
      @confirm="saveEdit"
    >
      <t-form label-width="72px">
        <t-form-item label="名称" required>
          <t-input v-model="editForm.name" maxlength="100" />
        </t-form-item>
        <t-form-item label="描述">
          <t-textarea v-model="editForm.description" maxlength="500" :autosize="{ minRows: 2, maxRows: 4 }" />
        </t-form-item>
      </t-form>
    </t-dialog>

    <t-dialog
      v-model:visible="shareVisible"
      header="分享个人知识库"
      :confirm-btn="{ content: '保存', loading: saving }"
      @confirm="saveShare"
    >
      <p class="kb-list__share-hint">被分享用户可查看与检索，不能修改或删除。</p>
      <t-select
        v-model="shareForm.share_user_ids"
        multiple
        filterable
        clearable
        placeholder="选择用户"
        :options="userOptions"
      />
    </t-dialog>
  </div>
</template>

<style scoped>
.kb-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-width: 1100px;
}

.kb-list__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.kb-list__title {
  margin: 0 0 6px;
  font-size: 22px;
  font-weight: 650;
}

.kb-list__lead {
  margin: 0;
  color: var(--wb-text-secondary);
  font-size: 14px;
}

.kb-list__toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
}

.kb-list__search {
  width: min(280px, 100%);
}

.kb-list__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 14px;
}

.kb-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px;
  background: var(--wb-surface);
  border: 1px solid var(--wb-border);
  border-radius: var(--wb-radius);
}

.kb-card__top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
}

.kb-card__name {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.kb-card__tag {
  flex-shrink: 0;
  padding: 2px 8px;
  border-radius: 4px;
  background: #ecf2fe;
  color: var(--wb-primary);
  font-size: 12px;
  font-weight: 600;
}

.kb-card__desc {
  margin: 0;
  min-height: 40px;
  color: var(--wb-text-secondary);
  font-size: 13px;
  line-height: 1.45;
}

.kb-card__meta {
  margin: 0;
  color: var(--wb-text-secondary);
  font-size: 12px;
}

.kb-card__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 4px;
}

.kb-list__pager {
  display: flex;
  justify-content: flex-end;
}

.kb-list__share-hint {
  margin: 0 0 12px;
  color: var(--wb-text-secondary);
  font-size: 13px;
}

@media (max-width: 640px) {
  .kb-list__header {
    flex-direction: column;
  }
}
</style>
