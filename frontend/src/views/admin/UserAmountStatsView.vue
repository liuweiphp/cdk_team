<template>
  <div class="page-container">
    <h1 class="page-title">用户领取统计</h1>

    <div class="filter-bar glass-card">
      <div class="filter-row">
        <div class="filter-item">
          <label>{{ periodLabel }}</label>
          <el-date-picker
            v-model="pickerValue"
            :type="pickerType"
            :key="period"
            range-separator="至"
            :start-placeholder="pickerPH.start"
            :end-placeholder="pickerPH.end"
            :value-format="pickerFormat"
            style="width: 280px"
          />
        </div>
        <div class="filter-item">
          <label>统计周期</label>
          <el-radio-group v-model="period" @change="onPeriodChange">
            <el-radio-button value="day">日</el-radio-button>
            <el-radio-button value="week">周</el-radio-button>
            <el-radio-button value="month">月</el-radio-button>
            <el-radio-button value="year">年</el-radio-button>
          </el-radio-group>
        </div>
        <div class="filter-item">
          <el-button type="primary" @click="fetchData" :loading="loading">查询</el-button>
        </div>
      </div>
    </div>

    <div class="table-card glass-card">
      <el-table :data="list" style="width:100%" v-loading="loading" stripe>
        <el-table-column prop="username" label="用户" width="160" />
        <el-table-column label="面额" width="140">
          <template #default="{ row }">{{ fmtMoney(row.amount) }} 元</template>
        </el-table-column>
        <el-table-column label="周期" width="200">
          <template #default="{ row }">{{ fmtPeriod(row.period) }}</template>
        </el-table-column>
        <el-table-column prop="count" label="领取张数" width="120" />
      </el-table>
      <div class="pagination-wrap">
        <el-pagination
          v-model:current-page="page"
          :page-size="pageSize"
          :total="total"
          layout="total, prev, pager, next"
          @current-change="fetchData"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { getStatsByUserAmount } from '@/api'

const loading = ref(false)
const list = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const period = ref('day')

const pickerValue = ref<any>(null)

function initPickerValue() {
  const now = new Date()
  const y = now.getFullYear()
  const m = String(now.getMonth() + 1).padStart(2, '0')
  const d = String(now.getDate()).padStart(2, '0')
  const today = `${y}-${m}-${d}`
  const thisMonth = `${y}-${m}`

  switch (period.value) {
    case 'day':
      pickerValue.value = [today, today]
      break
    case 'week':
      pickerValue.value = today
      break
    case 'month':
      pickerValue.value = [thisMonth, thisMonth]
      break
    case 'year':
      pickerValue.value = String(y)
      break
  }
}

const pickerType = computed(() => {
  const map: Record<string, string> = { day: 'daterange', week: 'week', month: 'monthrange', year: 'year' }
  return map[period.value] || 'daterange'
})

const pickerFormat = computed(() => {
  const map: Record<string, string> = { day: 'YYYY-MM-DD', week: 'YYYY-MM-DD', month: 'YYYY-MM', year: 'YYYY' }
  return map[period.value] || 'YYYY-MM-DD'
})

const periodLabel = computed(() => {
  const map: Record<string, string> = { day: '日期范围', week: '选择周', month: '月份范围', year: '选择年份' }
  return map[period.value] || '日期范围'
})

const pickerPH = computed(() => {
  const map: Record<string, { start: string; end: string }> = {
    day: { start: '开始日期', end: '结束日期' },
    week: { start: '选择周', end: '' },
    month: { start: '开始月份', end: '结束月份' },
    year: { start: '选择年份', end: '' },
  }
  return map[period.value] || map.day
})

initPickerValue()
onMounted(() => { fetchData() })

function onPeriodChange() {
  page.value = 1
  initPickerValue()
  fetchData()
}

async function fetchData() {
  if (!pickerValue.value) return
  loading.value = true
  try {
    let start = '', end = ''

    switch (period.value) {
      case 'day': {
        start = pickerValue.value[0]
        const d = new Date(pickerValue.value[1])
        d.setDate(d.getDate() + 1)
        end = d.toISOString().slice(0, 10)
        break
      }
      case 'week': {
        start = pickerValue.value
        const d = new Date(pickerValue.value)
        d.setDate(d.getDate() + 7)
        end = d.toISOString().slice(0, 10)
        break
      }
      case 'month': {
        start = pickerValue.value[0] + '-01'
        const [y, m] = pickerValue.value[1].split('-').map(Number)
        const d = new Date(y, m, 1)
        end = d.toISOString().slice(0, 10)
        break
      }
      case 'year': {
        start = pickerValue.value + '-01-01'
        end = (Number(pickerValue.value) + 1) + '-01-01'
        break
      }
    }

    const data = await getStatsByUserAmount({
      start, end, period: period.value,
      page: page.value, page_size: pageSize.value,
    }) as any
    list.value = data.list
    total.value = data.total
  } catch {}
  loading.value = false
}

function fmtMoney(v: number) {
  if (!v) return '0.00'
  return Number(v).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

function fmtPeriod(p: string) {
  if (period.value === 'week') {
    const d = new Date(p)
    if (isNaN(d.getTime())) return p
    const end = new Date(d)
    end.setDate(end.getDate() + 6)
    return `${p} ~ ${end.toISOString().slice(0, 10)}`
  }
  return p
}
</script>

<style scoped lang="scss">
.filter-bar {
  padding: 20px 24px;
  margin-bottom: 20px;
}

.filter-row {
  display: flex;
  align-items: flex-end;
  gap: 24px;
  flex-wrap: wrap;
}

.filter-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  label {
    font-size: 13px;
    color: var(--foreground-muted);
  }
}

.table-card {
  padding: 24px;
}

.pagination-wrap {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
</style>
