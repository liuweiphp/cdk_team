<template>
  <div class="login-bg">
    <div class="login-card glass-card">
      <h1 class="login-title"><span class="accent">CDK</span> Exchange</h1>
      <p class="login-sub">兑换码管理系统</p>
      <el-form ref="formRef" :model="form" :rules="rules" @submit.prevent="handleLogin">
        <el-form-item prop="username">
          <el-input v-model="form.username" placeholder="用户名" size="large" />
        </el-form-item>
        <el-form-item prop="password">
          <el-input v-model="form.password" type="password" placeholder="密码" size="large"
            show-password @keyup.enter="handleLogin" />
        </el-form-item>
        <el-button type="success" size="large" :loading="loading" class="glow-btn login-btn"
          @click="handleLogin" native-type="submit">
          登 录
        </el-button>
      </el-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { login } from '@/api'
import { ElMessage } from 'element-plus'

const router = useRouter()
const loading = ref(false)
const form = reactive({ username: '', password: '' })
const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

async function handleLogin() {
  loading.value = true
  try {
    const data: any = await login(form.username, form.password)
    localStorage.setItem('token', data.token)
    localStorage.setItem('user', JSON.stringify(data.user))
    ElMessage.success('登录成功')
    if (data.user.role === 'admin') router.push('/admin')
    else router.push('/exchange')
  } catch {}
  loading.value = false
}
</script>

<style scoped lang="scss">
.login-bg {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: radial-gradient(ellipse at 50% 50%, rgba(34,197,94,0.04) 0%, transparent 60%),
              radial-gradient(ellipse at 80% 20%, rgba(99,102,241,0.06) 0%, transparent 50%),
              var(--bg-deep);
}

.login-card {
  width: 420px;
  padding: 48px 40px;
  text-align: center;
}

.login-title {
  font-family: var(--font-heading);
  font-size: 28px;
  color: var(--foreground);
  margin-bottom: 4px;
  .accent { color: var(--accent); text-shadow: 0 0 16px var(--accent-glow); }
}

.login-sub {
  color: var(--foreground-muted);
  margin-bottom: 32px;
  font-size: 14px;
}

.login-btn {
  width: 100%;
  height: 44px;
  font-size: 16px;
}
</style>
