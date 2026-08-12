import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: () => import('@/views/Login.vue') },
    {
      path: '/',
      component: () => import('@/views/Layout.vue'),
      redirect: '/dashboard',
      children: [
        { path: 'dashboard', name: 'dashboard', component: () => import('@/views/Dashboard.vue') },
        { path: 'players', name: 'players', component: () => import('@/views/Players.vue') },
        { path: 'guilds', name: 'guilds', component: () => import('@/views/Guilds.vue') },
        { path: 'whitelist', name: 'whitelist', component: () => import('@/views/Whitelist.vue') },
        { path: 'rcon', name: 'rcon', component: () => import('@/views/Rcon.vue') },
        { path: 'backups', name: 'backups', component: () => import('@/views/Backups.vue') },
        { path: 'banlist', name: 'banlist', component: () => import('@/views/Banlist.vue') },
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

router.beforeEach((to, _from, next) => {
  const token = localStorage.getItem('paladmin_token')
  if (to.name !== 'login' && !token) next({ name: 'login' })
  else next()
})

export default router
