<template>
  <div class="setup-wrap">
    <n-card title="初始化 PalAdmin" class="setup-card" :bordered="false">
      <n-space vertical :size="16">
        <n-alert type="info" :show-icon="false">
          首次使用，请设置管理员密码和游戏服务器连接信息。这些信息之后可在「系统设置」中修改。
        </n-alert>

        <n-divider style="margin: 4px 0">管理员账号</n-divider>
        <n-input
          v-model:value="form.web_password"
          type="password"
          show-password-on="click"
          placeholder="设置面板登录密码"
        />

        <n-divider style="margin: 4px 0">游戏服务器连接（可选，稍后配置）</n-divider>
        <n-input v-model:value="form.rest_address" placeholder="REST 地址，如 http://palworld:8212" />
        <n-input
          v-model:value="form.rest_password"
          type="password"
          show-password-on="click"
          placeholder="游戏 AdminPassword（REST 与 RCON 相同）"
        />
        <n-input v-model:value="form.rcon_address" placeholder="RCON 地址，如 palworld:25575" />

        <n-button
          type="primary"
          block
          size="large"
          :loading="loading"
          :disabled="!form.web_password"
          @click="submit"
        >
          完成初始化
        </n-button>
      </n-space>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { NCard, NSpace, NInput, NButton, NAlert, NDivider, useMessage } from 'naive-ui'
import { api } from '@/api'

const router = useRouter()
const message = useMessage()
const loading = ref(false)

const form = reactive({
  web_password: '',
  rest_address: 'http://palworld:8212',
  rest_password: '',
  rcon_address: 'palworld:25575',
  rcon_password: '',
})

async function submit() {
  loading.value = true
  try {
    // RCON 密码默认与 REST 相同
    const res = await api.setup({
      web_password: form.web_password,
      rest_address: form.rest_address,
      rest_password: form.rest_password,
      rcon_address: form.rcon_address,
      rcon_password: form.rest_password,
    })
    if (res.token) {
      localStorage.setItem('paladmin_token', res.token)
    }
    message.success('初始化完成')
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
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1e3a8a 0%, #312e81 100%);
}
.setup-card {
  width: 460px;
}
</style>
