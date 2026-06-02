<template>
  <div class="page-container">
    <h1 class="page-title">模板管理</h1>
    <div class="toolbar">
      <el-input v-model="keyword" placeholder="搜索模板名称" clearable style="width:240px" @change="fetchData" />
      <el-button type="success" @click="openCreate">新增模板</el-button>
    </div>

    <div class="glass-card" style="margin-top:16px">
      <el-table :data="list" v-loading="loading" style="width:100%">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="名称" width="180" />
        <el-table-column prop="external_target_code" label="目标编码" width="160" />
        <el-table-column prop="external_target_name" label="目标名称" width="180" />
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
        <el-table-column label="内容预览" min-width="360">
          <template #default="{ row }">
            <div class="content-preview">{{ row.content }}</div>
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

    <el-dialog v-model="dialogVisible" :title="editingId ? '编辑模板' : '新增模板'" width="760px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="名称" required>
          <el-input v-model="form.name" placeholder="模板名称" />
        </el-form-item>
        <el-form-item label="目标编码" required>
          <el-input v-model="form.external_target_code" placeholder="固定购买目标编码" />
        </el-form-item>
        <el-form-item label="目标名称">
          <el-input v-model="form.external_target_name" placeholder="固定购买目标名称" />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="form.status" style="width:160px">
            <el-option label="启用" value="active" />
            <el-option label="禁用" value="disabled" />
          </el-select>
        </el-form-item>
        <el-form-item label="模板内容" required>
          <el-input v-model="form.content" type="textarea" :rows="18" />
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
import { createTemplate, deleteTemplate, getTemplates, updateTemplate } from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'

const defaultTemplate = `配 置 链 接↓ ↓ ↓ ↓ ↓ ↓↓ ↓ ↓ ↓ ↓ ↓↓ ↓ ↓ ↓ ↓ ↓
{{content}}

有其他账号需求的话，可以咨询。
ID成品号（永久）￥46
chatgpt plus充值会员￥60
tiktok账号￥48
推特新号￥36
推特老号￥66
飞机新号￥36
飞机老号（稳定）￥66
谷歌账号￥36
谷歌老号￥66
ins账号￥25
脸书账号￥25

免费收徒
免费收徒
免费收徒
免费收徒`

const list = ref<any[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const keyword = ref('')
const dialogVisible = ref(false)
const editingId = ref(0)
const form = reactive({
  name: '',
  content: '',
  status: 'active',
  external_target_code: '',
  external_target_name: '',
})
const currentUser = JSON.parse(localStorage.getItem('user') || '{}')

onMounted(() => fetchData())

async function fetchData() {
  loading.value = true
  try {
    const data: any = await getTemplates({ page: page.value, page_size: pageSize.value, keyword: keyword.value })
    list.value = data.list
    total.value = data.total
  } catch {}
  loading.value = false
}

function openCreate() {
  editingId.value = 0
  form.name = '默认账号模板'
  form.content = defaultTemplate
  form.status = 'active'
  form.external_target_code = ''
  form.external_target_name = ''
  dialogVisible.value = true
}

function openEdit(row: any) {
  editingId.value = row.id
  form.name = row.name
  form.content = row.content
  form.status = row.status
  form.external_target_code = row.external_target_code || ''
  form.external_target_name = row.external_target_name || ''
  dialogVisible.value = true
}

async function handleSave() {
  if (!form.name.trim()) { ElMessage.warning('请输入名称'); return }
  if (!form.content.includes('{{content}}')) { ElMessage.warning('模板必须包含 {{content}}'); return }
  if (!form.external_target_code.trim()) { ElMessage.warning('请输入固定购买目标编码'); return }
  const payload = {
    name: form.name,
    content: form.content,
    status: form.status,
    external_target_code: form.external_target_code,
    external_target_name: form.external_target_name,
  }
  try {
    if (editingId.value) await updateTemplate(editingId.value, payload)
    else await createTemplate(payload)
    ElMessage.success('保存成功')
    dialogVisible.value = false
    fetchData()
  } catch {}
}

async function handleDelete(id: number) {
  try {
    await ElMessageBox.confirm('确认删除该模板?', '提示', { type: 'warning' })
    await deleteTemplate(id)
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

.content-preview {
  max-height: 58px;
  overflow: hidden;
  color: var(--foreground-muted);
  white-space: pre-wrap;
}

.pagination-wrap {
  display: flex;
  justify-content: flex-end;
  padding: 16px;
}
</style>
