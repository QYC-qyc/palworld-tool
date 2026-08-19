import { ref, computed } from 'vue'
import { useDialog, useMessage } from 'naive-ui'
import { api } from '@/api'

export interface RunModeInfo {
  wine_mode: boolean
  linux_installed: boolean
  windows_installed: boolean
  running: boolean
}

// 模块级单例状态
const loading = ref(false)
const info = ref<RunModeInfo>({
  wine_mode: false,
  linux_installed: false,
  windows_installed: false,
  running: false,
})
const loaded = ref(false)
let inflight: Promise<void> | null = null

async function fetchMode(force = false) {
  if (inflight && !force) return inflight
  loading.value = true
  inflight = (async () => {
    try {
      const data = await api.getRunMode()
      info.value = data
      loaded.value = true
    } catch {
      // 静默失败，保留默认值
    } finally {
      loading.value = false
    }
  })()
  return inflight
}

// 首次自动拉取（模块加载即触发，组件内可 await refresh）
fetchMode()

export function useRunMode() {
  const dialog = useDialog()
  const message = useMessage()

  const isWindows = computed(() => info.value.wine_mode)
  const isLinux = computed(() => !info.value.wine_mode)
  const linuxInstalled = computed(() => info.value.linux_installed)
  const windowsInstalled = computed(() => info.value.windows_installed)
  const running = computed(() => info.value.running)

  function refresh() {
    return fetchMode(true)
  }

  function setMode(windows: boolean) {
    if (windows === info.value.wine_mode) return
    dialog.warning({
      title: windows ? '切换到 Windows（Wine）模式？' : '切换到原生 Linux 模式？',
      content: windows
        ? '用 Wine 启动 Windows 版服务端以加载 PalDefender 反作弊，需已安装 Wine/Windows版游戏服/PalDefender DLL，切换后需重启游戏服。'
        : '用原生 PalServer.sh 启动，性能更好但无法使用 PalDefender，需已安装 Linux 版游戏服，切换后需重启。',
      positiveText: '确认切换',
      negativeText: '取消',
      onPositiveClick: async () => {
        try {
          await api.saveSettings({
            'paldefender.wine_mode': windows ? 'true' : 'false',
          })
          message.success('已切换，请重启游戏服生效')
          await refresh()
        } catch (e: any) {
          message.error(e.message || '切换失败')
        }
      },
    })
  }

  return {
    isWindows,
    isLinux,
    loading,
    loaded,
    linuxInstalled,
    windowsInstalled,
    running,
    refresh,
    setMode,
  }
}
