<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { usersApi } from './api'
import { useUser } from './composables/useUser'

const route = useRoute()
const router = useRouter()
const { user } = useUser()
const { reload } = useUser()

const isAuthed = computed(() => Boolean(user.value?.email))
const year = new Date().getFullYear()
const busyLogout = ref(false)

async function logout() {
  if (busyLogout.value) return
  busyLogout.value = true
  try {
    await usersApi.logout()
  } finally {
    await reload()
    busyLogout.value = false
    await router.push({ name: 'home' })
  }
}
const isAuthRoute = computed(() => {
  const name = String(route.name ?? '')
  return (
    name === 'login' ||
    name === 'register' ||
    name === 'confirm' ||
    name === 'forgot-password' ||
    name === 'reset-password' ||
    name === 'account' ||
    name === 'change-password'
  )
})
</script>

<template>
  <div class="shell">
    <header class="header">
      <nav class="nav" aria-label="Primary">
        <RouterLink class="navLink brand" to="/">QR Codes</RouterLink>

        <div class="spacer" />

        <RouterLink v-if="!isAuthed" class="navLink" to="/register">Create account</RouterLink>
        <RouterLink v-if="!isAuthed" class="navLink" to="/login">Login</RouterLink>

        <template v-if="isAuthed">
          <span class="navUser" aria-label="Signed in user">{{ user?.email }}</span>
          <RouterLink class="navLink" to="/subscription">Subscription</RouterLink>
          <RouterLink class="navLink" to="/account">Account</RouterLink>
          <button class="navLink navButton" type="button" :disabled="busyLogout" @click="logout">
            {{ busyLogout ? 'Logging out…' : 'Logout' }}
          </button>
        </template>
      </nav>
    </header>

    <main class="content" :class="{ auth: isAuthRoute }">
      <RouterView />
    </main>

    <footer class="footer" aria-label="Footer">
      <div class="footerInner">
        <div class="footerLeft">
          <span class="footerText">© {{ year }} QR-Dragonfly. Made with love in Minnesota.</span>
        </div>

        <nav class="footerNav" aria-label="Footer links">
          <RouterLink class="footerLink" to="/terms">Terms of Service</RouterLink>
          <span class="footerSep" aria-hidden="true">•</span>
          <RouterLink class="footerLink" to="/privacy">Privacy Policy</RouterLink>
          <span class="footerSep" aria-hidden="true">•</span>
          <RouterLink class="footerLink" to="/cookies">Cookie Policy</RouterLink>
        </nav>
      </div>
    </footer>
  </div>
</template>

<style scoped lang="scss">
.shell {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.header {
  border-bottom: 1px solid color-mix(in srgb, $color-fg 12%, transparent);
  background: $color-nav-bg;
}

.content {
  flex: 1;
}

.nav {
  max-width: 1100px;
  margin: 0 auto;
  padding: 18px 16px;
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
}

.spacer {
  flex: 1;
}

.navLink {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 6px 10px;
  border-radius: $radius-md;
  border: 1px solid color-mix(in srgb, $color-fg 12%, transparent);
  text-decoration: none;
  opacity: 0.95;
  background: color-mix(in srgb, $color-bg 30%, transparent);
}

.navButton {
  cursor: pointer;
  font: inherit;
  color: inherit;
}

.navButton:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.navUser {
  opacity: 0.85;
  font-weight: 600;
  max-width: 320px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.navLink:hover {
  border-color: color-mix(in srgb, $color-link 55%, transparent);
  text-decoration: none;
}

.navLink.router-link-active {
  opacity: 1;
  border-color: color-mix(in srgb, $color-link 70%, transparent);
}

.brand {
  font-weight: 700;
}

.footer {
  padding: 18px 16px 28px;
  border-top: 1px solid color-mix(in srgb, $color-fg 12%, transparent);
  background: $color-footer-bg;
}

.footerInner {
  max-width: 1100px;
  margin: 0 auto;
  display: flex;
  gap: 14px;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
}

.footerLeft {
  display: inline-flex;
  align-items: baseline;
  gap: 10px;
}

.footerText {
  opacity: 0.8;
}

.footerSep {
  opacity: 0.5;
}

.footerNav {
  display: inline-flex;
  gap: 12px;
  flex-wrap: wrap;
  align-items: center;
}

.footerLink {
  opacity: 0.85;
  text-decoration: none;
}

.footerLink:hover {
  opacity: 1;
}

@media (max-width: 768px) {
  .nav {
    justify-content: center;
    padding: 12px 12px;
  }

  .spacer {
    display: none;
  }

  .brand {
    flex-basis: 100%;
    text-align: center;
    margin-bottom: 4px;
  }

  .navUser {
    flex-basis: 100%;
    text-align: center;
    max-width: 100%;
    order: -1;
    margin-bottom: 4px;
    font-size: $font-size-sm;
  }

  .navLink {
    padding: 8px 12px;
    font-size: $font-size-sm;
  }

  .footer {
    padding: 16px 12px 24px;
  }

  .footerInner {
    flex-direction: column;
    text-align: center;
    gap: 12px;
  }

  .footerLeft {
    width: 100%;
    justify-content: center;
  }

  .footerText {
    font-size: $font-size-xs;
  }

  .footerNav {
    width: 100%;
    justify-content: center;
    font-size: $font-size-xs;
    gap: 8px;
  }

  .footerLink {
    padding: 4px 8px;
  }
}
</style>
