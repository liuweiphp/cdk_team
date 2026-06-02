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
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <template v-if="canEdit(row)">
              <el-button text size="small" type="primary" @click="openManualComplete(row)">手动补录</el-button>
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
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { getPurchaseTasks, manualCompletePurchaseTask } from '@/api'

const list = ref<any[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const filters = reactive({ status: '', payment_status: '' })
const dialogVisible = ref(false)
const saving = ref(false)
const selectedTask = ref<any>(null)
const form = reactive({ subscribe_url: '' })
const currentUser = JSON.parse(localStorage.getItem('user') || '{}')

onMounted(() => fetchData())

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

function canEdit(row: any) {
  return Number(row.team_owner_id) === Number(currentUser.id)
}

function openManualComplete(row: any) {
  selectedTask.value = row
  form.subscribe_url = row.subscribe_url || ''
  dialogVisible.value = true
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
</style>
