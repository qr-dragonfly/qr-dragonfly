<script setup lang="ts">
import { computed, ref, watchEffect } from 'vue'
import { requestJson } from '../../api'
import type { QrCodeItem } from '../../types/qrCodeItem'
import { generateQrDataUrl } from '../../lib/qr'
import { trackingUrlForQrId } from '../../lib/tracking'

const CLICK_API_BASE_URL = (import.meta as { env?: Record<string, string> }).env?.VITE_CLICK_API_BASE_URL || ''

type DailyClicks = {
  qrCodeId: string
  dayIso: string
  total: number
  hour00: number
  hour01: number
  hour02: number
  hour03: number
  hour04: number
  hour05: number
  hour06: number
  hour07: number
  hour08: number
  hour09: number
  hour10: number
  hour11: number
  hour12: number
  hour13: number
  hour14: number
  hour15: number
  hour16: number
  hour17: number
  hour18: number
  hour19: number
  hour20: number
  hour21: number
  hour22: number
  hour23: number
}

type Props = {
  qrCodes: QrCodeItem[]
  updatingId: string | null
  errorMessage: string | null
  showSampleWhenEmpty?: boolean
}

const props = defineProps<Props>()

const isShowingSamples = computed(() => props.qrCodes.length === 0 && Boolean(props.showSampleWhenEmpty))

const sampleQrCodes: QrCodeItem[] = [
  {
    id: 'sample-1',
    label: 'Example menu',
    url: 'https://example.com/menu',
    active: true,
    createdAtIso: new Date().toISOString(),
    qrDataUrl: '',
  },
  {
    id: 'sample-2',
    label: 'Product page',
    url: 'https://example.com/product?utm_source=qr',
    active: true,
    createdAtIso: new Date().toISOString(),
    qrDataUrl: '',
  },
  {
    id: 'sample-3',
    label: 'Support link',
    url: 'https://example.com/support',
    active: false,
    createdAtIso: new Date().toISOString(),
    qrDataUrl: '',
  },
]

const rows = computed(() => {
  if (props.qrCodes.length > 0) return props.qrCodes
  if (isShowingSamples.value) return sampleQrCodes
  return []
})

const searchText = ref('')

const filteredRows = computed(() => {
  const q = searchText.value.trim().toLowerCase()
  if (!q) return rows.value
  return rows.value.filter((r) => {
    const label = (r.label ?? '').toLowerCase()
    const url = (r.url ?? '').toLowerCase()
    return label.includes(q) || url.includes(q)
  })
})

const PAGE_SIZE_KEY = 'qrCodesTable.pageSize'
const SEARCH_TEXT_KEY = 'qrCodesTable.searchText'
const PAGE_KEY = 'qrCodesTable.page'
const pageSize = ref<number>(10)
const page = ref<number>(1)

function loadSearchText(): string {
  try {
    const raw = window.localStorage.getItem(SEARCH_TEXT_KEY)
    return typeof raw === 'string' ? raw : ''
  } catch {
    return ''
  }
}

function saveSearchText(value: string) {
  try {
    window.localStorage.setItem(SEARCH_TEXT_KEY, value)
  } catch {
    // ignore
  }
}

function loadPage(): number {
  try {
    const raw = window.localStorage.getItem(PAGE_KEY)
    const n = raw ? Number(raw) : 1
    return Number.isFinite(n) && n >= 1 ? Math.floor(n) : 1
  } catch {
    return 1
  }
}

function savePage(n: number) {
  try {
    window.localStorage.setItem(PAGE_KEY, String(n))
  } catch {
    // ignore
  }
}

function loadPageSize(): number {
  try {
    const raw = window.localStorage.getItem(PAGE_SIZE_KEY)
    const n = raw ? Number(raw) : 10
    return n === 10 || n === 25 || n === 50 ? n : 10
  } catch {
    return 10
  }
}

