<script setup lang="ts">
import { ref } from 'vue'
import { useQrCodes } from '../../composables/useQrCodes'
import { useUser } from '../../composables/useUser'
import CreateQrCodeForm from '../../components/CreateQrCodeForm/CreateQrCodeForm.vue'
import DefaultRedirectSettings from '../../components/DefaultRedirectSettings/DefaultRedirectSettings.vue'
import QrCodesTable from '../../components/QrCodesTable/QrCodesTable.vue'
import FormatSelectorModal from '../../components/QrCodesTable/FormatSelectorModal.vue'
import type { QrCodeItem } from '../../types/qrCodeItem'
import type { QrFormat } from '../../lib/qr'

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
  downloadQrCodeInFormat,
  deleteQrCode,
} = useQrCodes()

const showFormatSelector = ref(false)
const qrCodeToDownload = ref<QrCodeItem | null>(null)

function openFormatSelector(qrCode: QrCodeItem) {
  qrCodeToDownload.value = qrCode
  showFormatSelector.value = true
}

function closeFormatSelector() {
  showFormatSelector.value = false
  qrCodeToDownload.value = null
}

async function handleDownload(format: QrFormat) {
  if (qrCodeToDownload.value) {
    await downloadQrCodeInFormat(qrCodeToDownload.value, format)
  }
  closeFormatSelector()
}
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

    <DefaultRedirectSettings v-if="isAuthed" />

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
      @download="openFormatSelector"
      @remove="deleteQrCode"
      @update="updateQrCode"
      @set-active="setQrCodeActive"
    />

    <FormatSelectorModal
      v-if="showFormatSelector"
      @close="closeFormatSelector"
      @download="handleDownload"
    />
  </main>
</template>

<style scoped src="./HomePage.scss" lang="scss"></style>
