<script setup lang="ts">
import { computed, ref, watchEffect } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { qrCodesApi, requestJson } from '../../api'
import { useUser } from '../../composables/useUser'
import { trackingUrlForQrId } from '../../lib/tracking'

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

function dateIsoUTC(daysAgo: number): string {
  const d = new Date()
  d.setUTCDate(d.getUTCDate() - daysAgo)
  return d.toISOString().slice(0, 10)
}

async function fetchDailyClicksBatch(qrId: string, dayIsos: string[]): Promise<Record<string, DailyClicks>> {
  try {
    return await requestJson<Record<string, DailyClicks>>({
      method: 'GET',
      path: '/api/clicks/daily-batch',
      query: { qrId, days: dayIsos.join(',') },
    })
  } catch {
    return {}
  }
}

function hoursArray(dc: DailyClicks): number[] {
  return [
    dc.hour00,
    dc.hour01,
    dc.hour02,
    dc.hour03,
    dc.hour04,
    dc.hour05,
    dc.hour06,
    dc.hour07,
    dc.hour08,
    dc.hour09,
    dc.hour10,
    dc.hour11,
    dc.hour12,
    dc.hour13,
    dc.hour14,
    dc.hour15,
    dc.hour16,
    dc.hour17,
    dc.hour18,
    dc.hour19,
    dc.hour20,
    dc.hour21,
    dc.hour22,
    dc.hour23,
  ]
}

const router = useRouter()
const route = useRoute()

const id = computed(() => String(route.params.id ?? ''))

const { isAuthed, isLoaded, userType } = useUser()

const isFreeUser = computed(() => userType.value === 'free')

watchEffect(() => {
  if (!isLoaded.value) return
  if (isAuthed.value) return
  const redirect = route.fullPath || '/'
  void router.replace({ name: 'login', query: { redirect } })
})

const qrCode = ref<{ id: string; label: string; url: string; active: boolean } | null>(null)
const isLoading = ref(false)
const errorMessage = ref<string | null>(null)

const last7Days = computed(() => [dateIsoUTC(6), dateIsoUTC(5), dateIsoUTC(4), dateIsoUTC(3), dateIsoUTC(2), dateIsoUTC(1), dateIsoUTC(0)])
const selectedDayIso = ref<string>(dateIsoUTC(0))

const dailyByDay = ref<Record<string, DailyClicks | null>>({})

const last7Total = computed(() => {
  return last7Days.value.reduce((sum, day) => sum + (dailyByDay.value[day]?.total ?? 0), 0)
})

const selectedDaily = computed(() => dailyByDay.value[selectedDayIso.value] ?? null)

const hourlyRows = computed(() => {
  const d = selectedDaily.value
  if (!d) return []
  return hoursArray(d).map((count, hour) => ({ hour, count }))
})

// Sample data for free users
const sampleHourlyRows = [
  { hour: 0, count: 12 },
  { hour: 1, count: 8 },
  { hour: 2, count: 3 },
  { hour: 3, count: 5 },
  { hour: 4, count: 7 },
  { hour: 5, count: 15 },
  { hour: 6, count: 24 },
  { hour: 7, count: 38 },
  { hour: 8, count: 52 },
  { hour: 9, count: 67 },
  { hour: 10, count: 71 },
  { hour: 11, count: 68 },
  { hour: 12, count: 75 },
  { hour: 13, count: 82 },
  { hour: 14, count: 91 },
  { hour: 15, count: 88 },
  { hour: 16, count: 79 },
  { hour: 17, count: 84 },
  { hour: 18, count: 72 },
  { hour: 19, count: 61 },
  { hour: 20, count: 48 },
  { hour: 21, count: 35 },
  { hour: 22, count: 28 },
  { hour: 23, count: 19 },
]

const displayLast7Total = computed(() => (isFreeUser.value ? 1247 : last7Total.value))
const displaySelectedDayTotal = computed(() => (isFreeUser.value ? 342 : (selectedDaily.value?.total ?? 0)))
const displayHourlyRows = computed(() => (isFreeUser.value ? sampleHourlyRows : hourlyRows.value))

