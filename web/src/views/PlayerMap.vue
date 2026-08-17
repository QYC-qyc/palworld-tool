<template>
  <div class="playermap-wrap">
    <div class="map-header">
      <n-space align="center" justify="space-between">
        <n-space align="center">
          <span class="map-title">玩家地图</span>
          <n-tag size="small" type="info" :bordered="false">{{ currentMapLabel }}</n-tag>
          <n-tag size="small" type="success" :bordered="false">在线 {{ onlineCount }} 人</n-tag>
          <n-tag size="small" :bordered="false">可见玩家 {{ visiblePlayerCount }} 人</n-tag>
          <n-tag size="small" :bordered="false">据点 {{ baseCount }} 个</n-tag>
        </n-space>
        <n-button size="small" :loading="loading" @click="loadAll">刷新</n-button>
      </n-space>
    </div>

    <div class="map-stage">
      <div ref="mapEl" class="map-container"></div>

      <!-- 右下角控制面板 -->
      <div class="control-panel" :class="{ 'control-panel--collapsed': collapsed }">
        <div class="control-panel__head" @click="collapsed = !collapsed">
          <span>地图控制</span>
          <n-icon size="16">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline v-if="collapsed" points="6 9 12 15 18 9" />
              <polyline v-else points="18 15 12 9 6 15" />
            </svg>
          </n-icon>
        </div>

        <div v-show="!collapsed" class="control-panel__body">
          <div class="control-row">
            <n-radio-group v-model:value="mapMode" size="small" name="mapmode">
              <n-radio-button value="palpagos">主世界</n-radio-button>
              <n-radio-button value="feybreak">天坠之地</n-radio-button>
              <n-radio-button value="auto">自动</n-radio-button>
            </n-radio-group>
          </div>

          <div class="control-row">
            <n-select
              v-model:value="searchValue"
              filterable
              clearable
              size="small"
              placeholder="搜索玩家 / 据点"
              :options="searchOptions"
              @update:value="onSearchSelect"
            />
          </div>

          <div class="control-row">
            <n-radio-group v-model:value="visibility" size="small" name="vis">
              <n-radio-button value="online">在线玩家</n-radio-button>
              <n-radio-button value="all">全部玩家</n-radio-button>
            </n-radio-group>
          </div>

          <div class="control-switches">
            <div class="switch-row">
              <span>显示玩家</span>
              <n-switch v-model:value="showPlayer" size="small" />
            </div>
            <div class="switch-row">
              <span>显示据点</span>
              <n-switch v-model:value="showBase" size="small" />
            </div>
            <div class="switch-row">
              <span>显示 Boss 塔</span>
              <n-switch v-model:value="showBoss" size="small" />
            </div>
            <div class="switch-row">
              <span>显示传送点</span>
              <n-switch v-model:value="showFastTravel" size="small" />
            </div>
          </div>

          <div class="control-coords">
            <span class="coords-label">游戏坐标</span>
            <span class="coords-value">{{ mouseCoords }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch, nextTick } from 'vue'
import {
  NSpace, NTag, NButton, NSwitch, NSelect, NRadioGroup, NRadioButton, NIcon,
} from 'naive-ui'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import { api } from '@/api'
import {
  MAP_CONFIGS,
  toMapPosition,
  fromMapPosition,
  toMapDistance,
  hasMapLocation,
  mergeMapPlayers,
  selectVisibleMapPlayers,
  buildPlayerGuildMap,
  detectMap,
  pointInMap,
  type MapKey,
} from '@/utils/mapCoords'
import { clusterBases, type BaseMarker } from '@/utils/mapCluster'

// 两张图共用同一 Leaflet Simple CRS 归一化空间
const MAP_BOUNDS: L.LatLngBoundsExpression = [
  [0, 0],
  [-256, 256],
]

// ---------- 响应式状态 ----------
const mapEl = ref<HTMLElement>()
const loading = ref(false)
const collapsed = ref(false)

const players = ref<any[]>([])
const guilds = ref<any[]>([])
const onlineUids = ref<Set<string>>(new Set())
const onlinePlayers = ref<any[]>([])

