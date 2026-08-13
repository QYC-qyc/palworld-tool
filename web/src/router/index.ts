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
        { path: 'guilds', name: 'guilds', component: () => import('@/views/Guilds.vue') },
        { path: 'whitelist', name: 'whitelist', component: () => import('@/views/Whitelist.vue') },
        { path: 'banlist', name: 'banlist', component: () => import('@/views/Banlist.vue') },
        { path: 'rcon', name: 'rcon', component: () => import('@/views/Rcon.vue') },
        { path: 'backups', name: 'backups', component: () => import('@/views/Backups.vue') },
        { path: 'settings', name: 'settings', component: () => import('@/views/Settings.vue') },
        {
          path: 'anticheat',
          name: 'anticheat',
          component: () => import('@/views/anticheat/Alerts.vue'),
        },
        {
          path: 'anticheat/rules',
          name: 'ac-rules',
          component: () => import('@/views/anticheat/Rules.vue'),
        },
        {
          path: 'anticheat/audit',
          name: 'ac-audit',
          component: () => import('@/views/anticheat/Audit.vue'),
        },
      ],
    },
  ],
})

router.beforeEach(async (to, _from, next) => {
  // 公开页面直接放行
  if (to.meta.public) return next()

  // 检查是否已初始化
  try {
    const status = await api.setupStatus()
    if (!status.initialized) {
      return next({ name: 'setup' })
    }
  } catch {
    // 接口异常不阻塞，继续走登录校验
  }

  // 登录校验
  const token = localStorage.getItem('paladmin_token')
  if (!token) return next({ name: 'login' })
  next()
})

export default router