watchEffect(() => {
  if (!isAuthed.value) return
  if (!id.value) return
  if (isLoading.value) return // Prevent duplicate requests

  errorMessage.value = null
  isLoading.value = true
  void (async () => {
    try {
      const item = await qrCodesApi.getById(id.value, userType.value)
      qrCode.value = { id: item.id, label: item.label, url: item.url, active: item.active }

      // Fetch daily click buckets for the last 7 days using batch endpoint.
      const batchResult = await fetchDailyClicksBatch(id.value, last7Days.value)
      const next: Record<string, DailyClicks | null> = {}
      last7Days.value.forEach((dayIso) => {
        next[dayIso] = batchResult[dayIso] ?? null
      })
      dailyByDay.value = next

      // Ensure selected day is still in range.
      if (!last7Days.value.includes(selectedDayIso.value)) {
        selectedDayIso.value = last7Days.value[last7Days.value.length - 1] ?? dateIsoUTC(0)
      }
    } catch {
      errorMessage.value = 'Failed to load stats for this QR code.'
      qrCode.value = null
      dailyByDay.value = {}
    } finally {
      isLoading.value = false
    }
  })()
})
</script>

<template>
  <main class="page">
    <header class="header">
      <h1 class="title">QR Code Stats</h1>
      <p class="subtitle" v-if="qrCode">{{ qrCode.label }}</p>
      <p class="subtitle" v-else>Performance details for your QR code.</p>
    </header>

    <section class="card">
      <div class="topRow">
        <RouterLink class="link" to="/">← Back to QR codes</RouterLink>
        <div class="spacer" />
        <span v-if="qrCode" class="pill" :class="qrCode.active ? 'ok' : 'off'">{{ qrCode.active ? 'Active' : 'Inactive' }}</span>
      </div>

      <p v-if="errorMessage" class="error">{{ errorMessage }}</p>

      <div v-if="isLoading" class="muted">Loading…</div>

      <template v-else-if="qrCode">
        <div v-if="isFreeUser" class="upgradePrompt">
          <h3 class="upgradeTitle">📊 Unlock Detailed Analytics</h3>
          <p class="upgradeText">Upgrade to view hourly breakdowns, tracking URLs, and detailed click statistics.</p>
          <RouterLink to="/subscription" class="upgradeButton">Upgrade Now</RouterLink>
        </div>

        <div class="kv" :class="{ blurred: isFreeUser }">
          <div class="kvRow">
            <span class="kvKey">Target</span>
            <a class="link" :href="qrCode.url" target="_blank" rel="noreferrer">{{ qrCode.url }}</a>
          </div>
          <div class="kvRow">
            <span class="kvKey">Tracking</span>
            <span class="mono">{{ trackingUrlForQrId(qrCode.id) }}</span>
          </div>
        </div>

        <div class="statsGrid" :class="{ blurred: isFreeUser }">
          <div class="statCard">
            <div class="statLabel">Clicks (last 7 days)</div>
            <div class="statValue">{{ displayLast7Total }}</div>
          </div>

          <div class="statCard">
            <div class="statLabel">Day</div>
            <select v-model="selectedDayIso" class="select" aria-label="Select day">
              <option v-for="d in last7Days" :key="d" :value="d">{{ d }}</option>
            </select>
          </div>

          <div class="statCard">
            <div class="statLabel">Clicks (selected day)</div>
            <div class="statValue">{{ displaySelectedDayTotal }}</div>
          </div>
        </div>

        <h2 class="sectionTitle">Hourly breakdown</h2>
        <p class="muted" v-if="!selectedDaily && !isFreeUser">No click data available for this day.</p>

        <table v-if="selectedDaily || isFreeUser" class="table" :class="{ blurred: isFreeUser }">
          <thead>
            <tr>
              <th>Hour (UTC)</th>
              <th>Clicks</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in displayHourlyRows" :key="row.hour">
              <td class="mono">{{ String(row.hour).padStart(2, '0') }}:00</td>
              <td class="mono">{{ row.count }}</td>
            </tr>
          </tbody>
        </table>
      </template>

      <template v-else>
        <p class="muted">No QR code found.</p>
      </template>
    </section>
  </main>
</template>

<style scoped src="../HomePage/HomePage.scss" lang="scss"></style>
<style scoped src="./QrCodeStatsPage.scss" lang="scss"></style>
