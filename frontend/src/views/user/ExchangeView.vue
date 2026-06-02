<template>
  <div>
    <div class="exchange-container glass-card">
      <h2 class="section-title">兑换文本文件</h2>
      <div class="exchange-form">
        <div class="form-group">
          <label>兑换码</label>
          <el-input v-model="code" placeholder="请输入兑换码" size="large" clearable
            @keyup.enter="handleRedeem" />
        </div>
        <el-button type="success" size="large" class="glow-btn exchange-btn"
          :loading="redeeming" @click="handleRedeem">
          立即兑换
        </el-button>
      </div>
    </div>

    <el-dialog v-model="showResult" title="兑换成功" width="640px" class="glass-dialog">
      <div class="result-info">
        <p>内容: <strong>{{ result.item_name }}</strong></p>
        <p>文件: <strong>{{ result.filename }}</strong></p>
      </div>
      <div class="text-preview">
        <pre>{{ result.content }}</pre>
      </div>
      <template #footer>
        <el-button @click="copyContent">复制文本</el-button>
        <el-button type="primary" @click="downloadFile">下载 TXT</el-button>
        <el-button type="success" @click="showResult = false">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { redeemCode } from '@/api'
import { ElMessage } from 'element-plus'

const code = ref('')
const redeeming = ref(false)
const showResult = ref(false)
const result = ref<{ item_name: string; filename: string; content: string }>({
  item_name: '', filename: '', content: '',
})

async function handleRedeem() {
  const value = code.value.trim()
  if (!value) {
    ElMessage.warning('请输入兑换码')
    return
  }
  redeeming.value = true
  try {
    result.value = await redeemCode(value) as any
    showResult.value = true
    code.value = ''
  } catch {}
  redeeming.value = false
}

function copyContent() {
  navigator.clipboard.writeText(result.value.content)
  ElMessage.success('文本已复制')
}

function downloadFile() {
  const blob = new Blob([result.value.content], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = result.value.filename || 'redeem.txt'
  a.click()
  URL.revokeObjectURL(url)
}
</script>

<style scoped lang="scss">
.exchange-container {
  padding: 32px;
}

.section-title {
  font-family: var(--font-heading);
  font-size: 20px;
  margin-bottom: 24px;
}

.exchange-form {
  max-width: 400px;
}

.form-group {
  margin-bottom: 20px;
  label {
    display: block;
    margin-bottom: 8px;
    color: var(--foreground-muted);
    font-size: 13px;
  }
}

.exchange-btn {
  width: 100%;
}

.result-info {
  display: flex;
  gap: 24px;
  margin-bottom: 20px;
  p strong { color: var(--accent); }
}

.text-preview {
  max-height: 360px;
  overflow-y: auto;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 16px;
  pre {
    white-space: pre-wrap;
    word-break: break-word;
    color: var(--foreground);
    font-family: var(--font-heading);
    font-size: 13px;
  }
}
</style>