function savePageSize(n: number) {
  try {
    window.localStorage.setItem(PAGE_SIZE_KEY, String(n))
  } catch {
    // ignore
  }
}

watchEffect(() => {
  // initialize once in the browser
  if (pageSize.value !== 10) return
  pageSize.value = loadPageSize()
})

watchEffect(() => {
  // initialize once in the browser
  if (searchText.value !== '') return
  searchText.value = loadSearchText()
})

watchEffect(() => {
  // initialize once in the browser
  if (page.value !== 1) return
  page.value = loadPage()
})

watchEffect(() => {
  savePageSize(pageSize.value)
})

watchEffect(() => {
  saveSearchText(searchText.value)
})

watchEffect(() => {
  savePage(page.value)
})

const totalRows = computed(() => filteredRows.value.length)
const totalPages = computed(() => Math.max(1, Math.ceil(totalRows.value / pageSize.value)))

watchEffect(() => {
  if (page.value > totalPages.value) page.value = totalPages.value
  if (page.value < 1) page.value = 1
})

const pageStartIndex = computed(() => (page.value - 1) * pageSize.value)
const pageEndIndexExclusive = computed(() => Math.min(totalRows.value, pageStartIndex.value + pageSize.value))
const pagedRows = computed(() => filteredRows.value.slice(pageStartIndex.value, pageEndIndexExclusive.value))

const canPrev = computed(() => page.value > 1)
const canNext = computed(() => page.value < totalPages.value)

function prevPage() {
  if (!canPrev.value) return
  page.value -= 1
}

function nextPage() {
  if (!canNext.value) return
  page.value += 1
}

watchEffect(() => {
  // Reset to page 1 on search changes.
  void searchText.value
  page.value = 1
})

const showError = computed(() => Boolean(props.errorMessage) && !isShowingSamples.value)

function isSampleId(id: string): boolean {
  return id.startsWith('sample-')
}

const sampleTrendPathById: Record<string, string> = {
  'sample-1': sparklinePath([1, 2, 3, 4, 6, 8, 11]),
  'sample-2': sparklinePath([0, 1, 1, 2, 3, 5, 7]),
  'sample-3': sparklinePath([2, 2, 3, 3, 4, 5, 6]),
}

function sampleTrendPath(id: string): string {
  return sampleTrendPathById[id] ?? sparklinePath([1, 2, 3, 4, 5, 6, 7])
}

const sampleQrDataUrlById = ref<Record<string, string>>({})

function qrImageSrc(qrCode: QrCodeItem): string {
  if (qrCode.qrDataUrl) return qrCode.qrDataUrl
  if (isSampleId(qrCode.id)) return sampleQrDataUrlById.value[qrCode.id] ?? ''
  return ''
}

function dateIsoUTC(daysAgo: number): string {
  const d = new Date()
  d.setUTCDate(d.getUTCDate() - daysAgo)
  return d.toISOString().slice(0, 10)
}

const todayIso = dateIsoUTC(0)

const last7DaysIso = computed(() => {
  // Oldest -> newest for left-to-right charts.
  return [dateIsoUTC(6), dateIsoUTC(5), dateIsoUTC(4), dateIsoUTC(3), dateIsoUTC(2), dateIsoUTC(1), dateIsoUTC(0)]
})

type TrendState = {
  dayIso: string
  counts: number[]
  total: number
  delta: number
  isUp: boolean
  pathD: string
}

const trendById = ref<Record<string, TrendState>>({})
const trendLoading = ref<Record<string, boolean>>({})

async function fetchDailyClicksBatch(qrId: string, dayIsos: string[]): Promise<Record<string, DailyClicks>> {
  try {
    return await requestJson<Record<string, DailyClicks>>({
      method: 'GET',
      path: '/api/clicks/daily-batch',
      query: { qrId, days: dayIsos.join(',') },
      baseUrl: CLICK_API_BASE_URL,
    })
  } catch {
    return {}
  }
}