// 当前地图与切换模式。默认进入页面显示主世界。
const currentMap = ref<MapKey>('palpagos')
const mapMode = ref<'palpagos' | 'feybreak' | 'auto'>('palpagos')

const showPlayer = ref(true)
const showBase = ref(true)
const showBoss = ref(false)
const showFastTravel = ref(false)
const visibility = ref<'online' | 'all'>('online')

const searchValue = ref<string | null>(null)
const mouseCoords = ref('-')

// ---------- Leaflet 实例 ----------
let map: L.Map | null = null
let tileLayer: L.TileLayer | null = null
let playerLayer: L.LayerGroup | null = null
let baseLayer: L.LayerGroup | null = null
let bossLayer: L.LayerGroup | null = null
let fastTravelLayer: L.LayerGroup | null = null
let timer: number | null = null

// POI 按地图缓存，避免重复 fetch
const pointsCache: Record<MapKey, any> = {
  palpagos: null,
  feybreak: null,
}
const pointsLoading = new Set<MapKey>()

// ---------- 图标 ----------
const playerIcon = L.icon({
  iconUrl: '/map/icons/player.webp',
  iconSize: [45, 45],
  iconAnchor: [22, 40],
  popupAnchor: [0, -35],
})
const baseIcon = L.icon({
  iconUrl: '/map/icons/base.webp',
  iconSize: [55, 55],
  iconAnchor: [27, 50],
  className: 'base-marker',
})
const bossIcon = L.icon({
  iconUrl: '/map/icons/boss_tower.webp',
  iconSize: [48, 48],
  iconAnchor: [24, 24],
})
const fastTravelIcon = L.icon({
  iconUrl: '/map/icons/fast_travel.webp',
  iconSize: [48, 48],
  iconAnchor: [24, 24],
})

// ---------- 统计 ----------
const onlineCount = computed(() => onlineUids.value.size)
const currentMapLabel = computed(() => MAP_CONFIGS[currentMap.value].label)
const visiblePlayerCount = computed(() => {
  const m = currentMap.value
  return selectVisibleMapPlayers(players.value, onlineUids.value, visibility.value).filter(
    (p) => pointInMap(Number(p.location_x), Number(p.location_y), m),
  ).length
})
const rawBaseMarkers = computed<BaseMarker[]>(() => {
  const m = currentMap.value
  const out: BaseMarker[] = []
  guilds.value.forEach((g) => {
    const camps: any[] = Array.isArray(g?.base_camp) ? g.base_camp : []
    camps.forEach((c) => {
      const x = Number(c.location_x)
      const y = Number(c.location_y)
      if (!Number.isFinite(x) || !Number.isFinite(y) || (x === 0 && y === 0)) return
      // 只保留落在当前地图边界内的据点
      if (!pointInMap(x, y, m)) return
      out.push({
        key: `${g.admin_player_uid}:${c.id}`,
        position: toMapPosition([x, y], m),
        camp: c,
        guildName: g.name,
        level: g.base_camp_level,
        members: (g.players || []).map((p: any) => p.nickname).filter(Boolean),
      })
    })
  })
  return out
})
const baseCount = computed(() => rawBaseMarkers.value.length)

// ---------- 搜索选项（仅当前地图内的玩家/据点） ----------
const searchOptions = computed(() => {
  const m = currentMap.value
  const opts: { label: string; value: string }[] = []
  players.value
    .filter(
      (p) =>
        hasMapLocation(p) &&
        pointInMap(Number(p.location_x), Number(p.location_y), m),
    )
    .forEach((p) => {
      opts.push({
        label: `玩家: ${p.nickname || p.player_uid}`,
        value: `player:${p.player_uid}`,
      })
    })
  rawBaseMarkers.value.forEach((bm) => {
    opts.push({
      label: `据点: ${bm.guildName || '未知公会'}-${bm.camp?.id ?? ''}`,
      value: `base:${bm.key}`,
    })
  })
  return opts
})

