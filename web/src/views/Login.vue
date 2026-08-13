<template>
  <div class="login-wrap">
    <n-card title="PalAdmin 登录" class="login-card" :bordered="false">
      <n-space vertical :size="16">
        <n-input
          v-model:value="password"
          type="password"
          show-password-on="click"
          placeholder="管理员密码"
          @keyup.enter="login"
        />
        <n-button type="primary" block :loading="loading" @click="login">登录</n-button>
      </n-space>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { NCard, NSpace, NInput, NButton, useMessage } from 'naive-ui'
import { api } from '@/api'

const password = ref('')
const loading = ref(false)
const router = useRouter()
const message = useMessage()

async function login() {
  if (!password.value) return
  loading.value = true
  try {
    const { token } = await api.login(password.value)
    localStorage.setItem('paladmin_token', token)
    message.success('登录成功')
    router.push('/dashboard')
  } catch (e: any) {
    message.error(e.message || '登录失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-wrap {
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1e3a8a 0%, #312e81 100%);
}
.login-card {
  width: 100%;
  max-width: 360px;
  margin: 0 12px;
}
</style>
