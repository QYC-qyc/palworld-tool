// 地图坐标换算工具
// 参考 _palworld-server-tool MapView.vue / utils/mapPlayers.js
// 世界坐标使用虚幻引擎坐标，范围约 -1.4M ~ +1.4M
// 映射到 Leaflet Simple CRS 的 [0,256] x [-256,0] 坐标空间

// 世界坐标边界：[maxX, maxY, minX, minY]
export const LAND_SCAPE = [349400, 724400, -1099400, -724400] as const

const [MAX_X, MAX_Y, MIN_X, MIN_Y] = LAND_SCAPE
const SPAN_X = MAX_X - MIN_X
const SPAN_Y = MAX_Y - MIN_Y

/**
 * 世界坐标 -> Leaflet 地图坐标 [x_map, y_map]
 * x_map = -256 + 256 * (wx - minX) / (maxX - minX)  -> [-256, 0]
 * y_map =  256 * (wy - minY) / (maxY - minY)        -> [0, 256]
 *
 * 注意：Leaflet 用 [lat, lng] = [x_map, y_map]，lat 为负值、lng 为正值，
 * 与瓦片 bounds [[0,0],[-256,256]] 一致。
 *
 * hack：若输入已经在 [-256, 256] 范围内，视为已转换坐标直接透传。
 */
export function toMapPosition(worldPos: [number, number]): [number, number] {
  const [wx, wy] = worldPos
  if (wx >= -256 && wx <= 256 && wy >= -256 && wy <= 256) {
    return [wx, wy]
  }
  const x = -256 + (256 * (wx - MIN_X)) / SPAN_X
  const y = (256 * (wy - MIN_Y)) / SPAN_Y
  return [x, y]
}

/**
 * Leaflet 地图坐标 -> 世界坐标（逆变换）
 */
export function fromMapPosition(mapPos: [number, number]): [number, number] {
  const [mx, my] = mapPos
  const wx = ((mx + 256) * SPAN_X) / 256 + MIN_X
  const wy = (my * SPAN_Y) / 256 + MIN_Y
  return [wx, wy]
}

/**
 * 游戏世界距离 -> 地图坐标距离（用于据点范围圆圈半径）
 */
export function toMapDistance(d: number): number {
  return (256 * d) / SPAN_X
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