function sparklinePath(counts: number[], width = 96, height = 24, pad = 2): string {
  const n = counts.length
  if (n === 0) return ''

  const first = counts[0] ?? 0
  let min = first
  let max = first
  for (const v of counts) {
    if (v < min) min = v
    if (v > max) max = v
  }

  const usableW = Math.max(1, width - pad * 2)
  const usableH = Math.max(1, height - pad * 2)
  const dx = n === 1 ? 0 : usableW / (n - 1)

  const yFor = (v: number) => {
    if (max === min) return pad + usableH / 2
    const t = (v - min) / (max - min)
    return pad + (1 - t) * usableH
  }

  let d = ''
  for (let i = 0; i < n; i++) {
    const v = counts[i] ?? 0
    const x = pad + i * dx
    const y = yFor(v)
    d += i === 0 ? `M ${x.toFixed(2)} ${y.toFixed(2)}` : ` L ${x.toFixed(2)} ${y.toFixed(2)}`
  }
  return d
}

function trendTotal(qrId: string): number {
  return trendById.value[qrId]?.total ?? 0
}

function trendPath(qrId: string): string {
  return trendById.value[qrId]?.pathD ?? ''
}

function trendIsUp(qrId: string): boolean {
  return trendById.value[qrId]?.isUp ?? true
}

async function loadTrend(qrId: string) {
  if (trendLoading.value[qrId]) return
  if (trendById.value[qrId]?.dayIso === todayIso) return
  trendLoading.value = { ...trendLoading.value, [qrId]: true }

  try {
    const days = last7DaysIso.value
    const batch = await fetchDailyClicksBatch(qrId, days)
    const counts = days.map((d) => batch[d]?.total ?? 0)
    const total = counts.reduce((a, b) => a + b, 0)
    const delta = counts.length >= 2 ? (counts[counts.length - 1] ?? 0) - (counts[counts.length - 2] ?? 0) : 0
    const isUp = delta >= 0

    trendById.value = {
      ...trendById.value,
      [qrId]: {
        dayIso: todayIso,
        counts,
        total,
        delta,
        isUp,
        pathD: sparklinePath(counts),
      },
    }
  } finally {
    trendLoading.value = { ...trendLoading.value, [qrId]: false }
  }
}

watchEffect(() => {
  for (const row of pagedRows.value) {
    if (isSampleId(row.id)) continue
    void loadTrend(row.id)
  }
})

watchEffect(() => {
  if (!isShowingSamples.value) return
  for (const row of sampleQrCodes) {
    if (sampleQrDataUrlById.value[row.id]) continue
    void (async () => {
      try {
        const dataUrl = await generateQrDataUrl(row.url)
        sampleQrDataUrlById.value = { ...sampleQrDataUrlById.value, [row.id]: dataUrl }
      } catch {
        // ignore
      }
    })()
  }
})

const emit = defineEmits<{
  (e: 'copy-url', url: string): void
  (e: 'download', qrCode: QrCodeItem): void
  (e: 'remove', id: string): void
  (e: 'update', id: string, input: { label: string }): void
  (e: 'set-active', id: string, active: boolean): void
}>()

const editingId = ref<string | null>(null)
const editLabel = ref('')

function startEdit(qrCode: QrCodeItem) {
  if (isSampleId(qrCode.id)) return
  editingId.value = qrCode.id
  editLabel.value = qrCode.label
}

function cancelEdit() {
  editingId.value = null
  editLabel.value = ''
}

function saveEdit() {
  const id = editingId.value
  if (!id) return
  if (isSampleId(id)) return
  emit('update', id, { label: editLabel.value })
}

function isEditing(tabId: string): boolean {
  return editingId.value === tabId
}

function isBusy(tabId: string): boolean {
  return props.updatingId === tabId
}

