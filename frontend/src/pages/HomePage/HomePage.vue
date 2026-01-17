<script setup lang="ts">
import { useQrCodes } from '../../composables/useQrCodes'
import { useUser } from '../../composables/useUser'
import CreateQrCodeForm from '../../components/CreateQrCodeForm/CreateQrCodeForm.vue'
import QrCodesTable from '../../components/QrCodesTable/QrCodesTable.vue'

const { isAuthed } = useUser()

const {
  qrCodes,
  labelInput,
  urlInput,
  isCreating,
  updatingId,
  errorMessage,
  createQrCode,
  updateQrCode,
  setQrCodeActive,
  copyToClipboard,
  downloadQrCode,
  deleteQrCode,
} = useQrCodes()
</script>

<template>
  <main class="page">
    <header class="header">
      <h1 class="title">QR Codes</h1>
      <p class="subtitle">Create QR codes for user-inputted URLs.</p>
    </header>

    <section v-if="!isAuthed" class="authPrompt">
      <p class="muted">Sign in to create and manage your own QR codes.</p>
      <div class="actions">
        <RouterLink class="button" to="/login">Login</RouterLink>
        <RouterLink class="button secondary" to="/register">Create account</RouterLink>
      </div>
    </section>

    <CreateQrCodeForm
      v-if="isAuthed"
      v-model:label="labelInput"
      v-model:url="urlInput"
      :isCreating="isCreating"
      :errorMessage="errorMessage"
      @submit="createQrCode"
    />

    <QrCodesTable
      :qrCodes="qrCodes"
      :updatingId="updatingId"
      :errorMessage="errorMessage"
      :showSampleWhenEmpty="!isAuthed"
      @copy-url="copyToClipboard"
      @download="downloadQrCode"
      @remove="deleteQrCode"
      @update="updateQrCode"
      @set-active="setQrCodeActive"
    />
  </main>
</template>

<style scoped src="./HomePage.scss" lang="scss"></style>
