<template>
  <div class="page-container">
    <h1 class="page-title">用户管理</h1>
    <div class="toolbar">
      <el-input v-model="keyword" placeholder="搜索用户名" clearable style="width:220px" @change="fetchData" />
      <el-button type="success" @click="openCreate">新增用户</el-button>
    </div>
    <div class="glass-card" style="margin-top:16px">
      <el-table :data="list" v-loading="loading" style="width:100%">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="username" label="用户名" width="180" />
        <el-table-column prop="file_prefix" label="文件前缀" width="120">
          <template #default="{ row }">{{ row.file_prefix || '-' }}</template>
        </el-table-column>
        <el-table-column prop="file_sequence_next" label="下个序号" width="110" />
        <el-table-column prop="role" label="角色" width="100">
          <template #default="{ row }">
            <el-tag :type="row.role==='admin'?'warning':''" size="small">
              {{ row.role==='admin'?'管理员':'用户' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status==='active'?'success':'danger'" size="small">
              {{ row.status==='active'?'正常':'禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="last_login_at" label="最后登录" width="180" />
        <el-table-column prop="created_at" label="创建时间" width="180" />
        <el-table-column label="操作" width="280">
          <template #default="{ row }">
            <el-button text size="small" type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button text size="small"
              :type="row.status==='active'?'danger':''" @click="toggleStatus(row)">
              {{ row.status==='active'?'禁用':'启用' }}
            </el-button>
            <el-button text size="small" type="warning" @click="resetPwd(row)">重置密码</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination-wrap">
        <el-pagination v-model:current-page="page" :page-size="pageSize" :total="total"
          layout="prev, pager, next" @current-change="fetchData" />
      </div>
    </div>

    <!-- 新增/编辑弹窗 -->
    <el-dialog v-model="dialogVisible" :title="editingId ? '编辑用户' : '新增用户'" width="440px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="用户名" required>
          <el-input v-model="form.username" :disabled="!!editingId" />
        </el-form-item>
        <el-form-item label="密码" :required="!editingId">
          <el-input v-model="form.password" type="password" show-password
            :placeholder="editingId ? '留空则不改密码' : '至少8位'" />
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="form.role">
            <el-option label="普通用户" value="user" />
            <el-option label="管理员" value="admin" />
          </el-select>
        </el-form-item>
        <el-form-item label="文件前缀">
          <el-input v-model="form.file_prefix" placeholder="仅支持字母、数字、-、_" />
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
import { ref, reactive, onMounted } from 'vue'
import { getUsers, createUser, updateUser } from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'

const list = ref<any[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const keyword = ref('')

const dialogVisible = ref(false)
const editingId = ref(0)
const originalFilePrefix = ref('')
const form = reactive({ username: '', password: '', role: 'user', file_prefix: '' })

onMounted(() => fetchData())

async function fetchData() {
  loading.value = true
  try {
    const data: any = await getUsers({ page: page.value, page_size: pageSize.value, keyword: keyword.value })
    list.value = data.list
    total.value = data.total
  } catch {}
  loading.value = false
}

function openCreate() {
  editingId.value = 0
  form.username = ''
  form.password = ''
  form.role = 'user'
  form.file_prefix = ''
  originalFilePrefix.value = ''
  dialogVisible.value = true
}

function openEdit(row: any) {
  editingId.value = row.id
  form.username = row.username
  form.password = ''
  form.role = row.role
  form.file_prefix = row.file_prefix || ''
  originalFilePrefix.value = row.file_prefix || ''
  dialogVisible.value = true
}

async function handleSave() {
  if (!form.username) { ElMessage.warning('请输入用户名'); return }
  if (!editingId.value && form.password.length < 8) { ElMessage.warning('密码至少8位'); return }
  try {
    const payload: any = { role: form.role }
    if (form.password) payload.password = form.password
    if (editingId.value) {
      if (form.file_prefix !== originalFilePrefix.value) {
        await ElMessageBox.confirm('修改生成文件前缀后，后续生成文件序号将从 1001 重新开始，确认修改？', '提示', { type: 'warning' })
        payload.file_prefix = form.file_prefix
      }
      await updateUser(editingId.value, payload)
    } else {
      payload.username = form.username
      payload.file_prefix = form.file_prefix
      await createUser(payload)
    }
    ElMessage.success('保存成功')
    dialogVisible.value = false
    fetchData()
  } catch {}
}

async function toggleStatus(row: any) {
  const newStatus = row.status === 'active' ? 'disabled' : 'active'
  await updateUser(row.id, { status: newStatus })
  ElMessage.success('已更新')
  fetchData()
}

async function resetPwd(row: any) {
  try {
    const { value } = await ElMessageBox.prompt('请输入新密码(至少8位)', '重置密码', {
      inputType: 'password',
      inputValidator: (v: string) => v.length >= 8 ? true : '密码至少8位',
    })
    await updateUser(row.id, { password: value })
    ElMessage.success('密码已重置')
  } catch {}
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
