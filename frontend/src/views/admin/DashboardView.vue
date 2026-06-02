<template>
  <div class="page-container">
    <h1 class="page-title">仪表盘</h1>

    <!-- 概览卡片 -->
    <div class="stat-grid">
      <div class="stat-card">
        <div class="stat-value">{{ overview.user_count }}</div>
        <div class="stat-label">总用户数</div>
      </div>
      <div class="stat-card">
        <div class="stat-value">{{ overview.cdk_total }}</div>
        <div class="stat-label">CDK 总数</div>
      </div>
      <div class="stat-card">
        <div class="stat-value" style="color: var(--info)">
          {{ overview.cdk_exchanged }}</div>
        <div class="stat-label">已兑换</div>
      </div>
      <div class="stat-card">
        <div class="stat-value" style="color: var(--warning)">
          {{ overview.cdk_remaining }}</div>
        <div class="stat-label">剩余</div>
      </div>
    </div>

    <!-- 按内容柱状图 -->
    <div class="chart-row">
      <div class="chart-card glass-card">
        <h3>按兑换内容统计</h3>
        <v-chart :option="byItemOption" style="height:340px" autoresize />
      </div>
      <div class="chart-card glass-card">
        <h3>每日兑换趋势</h3>
        <v-chart :option="dailyOption" style="height:340px" autoresize />
      </div>
    </div>

    <!-- Top 用户 -->
    <div class="chart-card glass-card" style="margin-top:20px">
      <h3>兑换排行 TOP 10</h3>
      <el-table :data="topUsers" style="width:100%" v-loading="loadingTop">
        <el-table-column prop="username" label="用户" width="200" />
        <el-table-column prop="count" label="兑换次数" width="150" />
      </el-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { getStatsOverview, getStatsByItem, getStatsDaily, getTopUsers } from '@/api'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { BarChart, LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'

use([CanvasRenderer, BarChart, LineChart, GridComponent, TooltipComponent, LegendComponent])

const overview = ref<Record<string, number>>({})
const byItem = ref<any[]>([])
const daily = ref<any[]>([])
const topUsers = ref<any[]>([])
const loadingTop = ref(false)

onMounted(async () => {
  try {
    overview.value = await getStatsOverview() as any
    byItem.value = await getStatsByItem() as any
    // 默认查最近30天
    const end = new Date().toISOString().slice(0, 10)
    const start = new Date(Date.now() - 30*86400000).toISOString().slice(0, 10)
    daily.value = await getStatsDaily(start, end) as any
    loadingTop.value = true
    topUsers.value = await getTopUsers(10) as any
    loadingTop.value = false
  } catch {}
})

const byItemOption = computed(() => ({
  tooltip: { trigger: 'axis' },
  legend: { data: ['总数', '已兑换', '剩余'], textStyle: { color: '#94a3b8' } },
  grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
  xAxis: { type: 'category', data: byItem.value.map(a => a.item_name), axisLabel: { color: '#94a3b8' } },
  yAxis: { type: 'value', axisLabel: { color: '#94a3b8' } },
  series: [
    { name: '总数', type: 'bar', data: byItem.value.map(a => a.total), itemStyle: { color: '#3b82f6' }, barGap: '10%' },
    { name: '已兑换', type: 'bar', data: byItem.value.map(a => a.exchanged), itemStyle: { color: '#22c55e' } },
    { name: '剩余', type: 'bar', data: byItem.value.map(a => a.remaining), itemStyle: { color: '#f59e0b' } },
  ],
}))

const dailyOption = computed(() => ({
  tooltip: { trigger: 'axis' },
  grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
  xAxis: { type: 'category', data: daily.value.map(d => d.date), axisLabel: { color: '#94a3b8' } },
  yAxis: { type: 'value', axisLabel: { color: '#94a3b8' } },
  series: [{
    name: '兑换次数', type: 'line', smooth: true,
    data: daily.value.map(d => d.count),
    itemStyle: { color: '#22c55e' },
    areaStyle: { color: 'rgba(34,197,94,0.1)' },
  }],
}))

</script>

<style scoped lang="scss">
.stat-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 20px;
  margin-bottom: 20px;
  @media (max-width: 1024px) { grid-template-columns: repeat(2, 1fr); }
}

.chart-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
  @media (max-width: 1024px) { grid-template-columns: 1fr; }
}

.chart-card {
  padding: 24px;
  h3 {
    font-family: var(--font-heading);
    font-size: 16px;
    margin-bottom: 16px;
  }
}
</style>
