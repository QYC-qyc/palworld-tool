<template>
  <n-space vertical>
    <n-card title="在线玩家位置" size="small">
      <template #header-extra>
        <n-space align="center">
          <n-tag :bordered="false" size="small" type="info">
            在线: {{ online.length }} 人
          </n-tag>
          <n-button size="small" @click="loadOnline" :loading="loading">刷新</n-button>
        </n-space>
      </template>
      <div class="map-container" ref="mapRef">
        <div class="map-grid">
          <!-- 坐标刻度 -->
          <div class="axis axis-top">
            <span v-for="g in gridLines" :key="'tx'+g" :style="{ left: g.pos + '%' }">{{ g.label }}</span>
          </div>
          <div class="axis axis-left">
            <span v-for="g in gridLines" :key="'ty'+g" :style="{ top: g.pos + '%' }">{{ -g.label }}</span>
          </div>
          <!-- 网格线 -->
          <div v-for="i in 8" :key="'v'+i" class="grid-line-v" :style="{ left: (i * 100 / 8) + '%' }" />
          <div v-for="i in 8" :key="'h'+i" class="grid-line-h" :style="{ top: (i * 100 / 8) + '%' }" />
          <!-- 原点 -->
          <div class="origin"></div>
          <!-- 玩家标记 -->
          <div
            v-for="(p, idx) in online"
            :key="p.player_uid || idx"
            class="player-marker"
            :style="markerStyle(p)"
          >
            <n-tooltip placement="top">
              <template #trigger>
                <div class="marker-dot" :style="{ background: colors[idx % colors.length] }"></div>
              </template>
              <div>
                <strong>{{ p.nickname }}</strong><br />
                等级: {{ p.level }}<br />
                坐标: {{ Math.round(p.location_x) }}, {{ Math.round(p.location_y) }}<br />
                Ping: {{ Math.round(p.ping) }}ms
              </div>
            </n-tooltip>
          </div>
        </div>
      </div>
      <n-text depth="3" style="font-size:12px;display:block;margin-top:8px">
        坐标范围约 -1,400,000 至 1,400,000。每 10 秒自动刷新。
      </n-text>
    </n-card>
  </n-space>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { NSpace, NCard, NButton, NTag, NTooltip, NText } from 'naive-ui'
import { api } from '@/api'

const online = ref<any[]>([])
const loading = ref(false)

const WORLD_HALF = 1400000

const colors = ['#ff4444', '#44ff44', '#4488ff', '#ffaa00', '#ff44ff', '#44ffff', '#ffff44', '#ff8844']

async function loadOnline() {
  loading.value = true
  try {
    online.value = await api.getOnline()
  } catch {
    // 忽略
  } finally {
    loading.value = false
  }
}

function markerStyle(p: any) {
  const x = ((Number(p.location_x) + WORLD_HALF) / (WORLD_HALF * 2)) * 100
  const y = ((Number(p.location_y) + WORLD_HALF) / (WORLD_HALF * 2)) * 100
  return { left: `${x}%`, top: `${y}%` }
}

const gridLines = [
  { pos: 0, label: '-1400k' },
  { pos: 25, label: '-700k' },
  { pos: 50, label: '0' },
  { pos: 75, label: '700k' },
  { pos: 100, label: '1400k' },
]

let timer: any
onMounted(async () => {
  await loadOnline()
  timer = setInterval(loadOnline, 10000)
})
onUnmounted(() => clearInterval(timer))
</script>

<style scoped>
.map-container {
  position: relative;
  width: 100%;
  padding-top: 75%;
  background: #1a2332;
  border-radius: 6px;
  overflow: hidden;
}
.map-grid {
  position: absolute;
  inset: 30px;
  top: 25px;
  left: 60px;
  right: 20px;
  bottom: 30px;
  border: 1px solid #2a4a6a;
}
.grid-line-v {
  position: absolute;
  top: 0;
  bottom: 0;
  width: 1px;
  background: rgba(255,255,255,0.08);
}
.grid-line-h {
  position: absolute;
  left: 0;
  right: 0;
  height: 1px;
  background: rgba(255,255,255,0.08);
}
.origin {
  position: absolute;
  left: 50%;
  top: 50%;
  width: 6px;
  height: 6px;
  background: #ff6600;
  border-radius: 50%;
  transform: translate(-50%, -50%);
}
.axis {
  position: absolute;
  color: #6a8aaa;
  font-size: 10px;
}
.axis-top {
  top: 5px;
  left: 60px;
  right: 20px;
  height: 16px;
}
.axis-top span {
  position: absolute;
  transform: translateX(-50%);
}
.axis-left {
  left: 5px;
  top: 25px;
  bottom: 30px;
  width: 50px;
}
.axis-left span {
  position: absolute;
  transform: translateY(-50%);
  text-align: right;
  width: 45px;
}
.player-marker {
  position: absolute;
  transform: translate(-50%, -50%);
  z-index: 10;
}
.marker-dot {
  width: 14px;
  height: 14px;
  border: 2px solid #fff;
  border-radius: 50%;
  cursor: pointer;
  box-shadow: 0 0 10px rgba(255,255,255,0.3);
  animation: pulse 2s infinite;
}
@keyframes pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.7; transform: scale(1.3); }
}
</style>
