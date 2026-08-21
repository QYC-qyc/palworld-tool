import { ref, computed } from 'vue'
import { darkTheme } from '@/theme'

const STORAGE_KEY = 'palworld-panel-theme'

export type ThemeMode = 'dark' | 'light'

function getInitialMode(): ThemeMode {
  const saved = localStorage.getItem(STORAGE_KEY) as ThemeMode | null
  if (saved === 'dark' || saved === 'light') return saved
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

// 模块级单例
const mode = ref<ThemeMode>(getInitialMode())

function applyBodyBg(m: ThemeMode) {
  document.body.style.background = m === 'dark' ? '#18181c' : '#f5f6f8'
}
applyBodyBg(mode.value)

export function useTheme() {
  const isDark = computed(() => mode.value === 'dark')
  const theme = computed(() => (isDark.value ? darkTheme : null))

  function setMode(m: ThemeMode) {
    mode.value = m
    localStorage.setItem(STORAGE_KEY, m)
    applyBodyBg(m)
  }

  function toggle() {
    setMode(isDark.value ? 'light' : 'dark')
  }

  return { mode, isDark, theme, setMode, toggle }
}
