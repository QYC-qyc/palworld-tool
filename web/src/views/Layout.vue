<template>
  <n-layout has-sider style="height: 100vh">
    <!-- 桌面端固定侧边栏 -->
    <n-layout-sider
      v-if="!isMobile"
      bordered
      :collapsed="collapsed"
      :width="220"
      :collapsed-width="64"
      show-trigger
      collapse-mode="width"
      @collapse="collapsed = true"
      @expand="collapsed = false"
    >
      <div class="logo">{{ collapsed ? 'PA' : 'PalAdmin' }}</div>
      <n-menu
        :collapsed="collapsed"
        :collapsed-width="64"
        :collapsed-icon-size="22"
        :options="menuOptions"
        :value="activeKey"
        @update:value="navigate"
      />
    </n-layout-sider>

    <!-- 手机端抽屉菜单 -->
    <n-drawer v-if="isMobile" v-model:show="showDrawer" :width="260" placement="left">
      <n-drawer-content>
        <div class="logo">PalAdmin</div>
        <n-menu
          :options="menuOptions"
          :value="activeKey"
          @update:value="navigate"
        />
      </n-drawer-content>
    </n-drawer>

    <n-layout>
      <n-layout-header bordered class="header">
        <n-button text @click="toggleMenu" aria-label="菜单">
          <span style="font-size:18px;line-height:1">☰</span>
        </n-button>
        <n-space align="center" :size="isMobile ? 6 : 12" :wrap="false">
          <n-tag :type="serverOk ? 'success' : 'error'" size="small" round>
            {{ isMobile ? (serverOk ? '在线' : '离线') : (serverOk ? '服务器在线' : '服务器未连接') }}
          </n-tag>
          <n-dropdown :options="userMenu" @select="onUserMenu">
            <n-button quaternary :size="isMobile ? 'small' : 'medium'">管理员</n-button>
          </n-dropdown>
        </n-space>
      </n-layout-header>
      <n-layout-content class="content" content-style="padding: 12px;">
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
  NSpace, NTag, NDropdown, NDrawer, NDrawerContent, useMessage,
} from 'naive-ui'
import { api } from '@/api'

const collapsed = ref(false)
const showDrawer = ref(false)
const isMobile = ref(false)
const router = useRouter()
const route = useRoute()
const message = useMessage()
const serverOk = ref(false)

const activeKey = computed(() => route.path)
// 需要全宽展示的页面（地图等）
const WIDE_ROUTES = ['/playermap']
const isWide = computed(() => WIDE_ROUTES.includes(route.path))

const menuOptions = [
  { label: '仪表盘', key: '/dashboard' },
  { label: '游戏服', key: '/gameserver' },
  { label: '游戏配置', key: '/gamesettings' },
  { label: '玩家', key: '/players' },
  { label: '玩家地图', key: '/playermap' },
  { label: '公会', key: '/guilds' },
  { label: '白名单', key: '/whitelist' },
  { label: '备份', key: '/backups' },
  { label: 'PalDefender', key: '/paldefender' },
  { label: '审计', key: '/audit' },
  { label: '设置', key: '/settings' },
]

const userMenu = [{ label: '退出登录', key: 'logout' }]

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

onMounted(() => {
  checkMobile()
  window.addEventListener('resize', checkMobile)
  api.getServer().then((s: any) => (serverOk.value = !!s.version)).catch(() => (serverOk.value = false))
})

onUnmounted(() => window.removeEventListener('resize', checkMobile))
</script>

<style scoped>
.logo {
  height: 56px;
  line-height: 56px;
  text-align: center;
  font-size: 20px;
  font-weight: 700;
  color: #6366f1;
}
.header {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 12px;
}
.content {
  height: calc(100vh - 56px);
  overflow-y: auto;
}
.content-inner {
  max-width: 1200px;
  margin: 0 auto;
}
.content-inner--wide {
  max-width: none;
}
</style>