function onSearchSelect(value: string | null) {
  if (!value || !map) return
  const [type, id] = value.split(':')
  const m = currentMap.value
  let pos: [number, number] | undefined
  if (type === 'player') {
    const p = players.value.find((x) => x.player_uid === id)
    if (p) pos = toMapPosition([Number(p.location_x), Number(p.location_y)], m)
  } else if (type === 'base') {
    pos = rawBaseMarkers.value.find((bm) => bm.key === id)?.position
  }
  if (pos) map.setView(pos, 5)
}

// ---------- 绘制玩家 ----------
function drawPlayers() {
  if (!map || !playerLayer) return
  playerLayer.clearLayers()
  if (!showPlayer.value) return

  const m = currentMap.value
  const guildMap = buildPlayerGuildMap(guilds.value)
  const list = selectVisibleMapPlayers(players.value, onlineUids.value, visibility.value)
  list.forEach((p) => {
    const x = Number(p.location_x)
    const y = Number(p.location_y)
    // 只渲染落在当前地图内的玩家，避免另一张图的坐标映射到错误位置
    if (!pointInMap(x, y, m)) return
    const online = onlineUids.value.has(p.player_uid)
    const pos = toMapPosition([x, y], m)
    const icon = L.icon({
      ...playerIcon.options,
      className: online ? 'player-marker' : 'player-marker player-offline',
    })
    const marker = L.marker(pos, { icon })
    marker.bindTooltip(p.nickname || p.player_uid, {
      direction: 'top',
      offset: [0, -35],
      permanent: true,
      className: `player-tip${online ? ' player-tip--online' : ' player-tip--offline'}`,
    })
    const guild = guildMap.get(p.player_uid)
    const guildName = guild?.name || '无公会'
    const wx = Math.round(x)
    const wy = Math.round(y)
    const ping = p.ping != null ? Math.round(Number(p.ping)) : '-'
    marker.bindPopup(
      `<div class="map-popup">
        <div class="map-popup__title">${escapeHtml(p.nickname || p.player_uid)}</div>
        <div class="map-popup__row"><span>等级</span><b>Lv.${p.level ?? '-'}</b></div>
        <div class="map-popup__row"><span>公会</span><b>${escapeHtml(guildName)}</b></div>
        <div class="map-popup__row"><span>坐标</span><b>${wx}, ${wy}</b></div>
        <div class="map-popup__row"><span>Ping</span><b>${ping} ms</b></div>
        <div class="map-popup__status ${online ? 'is-online' : 'is-offline'}">
          ${online ? '在线' : '离线'}
        </div>
      </div>`,
    )
    marker.addTo(playerLayer!)
  })
}

// ---------- 绘制据点 ----------
function drawBases() {
  if (!map || !baseLayer) return
  baseLayer.clearLayers()
  if (!showBase.value) return

  const m = currentMap.value
  const zoom = map.getZoom()
  const clusters = clusterBases(rawBaseMarkers.value, zoom)

  clusters.forEach((c) => {
    if (c.markers.length > 1) {
      // 聚合标记
      const icon = L.divIcon({
        className: 'base-cluster',
        html: `<div class="base-cluster__badge">${c.markers.length}</div>`,
        iconSize: [62, 62],
        iconAnchor: [31, 31],
      })
      const marker = L.marker(c.position, { icon })
      const items = c.markers
        .map(
          (bm) =>
            `<div class="map-popup__row"><span>${escapeHtml(
              bm.guildName || '未知公会',
            )}</span><b>Lv.${bm.level ?? '-'} · #${bm.camp?.id ?? ''}</b></div>`,
        )
        .join('')
      marker.bindPopup(
        `<div class="map-popup"><div class="map-popup__title">${c.markers.length} 个据点</div>${items}</div>`,
      )
      marker.addTo(baseLayer!)
    } else {
      const bm = c.markers[0]
      const marker = L.marker(bm.position, { icon: baseIcon })
      const members = (bm.members || []).slice(0, 8).map(escapeHtml).join('、')
      marker.bindPopup(
        `<div class="map-popup">
          <div class="map-popup__title">${escapeHtml(bm.guildName || '未知公会')}</div>
          <div class="map-popup__row"><span>据点等级</span><b>Lv.${bm.level ?? '-'}</b></div>
          <div class="map-popup__row"><span>据点 ID</span><b>${bm.camp?.id ?? '-'}</b></div>
          <div class="map-popup__row"><span>范围</span><b>${Math.round(Number(bm.camp?.area) || 0)}</b></div>
          ${members ? `<div class="map-popup__members">成员：${members}</div>` : ''}
        </div>`,
      )
      marker.addTo(baseLayer!)

      // zoom >= 5 时画据点范围圆圈
      if (zoom >= 5 && bm.camp?.area) {
        L.circle(bm.position, {
          radius: toMapDistance(Number(bm.camp.area), m),
          color: '#18a058',
          fillColor: '#18a058',
          fillOpacity: 0.1,
          weight: 1,
        }).addTo(baseLayer!)
      }
    }
  })
}

