// 据点标记聚合
// 参考 _palworld-server-tool MapView.vue clusteredBaseMarkers 计算属性

export interface BaseMarker {
  key: string
  position: [number, number]
  camp?: any
  guildName?: string
  level?: number
  members?: string[]
}

export interface BaseCluster {
  key: string
  position: [number, number]
  markers: BaseMarker[]
}

/**
 * 按缩放等级聚合据点：
 * - zoom >= 5 不聚合，每个据点独立
 * - zoom <  5 按网格聚合，cellSize = 48 / 2^zoom，
 *   以 floor(position[0]/cellSize):floor(position[1]/cellSize) 为分桶 key，
 *   聚合位置取桶内所有点的平均值。
 */
export function clusterBases(markers: BaseMarker[], zoom: number): BaseCluster[] {
  if (zoom >= 5) {
    return markers.map((m) => ({ key: m.key, position: m.position, markers: [m] }))
  }

  const cellSize = 48 / 2 ** zoom
  const buckets = new Map<string, BaseMarker[]>()
  markers.forEach((m) => {
    const cx = Math.floor(m.position[0] / cellSize)
    const cy = Math.floor(m.position[1] / cellSize)
    const key = `${cx}:${cy}`
    const arr = buckets.get(key)
    if (arr) arr.push(m)
    else buckets.set(key, [m])
  })

  return Array.from(buckets.entries()).map(([key, arr]) => ({
    key,
    position: [
      arr.reduce((s, m) => s + m.position[0], 0) / arr.length,
      arr.reduce((s, m) => s + m.position[1], 0) / arr.length,
    ] as [number, number],
    markers: arr,
  }))
}
