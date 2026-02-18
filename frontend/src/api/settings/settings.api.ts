import { requestJson } from '../http'
import { QR_API_BASE_URL } from '../config'
import type { UserSettings } from './settings.types'

export const settingsApi = {
  async get(userType: string): Promise<UserSettings> {
    return requestJson<UserSettings>({
      baseUrl: QR_API_BASE_URL,
      method: 'GET',
      path: '/api/settings',
      headers: { 'X-User-Type': userType },
    })
  },

  async update(settings: UserSettings, userType: string): Promise<UserSettings> {
    return requestJson<UserSettings>({
      baseUrl: QR_API_BASE_URL,
      method: 'PUT',
      path: '/api/settings',
      headers: { 'X-User-Type': userType },
      body: settings,
    })
  },
}
