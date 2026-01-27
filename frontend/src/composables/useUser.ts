import { computed, onMounted, onUnmounted, ref } from 'vue'
import { usersApi } from '../api'
import type { User } from '../api'
import { AUTH_CHANGED_EVENT } from '../lib/authEvents'

const currentUser = ref<User | null>(null)
const isLoading = ref(false)
const isLoaded = ref(false)
const errorMessage = ref<string | null>(null)

async function loadCurrentUserOnce(): Promise<void> {
  if (isLoaded.value || isLoading.value) return
  isLoading.value = true
  errorMessage.value = null
  try {
    // If not authenticated, backend returns 401; treat as anonymous.
    const user = await usersApi.me().catch(() => null)
    currentUser.value = user
  } finally {
    isLoaded.value = true
    isLoading.value = false
  }
}

async function reloadCurrentUser(): Promise<void> {
  isLoaded.value = false
  await loadCurrentUserOnce()
}

export function useUser() {
  onMounted(() => {
    void loadCurrentUserOnce()
  })

  // Keep auth state fresh across the app.
  // - auth-changed: emitted after login/logout and on unexpected 401s
  // - focus/visibility: refresh when the tab becomes active
  const lastRefreshAt = ref<number>(0)

  function maybeRefresh(): void {
    const now = Date.now()
    if (isLoading.value) return
    if (!isLoaded.value) return
    if (now - lastRefreshAt.value < 5_000) return
    lastRefreshAt.value = now
    void reloadCurrentUser()
  }

  function onAuthChanged(): void {
    maybeRefresh()
  }

  function onVisibilityChange(): void {
    if (document.visibilityState === 'visible') maybeRefresh()
  }

  onMounted(() => {
    window.addEventListener(AUTH_CHANGED_EVENT, onAuthChanged)
    window.addEventListener('focus', maybeRefresh)
    document.addEventListener('visibilitychange', onVisibilityChange)
  })

  onUnmounted(() => {
    window.removeEventListener(AUTH_CHANGED_EVENT, onAuthChanged)
    window.removeEventListener('focus', maybeRefresh)
    document.removeEventListener('visibilitychange', onVisibilityChange)
  })

  const isAuthed = computed(() => Boolean(currentUser.value?.email))

  const userType = computed<"free" | "basic" | "enterprise" | "admin">(() => {
    const t = currentUser.value?.userType?.toLowerCase()
    if (t === 'basic' || t === 'enterprise' || t === 'admin') return t
    return 'free'
  })

  const entitlements = computed<string[]>(() => {
    const raw = currentUser.value?.entitlements
    if (!raw || raw.trim() === '') return []
    return raw.split('|').map(e => e.trim()).filter(e => e !== '')
  })

  function hasEntitlement(entitlement: string): boolean {
    return entitlements.value.includes(entitlement)
  }

  const isAdmin = computed(() => hasEntitlement('admin'))

  return {
    user: currentUser,
    isLoading,
    isLoaded,
    errorMessage,
    isAuthed,
    userType,
    entitlements,
    hasEntitlement,
    isAdmin,
    reload: reloadCurrentUser,
  }
}
