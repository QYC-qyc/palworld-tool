<template>
  <n-layout has-sider style="height: 100vh">
    <!-- 桌面端固定侧边栏 -->
    <n-layout-sider
      v-if="!isMobile"
      bordered
      :collapsed="collapsed"
      :width="224"
      :collapsed-width="64"
      show-trigger
      collapse-mode="width"
      @collapse="collapsed = true"
      @expand="collapsed = false"
      class="sider"
    >
      <div class="logo">
        <n-icon :component="GameControllerOutline" size="22" />
        <span v-if="!collapsed" class="logo-text">PalAdmin</span>
      </div>
      <n-menu
        :collapsed="collapsed"
        :collapsed-width="64"
        :collapsed-icon-size="20"
        :options="menuOptions"
        :value="activeKey"
        @update:value="navigate"
      />
    </n-layout-sider>

    <!-- 手机端抽屉菜单 -->
    <n-drawer v-if="isMobile" v-model:show="showDrawer" :width="260" placement="left">
      <n-drawer-content>
        <div class="logo">
          <n-icon :component="GameControllerOutline" size="22" />
          <span class="logo-text">PalAdmin</span>
        </div>
        <n-menu
          :options="menuOptions"
          :value="activeKey"
          @update:value="navigate"
        />
      </n-drawer-content>
    </n-drawer>

    <n-layout>
      <n-layout-header bordered class="header">
        <div class="header-left">
          <n-button quaternary circle @click="toggleMenu" aria-label="菜单">
            <template #icon>
              <n-icon :component="MenuOutline" />
            </template>
          </n-button>
          <n-breadcrumb>
            <n-breadcrumb-item>{{ currentLabel }}</n-breadcrumb-item>
          </n-breadcrumb>
        </div>
        <n-space align="center" :size="isMobile ? 6 : 12" :wrap="false">
          <n-tooltip trigger="hover" placement="bottom">
            <template #trigger>
              <span class="server-dot" :class="{ 'server-dot--on': serverOk }">
                <span class="pulse" v-if="serverOk" />
              </span>
            </template>
            {{ serverOk ? `服务器在线 · ${serverVersion}` : '服务器未连接' }}
          </n-tooltip>

          <n-tag type="info" size="small" round :bordered="false" class="mode-tag">
            <n-icon :component="LogoWindows" size="14" style="vertical-align:-2px;margin-right:4px" />
            Windows
          </n-tag>

          <n-button quaternary circle @click="toggleTheme" aria-label="切换主题">
            <template #icon>
              <n-icon :component="isDark ? SunnyOutline : MoonOutline" />
            </template>
          </n-button>

          <n-dropdown :options="userMenu" @select="onUserMenu">
            <n-button quaternary :size="isMobile ? 'small' : 'medium'">
              <template #icon>
                <n-icon :component="PersonCircleOutline" />
              </template>
              <span v-if="!isMobile">管理员</span>
            </n-button>
          </n-dropdown>
        </n-space>
      </n-layout-header>
      <n-layout-content class="content" content-style="padding: 20px;">
        <div class="content-inner" :class="{ 'content-inner--wide': isWide }">
          <router-view />
        </div>
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<script setup lang="ts">
import { computed, h, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  NLayout, NLayoutSider, NLayoutHeader, NLayoutContent, NMenu, NButton,
  NSpace, NTag, NDropdown, NDrawer, NDrawerContent, NIcon, NTooltip,
  NBreadcrumb, NBreadcrumbItem, useMessage,
} from 'naive-ui'
import type { MenuOption } from 'naive-ui'
import {
  GridOutline, ServerOutline, SettingsOutline, PeopleOutline, MapOutline,
  GitNetworkOutline, ShieldCheckmarkOutline, ArchiveOutline, ListOutline,
  ShieldOutline, CogOutline, GameControllerOutline, MenuOutline,
  MoonOutline, SunnyOutline, LogoWindows, PersonCircleOutline,
  LogOutOutline,
} from '@vicons/ionicons5'
import { api } from '@/api'
import { useTheme } from '@/composables/useTheme'

const collapsed = ref(false)
const showDrawer = ref(false)
const isMobile = ref(false)
const router = useRouter()
const route = useRoute()
const message = useMessage()
const serverOk = ref(false)
const serverVersion = ref('')