const viewDialog = ref<HTMLDialogElement | null>(null)
const viewingQrCode = ref<QrCodeItem | null>(null)
const viewingOverrideQrSrc = ref('')
const viewingIsGenerating = ref(false)

const viewingQrSrc = computed(() => {
  if (viewingOverrideQrSrc.value) return viewingOverrideQrSrc.value
  if (!viewingQrCode.value) return ''
  return qrImageSrc(viewingQrCode.value)
})

async function openViewCode(qrCode: QrCodeItem) {
  viewingQrCode.value = qrCode
  viewingOverrideQrSrc.value = ''
  viewingIsGenerating.value = false
  viewDialog.value?.showModal()

  // If this row doesn't have a hydrated QR image yet, generate one on-demand.
  if (qrImageSrc(qrCode)) return
  const id = qrCode.id
  viewingIsGenerating.value = true
  try {
    const dataUrl = await generateQrDataUrl(qrCode.url)
    if (viewingQrCode.value?.id === id) {
      viewingOverrideQrSrc.value = dataUrl
    }
  } catch {
    // ignore
  } finally {
    if (viewingQrCode.value?.id === id) {
      viewingIsGenerating.value = false
    }
  }
}

function closeViewCode() {
  viewDialog.value?.close()
  viewingQrCode.value = null
  viewingOverrideQrSrc.value = ''
  viewingIsGenerating.value = false
}

// Disable confirmation modal
const disableDialog = ref<HTMLDialogElement | null>(null)
const pendingDisableId = ref<string | null>(null)
const pendingDisableLabel = ref<string>('')

function openDisableDialog(qrCode: QrCodeItem) {
  if (isSampleId(qrCode.id)) return
  pendingDisableId.value = qrCode.id
  pendingDisableLabel.value = qrCode.label
  disableDialog.value?.showModal()
}

function closeDisableDialog() {
  disableDialog.value?.close()
  pendingDisableId.value = null
  pendingDisableLabel.value = ''
}

function confirmDisable() {
  const id = pendingDisableId.value
  if (!id) return
  emit('set-active', id, false)
  closeDisableDialog()
}

// Remove confirmation modal
const removeDialog = ref<HTMLDialogElement | null>(null)
const pendingRemoveId = ref<string | null>(null)
const pendingRemoveLabel = ref<string>('')

function openRemoveDialog(qrCode: QrCodeItem) {
  if (isSampleId(qrCode.id)) return
  pendingRemoveId.value = qrCode.id
  pendingRemoveLabel.value = qrCode.label
  removeDialog.value?.showModal()
}

function closeRemoveDialog() {
  removeDialog.value?.close()
  pendingRemoveId.value = null
  pendingRemoveLabel.value = ''
}

function confirmRemove() {
  const id = pendingRemoveId.value
  if (!id) return
  emit('remove', id)
  closeRemoveDialog()
}
</script>

