<template>
  <div class="admin-layout">
    <aside class="admin-sidebar glass-panel">
      <div class="sidebar-brand">
        <h2><span class="accent">CDK</span> Admin</h2>
      </div>
      <nav class="sidebar-nav">
        <router-link v-if="isAdmin" to="/admin" class="nav-item" exact-active-class="active">
          <span class="nav-icon">◆</span> 仪表盘
        </router-link>
        <router-link to="/admin/cdk" class="nav-item" active-class="active">
          <span class="nav-icon">▣</span> CDK 管理
        </router-link>
        <router-link to="/admin/redeem-categories" class="nav-item" active-class="active">
          <span class="nav-icon">◫</span> 分类管理
        </router-link>
        <router-link to="/admin/redeem-items" class="nav-item" active-class="active">
          <span class="nav-icon">◇</span> 兑换内容
        </router-link>
        <router-link to="/admin/templates" class="nav-item" active-class="active">
          <span class="nav-icon">▤</span> 模板管理
        </router-link>
        <router-link to="/admin/teams" class="nav-item" active-class="active">
          <span class="nav-icon">▥</span> 团队管理
        </router-link>
        <router-link v-if="isAdmin" to="/admin/users" class="nav-item" active-class="active">
          <span class="nav-icon">👥</span> 用户管理
        </router-link>
        <router-link v-if="isAdmin" to="/admin/announcements" class="nav-item" active-class="active">
          <span class="nav-icon">📢</span> 公告管理
        </router-link>
      </nav>
      <div class="sidebar-footer">
        <a class="nav-item" @click="showPwd = true">
          <span class="nav-icon">🔑</span> 修改密码
        </a>
        <router-link to="/exchange" class="nav-item">
          <span class="nav-icon">←</span> 返回用户端
        </router-link>
        <a class="nav-item logout" @click="handleLogout">
          <span class="nav-icon">⏻</span> 退出
        </a>
      </div>
    </aside>
    <main class="admin-main">
      <router-view />
    </main>

    <el-dialog v-model="showPwd" title="修改密码" width="400px">
      <el-form :model="pwdForm" label-width="80px">
        <el-form-item label="旧密码">
          <el-input v-model="pwdForm.oldPwd" type="password" show-password />
        </el-form-item>
        <el-form-item label="新密码">
          <el-input v-model="pwdForm.newPwd" type="password" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showPwd = false">取消</el-button>
        <el-button type="success" :loading="pwdLoading" @click="handleChangePwd">确认</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { changePassword } from '@/api'
import { ElMessage } from 'element-plus'

const router = useRouter()
const showPwd = ref(false)
const pwdLoading = ref(false)
const pwdForm = reactive({ oldPwd: '', newPwd: '' })
const currentUser = JSON.parse(localStorage.getItem('user') || '{}')
const isAdmin = currentUser.role === 'admin'

function handleLogout() {
  localStorage.removeItem('token')
  localStorage.removeItem('user')
  router.push('/login')
}

async function handleChangePwd() {
  if (!pwdForm.oldPwd || !pwdForm.newPwd) { ElMessage.warning('请填写完整'); return }
  if (pwdForm.newPwd.length < 8) { ElMessage.warning('新密码至少8位'); return }
  pwdLoading.value = true
  try {
    await changePassword(pwdForm.oldPwd, pwdForm.newPwd)
    ElMessage.success('密码已修改')
    showPwd.value = false
    pwdForm.oldPwd = ''
    pwdForm.newPwd = ''
  } catch {}
  pwdLoading.value = false
}
</script>

<style scoped lang="scss">
.admin-layout {
  display: flex;
  height: 100vh;
  background: var(--bg-deep);
}

.admin-sidebar {
  width: 240px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  margin: 8px 0 8px 8px;
  padding: 20px 0;
  border-radius: var(--radius);
}

.sidebar-brand h2 {
  font-family: var(--font-heading);
  font-size: 18px;
  padding: 0 20px 20px;
  color: var(--foreground);
  .accent { color: var(--accent); }
}

.sidebar-nav {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 0 12px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: var(--radius-sm);
  color: var(--foreground-muted);
  text-decoration: none;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
  &:hover { background: var(--surface-hover); color: var(--foreground); }
  &.active, &.router-link-exact-active {
    background: var(--accent-glow);
    color: var(--accent);
    .nav-icon { text-shadow: 0 0 8px var(--accent-glow); }
  }
}

.sidebar-footer {
  padding: 20px 12px 0;
  border-top: 1px solid var(--border);
  margin-top: 8px;
}

.admin-main {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
}
</style>
