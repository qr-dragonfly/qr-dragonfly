import { CLICK_BASE_URL } from '../api/config'

export function trackingUrlForQrId(id: string): string {
  const base = CLICK_BASE_URL.replace(/\/+$/, '')
  return `${base}/r/${encodeURIComponent(id)}`
}
