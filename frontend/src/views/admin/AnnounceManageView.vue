<template>
  <div class="page-container">
    <h1 class="page-title">公告管理</h1>
    <el-button type="success" @click="openCreate" style="margin-bottom:16px">新增公告</el-button>

    <div class="glass-card">
      <el-table :data="list" v-loading="loading" style="width:100%">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="title" label="标题" width="200" />
        <el-table-column prop="is_pinned" label="置顶" width="80">
          <template #default="{ row }">
            <el-tag :type="row.is_pinned ? 'success' : 'info'" size="small">
              {{ row.is_pinned ? '是' : '否' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="内容" min-width="300">
          <template #default="{ row }">
            <div class="content-preview" v-html="row.content" />
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="时间" width="180" />
        <el-table-column label="操作" width="160">
          <template #default="{ row }">
            <el-button text size="small" type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button text size="small" type="danger" @click="handleDelete(row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination-wrap">
        <el-pagination v-model:current-page="page" :page-size="pageSize" :total="total"
          layout="prev, pager, next" @current-change="fetchData" />
      </div>
    </div>

    <!-- 新增/编辑弹窗 -->
    <el-dialog v-model="dialogVisible" :title="editingId ? '编辑公告' : '新增公告'" width="720px">
      <el-form :model="form" label-width="60px">
        <el-form-item label="标题">
          <el-input v-model="form.title" placeholder="公告标题" />
        </el-form-item>
        <el-form-item label="置顶">
          <el-switch v-model="form.is_pinned" />
        </el-form-item>
        <el-form-item label="内容">
          <div class="editor-wrap">
            <div v-if="editor" class="editor-toolbar">
              <button @click="editor.chain().focus().toggleBold().run()" :class="{ active: editor.isActive('bold') }">B</button>
              <button @click="editor.chain().focus().toggleItalic().run()" :class="{ active: editor.isActive('italic') }">I</button>
              <button @click="editor.chain().focus().toggleHeading({ level: 2 }).run()" :class="{ active: editor.isActive('heading', { level: 2 }) }">H2</button>
              <button @click="editor.chain().focus().toggleBulletList().run()" :class="{ active: editor.isActive('bulletList') }">• 列表</button>
            </div>
            <editor-content :editor="editor" />
          </div>
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
import { ref, reactive, onMounted, onBeforeUnmount } from 'vue'
import { getAnnouncements, createAnnouncement, updateAnnouncement, deleteAnnouncement } from '@/api'
import { useEditor, EditorContent } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import { ElMessage, ElMessageBox } from 'element-plus'

const list = ref<any[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

const dialogVisible = ref(false)
const editingId = ref(0)
const form = reactive({ title: '', content: '', is_pinned: false })

const editor = useEditor({
  content: '',
  extensions: [StarterKit],
})

onMounted(() => fetchData())
onBeforeUnmount(() => editor.value?.destroy())

async function fetchData() {
  loading.value = true
  try {
    const data: any = await getAnnouncements({ page: page.value, page_size: pageSize.value })
    list.value = data.list
    total.value = data.total
  } catch {}
  loading.value = false
}

function openCreate() {
  editingId.value = 0
  form.title = ''
  form.is_pinned = false
  editor.value?.commands.setContent('')
  dialogVisible.value = true
}

function openEdit(row: any) {
  editingId.value = row.id
  form.title = row.title
  form.is_pinned = row.is_pinned
  editor.value?.commands.setContent(row.content)
  dialogVisible.value = true
}

async function handleSave() {
  if (!form.title) { ElMessage.warning('请输入标题'); return }
  try {
    const payload = {
      title: form.title,
      content: editor.value?.getHTML() || '',
      is_pinned: form.is_pinned,
    }
    if (editingId.value) {
      await updateAnnouncement(editingId.value, payload)
    } else {
      await createAnnouncement(payload)
    }
    ElMessage.success('保存成功')
    dialogVisible.value = false
    fetchData()
  } catch {}
}

async function handleDelete(id: number) {
  try {
    await ElMessageBox.confirm('确认删除?', '提示', { type: 'warning' })
    await deleteAnnouncement(id)
    ElMessage.success('已删除')
    fetchData()
  } catch {}
}
</script>

<style scoped lang="scss">
.content-preview {
  max-height: 40px;
  overflow: hidden;
  color: var(--foreground-muted);
  font-size: 13px;
}

.editor-wrap {
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-elevated);
  width: 100%;
}

.editor-toolbar {
  display: flex;
  gap: 4px;
  padding: 8px;
  border-bottom: 1px solid var(--border);
  button {
    padding: 4px 10px;
    border: 1px solid var(--border);
    border-radius: 4px;
    background: transparent;
    color: var(--foreground-muted);
    cursor: pointer;
    font-size: 13px;
    &.active {
      background: var(--accent-glow);
      color: var(--accent);
      border-color: var(--accent);
    }
  }
}

:deep(.tiptap) {
  padding: 12px;
  min-height: 200px;
  outline: none;
  color: var(--foreground);
  h2 { font-size: 20px; margin: 8px 0; }
  ul { padding-left: 24px; }
  p { margin: 4px 0; }
}

:deep(.ProseMirror) {
  min-height: 200px;
  outline: none;
}

.pagination-wrap {
  display: flex;
  justify-content: flex-end;
  padding: 16px;
}
</style>
