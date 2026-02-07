<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { usersApi } from '../../api'
import { createPortalSession } from '../../api/stripe/stripe.api'
import { authErrorMessage } from '../../lib/authErrors'
import { useUser } from '../../composables/useUser'

const router = useRouter()
const { user, userType, reload } = useUser()

const isAuthed = computed(() => Boolean(user.value?.email))

const busyLogout = ref(false)
const busyManageSubscription = ref(false)
const busyRefresh = ref(false)
const errorMessage = ref<string | null>(null)

const changePasswordDialog = ref<HTMLDialogElement | null>(null)
const oldPassword = ref('')
const newPassword = ref('')
const busyChangePassword = ref(false)
const changePasswordError = ref<string | null>(null)
const changePasswordStatus = ref<string | null>(null)

function openChangePassword() {
  changePasswordError.value = null
  changePasswordStatus.value = null
  oldPassword.value = ''
  newPassword.value = ''
  changePasswordDialog.value?.showModal()
}

function closeChangePassword() {
  changePasswordDialog.value?.close()
}

async function submitChangePassword() {
  changePasswordError.value = null
  changePasswordStatus.value = null

  if (!isAuthed.value) {
    await router.push({ name: 'login' })
    return
  }

  const oldPwd = oldPassword.value.trim()
  const newPwd = newPassword.value.trim()
  if (!oldPwd || !newPwd) {
    changePasswordError.value = 'Old and new passwords are required.'
    return
  }

  busyChangePassword.value = true
  try {
    await usersApi.changePassword({ oldPassword: oldPwd, newPassword: newPwd })
    await reload()
    changePasswordStatus.value = 'Password changed.'
    oldPassword.value = ''
    newPassword.value = ''
  } catch (err) {
    changePasswordError.value = authErrorMessage(err)
  } finally {
    busyChangePassword.value = false
  }
}

async function refreshAccount() {
  errorMessage.value = null
  busyRefresh.value = true
  try {
    await reload()
  } catch (err) {
    errorMessage.value = authErrorMessage(err)
  } finally {
    busyRefresh.value = false
  }
}

async function logout() {
  errorMessage.value = null
  busyLogout.value = true
  try {
    await usersApi.logout()
    await reload()
    await router.push({ name: 'home' })
  } catch (err) {
    errorMessage.value = authErrorMessage(err)
  } finally {
    busyLogout.value = false
  }
}

async function manageSubscription() {
  errorMessage.value = null
  busyManageSubscription.value = true
  try {
    const response = await createPortalSession()
    window.location.href = response.url
  } catch (err) {
    errorMessage.value = 'Failed to open subscription management. Please try again.'
    busyManageSubscription.value = false
  }
}
</script>

<template>
  <main class="page">
    <header class="header">
      <h1 class="title">Account</h1>
      <p class="subtitle">Manage your account settings.</p>
    </header>

    <section class="card">
      <p v-if="!isAuthed" class="muted">You are not logged in.</p>

      <div v-if="isAuthed" class="kv">
        <div class="kvRow" v-if="user?.email">
          <span class="kvKey">Email</span>
          <span class="kvVal">{{ user?.email }}</span>
        </div>
        <div class="kvRow" v-if="userType">
          <span class="kvKey">Plan</span>
          <span class="kvVal">{{ userType === 'free' ? 'Free' : userType === 'basic' ? 'Basic' : userType === 'enterprise' ? 'Enterprise' : 'Admin' }}</span>
        </div>
      </div>

      <div class="links">
        <RouterLink v-if="!isAuthed" to="/login">Login</RouterLink>
        <RouterLink v-if="!isAuthed" to="/register">Create account</RouterLink>
        <RouterLink v-if="!isAuthed" to="/confirm">Confirm account</RouterLink>
        <RouterLink v-if="!isAuthed" to="/forgot-password">Forgot password</RouterLink>
        <RouterLink v-if="!isAuthed" to="/reset-password">Reset password</RouterLink>
        <RouterLink v-if="isAuthed" to="/subscription">View plans</RouterLink>
      </div>

      <div v-if="isAuthed" class="divider" />

      <div v-if="isAuthed">
        <h2 class="sectionTitle" style="margin-top: 0">Subscription</h2>
        <p class="muted">Manage your subscription, payment methods, and billing history.</p>
        
        <div class="actions">
          <button class="button" type="button" :disabled="busyManageSubscription" @click="manageSubscription">
            {{ busyManageSubscription ? 'Opening…' : 'Manage Subscription' }}
          </button>
        </div>
      </div>

      <div v-if="isAuthed" class="divider" />

      <div v-if="isAuthed">
        <h2 class="sectionTitle" style="margin-top: 0">Support</h2>
        <p class="muted">
          Need help? Contact us at 
          <a href="mailto:support@qr-dragonfly.com" class="link">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="vertical-align: middle; margin-right: 2px;">
              <rect x="2" y="4" width="20" height="16" rx="2"/>
              <path d="m2 7 10 7 10-7"/>
            </svg>
            support@qr-dragonfly.com
          </a>
        </p>
      </div>

      <div v-if="isAuthed" class="divider" />

      <div v-if="isAuthed">
        <h2 class="sectionTitle" style="margin-top: 0">Security</h2>

        <div class="actions">
          <button class="button" type="button" @click="openChangePassword">Change password</button>
          <button class="button secondary" type="button" :disabled="busyRefresh" @click="refreshAccount">
            {{ busyRefresh ? 'Refreshing…' : 'Refresh' }}
          </button>
        </div>

        <div class="actions" style="margin-top: 12px">
          <button class="button" type="button" :disabled="busyLogout" @click="logout">
            {{ busyLogout ? 'Logging out…' : 'Logout' }}
          </button>
        </div>
      </div>

      <div v-if="!isAuthed" class="actions" style="margin-top: 12px">
        <RouterLink class="button" to="/login">Go to login</RouterLink>
      </div>

      <p v-if="errorMessage" class="error">{{ errorMessage }}</p>
    </section>

    <dialog ref="changePasswordDialog" class="dialog" @cancel.prevent="closeChangePassword">
      <div class="dialogInner">
        <div class="dialogHeader">
          <h2 class="dialogTitle">Change password</h2>
          <button class="iconButton" type="button" aria-label="Close" @click="closeChangePassword">×</button>
        </div>

        <form class="form" @submit.prevent="submitChangePassword">
          <label class="field">
            <span class="label">Old password</span>
            <input v-model="oldPassword" class="input" type="password" autocomplete="current-password" />
          </label>

          <label class="field">
            <span class="label">New password</span>
            <input v-model="newPassword" class="input" type="password" autocomplete="new-password" />
          </label>

          <div class="actions">
            <button class="button" type="submit" :disabled="busyChangePassword">
              {{ busyChangePassword ? 'Updating…' : 'Update password' }}
            </button>
            <button class="button secondary" type="button" :disabled="busyChangePassword" @click="closeChangePassword">
              Close
            </button>
          </div>
        </form>

        <p v-if="changePasswordStatus" class="status">{{ changePasswordStatus }}</p>
        <p v-if="changePasswordError" class="error">{{ changePasswordError }}</p>
      </div>
    </dialog>
  </main>
</template>

<style scoped src="./AuthPage.scss" lang="scss"></style>
<style scoped src="./AccountPage.scss" lang="scss"></style>