// ---------- 静态点（Boss/传送），按地图缓存 ----------
function renderPoints(mapKey: MapKey) {
  // 仅渲染当前地图，避免切图竞态把旧图 POI 画进来
  if (mapKey !== currentMap.value) return
  const data = pointsCache[mapKey]
  if (!data) return
  bossLayer?.clearLayers()
  ;(data.boss_tower || []).forEach((pt: [number, number]) => {
    L.marker(toMapPosition(pt, mapKey), { icon: bossIcon }).addTo(bossLayer!)
  })
  fastTravelLayer?.clearLayers()
  ;(data.fast_travel || []).forEach((pt: [number, number]) => {
    L.marker(toMapPosition(pt, mapKey), { icon: fastTravelIcon }).addTo(fastTravelLayer!)
  })
}

async function loadPoints(mapKey: MapKey) {
  if (pointsCache[mapKey]) {
    renderPoints(mapKey)
    return
  }
  if (pointsLoading.has(mapKey)) return
  pointsLoading.add(mapKey)
  try {
    const resp = await fetch(MAP_CONFIGS[mapKey].pointsUrl)
    const data = await resp.json()
    pointsCache[mapKey] = data
    renderPoints(mapKey)
  } catch (e) {
    // 静态点加载失败不影响主功能
    console.warn(`加载地图静态点失败 (${mapKey})`, e)
  } finally {
    pointsLoading.delete(mapKey)
  }
}

// ---------- 瓦片层构建（切换地图时 remove + 重建） ----------
function buildTileLayer(mapKey: MapKey) {
  if (tileLayer) {
    map?.removeLayer(tileLayer)
    tileLayer = null
  }
  if (!map) return
  tileLayer = L.tileLayer(MAP_CONFIGS[mapKey].tilesUrl, {
    tileSize: 256,
    noWrap: true,
    bounds: MAP_BOUNDS,
  }).addTo(map)
}

/**
 * 切换到指定地图：重建瓦片、清空并重绘所有图层、保持当前缩放。
 */
async function switchMap(mapKey: MapKey) {
  if (!map) return
  const zoom = map.getZoom()
  const needRebuild = mapKey !== currentMap.value || !tileLayer

  currentMap.value = mapKey
  if (needRebuild) {
    buildTileLayer(mapKey)
  }

  playerLayer?.clearLayers()
  baseLayer?.clearLayers()
  bossLayer?.clearLayers()
  fastTravelLayer?.clearLayers()

  drawPlayers()
  drawBases()

  // POI 图层若处于显示状态，重建当前地图的标记
  if (showBoss.value || showFastTravel.value) {
    await loadPoints(mapKey)
  }

  map.setView([-128, 128], zoom)
  map.invalidateSize()
}

/**
 * 自动模式：根据在线玩家坐标判定多数所在地图。
 * 平票或无在线玩家时保持当前地图。
 */
function detectAutoMap() {
  const list = onlinePlayers.value.filter(hasMapLocation)
  if (list.length === 0) {
    drawPlayers()
    return
  }
  const counts: Record<MapKey, number> = { palpagos: 0, feybreak: 0 }
  list.forEach((p) => {
    counts[detectMap(Number(p.location_x), Number(p.location_y))]++
  })
  let winner: MapKey = currentMap.value
  if (counts.feybreak > counts.palpagos) winner = 'feybreak'
  else if (counts.palpagos > counts.feybreak) winner = 'palpagos'
  if (winner !== currentMap.value) {
    switchMap(winner)
  } else {
    drawPlayers()
  }
}

