<template>
  <div class="page-container">
    <h1 class="page-title">兑换内容</h1>
    <div class="glass-card upload-panel">
      <h3>导入兑换内容</h3>
      <el-form :model="uploadForm" label-width="80px" class="upload-form">
        <el-form-item label="方式" required>
          <el-segmented v-model="uploadForm.mode" :options="importModeOptions" />
        </el-form-item>
        <el-form-item label="分类" required>
          <el-select v-model="uploadForm.category_id" placeholder="请选择分类" style="width:320px">
            <el-option v-for="category in categories" :key="category.id" :label="category.name" :value="category.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="模板" required>
          <el-select v-model="uploadForm.template_id" placeholder="请选择模板" style="width:320px">
            <el-option v-for="tpl in templates" :key="tpl.id" :label="tpl.name" :value="tpl.id" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="uploadForm.mode === 'text'" label="文本" required>
          <el-input
            v-model="uploadForm.text"
            type="textarea"
            :rows="8"
            placeholder="每一行会生成一个兑换内容并自动生成一个 CDK"
          />
        </el-form-item>
        <el-form-item v-else label="文件" required>
          <el-upload ref="uploadRef" :auto-upload="false" :limit="1" accept=".txt,.csv"
            :on-change="handleUploadChange" :on-remove="handleUploadRemove">
            <el-button type="primary">选择文件</el-button>
            <template #tip>单文件上传，每一行会生成一个兑换内容并自动生成一个 CDK</template>
          </el-upload>
        </el-form-item>
        <el-form-item>
          <el-button type="success" :loading="uploading" @click="handleLineUpload">
            开始生成
          </el-button>
        </el-form-item>
      </el-form>
      <div v-if="uploadResult" class="upload-result">
        共 {{ uploadResult.total }} 行，成功 {{ uploadResult.inserted }} 行，失败 {{ uploadResult.invalid?.length || 0 }} 行
        <div v-if="uploadResult.codes?.length" class="code-result">
          <div v-for="item in uploadResult.codes" :key="item.code" class="code-row">
            <code>{{ item.code }}</code>
            <span>{{ item.item_name }}</span>
          </div>
        </div>
        <div v-for="msg in uploadResult.invalid" :key="msg" class="invalid-row">{{ msg }}</div>
      </div>
    </div>

    <div class="toolbar">
      <el-input v-model="keyword" placeholder="搜索名称或文件名" clearable style="width:240px" @change="fetchData" />
    </div>

    <div class="glass-card" style="margin-top:16px">
      <el-table :data="list" v-loading="loading" style="width:100%">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="名称" width="180" />
        <el-table-column label="归属" width="130">
          <template #default="{ row }">{{ row.creator?.username || (canEdit(row) ? '我的' : '团队') }}</template>
        </el-table-column>
        <el-table-column label="分类" width="160">
          <template #default="{ row }">{{ row.category?.name || '-' }}</template>
        </el-table-column>
        <el-table-column prop="filename" label="文件名" width="220" />
        <el-table-column label="CDK" width="190">
          <template #default="{ row }">
            <code class="cdk-code">{{ row.cdk?.code || '-' }}</code>
          </template>
        </el-table-column>
        <el-table-column label="模板" width="160">
          <template #default="{ row }">{{ row.template?.name || '-' }}</template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'danger'" size="small">
              {{ row.status === 'active' ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="内容预览" min-width="260">
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

    <el-dialog v-model="dialogVisible" :title="editingId ? '编辑兑换内容' : '新增兑换内容'" width="720px">
      <el-form :model="form" label-width="90px">
        <el-form-item label="名称" required>
          <el-input v-model="form.name" placeholder="例如: 教程资料" />
        </el-form-item>
        <el-form-item label="文件名">
          <el-input v-model="form.filename" placeholder="例如: guide.txt" />
        </el-form-item>
        <el-form-item label="分类" required>
          <el-select v-model="form.category_id" placeholder="请选择分类" style="width:240px">
            <el-option v-for="category in categories" :key="category.id" :label="category.name" :value="category.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="form.status" style="width:160px">
            <el-option label="启用" value="active" />
            <el-option label="禁用" value="disabled" />
          </el-select>
        </el-form-item>
        <el-form-item label="文本内容" required>
          <el-input v-model="form.content" type="textarea" :rows="12" placeholder="兑换成功后下载的 txt 内容" />
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
import type { UploadInstance } from 'element-plus'
import { deleteRedeemItem, getRedeemCategories, getRedeemItems, getTemplates, importRedeemItemFiles, updateRedeemItem } from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'

const list = ref<any[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const keyword = ref('')

const dialogVisible = ref(false)
const editingId = ref(0)
const form = reactive({ name: '', filename: '', content: '', category_id: undefined as number | undefined, status: 'active' })
const importModeOptions = [
  { label: '输入文本', value: 'text' },
  { label: '上传文件', value: 'file' },
]
const uploadFile = ref<File | null>(null)
const uploadRef = ref<UploadInstance>()
const uploadForm = reactive({
  mode: 'text',
  text: '',
  category_id: undefined as number | undefined,
  template_id: undefined as number | undefined,
})
const uploading = ref(false)
const uploadResult = ref<any>(null)
const templates = ref<any[]>([])
const categories = ref<any[]>([])
const currentUser = JSON.parse(localStorage.getItem('user') || '{}')

onMounted(async () => {
  await fetchCategories()
  await fetchTemplates()
  fetchData()
})

async function fetchCategories() {
  try {
    const data: any = await getRedeemCategories({ page: 1, page_size: 100, status: 'active' })
    categories.value = data.list
    if (!uploadForm.category_id && categories.value.length) uploadForm.category_id = categories.value[0].id
  } catch {}
}

async function fetchTemplates() {
  try {
    const data: any = await getTemplates({ page: 1, page_size: 100, status: 'active' })
    templates.value = data.list
    if (!uploadForm.template_id && templates.value.length) uploadForm.template_id = templates.value[0].id
  } catch {}
}

async function fetchData() {
  loading.value = true
  try {
    const data: any = await getRedeemItems({ page: page.value, page_size: pageSize.value, keyword: keyword.value })
    list.value = data.list
    total.value = data.total
  } catch {}
  loading.value = false
}

function handleUploadChange(upload: any) {
  uploadFile.value = upload.raw
}

function handleUploadRemove() {
  uploadFile.value = null
}

async function handleLineUpload() {
  if (!uploadForm.category_id) { ElMessage.warning('请选择分类'); return }
  if (!uploadForm.template_id) { ElMessage.warning('请选择模板'); return }
  if (uploadForm.mode === 'text' && !uploadForm.text.trim()) { ElMessage.warning('请输入文本内容'); return }
  if (uploadForm.mode === 'file' && !uploadFile.value) { ElMessage.warning('请选择文件'); return }
  uploading.value = true
  try {
    const fd = new FormData()
    fd.append('category_id', String(uploadForm.category_id))
    fd.append('template_id', String(uploadForm.template_id))
    if (uploadForm.mode === 'text') {
      fd.append('text', uploadForm.text)
    } else if (uploadFile.value) {
      fd.append('file', uploadFile.value)
    }
    uploadResult.value = await importRedeemItemFiles(fd)
    ElMessage.success('生成完成')
    if (uploadForm.mode === 'text') {
      uploadForm.text = ''
    } else {
      uploadFile.value = null
      uploadRef.value?.clearFiles()
    }
    fetchData()
  } catch {}
  uploading.value = false
}

function openEdit(row: any) {
  editingId.value = row.id
  form.name = row.name
  form.filename = row.filename
  form.content = row.content
  form.category_id = row.category_id
  form.status = row.status
  dialogVisible.value = true
}

async function handleSave() {
  if (!form.name.trim()) { ElMessage.warning('请输入名称'); return }
  if (!form.category_id) { ElMessage.warning('请选择分类'); return }
  if (!form.content) { ElMessage.warning('请输入文本内容'); return }
  const payload = { name: form.name, filename: form.filename, content: form.content, category_id: form.category_id, status: form.status }
  try {
    await updateRedeemItem(editingId.value, payload)
    ElMessage.success('保存成功')
    dialogVisible.value = false
    fetchData()
  } catch {}
}

async function handleDelete(id: number) {
  try {
    await ElMessageBox.confirm('确认删除该兑换内容?', '提示', { type: 'warning' })
    await deleteRedeemItem(id)
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
  margin-top: 16px;
}

.upload-panel {
  padding: 20px 24px;
  margin-bottom: 16px;
  display: flex;
  align-items: flex-start;
  gap: 20px;
  flex-wrap: wrap;
  h3 {
    width: 100%;
    font-family: var(--font-heading);
    font-size: 16px;
  }
}

.upload-form {
  width: 100%;
}

.upload-result {
  width: 100%;
  color: var(--foreground-muted);
}

.invalid-row {
  font-size: 12px;
  color: var(--danger);
  padding-top: 4px;
}

.code-result {
  margin-top: 8px;
  display: grid;
  gap: 6px;
}

.code-row {
  display: flex;
  gap: 12px;
  align-items: center;
  color: var(--foreground-muted);
}

.cdk-code,
.code-row code {
  font-family: var(--font-heading);
  color: var(--accent);
}

.content-preview {
  max-height: 44px;
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
