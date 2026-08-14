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

  getRconCommands: () => request<any[]>('/api/rcon'),
  sendRcon: (command: string, content = '') =>
    request('/api/rcon/send', { method: 'POST', body: JSON.stringify({ command, content }) }),

  getBackups: () => request<any[]>('/api/backup'),
  deleteBackup: (id: string) => request(`/api/backup/${id}`, { method: 'DELETE' }),
  restoreBackup: (id: string) =>
    request(`/api/backup/restore/${id}`, { method: 'POST' }),

  getBanlist: () => request<any[]>('/api/banlist?active=true'),
  banIP: (ip: string) =>
    request('/api/banip', { method: 'POST', body: JSON.stringify({ ip }) }),
  unbanIP: (ip: string) =>
    request('/api/unbanip', { method: 'POST', body: JSON.stringify({ ip }) }),

  getAlerts: (params = '') => request<any>(`/api/anticheat/alert?${params}`),
  getAlert: (id: number) => request<any>(`/api/anticheat/alert/${id}`),
  alertAction: (id: number, status: string, note = '') =>
    request(`/api/anticheat/alert/${id}/action`, {
      method: 'POST',
      body: JSON.stringify({ status, note }),
    }),
  getRules: () => request<any[]>('/api/anticheat/rule'),
  updateRule: (id: string, body: any) =>
    request(`/api/anticheat/rule/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  runScan: () => request('/api/anticheat/scan', { method: 'POST' }),
  getStats: () => request<any>('/api/anticheat/stats'),
  getAudit: () => request<any[]>('/api/anticheat/audit'),
  reloadAC: () => request('/api/anticheat/reload', { method: 'POST' }),

  sync: (from = 'all') => request(`/api/sync?from=${from}`, { method: 'POST' }),

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
