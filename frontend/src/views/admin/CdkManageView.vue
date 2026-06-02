<template>
  <div class="page-container">
    <h1 class="page-title">CDK 管理</h1>
    <!-- 筛选 -->
    <div class="filters glass-card">
      <el-input v-model="filters.code" placeholder="搜索 CDK 码" clearable style="width:200px" @change="search" />
      <el-select v-model="filters.item_id" placeholder="兑换内容" clearable style="width:220px" @change="search">
        <el-option v-for="item in items" :key="item.id" :label="item.name" :value="item.id" />
      </el-select>
      <el-select v-model="filters.status" placeholder="状态" clearable style="width:130px" @change="search">
        <el-option label="未使用" value="unused" />
        <el-option label="已领取" value="exchanged" />
      </el-select>
      <el-button type="success" @click="search">查询</el-button>
    </div>
    <!-- 表格 -->
    <div class="glass-card" style="margin-top:16px">
      <el-table :data="list" v-loading="loading" style="width:100%">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="code" label="CDK 码" width="200">
          <template #default="{ row }">
            <code style="color:var(--accent);font-family:var(--font-heading)">{{ row.code }}</code>
          </template>
        </el-table-column>
        <el-table-column label="兑换内容" width="180">
          <template #default="{ row }">{{ row.redeem_item?.name || '-' }}</template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status==='unused'?'info':'success'" size="small">
              {{ row.status==='unused'?'未使用':'已领取' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="import_id" label="导入批次" width="100" />
        <el-table-column prop="exchanged_at" label="领取时间" width="180">
          <template #default="{ row }">{{ row.exchanged_at || '-' }}</template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180" />
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button text size="small" type="primary" @click="downloadItem(row)">下载</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination-wrap">
        <el-pagination v-model:current-page="page" :page-size="pageSize" :total="total"
          layout="prev, pager, next" @current-change="fetchData" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getCdkList, getRedeemItems } from '@/api'

const list = ref<any[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const items = ref<any[]>([])
const filters = ref({ code: '', item_id: '', status: '' })

onMounted(async () => {
  await fetchData()
  try {
    const data: any = await getRedeemItems({ page: 1, page_size: 100 })
    items.value = data.list
  } catch {}
})

async function fetchData() {
  loading.value = true
  try {
    const params: any = { page: page.value, page_size: pageSize.value }
    if (filters.value.code) params.code = filters.value.code
    if (filters.value.item_id) params.item_id = Number(filters.value.item_id)
    if (filters.value.status) params.status = filters.value.status
    const data: any = await getCdkList(params)
    list.value = data.list
    total.value = data.total
  } catch {}
  loading.value = false
}

function search() {
  page.value = 1
  fetchData()
}

function downloadItem(row: any) {
  const item = row.redeem_item
  if (!item?.content) {
    return
  }
  const filename = item.filename || `${row.code}.txt`
  const blob = new Blob([item.content], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

</script>

<style scoped lang="scss">
.filters {
  padding: 16px;
  display: flex;
  gap: 12px;
  align-items: center;
}
.pagination-wrap {
  display: flex;
  justify-content: flex-end;
  padding: 16px;
}
</style>
