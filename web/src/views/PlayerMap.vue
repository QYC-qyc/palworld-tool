<template>
  <n-space vertical>
    <n-card title="在线玩家位置" size="small">
      <template #header-extra>
        <n-space align="center">
          <n-tag :bordered="false" size="small" type="info">在线: {{ online.length }} 人</n-tag>
          <n-button size="small" @click="loadOnline" :loading="loading">刷新</n-button>
        </n-space>
      </template>
      <div ref="mapEl" class="map-container"></div>
    </n-card>
  </n-space>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref, nextTick } from 'vue'
import { NSpace, NCard, NButton, NTag, useMessage } from 'naive-ui'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import { api } from '@/api'

const message = useMessage()
const mapEl = ref<HTMLElement>()
const online = ref<any[]>([])
const loading = ref(false)

const WORLD_HALF = 1400000 // 游戏坐标半范围
const IMG_SIZE = 4096      // 地图图片像素
let map: L.Map | null = null
let markersLayer: L.LayerGroup | null = null
let timer: any

// 游戏坐标 -> Leaflet 坐标（Simple CRS, y轴翻转）
function gameToMap(x: number, y: number): [number, number] {
  const px = ((x + WORLD_HALF) / (WORLD_HALF * 2)) * IMG_SIZE
  const py = ((-y + WORLD_HALF) / (WORLD_HALF * 2)) * IMG_SIZE
  return [py, px] // Leaflet uses [lat, lng] = [y, x] in pixel CRS
}

async function loadOnline() {
  loading.value = true
  try {
    online.value = await api.getOnline()
    updateMarkers()
  } catch (e: any) {
    message.error(e.message)
  } finally {
    loading.value = false
  }
}

function updateMarkers() {
  if (!markersLayer || !map) return
  markersLayer.clearLayers()

  const colors = ['#ff4444', '#44ff44', '#4488ff', '#ffaa00', '#ff44ff', '#44ffff']
  online.value.forEach((p, i) => {
    if (p.location_x === undefined || p.location_y === undefined) return
    const [lat, lng] = gameToMap(Number(p.location_x), Number(p.location_y))
    const color = colors[i % colors.length]
    const icon = L.divIcon({
      className: 'player-marker',
      html: `<div style="width:14px;height:14px;background:${color};border:2px solid #fff;border-radius:50%;box-shadow:0 0 8px ${color}"></div>`,
      iconSize: [14, 14],
      iconAnchor: [7, 7],
    })
    const marker = L.marker([lat, lng], { icon })
    marker.bindPopup(`
      <strong>${p.nickname}</strong><br/>
      等级: ${p.level}<br/>
      坐标: ${Math.round(p.location_x)}, ${Math.round(p.location_y)}<br/>
      Ping: ${Math.round(p.ping)}ms
    `)
    marker.addTo(markersLayer)
  })
}

onMounted(async () => {
  await nextTick()
  if (!mapEl.value) return

  // Simple CRS: 像素坐标系，y 向下为正
  map = L.map(mapEl.value, {
    crs: L.CRS.Simple,
    minZoom: -2,
    maxZoom: 3,
    zoomSnap: 0.5,
    center: [IMG_SIZE / 2, IMG_SIZE / 2],
    zoom: -1,
    attributionControl: false,
  })

  // 地图边界
  const bounds: L.LatLngBoundsExpression = [
    [0, 0],
    [IMG_SIZE, IMG_SIZE],
  ]
  L.imageOverlay('/map/world.webp', bounds).addTo(map)
  map.fitBounds(bounds)
  map.setMaxBounds(bounds)

  markersLayer = L.layerGroup().addTo(map)

  await loadOnline()
  timer = setInterval(loadOnline, 10000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
  if (map) map.remove()
})
</script>

<style scoped>
.map-container {
  width: 100%;
  height: 70vh;
  background: #1a2332;
  border-radius: 4px;
  overflow: hidden;
}
</style>

<style>
/* Leaflet 暗色主题 */
.leaflet-container {
  background: #1a2332 !important;
}
.leaflet-bar a {
  background: #2a3a4a !important;
  color: #ccc !important;
  border-color: #3a4a5a !important;
}
.leaflet-bar a:hover {
  background: #3a5a7a !important;
}
.leaflet-popup-content-wrapper {
  background: #1f2937 !important;
  color: #e5e7eb !important;
  border-radius: 6px;
}
.leaflet-popup-tip {
  background: #1f2937 !important;
}
.player-marker {
  background: transparent !important;
  border: none !important;
}
</style>
