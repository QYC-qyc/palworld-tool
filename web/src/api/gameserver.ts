import { request } from './index'

export interface GameServerStatus {
  available: boolean
  message?: string
  status?: {
    installed: boolean
    running: boolean
    container: string
    image: string
    game_port: string
    data_dir: string
    state?: string
    version?: string
  }
}

export interface InstallConfig {
  admin_password: string
  server_name?: string
  game_port?: string
  rcon_port?: string
  rest_port?: string
  data_dir?: string
}

export const gameApi = {
  status: () => request<GameServerStatus>('/api/gameserver'),
  install: (cfg: InstallConfig) =>
    request<{ success: boolean; message?: string }>('/api/gameserver/install', {
      method: 'POST',
      body: JSON.stringify(cfg),
    }),
  start: () => request<{ success: boolean }>('/api/gameserver/start', { method: 'POST' }),
  stop: () => request<{ success: boolean }>('/api/gameserver/stop', { method: 'POST' }),
  restart: () => request<{ success: boolean }>('/api/gameserver/restart', { method: 'POST' }),
  update: (cfg: Partial<InstallConfig>) =>
    request<{ success: boolean; message?: string }>('/api/gameserver/update', {
      method: 'POST',
      body: JSON.stringify(cfg),
    }),
  logs: () => request<{ logs: string }>('/api/gameserver/logs'),
}
