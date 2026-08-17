const BASE = ''

function getToken(): string {
  return localStorage.getItem('paladmin_token') || ''
}

export async function request<T = any>(
  url: string,
  options: RequestInit = {}
): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string>),
  }
  const token = getToken()
  if (token) headers['Authorization'] = 'Bearer ' + token

  const resp = await fetch(BASE + url, { ...options, headers })
  if (resp.status === 401) {
    localStorage.removeItem('paladmin_token')
    if (location.pathname !== '/login') location.href = '/login'
    throw new Error('未授权')
  }
  const data = await resp.json().catch(() => ({}))
  if (!resp.ok) {
    throw new Error(data.error || data.message || '请求失败')
  }
  return data as T
}

export const api = {
  // 初始化向导
  setupStatus: () => request<{ initialized: boolean }>('/api/setup/status'),
  setup: (data: Record<string, string>) =>
    request<{ token: string }>('/api/setup', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  login: (password: string) =>
    request<{ token: string }>('/api/login', {
      method: 'POST',
      body: JSON.stringify({ password }),
    }),

  getServer: () => request<any>('/api/server'),
  getMetrics: () => request<any>('/api/server/metrics'),
  broadcast: (message: string) =>
    request('/api/server/broadcast', { method: 'POST', body: JSON.stringify({ message }) }),
  shutdown: (waittime: number, message: string) =>
    request('/api/server/shutdown', {
      method: 'POST',
      body: JSON.stringify({ waittime, message }),
    }),

  getPlayers: () => request<any[]>('/api/player'),
  getPlayer: (uid: string) => request<any>(`/api/player/${uid}`),
  getOnline: () => request<any[]>('/api/online_player'),
  kickPlayer: (uid: string) => request(`/api/player/${uid}/kick`, { method: 'POST' }),
  banPlayer: (uid: string) => request(`/api/player/${uid}/ban`, { method: 'POST' }),
  unbanPlayer: (uid: string) => request(`/api/player/${uid}/unban`, { method: 'POST' }),
  ipBanPlayer: (uid: string) => request(`/api/player/${uid}/ipban`, { method: 'POST' }),

  getGuilds: () => request<any[]>('/api/guild'),
  getGuild: (uid: string) => request<any>(`/api/guild/${uid}`),

  getWhitelist: () => request<any[]>('/api/whitelist'),
  addWhitelist: (p: any) =>
    request('/api/whitelist', { method: 'POST', body: JSON.stringify(p) }),
  removeWhitelist: (p: any) =>
    request('/api/whitelist', { method: 'DELETE', body: JSON.stringify(p) }),

  getBackups: () => request<any[]>('/api/backup'),
  deleteBackup: (id: string) => request(`/api/backup/${id}`, { method: 'DELETE' }),
  restoreBackup: (id: string) =>
    request(`/api/backup/restore/${id}`, { method: 'POST' }),

  getBanlist: () => request<any[]>('/api/banlist?active=true'),
  banIP: (ip: string) =>
    request('/api/banip', { method: 'POST', body: JSON.stringify({ ip }) }),
  unbanIP: (ip: string) =>
    request('/api/unbanip', { method: 'POST', body: JSON.stringify({ ip }) }),

  sync: (from = 'all') => request(`/api/sync?from=${from}`, { method: 'POST' }),

  // PalDefender REST API 代理（前缀 /api/paldefender/api）
  pdVersion: () => request<any>('/api/paldefender/api/version'),
  pdGetPlayers: () => request<any>('/api/paldefender/api/players'),
  pdGetPlayer: (id: string) => request<any>(`/api/paldefender/api/players/${encodeURIComponent(id)}`),
  pdKick: (id: string, reason = '') =>
    request(`/api/paldefender/api/players/${encodeURIComponent(id)}/kick`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    }),
  pdBan: (id: string, reason = '', ip = false) =>
    request(`/api/paldefender/api/players/${encodeURIComponent(id)}/ban`, {
      method: 'POST',
      body: JSON.stringify({ reason, ip }),
    }),
  pdUnban: (userId: string, reason = '') =>
    request(`/api/paldefender/api/unban/${encodeURIComponent(userId)}`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    }),
  pdBanIP: (ip: string, reason = '') =>
    request(`/api/paldefender/api/banip/${encodeURIComponent(ip)}`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    }),
  pdUnbanIP: (ip: string, reason = '') =>
    request(`/api/paldefender/api/unbanip/${encodeURIComponent(ip)}`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    }),
  pdBanlist: (query?: Record<string, string | number | boolean>) => {
    const qs = query
      ? '?' + new URLSearchParams(Object.fromEntries(Object.entries(query).map(([k, v]) => [k, String(v)]))).toString()
      : ''
    return request<any>(`/api/paldefender/api/banlist${qs}`)
  },
  pdBroadcast: (message: string) =>
    request('/api/paldefender/api/broadcast', {
      method: 'POST',
      body: JSON.stringify({ message }),
    }),
  pdAlert: (message: string) =>
    request('/api/paldefender/api/alert', {
      method: 'POST',
      body: JSON.stringify({ message }),
    }),
  pdMessage: (userId: string, message: string, sendType?: string) =>
    request('/api/paldefender/api/message', {
      method: 'POST',
      body: JSON.stringify({ userId, message, sendType: sendType || 'PlayerChat' }),
    }),
  pdGuilds: () => request<any>('/api/paldefender/api/guilds'),
  pdGuild: (id: string) => request<any>(`/api/paldefender/api/guilds/${encodeURIComponent(id)}`),
  pdDeleteBase: (id: string) =>
    request(`/api/paldefender/api/deletebase/${encodeURIComponent(id)}`, {
      method: 'POST',
      body: JSON.stringify({}),
    }),
  pdReloadConfig: () =>
    request('/api/paldefender/api/reload-config', { method: 'POST', body: JSON.stringify({}) }),

  // 面板动态设置
  getSettings: () => request<Record<string, any>>('/api/settings'),
  saveSettings: (settings: Record<string, any>) =>
    request('/api/settings', { method: 'PUT', body: JSON.stringify(settings) }),

  // 面板在线更新
  checkUpdate: () =>
    request<{
      current: string
      has_update: boolean
      latest?: string
      name?: string
      body?: string
      published?: string
      error?: string
    }>('/api/updater/check'),
  doUpdateURL: () => '/api/updater/do',
}