const { isDark, toggle: toggleTheme } = useTheme()

const activeKey = computed(() => route.path)
// 需要全宽展示的页面（地图、游戏服日志/配置）
const WIDE_ROUTES = ['/playermap', '/gameserver']
const isWide = computed(() => WIDE_ROUTES.includes(route.path))

const menuOptions = computed<MenuOption[]>(() => {
  const icon = (c: any) => () => h(NIcon, { component: c, size: 18 })
  const item = (label: string, key: string, c: any): MenuOption => ({
    label, key, icon: icon(c),
  })
  const groups: MenuOption[] = [
    {
      type: 'group', label: '概览', key: 'g-overview',
      children: [item('仪表盘', '/dashboard', GridOutline)],
    },
    {
      type: 'group', label: '游戏服', key: 'g-server',
      children: [
        item('游戏服', '/gameserver', ServerOutline),
        item('游戏配置', '/gamesettings', SettingsOutline),
      ],
    },
    {
      type: 'group', label: '玩家', key: 'g-player',
      children: [
        item('玩家', '/players', PeopleOutline),
        item('玩家地图', '/playermap', MapOutline),
        item('公会', '/guilds', GitNetworkOutline),
        item('白名单', '/whitelist', ShieldCheckmarkOutline),
      ],
    },
    {
      type: 'group', label: '维护', key: 'g-maint',
      children: [
        item('备份', '/backups', ArchiveOutline),
        item('审计', '/audit', ListOutline),
      ],
    },
  ]
  groups.push({
    type: 'group', label: '反作弊', key: 'g-ac',
    children: [item('PalDefender', '/paldefender', ShieldOutline)],
  })
  groups.push({
    type: 'group', label: '系统', key: 'g-sys',
    children: [item('设置', '/settings', CogOutline)],
  })
  return groups
})

const labelMap = computed<Record<string, string>>(() => {
  const m: Record<string, string> = {}
  menuOptions.value.forEach((g: any) => {
    g.children?.forEach((c: any) => { m[c.key] = c.label })
  })
  return m
})
const currentLabel = computed(() => labelMap.value[route.path] || 'PalAdmin')

const userMenu = [
  { label: '退出登录', key: 'logout', icon: () => h(NIcon, { component: LogOutOutline }) },
]

function checkMobile() {
  isMobile.value = window.innerWidth < 768
}

function toggleMenu() {
  if (isMobile.value) {
    showDrawer.value = true
  } else {
    collapsed.value = !collapsed.value
  }
}

function navigate(v: string) {
  router.push(v)
  showDrawer.value = false
}

function onUserMenu(key: string) {
  if (key === 'logout') {
    localStorage.removeItem('paladmin_token')
    router.push('/login')
  }
}

async function checkServer() {
  try {
    const s: any = await api.getServer()
    serverOk.value = !!s.version
    serverVersion.value = s.version || ''
  } catch {
    serverOk.value = false
    serverVersion.value = ''
  }
}

onMounted(() => {
  checkMobile()
  window.addEventListener('resize', checkMobile)
  checkServer()
})

onUnmounted(() => window.removeEventListener('resize', checkMobile))
</script>

<style scoped>
.sider :deep(.n-layout-sider-scroll-container) {
  /* 让菜单背景统一 */
}
.logo {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font-size: 20px;
  font-weight: 700;
  color: var(--n-primary-color, #4f46e5);
}
.logo-text {
  letter-spacing: 0.5px;
}
.header {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
}
.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}
.content {
  height: calc(100vh - 56px);
  overflow-y: auto;
}
.content-inner {
  max-width: 1280px;
  margin: 0 auto;
}
.content-inner--wide {
  max-width: none;
}
.server-dot {
  position: relative;
  display: inline-block;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #909399;
  cursor: help;
}
.server-dot--on {
  background: #18a058;
}
.pulse {
  position: absolute;
  inset: 0;
  border-radius: 50%;
  background: #18a058;
  animation: pulse 1.8s ease-out infinite;
}
@keyframes pulse {
  0% { transform: scale(1); opacity: 0.7; }
  100% { transform: scale(2.4); opacity: 0; }
}
.mode-tag {
  font-weight: 500;
}
</style>
