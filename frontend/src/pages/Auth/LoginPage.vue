<script setup lang="ts">
import { ref, watchEffect } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { usersApi } from '../../api'
import { authErrorMessage } from '../../lib/authErrors'
import { useUser } from '../../composables/useUser'
import AppButton from '../../components/ui/AppButton.vue'

const route = useRoute()
const router = useRouter()
const { reload } = useUser()

const email = ref('')
const password = ref('')

const busy = ref(false)
const errorMessage = ref<string | null>(null)

watchEffect(() => {
  const q = route.query.email
  if (typeof q === 'string' && !email.value) email.value = q
})

async function submit() {
  errorMessage.value = null

  const e = email.value.trim().toLowerCase()
  const p = password.value.trim()
  if (!e || !p) {
    errorMessage.value = 'Email and password are required.'
    return
  }

  busy.value = true
  try {
    await usersApi.login({ email: e, password: p })
    await reload()

    const redirect = route.query.redirect
    const redirectPath = typeof redirect === 'string' ? redirect : ''
    const isSafeInternalPath = redirectPath.startsWith('/') && !redirectPath.startsWith('//')

    if (isSafeInternalPath && !redirectPath.startsWith('/login')) {
      await router.push(redirectPath)
    } else {
      await router.push({ name: 'home' })
    }
  } catch (err) {
    errorMessage.value = authErrorMessage(err)

    // If confirmation is required, push the user into the right flow.
    if (String(errorMessage.value).toLowerCase().includes('not confirmed')) {
      await router.push({ name: 'confirm', query: { email: e } })
    }
  } finally {
    busy.value = false
    password.value = ''
  }
}
</script>

<template>
  <main class="page">
    <header class="header">
      <h1 class="title">Login</h1>
      <p class="subtitle">Sign in to manage your QR codes.</p>
    </header>

    <section class="card">
      <h2 class="sectionTitle">Login</h2>

      <form class="form" @submit.prevent="submit">
        <label class="field">
          <span class="label">Email</span>
          <input v-model="email" class="input" type="email" autocomplete="email" />
        </label>

        <label class="field">
          <span class="label">Password</span>
          <input v-model="password" class="input" type="password" autocomplete="current-password" />
        </label>

        <div class="actions">
          <AppButton type="submit" :disabled="busy">{{ busy ? 'Signing in…' : 'Login' }}</AppButton>
        </div>
      </form>

      <p v-if="errorMessage" class="error">{{ errorMessage }}</p>

      <div class="links">
        <RouterLink to="/register">Create account</RouterLink>
        <RouterLink to="/confirm">Confirm email</RouterLink>
        <RouterLink to="/forgot-password">Forgot password</RouterLink>
      </div>
    </section>
  </main>
</template>

<style scoped src="./AuthPage.scss" lang="scss"></style>
