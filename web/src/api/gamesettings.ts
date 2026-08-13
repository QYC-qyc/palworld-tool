import { request } from './index'

export interface ConfigField {
  key: string
  label: string
  type: string
  default: string
  description?: string
  options?: string[]
  group: string
  requires_restart: boolean
}

export interface GameSettingsData {
  settings: Record<string, string>
  path: string
  exists: boolean
}

export const gameSettingsApi = {
  schema: () =>
    request<{ fields: ConfigField[]; iniPath: string }>('/api/gamesettings/schema'),
  get: () => request<GameSettingsData>('/api/gamesettings'),
  save: (settings: Record<string, string>, restart = false) =>
    request<{ success: boolean; message?: string }>('/api/gamesettings', {
      method: 'PUT',
      body: JSON.stringify({ settings, restart }),
    }),
}
