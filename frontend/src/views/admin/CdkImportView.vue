<template>
  <div class="page-container">
    <h1 class="page-title">兑换码导入</h1>

    <div class="glass-card import-form">
      <h3>上传兑换码映射文件</h3>
      <el-form :model="form" label-width="80px">
        <el-form-item label="文件">
          <el-upload :auto-upload="false" :limit="1" accept=".csv,.txt,.xlsx,.xls"
            :on-change="handleFileChange" :on-remove="() => file = null">
            <el-button type="primary">选择文件</el-button>
            <template #tip>支持 CSV/TXT: 每行“兑换码,文件名”；Excel: 第一列兑换码、第二列文件名。文件名需先在“兑换内容”中上传。</template>
          </el-upload>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" placeholder="可选备注" />
        </el-form-item>
        <el-form-item>
          <el-button type="success" size="large" :loading="importing" @click="handleImport">
            开始导入
          </el-button>
        </el-form-item>
      </el-form>

      <!-- 导入结果 -->
      <div v-if="result" class="import-result glass-panel">
        <h4>导入结果</h4>
        <p>总行数: {{ result.total }} | 成功: <span style="color:var(--accent)">{{ result.inserted }}</span>
          | 跳过: {{ result.skipped }} | 无效: <span style="color:var(--danger)">{{ result.invalid?.length }}</span></p>
        <div v-if="result.invalid?.length" style="margin-top:12px">
          <h4>无效行:</h4>
          <div v-for="inv in result.invalid" :key="inv.line" class="invalid-row">
            #{{ inv.line }}: {{ inv.code }} - {{ inv.reason }}
          </div>
        </div>
      </div>
    </div>

    <!-- 导入历史 -->
    <div class="glass-card" style="margin-top:20px">
      <h3 style="padding:16px 16px 0">导入历史</h3>
      <el-table :data="history" v-loading="loadingHistory" style="width:100%">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="filename" label="文件名" />
        <el-table-column label="兑换内容" width="200">
          <template #default="{ row }">{{ row.redeem_item?.name || '-' }}</template>
        </el-table-column>
        <el-table-column prop="inserted" label="成功" width="80" />
        <el-table-column prop="skipped" label="重复" width="80" />
        <el-table-column prop="invalid" label="无效" width="80" />
        <el-table-column prop="remark" label="备注" width="150" />
        <el-table-column prop="created_at" label="时间" width="180" />
      </el-table>
      <div class="pagination-wrap">
        <el-pagination v-model:current-page="hPage" :page-size="20" :total="hTotal"
          layout="prev, pager, next" @current-change="fetchHistory" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { importCdk, getImportHistory } from '@/api'
import { ElMessage } from 'element-plus'

const form = ref({ remark: '' })
const file = ref<File | null>(null)
const importing = ref(false)
const result = ref<any>(null)

const history = ref<any[]>([])
const loadingHistory = ref(false)
const hPage = ref(1)
const hTotal = ref(0)

onMounted(() => fetchHistory())

async function fetchHistory() {
  loadingHistory.value = true
  try {
    const data: any = await getImportHistory({ page: hPage.value, page_size: 20 })
    history.value = data.list
    hTotal.value = data.total
  } catch {}
  loadingHistory.value = false
}

function handleFileChange(uploadFile: any) {
  file.value = uploadFile.raw
}

async function handleImport() {
  if (!file.value) { ElMessage.warning('请选择文件'); return }
  importing.value = true
  try {
    const fd = new FormData()
    fd.append('file', file.value)
    fd.append('remark', form.value.remark)
    result.value = await importCdk(fd)
    ElMessage.success('导入完成')
    fetchHistory()
  } catch {}
  importing.value = false
}
</script>

<style scoped lang="scss">
.import-form { padding: 24px; }
.import-result {
  margin-top: 20px;
  padding: 16px;
  h4 { margin-bottom: 8px; }
}
.invalid-row {
  font-size: 12px;
  color: var(--foreground-muted);
  padding: 2px 0;
}
.pagination-wrap {
  display: flex;
  justify-content: flex-end;
  padding: 16px;
}
</style>
