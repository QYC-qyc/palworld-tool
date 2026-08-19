<template>
  <div class="setup-wrap" :class="{ 'setup-wrap--dark': isDark }">
    <div class="setup-card">
      <div class="brand">
        <div class="brand__logo">
          <n-icon :component="GameControllerOutline" size="30" color="#fff" />
        </div>
        <h1 class="brand__title">初始化 PalAdmin</h1>
        <p class="brand__subtitle">首次使用，请设置管理员密码与服务器连接</p>
      </div>

      <n-alert type="info" :show-icon="false" style="margin-bottom:8px">
        这些信息之后可在「系统设置」中修改。游戏服务器连接可稍后配置。
      </n-alert>

      <n-space vertical :size="16">
        <div class="step">
          <div class="step__no">1</div>
          <div class="step__body">
            <div class="step__title">管理员账号</div>
            <n-input
              v-model:value="form.web_password"
              type="password"
              show-password-on="click"
              placeholder="设置面板登录密码"
            />
          </div>
        </div>

        <div class="step">
          <div class="step__no">2</div>
          <div class="step__body">
            <div class="step__title">游戏服务器连接（可选）</div>
            <n-space vertical :size="10">
              <n-input v-model:value="form.rest_address" placeholder="REST 地址，如 http://palworld:8212" />
              <n-input
                v-model:value="form.rest_password"
                type="password"
                show-password-on="click"
                placeholder="REST API 管理员密码"
              />
            </n-space>
          </div>
        </div>

        <n-button
          type="primary"
          size="large"
          block
          :loading="loading"
          :disabled="!form.web_password"
          @click="submit"
        >
          完成初始化
        </n-button>
      </n-space>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { NSpace, NInput, NButton, NAlert, NIcon, useMessage } from 'naive-ui'
import { GameControllerOutline } from '@vicons/ionicons5'
import { api } from '@/api'
import { useTheme } from '@/composables/useTheme'

const { isDark } = useTheme()
const router = useRouter()
const message = useMessage()
const loading = ref(false)

const form = reactive({
  web_password: '',
  rest_address: 'http://palworld:8212',
  rest_password: '',
})

async function submit() {
  loading.value = true
  try {
    const res = await api.setup({
      web_password: form.web_password,
      rest_address: form.rest_address,
      rest_password: form.rest_password,
    })
    if (res.token) {
      localStorage.setItem('paladmin_token', res.token)
    }
    router.push('/dashboard')
  } catch (e: any) {
    message.error(e.message || '初始化失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.setup-wrap {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px 16px;
  background: linear-gradient(135deg, #312e81 0%, #1e1b4b 50%, #0f172a 100%);
}
.setup-card {
  width: 100%;
  max-width: 480px;
  padding: 32px 28px;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.97);
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.35);
}
.setup-wrap--dark .setup-card {
  background: rgba(30, 30, 36, 0.97);
}
.brand {
  text-align: center;
  margin-bottom: 20px;
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
  font-size: 22px;
  font-weight: 700;
  margin: 0;
}
.brand__subtitle {
  font-size: 13px;
  color: #888;
  margin: 6px 0 0;
}
.step {
  display: flex;
  gap: 12px;
  align-items: flex-start;
}
.step__no {
  flex-shrink: 0;
  width: 26px;
  height: 26px;
  border-radius: 50%;
  background: #4f46e5;
  color: #fff;
  font-size: 13px;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-top: 4px;
}
.step__title {
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 8px;
}
.step__body {
  flex: 1;
}
</style>