<template>
  <section class="card">
    <div class="tableHeader">
      <h2 class="sectionTitle">QR codes</h2>
      <div class="meta">{{ totalRows }} total</div>
    </div>

    <p v-if="showError" class="error">{{ errorMessage }}</p>

    <div v-if="rows.length === 0" class="empty">No QR codes yet. Create one above.</div>

    <div v-else class="tableWrap">
      <div v-if="isShowingSamples" class="sampleNote">
        Sample rows shown while signed out.
      </div>

      <div class="pager">
        <div class="pagerLeft">
          <span class="pagerText">Showing {{ pageStartIndex + 1 }}–{{ pageEndIndexExclusive }} of {{ totalRows }}</span>
        </div>
        <div class="pagerRight">
          <input
            v-model="searchText"
            class="pagerSearch"
            type="search"
            placeholder="Search label or URL"
            aria-label="Search QR codes"
          />

          <label class="pagerText">
            Per page
            <select v-model.number="pageSize" class="pagerSelect" aria-label="Rows per page">
              <option :value="10">10</option>
              <option :value="25">25</option>
              <option :value="50">50</option>
            </select>
          </label>

          <button class="buttonSmall" type="button" :disabled="!canPrev" @click="prevPage">Prev</button>
          <span class="pagerText">Page {{ page }} / {{ totalPages }}</span>
          <button class="buttonSmall" type="button" :disabled="!canNext" @click="nextPage">Next</button>
        </div>
      </div>

      <table class="table">
        <thead>
          <tr>
            <th>Label</th>
            <th>URL</th>
            <th>Active</th>
            <th>Trend</th>
            <th>QR</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="qrCode in pagedRows" :key="qrCode.id">
            <td class="mono">
              <template v-if="isEditing(qrCode.id)">
                <input v-model="editLabel" class="input" type="text" />
              </template>
              <template v-else>
                {{ qrCode.label }}
              </template>
            </td>
            <td class="urlCell">
              <template v-if="!isEditing(qrCode.id)">
                <span v-if="isSampleId(qrCode.id)" class="link sampleUrl" aria-label="Example URL">{{ qrCode.url }}</span>
                <span v-else class="linkWrap">
                  <a class="link" :href="qrCode.url" target="_blank" rel="noreferrer">{{ qrCode.url }}</a>
                  <button
                    class="iconButton"
                    type="button"
                    :title="'Copy app link'"
                    aria-label="Copy app link"
                    @click="emit('copy-url', trackingUrlForQrId(qrCode.id))"
                  >
                    ⧉
                  </button>
                </span>
              </template>
              <template v-else>
                <span class="link">{{ qrCode.url }}</span>
              </template>
            </td>
            <td>
              <template v-if="isSampleId(qrCode.id)">
                <span class="muted">{{ qrCode.active ? 'Active' : 'Inactive' }}</span>
              </template>
              <button
                v-else
                class="buttonSmall"
                type="button"
                :disabled="isBusy(qrCode.id)"
                @click="qrCode.active ? openDisableDialog(qrCode) : emit('set-active', qrCode.id, true)"
              >
                {{ qrCode.active ? 'Active' : 'Inactive' }}
              </button>
            </td>
            <td>
              <div v-if="isSampleId(qrCode.id)" class="sparkWrap" aria-hidden="true">
                <svg class="spark" viewBox="0 0 100 28">
                  <path :d="sampleTrendPath(qrCode.id)" class="sparkLine up" fill="none" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" />
                </svg>
              </div>
              <div v-else class="sparkWrap" :title="`Clicks last 7 days: ${trendTotal(qrCode.id)}`">
                <svg class="spark" viewBox="0 0 100 28" role="img" aria-label="Click trend (last 7 days)">
                  <path
                    v-if="trendPath(qrCode.id)"
                    :d="trendPath(qrCode.id)"
                    :class="trendIsUp(qrCode.id) ? 'sparkLine up' : 'sparkLine down'"
                    fill="none"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                  />
                  <path v-else d="M 2 14 L 98 14" class="sparkLine flat" fill="none" stroke-linecap="round" stroke-width="2" />
                </svg>
              </div>
            </td>
            <td>
              <img
                v-if="qrImageSrc(qrCode)"
                class="qr"
                :src="qrImageSrc(qrCode)"
                :alt="`QR for ${qrCode.url}`"
                role="button"
                :title="'View code'"
                @click="openViewCode(qrCode)"
              />
              <div
                v-else
                class="qrPlaceholder"
                role="button"
                aria-label="View code"
                @click="openViewCode(qrCode)"
              >
                QR
              </div>
            </td>
            <td class="actionsCell">
              <div class="rowActions">
                <template v-if="isSampleId(qrCode.id)">
                  <span class="muted">Sign in to manage</span>
                </template>
                <template v-else-if="isEditing(qrCode.id)">
                  <button class="buttonSmall" type="button" :disabled="isBusy(qrCode.id)" @click="saveEdit">
                    {{ isBusy(qrCode.id) ? 'Saving…' : 'Save' }}
                  </button>
                  <button class="buttonSmall" type="button" :disabled="isBusy(qrCode.id)" @click="cancelEdit">
                    Cancel
                  </button>
                </template>
                <template v-else>
                  <button class="buttonSmall" type="button" :disabled="isBusy(qrCode.id)" @click="startEdit(qrCode)">
                    Edit
                  </button>
                  <RouterLink class="buttonSmall" :to="{ name: 'qr-code-stats', params: { id: qrCode.id } }">Stats</RouterLink>
                  <button class="buttonSmall" type="button" :disabled="isBusy(qrCode.id)" @click="emit('download', qrCode)">
                    Download
                  </button>
                  <button class="buttonSmall danger" type="button" :disabled="isBusy(qrCode.id)" @click="openRemoveDialog(qrCode)">
                    Remove
                  </button>
                </template>
              </div>
            </td>
          </tr>
        </tbody>
      </table>

      <div v-if="rows.length > 0 && totalRows === 0" class="empty" style="margin-top: 12px">
        No matches. Try a different search.
      </div>
    </div>

    <dialog ref="viewDialog" class="dialog" @cancel.prevent="closeViewCode">
      <div class="dialogInner">
        <div class="dialogHeader">
          <div>
            <h2 class="dialogTitle">View code</h2>
            <div v-if="viewingQrCode" class="dialogMeta">{{ viewingQrCode.label }}</div>
          </div>
          <button class="iconButton" type="button" aria-label="Close" @click="closeViewCode">×</button>
        </div>

        <div class="qrModalBody">
          <img v-if="viewingQrSrc" class="qrLarge" :src="viewingQrSrc" :alt="viewingQrCode ? `QR for ${viewingQrCode.url}` : 'QR code'" />
          <div v-else class="qrLargePlaceholder">{{ viewingIsGenerating ? 'Generating…' : 'QR' }}</div>
        </div>

        <p v-if="viewingQrCode" class="dialogHint">
          Scan with your phone camera, or open:
          <span class="mono">{{ viewingQrCode.url }}</span>
        </p>
      </div>
    </dialog>

    <dialog ref="disableDialog" class="dialog" @cancel.prevent="closeDisableDialog">
      <div class="dialogInner">
        <div class="dialogHeader">
          <h2 class="dialogTitle">Disable code?</h2>
          <button class="iconButton" type="button" aria-label="Close" @click="closeDisableDialog">×</button>
        </div>

        <p class="dialogHint" v-if="pendingDisableLabel">{{ pendingDisableLabel }}</p>
        <p class="dialogHint">Users won’t be able to scan and open this code while disabled.</p>

        <div class="actions" style="margin-top: 12px">
          <button class="buttonSmall" type="button" @click="confirmDisable">Disable</button>
          <button class="buttonSmall" type="button" @click="closeDisableDialog">Cancel</button>
        </div>
      </div>
    </dialog>

    <dialog ref="removeDialog" class="dialog" @cancel.prevent="closeRemoveDialog">
      <div class="dialogInner">
        <div class="dialogHeader">
          <h2 class="dialogTitle">Remove code?</h2>
          <button class="iconButton" type="button" aria-label="Close" @click="closeRemoveDialog">×</button>
        </div>

        <p class="dialogHint" v-if="pendingRemoveLabel">{{ pendingRemoveLabel }}</p>
        <p class="dialogHint">This will permanently delete the QR code and its tracking link.</p>

        <div class="actions" style="margin-top: 12px">
          <button class="buttonSmall danger" type="button" @click="confirmRemove">Remove</button>
          <button class="buttonSmall" type="button" @click="closeRemoveDialog">Cancel</button>
        </div>
      </div>
    </dialog>
  </section>
</template>

<style scoped src="./QrCodesTable.scss" lang="scss"></style>
