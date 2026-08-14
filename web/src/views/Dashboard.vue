<template>
  <n-space vertical :size="16">
    <n-grid cols="1 s:2 m:4" responsive="screen" :x-gap="12" :y-gap="12">
      <n-gi>
        <n-card title="服务器名称" size="small"><n-statistic :value="server.name || '-'"/></n-card>
      </n-gi>
      <n-gi>
        <n-card title="版本" size="small"><n-statistic :value="server.version || '-'"/></n-card>
      </n-gi>
      <n-gi>
        <n-card title="在线人数" size="small">
          <n-statistic :value="metrics.current_player_num || 0">
            <template #suffix>/ {{ metrics.max_player_num || 0 }}</template>
          </n-statistic>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card title="服务器 FPS" size="small">
          <n-statistic :value="metrics.server_fps || 0"/>
        </n-card>
      </n-gi>
    </n-grid>

    <n-card title="快捷操作" size="small">
      <n-space>
        <n-button @click="openBroadcast">全服广播</n-button>
        <n-button @click="doSync" :loading="syncing">立即同步</n-button>
        <n-popconfirm @positive-click="doShutdown" positive-text="确定" negative-text="取消">
          <template #trigger>
            <n-button type="error" ghost>关闭服务器</n-button>
          </template>
          将发送 60 秒倒计时关服，确定？
        </n-popconfirm>
      </n-space>
    </n-card>

    <n-card title="广播" size="small">
      <n-space>
        <n-input v-model:value="broadcastMsg" placeholder="输入广播消息..." @keyup.enter="sendBroadcast" />
        <n-button type="primary" @click="sendBroadcast" :loading="sending">发送</n-button>
      </n-space>
    </n-card>

    <n-grid cols="1 s:2" responsive="screen" :x-gap="12" :y-gap="12">
      <n-gi>
        <n-card title="运行时长" size="small">
          <n-statistic :value="Math.round((metrics.uptime || 0) / 3600)">
            <template #suffix>小时</template>
          </n-statistic>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card title="服务器帧时间" size="small">
          <n-statistic :value="metrics.server_frame_time || 0">
            <template #suffix>ms</template>
          </n-statistic>
        </n-card>
      </n-gi>
    </n-grid>

    <n-modal v-model:show="showBroadcast" preset="card" title="全服广播" style="max-width: 500px">
      <n-input v-model:value="broadcastMsg" type="textarea" placeholder="输入广播消息..." />
      <template #footer>
        <n-button type="primary" @click="sendBroadcast" :loading="sending">发送</n-button>
      </template>
    </n-modal>
  </n-space>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import {
  NSpace, NCard, NGrid, NGi, NStatistic, NButton, NInput, NModal,
  NPopconfirm, useMessage,
} from 'naive-ui'
import { api } from '@/api'

const message = useMessage()
const server = ref<any>({})
const metrics = ref<any>({})
const broadcastMsg = ref('')
const showBroadcast = ref(false)
const sending = ref(false)
const syncing = ref(false)

async function loadAll() {
  try { server.value = await api.getServer() } catch {}
  try { metrics.value = await api.getMetrics() } catch {}
}

function openBroadcast() { showBroadcast.value = true }

async function sendBroadcast() {
  if (!broadcastMsg.value) return
  sending.value = true
  try {
    await api.broadcast(broadcastMsg.value)
    message.success('广播已发送')
    broadcastMsg.value = ''
    showBroadcast.value = false
  } catch (e: any) {
    message.error(e.message)
  } finally {
    sending.value = false
  }
}

async function doSync() {
  syncing.value = true
  try {
    await api.sync()
    message.success('同步已触发')
  } catch (e: any) {
    message.error(e.message)
  } finally {
    setTimeout(() => { syncing.value = false; loadAll() }, 2000)
  }
}

async function doShutdown() {
  try {
    await api.shutdown(60, '服务器将在 60 秒后关闭')
    message.success('关服指令已发送')
  } catch (e: any) {
    message.error(e.message)
  }
}

onMounted(loadAll)
</script>
