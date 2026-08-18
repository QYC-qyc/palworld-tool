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

    <n-card title="运行模式" size="small">
      <n-space align="center" :size="12" :wrap="false">
        <n-text strong>游戏服版本：</n-text>
        <n-radio-group v-model:value="runMode" @update:value="onModeChange">
          <n-radio-button value="linux">原生 Linux</n-radio-button>
          <n-radio-button value="windows">Windows（Wine/PalDefender）</n-radio-button>
        </n-radio-group>
        <n-tooltip trigger="hover" placement="top" :show-arrow="false" style="max-width:420px">
          <template #trigger>
            <n-icon :component="HelpCircleOutline" style="font-size:18px;color:var(--n-text-color-3,#999);cursor:help" />
          </template>
          <div style="line-height:1.7">
            <div><b>原生 Linux 版：</b>直接运行 PalServer.sh，<b>性能和稳定性最好、内存占用低</b>，无需 Wine；
              但无法使用 PalDefender（Windows DLL 反作弊）。</div>
            <div style="margin-top:6px"><b>Windows 版（Wine）：</b>通过 Wine 运行 Windows 服务端以加载
              PalDefender 反作弊（实时拦截作弊、封禁/私聊/删据点等）；<b>内存占用更高、性能略低</b>，
              需先安装 Wine、Windows 版游戏服和 PalDefender DLL。</div>
            <div style="margin-top:6px">切换版本后需在「游戏服」页安装对应版本，并重启游戏服。</div>
          </div>
        </n-tooltip>
      </n-space>
    </n-card>

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
  NPopconfirm, NRadioGroup, NRadioButton, NTooltip, NIcon, useMessage, useDialog,
} from 'naive-ui'
import { HelpCircleOutline } from '@vicons/ionicons5'
import { api } from '@/api'

const message = useMessage()
const dialog = useDialog()
const server = ref<any>({})
const metrics = ref<any>({})
const broadcastMsg = ref('')
const showBroadcast = ref(false)
const sending = ref(false)
const syncing = ref(false)
const runMode = ref<'linux' | 'windows'>('linux')

async function loadMode() {
  try {
    const s = await api.getSettings()
    runMode.value = s['paldefender.wine_mode'] === 'true' ? 'windows' : 'linux'
  } catch {}
}

function onModeChange(v: 'linux' | 'windows') {
  const wantWindows = v === 'windows'
  const previous = runMode.value
  dialog.warning({
    title: wantWindows ? '切换到 Windows（Wine）模式？' : '切换到原生 Linux 模式？',
    content: wantWindows
      ? '将用 Wine 启动 Windows 版服务端以加载 PalDefender 反作弊。请确保已安装 Wine、Windows 版游戏服和 PalDefender DLL，且已在「游戏配置」中检查对应版本配置，然后重启游戏服。'
      : '将用原生 PalServer.sh 启动（性能更好，但无法使用 PalDefender）。请在「游戏服」页确认已安装 Linux 版，并重启游戏服。',
    positiveText: '确认切换',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await api.saveSettings({ 'paldefender.wine_mode': wantWindows ? 'true' : 'false' })
        runMode.value = v
        message.success('已切换，请安装对应版本并重启游戏服')
      } catch (e: any) {
        runMode.value = previous
        message.error(e.message)
      }
    },
    onNegativeClick: () => {
      runMode.value = previous
    },
    onMaskClick: () => {
      runMode.value = previous
    },
  })
}

async function loadAll() {
  try { server.value = await api.getServer() } catch {}
  try { metrics.value = await api.getMetrics() } catch {}
}

function openBroadcast() { showBroadcast.value = true }

// 优先走 PalDefender 广播；若 PD 未配置/不可用，回退官方 REST broadcast
async function sendBroadcastMsg(msg: string) {
  try {
    await api.pdBroadcast(msg)
    return 'paldefender'
  } catch (e: any) {
    const msgText: string = e?.message || ''
    if (msgText.includes('未配置') || msgText.includes('PalDefender') || msgText.includes('paldefender')) {
      await api.broadcast(msg)
      return 'official'
    }
    throw e
  }
}

async function sendBroadcast() {
  if (!broadcastMsg.value) return
  sending.value = true
  try {
    await sendBroadcastMsg(broadcastMsg.value)
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

onMounted(() => { loadAll(); loadMode() })
</script>
