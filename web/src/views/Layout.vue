<template>
  <n-layout has-sider style="height: 100vh">
    <n-layout-sider bordered :collapsed="collapsed" :width="220" :collapsed-width="64">
      <div class="logo">{{ collapsed ? 'PA' : 'PalAdmin' }}</div>
      <n-menu
        :collapsed="collapsed"
        :collapsed-width="64"
        :collapsed-icon-size="22"
        :options="menuOptions"
        :value="activeKey"
        @update:value="(v: string) => router.push(v)"
      />
    </n-layout-sider>
    <n-layout>
      <n-layout-header bordered class="header">
        <n-button text @click="collapsed = !collapsed">
          <template #icon><span>☰</span></template>
        </n-button>
        <n-space align="center" :size="12">
          <n-tag :type="serverOk ? 'success' : 'error'" size="small" round>
            {{ serverOk ? '服务器在线' : '服务器未连接' }}
          </n-tag>
          <n-dropdown :options="userMenu" @select="onUserMenu">
            <n-button quaternary>管理员</n-button>
          </n-dropdown>
        </n-space>
      </n-layout-header>
      <n-layout-content class="content" content-style="padding: 16px;">
        <router-view />
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  NLayout,
  NLayoutSider,
  NLayoutHeader,
  NLayoutContent,
  NMenu,
  NButton,
  NSpace,
  NTag,
  NDropdown,
  useMessage,
} from 'naive-ui'
import { api } from '@/api'

const collapsed = ref(false)
const router = useRouter()
const route = useRoute()
const message = useMessage()
const serverOk = ref(false)

const activeKey = computed(() => route.path)

const menuOptions = [
  { label: '仪表盘', key: '/dashboard' },
  { label: '玩家管理', key: '/players' },
  { label: '公会', key: '/guilds' },
  { label: '白名单', key: '/whitelist' },
  { label: '封禁列表', key: '/banlist' },
  { label: 'PalDefender', key: '/paldefender' },
  { label: 'RCON 控制台', key: '/rcon' },
  { label: '备份管理', key: '/backups' },
  { label: '系统设置', key: '/settings' },
  { label: '反作弊告警', key: '/anticheat' },
  { label: '反作弊规则', key: '/anticheat/rules' },
  { label: '审计日志', key: '/anticheat/audit' },
]

const userMenu = [
  { label: '退出登录', key: 'logout' },
]

function onUserMenu(key: string) {
  if (key === 'logout') {
    localStorage.removeItem('paladmin_token')
    router.push('/login')
  }
}

onMounted(async () => {
  try {
    const s = await api.getServer()
    serverOk.value = !!s.version
  } catch {
    serverOk.value = false
  }
})
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
  padding: 0 16px;
}
.content {
  height: calc(100vh - 56px);
  overflow-y: auto;
}
</style>
