import { computed, ref, watchEffect } from 'vue'
import { ApiError, qrCodesApi } from '../api'
import { useUser } from './useUser'
import { generateQrDataUrl } from '../lib/qr'
import { trackingUrlForQrId } from '../lib/tracking'
import type { QrCodeItem } from '../types/qrCodeItem'

function validateTargetUrl(raw: string): { ok: true; url: string } | { ok: false; message: string } {
  const value = raw.trim()
  if (!value) return { ok: false, message: 'Enter a URL.' }

  // Helpful message for common input like "example.com" or "www.example.com".
  if (!value.includes('://')) {
    return { ok: false, message: 'URL must start with https:// (example: https://example.com).' }
  }

  let parsed: URL
  try {
    parsed = new URL(value)
  } catch {
    return { ok: false, message: 'That doesn’t look like a valid URL. Example: https://example.com' }
  }

  if (parsed.protocol !== 'https:') {
    return { ok: false, message: 'URL must start with https:// (http is not allowed).' }
  }
  if (!parsed.hostname) {
    return { ok: false, message: 'URL must include a hostname (example: https://example.com).' }
  }

  return { ok: true, url: parsed.toString() }
}

function qrCodesErrorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    const payload = err.payload as any
    const code = payload?.error
    if (typeof code === 'string' && code.trim()) {
      switch (code) {
        case 'url_required':
          return 'Enter a URL.'
        case 'url_invalid':
          return 'URL must be a valid https URL (example: https://example.com).'
        case 'quota_total_exceeded':
          return 'You’ve reached your QR code limit for your plan.'
        case 'quota_active_exceeded':
          return 'You’ve reached your active QR code limit for your plan.'
        case 'not_found':
          return 'That QR code no longer exists.'
        default:
          return code
      }
    }
    if (err.status === 401) return 'Please log in again.'
    return `${err.status} ${err.message}`
  }
  if (err instanceof Error) return err.message
  return 'Request failed.'
}

export function useQrCodes() {
  const qrCodes = ref<QrCodeItem[]>([])

  const labelInput = ref('')
  const urlInput = ref('')
  const isCreating = ref(false)
  const errorMessage = ref<string | null>(null)
  const isLoading = ref(false)
  const updatingId = ref<string | null>(null)

  const hasQrCodes = computed(() => qrCodes.value.length > 0)

  const { userType, isAuthed } = useUser()

  async function hydrateQrDataUrls(items: { id: string; url: string }[]): Promise<Record<string, string>> {
    const out: Record<string, string> = {}
    await Promise.all(
      items.map(async (i) => {
        try {
          out[i.id] = await generateQrDataUrl(trackingUrlForQrId(i.id))
        } catch {
          out[i.id] = ''
        }
      }),
    )
    return out
  }

  async function loadQrCodes(): Promise<void> {
    if (!isAuthed.value) {
      qrCodes.value = []
      errorMessage.value = null
      isLoading.value = false
      return
    }

    errorMessage.value = null
    isLoading.value = true
    try {
      const items = await qrCodesApi.list()
      const qrById = await hydrateQrDataUrls(items.map((i) => ({ id: i.id, url: i.url })))
      qrCodes.value = items.map((i) => ({
        id: i.id,
        label: i.label,
        url: i.url,
        active: i.active,
        createdAtIso: i.createdAtIso,
        qrDataUrl: qrById[i.id] || '',
      }))
    } catch {
      errorMessage.value = 'Failed to load QR codes from the server.'
    } finally {
      isLoading.value = false
    }
  }

  watchEffect(() => {
    if (!isAuthed.value) {
      qrCodes.value = []
      return
    }
    void loadQrCodes()
  })

  async function createQrCode(): Promise<void> {
    if (!isAuthed.value) {
      errorMessage.value = 'Please log in to create QR codes.'
      return
    }

    errorMessage.value = null

    const label = labelInput.value.trim() || 'Untitled'
    const validation = validateTargetUrl(urlInput.value)
    if (!validation.ok) {
      errorMessage.value = validation.message
      return
    }
    const url = validation.url

    isCreating.value = true
    try {
      const created = await qrCodesApi.create({ label, url, active: true }, userType.value)
      const qrDataUrl = await generateQrDataUrl(trackingUrlForQrId(created.id))
      const item: QrCodeItem = {
        id: created.id,
        label: created.label,
        url: created.url,
        active: created.active,
        createdAtIso: created.createdAtIso,
        qrDataUrl,
      }

      qrCodes.value = [item, ...qrCodes.value.filter((q) => q.id !== item.id)]
      labelInput.value = ''
      urlInput.value = ''
    } catch (err) {
      errorMessage.value = qrCodesErrorMessage(err)
    } finally {
      isCreating.value = false
    }
  }

  async function deleteQrCode(id: string): Promise<void> {
    if (!isAuthed.value) return
    errorMessage.value = null
    try {
      await qrCodesApi.delete(id, userType.value)
      qrCodes.value = qrCodes.value.filter((q) => q.id !== id)
    } catch (err) {
      errorMessage.value = qrCodesErrorMessage(err)
    }
  }

  async function updateQrCode(id: string, input: { label: string }): Promise<void> {
    if (!isAuthed.value) return
    errorMessage.value = null
    const label = input.label.trim()

    const current = qrCodes.value.find((q) => q.id === id)
    if (!current) return

    const patch: { label?: string } = {}
    if (label !== current.label) patch.label = label

    // No-op
    if (!patch.label) return

    updatingId.value = id
    try {
      const updated = await qrCodesApi.update(id, patch, userType.value)
      const nextQrDataUrl = current.qrDataUrl

      qrCodes.value = qrCodes.value.map((q) =>
        q.id === id
          ? {
              ...q,
              label: updated.label,
              url: q.url,
              active: updated.active,
              createdAtIso: updated.createdAtIso,
              qrDataUrl: nextQrDataUrl,
            }
          : q,
      )
    } catch (err) {
      errorMessage.value = qrCodesErrorMessage(err)
    } finally {
      updatingId.value = null
    }
  }

  async function setQrCodeActive(id: string, active: boolean): Promise<void> {
    if (!isAuthed.value) return
    errorMessage.value = null

    const current = qrCodes.value.find((q) => q.id === id)
    if (!current) return
    if (current.active === active) return

    updatingId.value = id
    try {
      const updated = await qrCodesApi.update(id, { active }, userType.value)
      qrCodes.value = qrCodes.value.map((q) => (q.id === id ? { ...q, active: updated.active } : q))
    } catch (err) {
      errorMessage.value = qrCodesErrorMessage(err)
    } finally {
      updatingId.value = null
    }
  }

  async function copyToClipboard(text: string): Promise<void> {
    try {
      await navigator.clipboard.writeText(text)
    } catch {
      // ignore
    }
  }

  function downloadQrCode(qrCode: QrCodeItem): void {
    const link = document.createElement('a')
    link.href = qrCode.qrDataUrl
    const safeLabel = qrCode.label.trim().replace(/[^a-z0-9_-]+/gi, '_')
    link.download = `${safeLabel || 'qr'}_${qrCode.id}.png`
    link.click()
  }

  return {
    qrCodes,
    hasQrCodes,
    labelInput,
    urlInput,
    isCreating,
    isLoading,
    updatingId,
    errorMessage,
    loadQrCodes,
    createQrCode,
    updateQrCode,
    setQrCodeActive,
    deleteQrCode,
    copyToClipboard,
    downloadQrCode,
  }
}
