<template>
  <div class="page-container">
    <h1 class="page-title">采购任务</h1>
    <div class="filters glass-card">
      <el-select v-model="filters.status" placeholder="任务状态" clearable style="width:180px" @change="search">
        <el-option label="待处理" value="pending" />
        <el-option label="待支付" value="pending_payment" />
        <el-option label="待人工复核" value="needs_manual_review" />
        <el-option label="已就绪" value="ready" />
        <el-option label="已手动完成" value="manual_completed" />
      </el-select>
      <el-select v-model="filters.payment_status" placeholder="支付状态" clearable style="width:180px" @change="search">
        <el-option label="未支付" value="unpaid" />
        <el-option label="已支付" value="paid" />
        <el-option label="未知" value="unknown" />
      </el-select>
      <el-button type="success" @click="search">查询</el-button>
      <el-button type="primary" @click="openCreateDialog">新建采购任务</el-button>
    </div>

    <div class="glass-card" style="margin-top:16px">
      <el-table :data="list" v-loading="loading" style="width:100%">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="account_name" label="外部账号" min-width="180" />
        <el-table-column prop="target_name" label="购买目标" min-width="160" />
        <el-table-column label="归属" width="130">
          <template #default="{ row }">{{ row.team_owner?.username || '-' }}</template>
        </el-table-column>
        <el-table-column prop="status" label="任务状态" width="160" />
        <el-table-column prop="payment_status" label="支付状态" width="120" />
        <el-table-column prop="subscribe_url" label="订阅链接" min-width="280">
          <template #default="{ row }">
            <div class="content-preview">{{ row.subscribe_url || '-' }}</div>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180" />
        <el-table-column label="操作" width="260" fixed="right">
          <template #default="{ row }">
            <template v-if="canEdit(row)">
              <div class="action-list">
                <el-button
                  v-if="canProcess(row)"
                  text
                  size="small"
                  type="primary"
                  :loading="actionLoadingId === row.id && actionLoadingType === 'process'"
                  @click="handleProcess(row)"
                >
                  {{ row.status === 'needs_manual_review' ? '重新处理' : '开始处理' }}
                </el-button>
                <el-button
                  v-if="canFetch(row)"
                  text
                  size="small"
                  type="warning"
                  :loading="actionLoadingId === row.id && actionLoadingType === 'fetch'"
                  @click="handleFetch(row)"
                >
                  已支付继续抓取
                </el-button>
                <el-button text size="small" type="success" @click="openManualComplete(row)">手动补录</el-button>
              </div>
            </template>
            <el-tag v-else size="small" type="info">只读</el-tag>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination-wrap">
        <el-pagination
          v-model:current-page="page"
          :page-size="pageSize"
          :total="total"
          layout="prev, pager, next"
          @current-change="fetchData"
        />
      </div>
    </div>

    <el-dialog v-model="dialogVisible" title="手动补录订阅链接" width="720px">
      <el-form :model="form" label-width="90px">
        <el-form-item label="外部账号">
          <el-input :model-value="selectedTask?.account_name || ''" disabled />
        </el-form-item>
        <el-form-item label="订阅链接" required>
          <el-input v-model="form.subscribe_url" type="textarea" :rows="5" placeholder="https://dash.yfjc.xyz/api/v1/client/subscribe?token=..." />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="success" :loading="saving" @click="handleManualComplete">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="createDialogVisible" title="新建采购任务" width="520px">
      <el-form label-width="90px">
        <el-form-item label="购买模板" required>
          <el-select v-model="createForm.template_id" placeholder="请选择模板" filterable style="width:100%">
            <el-option
              v-for="tpl in templates"
              :key="tpl.id"
              :label="templateLabel(tpl)"
              :value="tpl.id"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="handleCreate">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { createPurchaseTask, fetchPurchaseTaskSubscribe, getPurchaseTasks, getTemplates, manualCompletePurchaseTask, processPurchaseTask } from '@/api'

