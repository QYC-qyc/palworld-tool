<template>
  <div class="auth-wrap" :class="{ 'auth-wrap--dark': isDark }">
    <div class="auth-card">
      <div class="brand">
        <div class="brand__logo">
          <n-icon :component="GameControllerOutline" size="30" color="#fff" />
        </div>
        <h1 class="brand__title">PalAdmin</h1>
        <p class="brand__subtitle">幻兽帕鲁服务器管理面板</p>
      </div>
      <n-space vertical :size="16">
        <n-input
          v-model:value="password"
          type="password"
          show-password-on="click"
          placeholder="管理员密码"
          size="large"
          @keyup.enter="login"
        />
        <n-button type="primary" size="large" block :loading="loading" @click="login">
          登录
        </n-button>
      </n-space>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { NSpace, NInput, NButton, NIcon, useMessage } from 'naive-ui'
import { GameControllerOutline } from '@vicons/ionicons5'
import { api } from '@/api'
import { useTheme } from '@/composables/useTheme'

const { isDark } = useTheme()
const message = useMessage()
const password = ref('')
const loading = ref(false)
const router = useRouter()

async function login() {
  if (!password.value) return
  loading.value = true
  try {
    const { token } = await api.login(password.value)
    localStorage.setItem('paladmin_token', token)
    router.push('/dashboard')
  } catch (e: any) {
    message.error(e.message || '登录失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.auth-wrap {
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: linear-gradient(135deg, #312e81 0%, #1e1b4b 50%, #0f172a 100%);
}
.auth-card {
  width: 100%;
  max-width: 380px;
  padding: 32px 28px;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.97);
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.35);
}
.auth-wrap--dark .auth-card {
  background: rgba(30, 30, 36, 0.97);
}
.brand {
  text-align: center;
  margin-bottom: 24px;
}
.brand__logo {
  width: 60px;
  height: 60px;
  margin: 0 auto 14px;
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #6366f1, #4f46e5);
  box-shadow: 0 8px 20px rgba(79, 70, 229, 0.4);
}
.brand__title {
  font-size: 24px;
  font-weight: 700;
  margin: 0;
}
.brand__subtitle {
  font-size: 13px;
  color: #888;
  margin: 6px 0 0;
}
</style>
