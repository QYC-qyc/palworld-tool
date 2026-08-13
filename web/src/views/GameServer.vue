<template>
  <n-space vertical :size="16">
    <n-card title="幻兽帕鲁服务端" size="small">
      <n-space vertical>
        <!-- Docker 不可用 -->
        <n-alert v-if="!dockerAvailable" type="error" title="无法连接 Docker">
          {{ status?.message || '请确认面板容器已挂载 /var/run/docker.sock' }}
        </n-alert>

        <template v-else>
          <!-- 状态展示 -->
          <n-descriptions bordered :column="2" label-placement="left" size="small">
            <n-descriptions-item label="状态">
              <n-tag :type="running ? 'success' : installed ? 'warning' : 'default'" size="small">
                {{ running ? '运行中' : installed ? '已停止' : '未安装' }}
              </n-tag>
              <span v-if="state" style="margin-left:8px;color:#888;font-size:12px">{{ state }}</span>
            </n-descriptions-item>
            <n-descriptions-item label="容器">{{ status?.status?.container }}</n-descriptions-item>
            <n-descriptions-item label="镜像">{{ status?.status?.image }}</n-descriptions-item>
            <n-descriptions-item label="游戏端口">{{ status?.status?.game_port }}/udp</n-descriptions-item>
            <n-descriptions-item label="数据目录">{{ status?.status?.data_dir }}</n-descriptions-item>
          </n-descriptions>

          <!-- 未安装：部署表单 -->
          <n-card v-if="!installed" title="一键部署" size="small" embedded>
            <n-form label-placement="left" label-width="120">
              <n-form-item label="管理员密码">
                <n-input v-model:value="installForm.admin_password" type="password"
                  show-password-on="click" placeholder="游戏服 AdminPassword（REST/RCON 共用）" />
              </n-form-item>
              <n-form-item label="服务器名称">
                <n-input v-model:value="installForm.server_name" placeholder="My PalWorld Server" />
              </n-form-item>
              <n-form-item label="游戏端口">
                <n-input v-model:value="installForm.game_port" placeholder="8211" />
              </n-form-item>
              <n-button type="primary" :loading="acting" @click="doInstall">
                部署并启动
              </n-button>
            </n-form>
          </n-card>

          <!-- 已安装：控制按钮 -->
          <n-space v-else>
            <n-button type="success" :disabled="running" :loading="acting" @click="doStart">启动</n-button>
            <n-button type="warning" :disabled="!running" :loading="acting" @click="doStop">停止</n-button>
            <n-button :loading="acting" @click="doRestart">重启</n-button>
            <n-button type="error" :loading="acting" @click="doUpdate">更新</n-button>
          </n-space>
        </template>
      </n-space>
    </n-card>

    <n-card title="最近日志" size="small">
      <n-button size="tiny" @click="loadLogs" style="margin-bottom:8px">刷新</n-button>
      <pre class="logs">{{ logs || '暂无日志' }}</pre>
    </n-card>
  </n-space>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { NSpace, NCard, NAlert, NDescriptions, NDescriptionsItem, NTag,
  NForm, NFormItem, NInput, NButton, useMessage } from 'naive-ui'
import { gameApi, type GameServerStatus, type InstallConfig } from '@/api/gameserver'

const message = useMessage()
const status = ref<GameServerStatus | null>(null)
const logs = ref('')
const acting = ref(false)

const installForm = reactive<InstallConfig>({
  admin_password: '',
  server_name: 'My PalWorld Server',
  game_port: '8211',
})

const dockerAvailable = computed(() => status.value?.available !== false)
const installed = computed(() => !!status.value?.status?.installed)
const running = computed(() => !!status.value?.status?.running)
const state = computed(() => status.value?.status?.state)

async function loadStatus() {
  status.value = await gameApi.status()
}
async function loadLogs() {
  try {
    const r = await gameApi.logs()
    logs.value = r.logs || ''
  } catch { /* 忽略 */ }
}

async function doInstall() {
  if (!installForm.admin_password) {
    message.warning('请填写管理员密码')
    return
  }
  acting.value = true
  try {
    const r = await gameApi.install(installForm)
    message.success(r.message || '部署已开始，首次启动需下载服务端，请等待几分钟')
    setTimeout(loadStatus, 5000)
  } catch (e: any) {
    message.error(e.message)
  } finally {
    acting.value = false
  }
}

async function doAction(fn: () => Promise<any>, ok: string) {
  acting.value = true
  try {
    await fn()
    message.success(ok)
    await loadStatus()
  } catch (e: any) {
    message.error(e.message)
  } finally {
    acting.value = false
  }
}
const doStart = () => doAction(gameApi.start, '已启动')
const doStop = () => doAction(gameApi.stop, '已停止')
const doRestart = () => doAction(gameApi.restart, '已重启')
const doUpdate = () => doAction(() => gameApi.update({ admin_password: installForm.admin_password }), '已更新并重启')

onMounted(async () => {
  await loadStatus()
  if (installed.value) loadLogs()
})
</script>

<style scoped>
.logs {
  background: #1e1e1e;
  color: #d4d4d4;
  padding: 12px;
  border-radius: 6px;
  font-size: 12px;
  max-height: 400px;
  overflow: auto;
  white-space: pre-wrap;
}
</style>
