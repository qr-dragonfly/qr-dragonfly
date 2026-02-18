import { requestJson } from '../http'
import { QR_API_BASE_URL } from '../config'
import type { CreateQrCodeInput, QrCode, UpdateQrCodeInput } from './qrCodes.types'

export type ListQrCodesParams = {
  limit?: number
  cursor?: string
}

export const qrCodesApi = {
  list(params?: ListQrCodesParams, userType?: string): Promise<QrCode[]> {
    return requestJson<QrCode[]>({
      baseUrl: QR_API_BASE_URL,
      method: 'GET',
      path: '/api/qr-codes',
      query: params ? { limit: params.limit, cursor: params.cursor } : undefined,
      headers: userType ? { 'X-User-Type': userType } : undefined,
    })
  },

  getById(id: string, userType?: string): Promise<QrCode> {
    return requestJson<QrCode>({
      baseUrl: QR_API_BASE_URL,
      method: 'GET',
      path: `/api/qr-codes/${encodeURIComponent(id)}`,
      headers: userType ? { 'X-User-Type': userType } : undefined,
    })
  },

  create(input: CreateQrCodeInput, userType?: string): Promise<QrCode> {
    return requestJson<QrCode>({
      baseUrl: QR_API_BASE_URL,
      method: 'POST',
      path: '/api/qr-codes',
      body: input,
      headers: userType ? { 'X-User-Type': userType } : undefined,
    })
  },

  update(id: string, patch: UpdateQrCodeInput, userType?: string): Promise<QrCode> {
    return requestJson<QrCode>({
      baseUrl: QR_API_BASE_URL,
      method: 'PATCH',
      path: `/api/qr-codes/${encodeURIComponent(id)}`,
      body: patch,
      headers: userType ? { 'X-User-Type': userType } : undefined,
    })
  },

  delete(id: string, userType?: string): Promise<void> {
    return requestJson<void>({
      baseUrl: QR_API_BASE_URL,
      method: 'DELETE',
      path: `/api/qr-codes/${encodeURIComponent(id)}`,
      headers: userType ? { 'X-User-Type': userType } : undefined,
    })
  },
}
