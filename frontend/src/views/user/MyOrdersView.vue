<template>
  <div>
    <h2 class="page-title">我的领取记录</h2>
    <div class="glass-card">
      <el-table :data="list" style="width: 100%" v-loading="loading" row-key="id">
        <el-table-column prop="id" label="订单号" width="100" />
        <el-table-column prop="amount" label="面额" width="120">
          <template #default="{ row }">{{ row.amount }} 元</template>
        </el-table-column>
        <el-table-column prop="quantity" label="数量" width="100" />
        <el-table-column prop="total_amount" label="总金额" width="130">
          <template #default="{ row }">{{ row.total_amount }} 元</template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'success' ? 'success' : 'danger'" size="small">
              {{ row.status === 'success' ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="时间" width="180" />
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button text size="small" type="success" @click="showDetail(row)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination-wrap">
        <el-pagination v-model:current-page="page" :page-size="pageSize" :total="total"
          layout="prev, pager, next" @current-change="fetchData" />
      </div>
    </div>

    <el-dialog v-model="detailVisible" title="订单详情" width="500px">
      <div v-if="detail">
        <p>订单号: {{ detail.id }}</p>
        <p>面额: {{ detail.amount }} 元 × {{ detail.quantity }} 张</p>
        <p>总金额: {{ detail.total_amount }} 元</p>
        <div v-if="detail.items?.length" style="margin-top:16px">
          <h4>CDK 码:</h4>
          <div v-for="item in detail.items" :key="item.code"
            style="padding:6px 0;font-family:var(--font-heading);color:var(--accent)">
            {{ item.code }}
          </div>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import axios from 'axios'

const api = axios.create({ baseURL: '/api' })
const list = ref<any[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const detailVisible = ref(false)
const detail = ref<any>(null)

onMounted(() => fetchData())

async function fetchData() {
  loading.value = true
  try {
    // 直接用 axios 获取完整 order 数据包括 items
    const token = localStorage.getItem('token')
    const res = await api.get('/user/orders', {
      params: { page: page.value, page_size: pageSize.value },
      headers: { Authorization: `Bearer ${token}` },
    })
    const data = res.data
    if (data.code === 0) {
      list.value = data.data.list
      total.value = data.data.total
    }
  } catch {}
  loading.value = false
}

async function showDetail(row: any) {
  const token = localStorage.getItem('token')
  try {
    const res = await api.get(`/user/orders/${row.id}`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    detail.value = res.data.data
    detailVisible.value = true
  } catch {}
}
</script>

<style scoped lang="scss">
.pagination-wrap {
  display: flex;
  justify-content: flex-end;
  padding: 16px 0 0;
}
</style>
