<template>
  <div class="page-container">
    <h1 class="page-title">分类管理</h1>
    <div class="toolbar">
      <el-input v-model="keyword" placeholder="搜索分类名称" clearable style="width:240px" @change="fetchData" />
      <el-button type="success" @click="openCreate">新增分类</el-button>
    </div>

    <div class="glass-card" style="margin-top:16px">
      <el-table :data="list" v-loading="loading" style="width:100%">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="名称" min-width="220" />
        <el-table-column label="归属" width="130">
          <template #default="{ row }">{{ row.creator?.username || (canEdit(row) ? '我的' : '团队') }}</template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'danger'" size="small">
              {{ row.status === 'active' ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180" />
        <el-table-column label="操作" width="180">
          <template #default="{ row }">
            <template v-if="canEdit(row)">
              <el-button text size="small" type="primary" @click="openEdit(row)">编辑</el-button>
              <el-button text size="small" type="danger" @click="handleDelete(row.id)">删除</el-button>
            </template>
            <el-tag v-else size="small" type="info">只读</el-tag>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination-wrap">
        <el-pagination v-model:current-page="page" :page-size="pageSize" :total="total"
          layout="prev, pager, next" @current-change="fetchData" />
      </div>
    </div>

    <el-dialog v-model="dialogVisible" :title="editingId ? '编辑分类' : '新增分类'" width="520px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="名称" required>
          <el-input v-model="form.name" placeholder="分类名称" />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="form.status" style="width:160px">
            <el-option label="启用" value="active" />
            <el-option label="禁用" value="disabled" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="success" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { createRedeemCategory, deleteRedeemCategory, getRedeemCategories, updateRedeemCategory } from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'

const list = ref<any[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const keyword = ref('')
const dialogVisible = ref(false)
const editingId = ref(0)
const form = reactive({ name: '', status: 'active' })
const currentUser = JSON.parse(localStorage.getItem('user') || '{}')

onMounted(() => fetchData())

async function fetchData() {
  loading.value = true
  try {
    const data: any = await getRedeemCategories({ page: page.value, page_size: pageSize.value, keyword: keyword.value })
    list.value = data.list
    total.value = data.total
  } catch {}
  loading.value = false
}

function openCreate() {
  editingId.value = 0
  form.name = ''
  form.status = 'active'
  dialogVisible.value = true
}

function openEdit(row: any) {
  editingId.value = row.id
  form.name = row.name
  form.status = row.status
  dialogVisible.value = true
}

async function handleSave() {
  if (!form.name.trim()) { ElMessage.warning('请输入分类名称'); return }
  const payload = { name: form.name, status: form.status }
  try {
    if (editingId.value) await updateRedeemCategory(editingId.value, payload)
    else await createRedeemCategory(payload)
    ElMessage.success('保存成功')
    dialogVisible.value = false
    fetchData()
  } catch {}
}

async function handleDelete(id: number) {
  try {
    await ElMessageBox.confirm('确认删除该分类?', '提示', { type: 'warning' })
    await deleteRedeemCategory(id)
    ElMessage.success('已删除')
    fetchData()
  } catch {}
}

function canEdit(row: any) {
  return Number(row.created_by) === Number(currentUser.id)
}
</script>

<style scoped lang="scss">
.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.pagination-wrap {
  display: flex;
  justify-content: flex-end;
  padding: 16px;
}
</style>
