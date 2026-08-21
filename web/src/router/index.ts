import { createRouter, createWebHistory } from 'vue-router'
import { api } from '@/api'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/setup', name: 'setup', component: () => import('@/views/Setup.vue'), meta: { public: true } },
    { path: '/login', name: 'login', component: () => import('@/views/Login.vue'), meta: { public: true } },
    {
      path: '/',
      component: () => import('@/views/Layout.vue'),
      redirect: '/dashboard',
      children: [
        { path: 'dashboard', name: 'dashboard', component: () => import('@/views/Dashboard.vue') },
        { path: 'gameserver', name: 'gameserver', component: () => import('@/views/GameServer.vue') },
        { path: 'gamesettings', name: 'gamesettings', component: () => import('@/views/GameSettings.vue') },
        { path: 'players', name: 'players', component: () => import('@/views/Players.vue') },
        { path: 'playermap', name: 'playermap', component: () => import('@/views/PlayerMap.vue') },
        { path: 'guilds', name: 'guilds', component: () => import('@/views/Guilds.vue') },
        { path: 'whitelist', name: 'whitelist', component: () => import('@/views/Whitelist.vue') },
        { path: 'backups', name: 'backups', component: () => import('@/views/Backups.vue') },
        { path: 'paldefender', name: 'paldefender', component: () => import('@/views/PalDefender.vue') },
        { path: 'audit', name: 'audit', component: () => import('@/views/Audit.vue') },
        { path: 'settings', name: 'settings', component: () => import('@/views/Settings.vue') },
      ],
    },
  ],
})

// 缓存初始化状态，避免每次切换菜单都请求后端阻塞导航
let initCache: { initialized: boolean; checked: boolean } = { initialized: true, checked: false }

export function resetRouterCache() {
  initCache = { initialized: true, checked: false }
}

router.beforeEach(async (to, _from, next) => {
  if (to.meta.public) return next()

  const token = localStorage.getItem('palworld-panel_token')
  if (!token) return next({ name: 'login' })

  // 已初始化过则直接放行，不再串行请求阻塞菜单切换
  if (!initCache.checked) {
    try {
      const status = await api.setupStatus()
      initCache = { initialized: status.initialized, checked: true }
      if (!status.initialized) return next({ name: 'setup' })
    } catch {
      // 接口异常不阻塞
    }
  } else if (!initCache.initialized) {
    return next({ name: 'setup' })
  }

  next()
})

export default router
