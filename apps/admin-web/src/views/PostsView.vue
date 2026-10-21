<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { DialogPlugin } from 'tdesign-vue-next'
import PageStatus from '@/components/PageStatus.vue'
import {
  createPost,
  deletePost,
  listPosts,
  statusLabel,
  updatePost,
  type Post,
  type RecordStatus,
} from '@/api/identity'
import { ApiError } from '@/utils/http'
import { feedback } from '@/utils/feedback'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const canCreate = computed(() => auth.hasPermission('system:post:create'))
const canUpdate = computed(() => auth.hasPermission('system:post:update'))
const canDelete = computed(() => auth.hasPermission('system:post:delete'))
const pageStatus = ref<'loading' | 'error' | 'ready'>('loading')
const errorMessage = ref('')
const tenantId = computed(() => auth.tenantId)
const items = ref<Post[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const keyword = ref('')

const editorVisible = ref(false)
const saving = ref(false)
const editing = ref<Post | null>(null)
const form = ref({ code: '', name: '', sort: 0, remark: '', status: 0 as RecordStatus })

const columns = [
  { colKey: 'name', title: '名称' },
  { colKey: 'code', title: '编码' },
  { colKey: 'sort', title: '排序' },
  { colKey: 'remark', title: '备注' },
  { colKey: 'status', title: '状态' },
  { colKey: 'actions', title: '操作', width: 160 },
]

async function bootstrap() {
  if (!tenantId.value) {
    pageStatus.value = 'error'
    errorMessage.value = '当前登录未绑定租户'
    return
  }
  await load()
}

async function load() {
  if (!tenantId.value) {
    items.value = []
    total.value = 0
    pageStatus.value = 'ready'
    return
  }
  pageStatus.value = 'loading'
  errorMessage.value = ''
  try {
    const response = await listPosts(tenantId.value, {
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value.trim() || undefined,
    })
    items.value = response.data.items
    total.value = response.data.total
    pageStatus.value = 'ready'
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '无法加载岗位'
    pageStatus.value = 'error'
  }
}

function search() {
  page.value = 1
  void load()
}

function openCreate() {
  editing.value = null
  form.value = { code: '', name: '', sort: 0, remark: '', status: 0 }
  editorVisible.value = true
}

function openEdit(row: Post) {
  editing.value = row
  form.value = { code: row.code, name: row.name, sort: row.sort, remark: row.remark, status: row.status }
  editorVisible.value = true
}

async function save() {
  if (!tenantId.value || !form.value.name.trim() || !form.value.code.trim()) {
    feedback.warning('请填写名称和编码')
    return
  }
  saving.value = true
  try {
    if (editing.value) {
      await updatePost(tenantId.value, editing.value.id, {
        code: form.value.code.trim(),
        name: form.value.name.trim(),
        sort: Number(form.value.sort) || 0,
        remark: form.value.remark.trim(),
        status: form.value.status,
      })
      feedback.success('岗位已更新')
    } else {
      await createPost(tenantId.value, {
        code: form.value.code.trim(),
        name: form.value.name.trim(),
        sort: Number(form.value.sort) || 0,
        remark: form.value.remark.trim() || undefined,
      })
      feedback.success('岗位已创建')
    }
    editorVisible.value = false
    await load()
  } catch {
    // The HTTP client already shows the server message.
  } finally {
    saving.value = false
  }
}

function remove(row: Post) {
  const dialog = DialogPlugin.confirm({
    header: '删除岗位',
    body: `确定删除「${row.name}」？`,
    onConfirm: async () => {
      try {
        await deletePost(tenantId.value, row.id)
        feedback.success('岗位已删除')
        dialog.hide()
        await load()
      } catch {
        dialog.hide()
      }
    },
  })
}

onMounted(() => {
  void bootstrap()
})
</script>

<template>
  <div class="page-grid">
    <t-card :bordered="false">
      <div class="toolbar">
        <t-input v-model="keyword" clearable placeholder="搜索名称或编码" @enter="search" />
        <t-button theme="default" variant="outline" @click="search">查询</t-button>
        <t-button v-if="canCreate" theme="primary" variant="base" :disabled="!tenantId" @click="openCreate">新建岗位</t-button>
      </div>
      <PageStatus :status="pageStatus" :error-message="errorMessage" @retry="bootstrap">
        <t-table row-key="id" :data="items" :columns="columns" hover>
          <template #remark="{ row }">{{ row.remark || '—' }}</template>
          <template #status="{ row }">
            <t-tag :theme="row.status === 0 ? 'success' : 'warning'" variant="light">
              {{ statusLabel(row.status) }}
            </t-tag>
          </template>
          <template #actions="{ row }">
            <t-button v-if="canUpdate" variant="text" @click="openEdit(row)">编辑</t-button>
            <t-button v-if="canDelete" theme="danger" variant="text" @click="remove(row)">删除</t-button>
          </template>
        </t-table>
        <t-pagination v-model="page" v-model:page-size="pageSize" class="pager" :total="total" @change="load" />
      </PageStatus>
    </t-card>

    <t-dialog
      v-model:visible="editorVisible"
      :header="editing ? '编辑岗位' : '新建岗位'"
      :confirm-btn="{ content: '保存', loading: saving }"
      @confirm="save"
    >
      <t-form label-align="top">
        <t-form-item label="名称">
          <t-input v-model="form.name" maxlength="50" />
        </t-form-item>
        <t-form-item label="编码">
          <t-input v-model="form.code" maxlength="64" />
        </t-form-item>
        <t-form-item label="排序">
          <t-input-number v-model="form.sort" :min="0" :max="1000000" />
        </t-form-item>
        <t-form-item label="备注">
          <t-input v-model="form.remark" maxlength="500" />
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
