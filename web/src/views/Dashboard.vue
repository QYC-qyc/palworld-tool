<template>
  <n-space vertical :size="16">
    <PageHeader title="仪表盘" subtitle="服务器运行状态与控制台" />

    <n-grid cols="1 s:2 m:4" responsive="screen" :x-gap="14" :y-gap="14">
      <n-gi v-for="card in statCards" :key="card.label">
        <div class="stat-card" :style="{ '--accent': card.color }">
          <div class="stat-card__icon">
            <n-icon :component="card.icon" size="22" />
          </div>
          <div class="stat-card__body">
            <div class="stat-card__label">{{ card.label }}</div>
            <div class="stat-card__value">
              {{ card.value }}
              <span v-if="card.suffix" class="stat-card__suffix">{{ card.suffix }}</span>
            </div>
          </div>
        </div>
      </n-gi>
    </n-grid>

    <n-card title="控制台" size="small">
      <n-grid cols="1 m:2" responsive="screen" :x-gap="20" :y-gap="16" item-responsive>
        <n-gi>
          <n-space vertical :size="12">
            <n-text depth="3" style="font-size:12px">快捷操作</n-text>
            <n-space>
              <n-button @click="openBroadcast">
                <template #icon><n-icon :component="MegaphoneOutline" /></template>
                全服广播
              </n-button>
              <n-button @click="doSync" :loading="syncing">
                <template #icon><n-icon :component="RefreshOutline" /></template>
                立即同步
              </n-button>
              <n-popconfirm @positive-click="doShutdown" positive-text="确定关服" negative-text="取消">
                <template #trigger>
                  <n-button type="error" ghost>
                    <template #icon><n-icon :component="PowerOutline" /></template>
                    关闭服务器
                  </n-button>
                </template>
                将发送 60 秒倒计时关服，确定？
              </n-popconfirm>
            </n-space>
          </n-space>
        </n-gi>
        <n-gi>
          <n-space vertical :size="12">
            <n-text depth="3" style="font-size:12px">发送广播</n-text>
            <n-input
              v-model:value="broadcastMsg"
              type="textarea"
              :autosize="{ minRows: 2, maxRows: 4 }"
              placeholder="输入广播消息，回车发送..."
              @keydown.enter.exact.prevent="sendBroadcast"
            />
            <n-button type="primary" block :loading="sending" @click="sendBroadcast">
              <template #icon><n-icon :component="SendOutline" /></template>
              发送广播
            </n-button>
          </n-space>
        </n-gi>
      </n-grid>
    </n-card>

    <n-grid cols="1 s:2" responsive="screen" :x-gap="14" :y-gap="14">
      <n-gi>
        <n-card size="small">
          <n-statistic label="运行时长">
            {{ Math.round((metrics.uptime || 0) / 3600) }}
            <template #suffix>小时</template>
          </n-statistic>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card size="small">
          <n-statistic label="服务器帧时间">
            {{ metrics.server_frame_time || 0 }}
            <template #suffix>ms</template>
          </n-statistic>
        </n-card>
      </n-gi>
    </n-grid>
  </n-space>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  NSpace, NCard, NGrid, NGi, NStatistic, NButton, NInput,
  NPopconfirm, NIcon,
  useMessage,
} from 'naive-ui'
import {
  ServerOutline, GlobeOutline, PeopleOutline, SpeedometerOutline,
  MegaphoneOutline, RefreshOutline, PowerOutline, SendOutline,
} from '@vicons/ionicons5'
import { api } from '@/api'
import PageHeader from '@/components/PageHeader.vue'

const message = useMessage()

const server = ref<any>({})
const metrics = ref<any>({})
const broadcastMsg = ref('')
const sending = ref(false)
const syncing = ref(false)

const statCards = computed(() => [
  { label: '服务器名称', value: server.value.name || '-', icon: ServerOutline, color: '#4f46e5' },
  { label: '版本', value: server.value.version || '-', icon: GlobeOutline, color: '#0ea5e9' },
  {
    label: '在线人数',
    value: metrics.value.current_player_num || 0,
    suffix: `/ ${metrics.value.max_player_num || 0}`,
    icon: PeopleOutline,
    color: '#10b981',
  },
  { label: '服务器 FPS', value: metrics.value.server_fps || 0, icon: SpeedometerOutline, color: '#f59e0b' },
])

async function loadAll() {
  // 并行请求，避免游戏未运行时串行等待超时导致页面卡顿
  const [s, m] = await Promise.allSettled([api.getServer(), api.getMetrics()])
  if (s.status === 'fulfilled') server.value = s.value
  if (m.status === 'fulfilled') metrics.value = m.value
}

// 优先走 PalDefender 广播；若 PD 未配置/不可用，回退官方 REST broadcast
async function sendBroadcastMsg(msg: string) {
  try {
    await api.pdBroadcast(msg)
  } catch (e: any) {
    const msgText: string = e?.message || ''
    if (msgText.includes('未配置') || msgText.includes('PalDefender') || msgText.includes('paldefender')) {
      await api.broadcast(msg)
    } else {
      throw e
    }
  }
}

async function sendBroadcast() {
  if (!broadcastMsg.value.trim()) return
  sending.value = true
  try {
    await sendBroadcastMsg(broadcastMsg.value)
    message.success('广播已发送')
    broadcastMsg.value = ''
  } catch (e: any) {
    message.error(e.message)
  } finally {
    sending.value = false
  }
}

function openBroadcast() {
  // 焦点聚到广播输入框（已内联在控制台卡片）
  const el = document.querySelector('textarea') as HTMLTextAreaElement | null
  el?.focus()
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

<style scoped>
.stat-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px 18px;
  border-radius: 12px;
  background: var(--n-color, #fff);
  border: 1px solid var(--n-border-color, #efeff5);
  transition: box-shadow 0.2s, transform 0.2s;
}
.stat-card:hover {
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.06);
  transform: translateY(-1px);
}
.stat-card__icon {
  flex-shrink: 0;
  width: 44px;
  height: 44px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  background: var(--accent);
}
.stat-card__label {
  font-size: 12px;
  color: var(--n-text-color-3, #999);
}
.stat-card__value {
  font-size: 22px;
  font-weight: 600;
  margin-top: 2px;
  line-height: 1.2;
}
.stat-card__suffix {
  font-size: 13px;
  font-weight: 400;
  color: var(--n-text-color-3, #999);
  margin-left: 2px;
}
</style>
