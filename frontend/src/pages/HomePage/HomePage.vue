<script setup lang="ts">
import { watchEffect } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useQrCodes } from '../../composables/useQrCodes'
import { useUser } from '../../composables/useUser'
import CreateQrCodeForm from '../../components/CreateQrCodeForm/CreateQrCodeForm.vue'
import QrCodesTable from '../../components/QrCodesTable/QrCodesTable.vue'

const router = useRouter()
const route = useRoute()

const { isAuthed, isLoaded } = useUser()

watchEffect(() => {
  if (!isLoaded.value) return
  if (isAuthed.value) return

  const redirect = route.fullPath || '/'
  void router.replace({ name: 'login', query: { redirect } })
})

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

    <CreateQrCodeForm
      v-if="isAuthed"
      v-model:label="labelInput"
      v-model:url="urlInput"
      :isCreating="isCreating"
      :errorMessage="errorMessage"
      @submit="createQrCode"
    />

    <QrCodesTable
      v-if="isAuthed"
      :qrCodes="qrCodes"
      :updatingId="updatingId"
      :errorMessage="errorMessage"
      @copy-url="copyToClipboard"
      @download="downloadQrCode"
      @remove="deleteQrCode"
      @update="updateQrCode"
      @set-active="setQrCodeActive"
    />
  </main>
</template>

<style scoped src="./HomePage.scss" lang="scss"></style>