// ---------- 数据加载 ----------
async function loadAll() {
  loading.value = true
  try {
    const [ps, gs] = await Promise.all([api.getPlayers(), api.getGuilds()])
    players.value = Array.isArray(ps) ? ps : []
    guilds.value = Array.isArray(gs) ? gs : []
    await refreshOnline()
    drawBases()
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

async function refreshOnline() {
  try {
    const online = await api.getOnline()
    const list = Array.isArray(online) ? online : []
    onlineUids.value = new Set(
      list.map((p: any) => p.player_uid).filter(Boolean),
    )
    onlinePlayers.value = list
    players.value = mergeMapPlayers(players.value, list)
    if (mapMode.value === 'auto') {
      detectAutoMap()
    } else {
      drawPlayers()
    }
  } catch (e) {
    // 静默：在线接口偶发失败不打断
    console.warn('刷新在线玩家失败', e)
  }
}

// ---------- 工具 ----------
function escapeHtml(s: any): string {
  return String(s ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

// ---------- 开关联动 ----------
watch(showPlayer, (v) => {
  if (v) drawPlayers()
  else playerLayer?.clearLayers()
})
watch(showBase, (v) => {
  if (v) drawBases()
  else baseLayer?.clearLayers()
})
watch(showBoss, async (v) => {
  if (!map || !bossLayer) return
  if (v) {
    await loadPoints(currentMap.value)
    bossLayer.addTo(map)
  } else {
    map.removeLayer(bossLayer)
  }
})
watch(showFastTravel, async (v) => {
  if (!map || !fastTravelLayer) return
  if (v) {
    await loadPoints(currentMap.value)
    fastTravelLayer.addTo(map)
  } else {
    map.removeLayer(fastTravelLayer)
  }
})
watch(visibility, () => drawPlayers())

// 切换地图模式：手动选项直接切图；auto 则立即按在线玩家判定
watch(mapMode, (mode) => {
  if (mode === 'auto') {
    detectAutoMap()
  } else {
    switchMap(mode)
  }
})

// ---------- 生命周期 ----------
onMounted(async () => {
  await nextTick()
  if (!mapEl.value) return

  map = L.map(mapEl.value, {
    crs: L.CRS.Simple,
    center: [-128, 128],
    zoom: 2,
    minZoom: 0,
    maxZoom: 6,
    zoomControl: true,
    attributionControl: false,
    maxBounds: MAP_BOUNDS,
    maxBoundsViscosity: 1.0,
  })

  // 初始瓦片为主世界
  buildTileLayer(currentMap.value)

  playerLayer = L.layerGroup().addTo(map)
  baseLayer = L.layerGroup().addTo(map)
  bossLayer = L.layerGroup()
  fastTravelLayer = L.layerGroup()

  // zoom 变化时重绘据点聚合
  map.on('zoomend', () => drawBases())

  // 鼠标移动反算游戏坐标（按当前地图边界）
  map.on('mousemove', (e: L.LeafletMouseEvent) => {
    const [wx, wy] = fromMapPosition([e.latlng.lat, e.latlng.lng], currentMap.value)
    mouseCoords.value = `${Math.round(wx)}, ${Math.round(wy)}`
  })

  // 缩放控件放左下
  map.zoomControl.setPosition('bottomleft')

  await loadAll()
  timer = window.setInterval(refreshOnline, 5000)
})

onUnmounted(() => {
  if (timer != null) window.clearInterval(timer)
  if (map) {
    map.remove()
    map = null
  }
})
</script>

<style scoped>
.playermap-wrap {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.map-header {
  padding: 4px 2px;
}
.map-title {
  font-size: 15px;
  font-weight: 600;
  color: #e5e7eb;
}
.map-stage {
  position: relative;
}
.map-container {
  width: 100%;
  height: calc(100vh - 180px);
  min-height: 480px;
  background: #102536;
  border-radius: 8px;
  overflow: hidden;
}

/* 控制面板 */
.control-panel {
  position: absolute;
  right: 16px;
  bottom: 16px;
  width: 260px;
  background: rgba(24, 24, 28, 0.92);
  backdrop-filter: blur(14px);
  -webkit-backdrop-filter: blur(14px);
  border-radius: 12px;
  padding: 12px;
  z-index: 1000;
  color: #e5e7eb;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
  border: 1px solid rgba(255, 255, 255, 0.06);
}
.control-panel--collapsed .control-panel__body {
  display: none;
}
.control-panel__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  user-select: none;
  margin-bottom: 10px;
}
.control-panel__body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.control-row :deep(.n-base-selection) {
  background: rgba(255, 255, 255, 0.06);
}
.control-switches {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 13px;
  color: #d1d5db;
}
.control-coords {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 8px;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
  font-size: 12px;
}
.coords-label {
  color: #9ca3af;
}
.coords-value {
  color: #86efac;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

@media (max-width: 480px) {
  .control-panel {
    right: 12px;
    bottom: 12px;
    width: calc(100% - 24px);
    max-width: 280px;
  }
}
</style>

<style>
/* Leaflet 暗色控件样式（沿用现有 PlayerMap 暗色风格） */
.leaflet-container {
  background: #102536 !important;
}
.leaflet-bar a {
  background: #1f2937 !important;
  color: #ccc !important;
  border-color: #374151 !important;
}
.leaflet-bar a:hover {
  background: #374151 !important;
}

/* Popup 白色卡片风格（参考分析文档 2.12） */
.leaflet-popup-content-wrapper {
  background: #ffffff !important;
  color: #1f2937 !important;
  border-radius: 12px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.45);
  padding: 4px;
}
.leaflet-popup-content {
  margin: 10px 12px;
  font-size: 13px;
  line-height: 1.5;
}
.leaflet-popup-tip {
  background: #ffffff !important;
}
.map-popup__title {
  font-weight: 600;
  font-size: 14px;
  margin-bottom: 6px;
  color: #111827;
}
.map-popup__row {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  color: #4b5563;
}
.map-popup__row b {
  color: #111827;
  font-weight: 600;
}
.map-popup__members {
  margin-top: 6px;
  padding-top: 6px;
  border-top: 1px solid #eee;
  color: #6b7280;
  font-size: 12px;
}
.map-popup__status {
  display: inline-block;
  margin-top: 8px;
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 11px;
}
.map-popup__status.is-online {
  background: rgba(24, 160, 88, 0.12);
  color: #18a058;
}
.map-popup__status.is-offline {
  background: rgba(107, 114, 128, 0.15);
  color: #6b7280;
}

/* 玩家标记 */
.player-marker {
  background: transparent !important;
  border: none !important;
}
.player-marker.player-offline {
  filter: grayscale(0.9) saturate(0.35);
  opacity: 0.6;
}

/* 玩家昵称 tooltip */
.player-tip {
  background: rgba(17, 24, 39, 0.85) !important;
  border: 1px solid rgba(255, 255, 255, 0.12) !important;
  color: #d1d5db !important;
  font-size: 11px !important;
  padding: 2px 6px !important;
  border-radius: 6px !important;
  box-shadow: none !important;
}
.player-tip::before {
  display: none !important;
}
.player-tip--online {
  color: #86efac !important;
  border-color: rgba(24, 160, 88, 0.4) !important;
}
.player-tip--offline {
  color: #9ca3af !important;
}

/* 据点 */
.base-marker {
  background: transparent !important;
  border: none !important;
}
.base-cluster {
  background: transparent !important;
  border: none !important;
}
.base-cluster__badge {
  width: 62px;
  height: 62px;
  border-radius: 50%;
  background: #18a058;
  color: #fff;
  font-size: 18px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 4px 12px rgba(24, 160, 88, 0.5);
  border: 3px solid rgba(255, 255, 255, 0.9);
}
</style>
