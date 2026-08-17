// 地图坐标换算工具
// 参考 _palworld-server-tool MapView.vue / utils/mapPlayers.js
// 世界坐标使用虚幻引擎坐标，范围约 -1.4M ~ +1.4M
// 映射到 Leaflet Simple CRS 的 [0,256] x [-256,0] 坐标空间
//
// 双地图支持：主世界 Palpagos 与 1.0 新地图天坠之地 Feybreak。
// 两张图共用同一套游戏全局坐标系，仅世界边界不同。

/** 地图标识 */
export type MapKey = 'palpagos' | 'feybreak'

// 世界坐标边界：[maxX, maxY, minX, minY]
// 主世界
export const LAND_SCAPE = [349400, 724400, -1099400, -724400] as const
// 天坠之地（注意 maxY=-476400 比 minY=-818197 更靠近 0，SPAN_Y 仍为正数）
export const FEYBREAK_LAND_SCAPE = [689148.5, -476400, 347351.5, -818197] as const

/** 各地图配置 */
export const MAP_CONFIGS: Record<
  MapKey,
  {
    landScape: readonly [number, number, number, number] // [maxX, maxY, minX, minY]
    tilesUrl: string
    pointsUrl: string
    label: string
  }
> = {
  palpagos: {
    landScape: LAND_SCAPE,
    tilesUrl: '/map/tiles/{z}/{x}/{y}.webp',
    pointsUrl: '/data/map-points.json',
    label: '主世界',
  },
  feybreak: {
    landScape: FEYBREAK_LAND_SCAPE,
    tilesUrl: '/map/tiles-feybreak/{z}/{x}/{y}.webp',
    pointsUrl: '/data/map-points-feybreak.json',
    label: '世界树',
  },
}

// 主世界常量（保留向后兼容的模块级常量）
const [MAX_X, MAX_Y, MIN_X, MIN_Y] = LAND_SCAPE
const SPAN_X = MAX_X - MIN_X
const SPAN_Y = MAX_Y - MIN_Y

/** 取某地图的边界跨度 */
function getSpans(map: MapKey): {
  maxX: number
  maxY: number
  minX: number
  minY: number
  spanX: number
  spanY: number
} {
  const [mX, mY, nX, nY] = MAP_CONFIGS[map].landScape
  return {
    maxX: mX,
    maxY: mY,
    minX: nX,
    minY: nY,
    spanX: mX - nX,
    spanY: mY - nY,
  }
}

/**
 * 世界坐标 -> Leaflet 地图坐标 [x_map, y_map]
 * x_map = -256 + 256 * (wx - minX) / (maxX - minX)  -> [-256, 0]
 * y_map =  256 * (wy - minY) / (maxY - minY)        -> [0, 256]
 *
 * 注意：Leaflet 用 [lat, lng] = [x_map, y_map]，lat 为负值、lng 为正值，
 * 与瓦片 bounds [[0,0],[-256,256]] 一致。
 *
 * hack：若输入已经在 [-256, 256] 范围内，视为已转换坐标直接透传。
 *
 * @param map 目标地图，默认主世界 palpagos
 */
export function toMapPosition(
  worldPos: [number, number],
  map: MapKey = 'palpagos',
): [number, number] {
  const [wx, wy] = worldPos
  if (wx >= -256 && wx <= 256 && wy >= -256 && wy <= 256) {
    return [wx, wy]
  }
  const { minX, minY, spanX, spanY } = getSpans(map)
  const x = -256 + (256 * (wx - minX)) / spanX
  const y = (256 * (wy - minY)) / spanY
  return [x, y]
}

/**
 * Leaflet 地图坐标 -> 世界坐标（逆变换）
 * @param map 目标地图，默认主世界 palpagos
 */
export function fromMapPosition(
  mapPos: [number, number],
  map: MapKey = 'palpagos',
): [number, number] {
  const [mx, my] = mapPos
  const { minX, minY, spanX, spanY } = getSpans(map)
  const wx = ((mx + 256) * spanX) / 256 + minX
  const wy = (my * spanY) / 256 + minY
  return [wx, wy]
}

/**
 * 游戏世界距离 -> 地图坐标距离（用于据点范围圆圈半径）
 * @param map 目标地图，默认主世界 palpagos
 */
export function toMapDistance(d: number, map: MapKey = 'palpagos'): number {
  const { spanX } = getSpans(map)
  return (256 * d) / spanX
}

/**
 * 判定坐标属于哪张地图（严格边界，用于自动选图）。
 * 天坠之地范围：x∈[347351.5, 689148.5] 且 y∈[-818197, -476400]，
 * 其余归主世界。两图邻接带极小，边界外的一律归主世界。
 */
export function detectMap(x: number, y: number): MapKey {
  if (
    x >= 347351.5 &&
    x <= 689148.5 &&
    y >= -818197 &&
    y <= -476400
  ) {
    return 'feybreak'
  }
  return 'palpagos'
}

/**
 * 判断坐标是否落在指定地图边界内（含边界外 MARGIN 单位的余量），
 * 用于过滤玩家 / 据点，避免另一张图的坐标被映射到错误位置。
 */
export function pointInMap(
  x: number,
  y: number,
  map: MapKey,
  margin = 5000,
): boolean {
  if (!Number.isFinite(x) || !Number.isFinite(y)) return false
  const { maxX, maxY, minX, minY } = getSpans(map)
  return (
    x >= minX - margin &&
    x <= maxX + margin &&
    y >= minY - margin &&
    y <= maxY + margin
  )
}

/**
 * 玩家是否拥有有效地图坐标：location_x/y 都是有限数且不全为 0
 */
export function hasMapLocation(p: any): boolean {
  const x = Number(p?.location_x)
  const y = Number(p?.location_y)
  return Number.isFinite(x) && Number.isFinite(y) && (x !== 0 || y !== 0)
}

/**
 * 合并全量玩家与在线玩家：按 player_uid 合并，在线数据覆盖旧数据
 */
export function mergeMapPlayers(players: any[] = [], onlinePlayers: any[] = []): any[] {
  const merged = new Map<string, any>(
    players
      .filter((p) => p?.player_uid)
      .map((p) => [p.player_uid, { ...p }]),
  )
  onlinePlayers.forEach((p) => {
    if (!p?.player_uid) return
    merged.set(p.player_uid, {
      ...merged.get(p.player_uid),
      ...p,
    })
  })
  return Array.from(merged.values())
}

/**
 * 根据可见性筛选要显示的玩家
 * - online：只显示在线玩家
 * - all：显示所有有坐标的玩家
 */
export function selectVisibleMapPlayers(
  players: any[] = [],
  onlineUids: Set<string>,
  visibility: 'online' | 'all' = 'online',
): any[] {
  return players.filter(
    (p) =>
      hasMapLocation(p) &&
      (visibility === 'all' || onlineUids.has(p.player_uid)),
  )
}

/**
 * 建立 player_uid -> guild 的映射
 */
export function buildPlayerGuildMap(guilds: any[] = []): Map<string, any> {
  const m = new Map<string, any>()
  guilds.forEach((g) => {
    ;(g?.players || []).forEach((p: any) => {
      if (p?.player_uid) m.set(p.player_uid, g)
    })
  })
  return m
}
