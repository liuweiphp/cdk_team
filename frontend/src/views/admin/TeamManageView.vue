<template>
  <div class="page-container">
    <h1 class="page-title">团队管理</h1>

    <div class="glass-card team-panel">
      <div class="panel-head">
        <div>
          <h3>我的团队</h3>
          <p>{{ myTeam?.name || '-' }}</p>
        </div>
        <el-button type="success" @click="fetchData">刷新</el-button>
      </div>
      <el-table :data="myMembers" v-loading="loading" style="width:100%">
        <el-table-column label="成员" min-width="180">
          <template #default="{ row }">{{ row.member?.username || '-' }}</template>
        </el-table-column>
        <el-table-column label="角色" width="120">
          <template #default="{ row }">
            <el-tag :type="row.member_id === myTeam?.owner_id ? 'success' : 'info'" size="small">
              {{ row.member_id === myTeam?.owner_id ? '拥有者' : '成员' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="加入时间" width="180" />
        <el-table-column label="操作" width="120">
          <template #default="{ row }">
            <el-button v-if="row.member_id !== myTeam?.owner_id" text size="small" type="danger" @click="handleRemove(row.member_id)">
              移除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <div class="glass-card team-panel">
      <div class="panel-head">
        <div>
          <h3>加入团队</h3>
          <p>加入后可以查看该用户的 CDK、兑换内容和模板，不能修改。</p>
        </div>
      </div>
      <div class="join-row">
        <el-input v-model="ownerUsername" placeholder="团队拥有者用户名" clearable style="width:280px" />
        <el-button type="success" :loading="joining" @click="handleJoin">加入</el-button>
      </div>
    </div>

    <div class="glass-card team-panel">
      <h3>已加入的团队</h3>
      <el-table :data="joinedTeams" v-loading="loading" style="width:100%;margin-top:12px">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="团队" min-width="180" />
        <el-table-column label="拥有者" width="180">
          <template #default="{ row }">{{ row.owner?.username || '-' }}</template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180" />
      </el-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getJoinedTeams, getMyTeam, joinTeam, removeTeamMember } from '@/api'

const loading = ref(false)
const joining = ref(false)
const ownerUsername = ref('')
const myTeam = ref<any>(null)
const joinedTeams = ref<any[]>([])
const myMembers = computed(() => myTeam.value?.members || [])

onMounted(() => fetchData())

async function fetchData() {
  loading.value = true
  try {
    myTeam.value = await getMyTeam()
    const joined: any = await getJoinedTeams()
    joinedTeams.value = joined.list || []
  } catch {}
  loading.value = false
}

async function handleJoin() {
  if (!ownerUsername.value.trim()) {
    ElMessage.warning('请输入团队拥有者用户名')
    return
  }
  joining.value = true
  try {
    await joinTeam(ownerUsername.value.trim())
    ElMessage.success('已加入团队')
    ownerUsername.value = ''
    fetchData()
  } catch {}
  joining.value = false
}

async function handleRemove(memberId: number) {
  try {
    await ElMessageBox.confirm('确认移除该成员?', '提示', { type: 'warning' })
    await removeTeamMember(memberId)
    ElMessage.success('已移除')
    fetchData()
  } catch {}
}
</script>

<style scoped lang="scss">
.team-panel {
  padding: 20px 24px;
  margin-bottom: 16px;
}

.panel-head {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: center;
  margin-bottom: 14px;
}

h3 {
  font-family: var(--font-heading);
  font-size: 16px;
  margin-bottom: 6px;
}

p {
  color: var(--foreground-muted);
  font-size: 13px;
}

.join-row {
  display: flex;
  gap: 12px;
  align-items: center;
}
</style>
