<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { usersApi } from '../../api'
import { authErrorMessage } from '../../lib/authErrors'
import AppButton from '../../components/ui/AppButton.vue'

const router = useRouter()

const email = ref('')
const password = ref('')
const agreedToTerms = ref(false)

const busy = ref(false)
const errorMessage = ref<string | null>(null)
const statusMessage = ref<string | null>(null)

async function submit() {
  errorMessage.value = null
  statusMessage.value = null

  const e = email.value.trim().toLowerCase()
  const p = password.value.trim()
  if (!e || !p) {
    errorMessage.value = 'Email and password are required.'
    return
  }

  if (!agreedToTerms.value) {
    errorMessage.value = 'You must agree to the Terms of Service to continue.'
    return
  }

  busy.value = true
  try {
    await usersApi.register({ email: e, password: p })
    statusMessage.value = 'Account created. Check your email for a confirmation code.'
    await router.push({ name: 'confirm', query: { email: e } })
  } catch (err) {
    errorMessage.value = authErrorMessage(err)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <main class="page">
    <header class="header">
      <h1 class="title">Create account</h1>
      <p class="subtitle">You’ll need to confirm your email before logging in.</p>
    </header>

    <section class="card">
      <h2 class="sectionTitle">Register</h2>

      <form class="form" @submit.prevent="submit">
        <label class="field">
          <span class="label">Email</span>
          <input v-model="email" class="input" type="email" autocomplete="email" />
        </label>

        <label class="field">
          <span class="label">Password</span>
          <input v-model="password" class="input" type="password" autocomplete="new-password" />
        </label>

        <label class="checkboxField">
          <input v-model="agreedToTerms" type="checkbox" class="checkbox" />
          <span class="checkboxLabel">
            I agree to the <RouterLink to="/terms" target="_blank" class="link">Terms of Service</RouterLink>
          </span>
        </label>

        <div class="actions">
          <AppButton type="submit" :disabled="busy || !agreedToTerms">{{ busy ? 'Creating…' : 'Create account' }}</AppButton>
        </div>
      </form>

      <p v-if="statusMessage" class="status">{{ statusMessage }}</p>
      <p v-if="errorMessage" class="error">{{ errorMessage }}</p>

      <div class="links">
        <RouterLink to="/confirm">Confirm email</RouterLink>
        <RouterLink to="/login">Login</RouterLink>
      </div>
    </section>
  </main>
</template>

<style scoped src="./AuthPage.scss" lang="scss"></style>
