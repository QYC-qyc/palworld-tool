import { request } from './index'

export interface GameServerStatus {
  available: boolean
  status?: {
    installed: boolean
    linux_installed: boolean
    windows_installed: boolean
    steam_ready: boolean
    running: boolean
    updating: boolean
    pid?: number
    server_exe: string
    linux_exe: string
    windows_exe: string
    steam_exe: string
    install_dir: string
    wine_mode: boolean
    state?: string
  }
}

export interface GameServerConfig {
  steamcmd_path: string
  install_dir: string
  extra_args?: string
}

export const gameApi = {
  status: () => request<GameServerStatus>('/api/gameserver'),
  getConfig: () => request<GameServerConfig>('/api/gameserver/config'),
  saveConfig: (cfg: GameServerConfig) =>
    request<{ success: boolean }>('/api/gameserver/config', {
      method: 'PUT',
      body: JSON.stringify(cfg),
    }),
  verify: (cfg: GameServerConfig) =>
    request<{
      steam_ok: boolean
      steam_exe: string
      server_ok: boolean
      server_exe: string
      linux_server_ok: boolean
      linux_server_exe: string
      windows_server_ok: boolean
      windows_server_exe: string
      wine_mode: boolean
    }>('/api/gameserver/verify', { method: 'POST', body: JSON.stringify(cfg) }),
  install: (platform: 'linux' | 'windows' = 'linux') =>
    request<{ success: boolean; message?: string }>('/api/gameserver/install', {
      method: 'POST',
      body: JSON.stringify({ platform }),
    }),
  start: () => request<{ success: boolean; message?: string }>('/api/gameserver/start', { method: 'POST' }),
  stop: () => request<{ success: boolean; message?: string }>('/api/gameserver/stop', { method: 'POST' }),
  restart: () => request<{ success: boolean; message?: string }>('/api/gameserver/restart', { method: 'POST' }),
  logs: () => request<{ logs: string }>('/api/gameserver/logs'),
}
