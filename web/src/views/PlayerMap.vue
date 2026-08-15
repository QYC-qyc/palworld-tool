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

const TILE_SIZE = 256
const MAX_ZOOM = 4
// zoom=0 时瓦片数为 1x1，覆盖范围即游戏世界
// 游戏坐标范围约 -1,400,000 ~ +1,400,000
const WORLD_HALF = 1400000
const PX_PER_UNIT = TILE_SIZE / (WORLD_HALF * 2) // zoom 0 时 1 游戏单位 = 多少像素

let map: L.Map | null = null
let markersLayer: L.LayerGroup | null = null
let timer: any

function gameToLatLng(x: number, y: number): [number, number] {
  // Simple CRS: y 向下为正
  // 游戏坐标 (x, y) -> 地图像素 (px, py)
  const px = (x + WORLD_HALF) * PX_PER_UNIT
  const py = (-y + WORLD_HALF) * PX_PER_UNIT
  return [py, px]
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
    const [lat, lng] = gameToLatLng(Number(p.location_x), Number(p.location_y))
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
  const crs = L.Util.extend({}, L.CRS.Simple)
  map = L.map(mapEl.value, {
    crs,
    minZoom: 0,
    maxZoom: MAX_ZOOM,
    zoomSnap: 0.5,
    zoomDelta: 0.5,
    center: [TILE_SIZE / 2, TILE_SIZE / 2],
    zoom: 1,
    attributionControl: false,
  })

  // 主世界瓦片
  L.tileLayer('/map/tiles/{z}/{x}/{y}.webp', {
    tileSize: TILE_SIZE,
    minZoom: 0,
    maxZoom: MAX_ZOOM,
    noWrap: true,
    bounds: [[0, 0], [TILE_SIZE, TILE_SIZE]],
  }).addTo(map)

  // treemap 小地图叠加（左上角，约85x86像素）
  const miniBounds: L.LatLngBoundsExpression = [
    [0, 0],
    [86, 85],
  ]
  L.imageOverlay('/map/treemap.webp', miniBounds, { opacity: 0.9 }).addTo(map)

  map.setMaxBounds([[-256, -256], [TILE_SIZE + 256, TILE_SIZE + 256]])

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
  background: #0d1117;
  border-radius: 4px;
  overflow: hidden;
}
</style>

<style>
.leaflet-container {
  background: #0d1117 !important;
}
.leaflet-bar a {
  background: #1f2937 !important;
  color: #ccc !important;
  border-color: #374151 !important;
}
.leaflet-bar a:hover {
  background: #374151 !important;
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