const list = ref<any[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const filters = reactive({ status: '', payment_status: '' })
const dialogVisible = ref(false)
const createDialogVisible = ref(false)
const saving = ref(false)
const creating = ref(false)
const actionLoadingId = ref<number | null>(null)
const actionLoadingType = ref('')
const selectedTask = ref<any>(null)
const form = reactive({ subscribe_url: '' })
const createForm = reactive({ template_id: 0 })
const templates = ref<any[]>([])
const currentUser = JSON.parse(localStorage.getItem('user') || '{}')

onMounted(() => {
  fetchData()
  fetchTemplates()
})

async function fetchData() {
  loading.value = true
  try {
    const data: any = await getPurchaseTasks({
      page: page.value,
      page_size: pageSize.value,
      status: filters.status,
      payment_status: filters.payment_status,
    })
    list.value = data.list
    total.value = data.total
  } catch {}
  loading.value = false
}

function search() {
  page.value = 1
  fetchData()
}

async function fetchTemplates() {
  try {
    const data: any = await getTemplates({ page: 1, page_size: 100, status: 'active' })
    templates.value = data.list || []
  } catch {}
}

function canEdit(row: any) {
  return Number(row.team_owner_id) === Number(currentUser.id)
}

function canProcess(row: any) {
  return canEdit(row) && ['pending', 'needs_manual_review'].includes(row.status)
}

function canFetch(row: any) {
  return canEdit(row) && ['pending_payment', 'fetching_subscribe', 'needs_manual_review'].includes(row.status)
}

function openManualComplete(row: any) {
  selectedTask.value = row
  form.subscribe_url = row.subscribe_url || ''
  dialogVisible.value = true
}

function openCreateDialog() {
  if (!templates.value.length) {
    fetchTemplates()
  }
  createForm.template_id = templates.value[0]?.id || 0
  createDialogVisible.value = true
}

function templateLabel(tpl: any) {
  const target = tpl.external_target_name || tpl.external_target_code || '未配置目标'
  return `${tpl.name} / ${target}`
}

async function handleCreate() {
  if (!createForm.template_id) {
    ElMessage.warning('请选择模板')
    return
  }
  creating.value = true
  try {
    await createPurchaseTask(createForm.template_id)
    ElMessage.success('采购任务已创建')
    createDialogVisible.value = false
    fetchData()
  } catch {}
  creating.value = false
}

async function handleManualComplete() {
  if (!selectedTask.value) return
  if (!form.subscribe_url.trim()) {
    ElMessage.warning('请输入订阅链接')
    return
  }
  saving.value = true
  try {
    await manualCompletePurchaseTask(selectedTask.value.id, form.subscribe_url)
    ElMessage.success('补录完成')
    dialogVisible.value = false
    fetchData()
  } catch {}
  saving.value = false
}

async function handleProcess(row: any) {
  actionLoadingId.value = row.id
  actionLoadingType.value = 'process'
  try {
    const data: any = await processPurchaseTask(row.id)
    if (data.status === 'needs_manual_review') {
      ElMessage.warning(data.manual_review_reason || '自动化处理失败，已转人工复核')
    } else {
      ElMessage.success(data.status === 'pending_payment' ? '任务已推进到待支付' : '任务状态已更新')
    }
    fetchData()
  } catch {}
  actionLoadingId.value = null
  actionLoadingType.value = ''
}

async function handleFetch(row: any) {
  actionLoadingId.value = row.id
  actionLoadingType.value = 'fetch'
  try {
    const data: any = await fetchPurchaseTaskSubscribe(row.id)
    if (data.status === 'ready') {
      ElMessage.success('订阅链接已回填')
    } else if (data.status === 'needs_manual_review') {
      ElMessage.warning(data.manual_review_reason || '自动抓取失败，已转人工复核')
    } else {
      ElMessage.success('任务状态已更新')
    }
    fetchData()
  } catch {}
  actionLoadingId.value = null
  actionLoadingType.value = ''
}
</script>

<style scoped lang="scss">
.filters {
  padding: 16px;
  display: flex;
  gap: 12px;
  align-items: center;
}

.content-preview {
  max-height: 58px;
  overflow: hidden;
  color: var(--foreground-muted);
  word-break: break-all;
}

.pagination-wrap {
  display: flex;
  justify-content: flex-end;
  padding: 16px;
}

.action-list {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 8px;
}
</style>
