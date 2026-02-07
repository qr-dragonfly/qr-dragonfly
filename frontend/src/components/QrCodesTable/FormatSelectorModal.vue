<template>
  <div class="modal" @click.self="emit('close')">
    <div class="modalContent">
      <h2>Download QR Code</h2>
      <p class="modalDescription">Select a format to download your QR code</p>
      
      <div class="formatOptions">
        <label
          v-for="format in formats"
          :key="format.value"
          class="formatOption"
          :class="{ selected: selectedFormat === format.value }"
        >
          <input
            type="radio"
            :value="format.value"
            v-model="selectedFormat"
            name="format"
          />
          <div class="formatInfo">
            <div class="formatName">{{ format.name }}</div>
            <div class="formatDescription">{{ format.description }}</div>
          </div>
        </label>
      </div>

      <div class="modalActions">
        <button class="button" type="button" @click="handleDownload">
          Download
        </button>
        <button class="button secondary" type="button" @click="emit('close')">
          Cancel
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import type { QrFormat } from '../../lib/qr'

const FORMAT_STORAGE_KEY = 'qrDownloadFormat'

type FormatOption = {
  value: QrFormat
  name: string
  description: string
}

const formats: FormatOption[] = [
  { value: 'png', name: 'PNG', description: 'Best for web and digital use' },
  { value: 'jpeg', name: 'JPEG', description: 'Smaller file size, good for photos' },
  { value: 'svg', name: 'SVG', description: 'Vector format, scales without loss' },
  { value: 'eps', name: 'EPS', description: 'Professional print format' },
]

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'download', format: QrFormat): void
}>()

const selectedFormat = ref<QrFormat>('png')

function handleKeyDown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    emit('close')
  }
}

onMounted(() => {
  // Load last used format from localStorage
  const saved = localStorage.getItem(FORMAT_STORAGE_KEY)
  if (saved && ['png', 'jpeg', 'svg', 'eps'].includes(saved)) {
    selectedFormat.value = saved as QrFormat
  }
  
  // Add escape key listener
  window.addEventListener('keydown', handleKeyDown)
})

onUnmounted(() => {
  // Clean up event listener
  window.removeEventListener('keydown', handleKeyDown)
})

function handleDownload() {
  // Save selected format to localStorage
  localStorage.setItem(FORMAT_STORAGE_KEY, selectedFormat.value)
  emit('download', selectedFormat.value)
}
</script>

<style scoped lang="scss">
.modal {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0, 0, 0, 0.75);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modalContent {
  background: var(--color-surface);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  padding: var(--space-2xl);
  max-width: 500px;
  width: 90%;
  max-height: 90vh;
  overflow-y: auto;
  box-shadow: var(--shadow-lg);
}

h2 {
  margin: 0 0 var(--space-sm) 0;
  font-size: var(--font-size-2xl);
  font-weight: 600;
  color: var(--color-fg);
}

.modalDescription {
  margin: 0 0 var(--space-2xl) 0;
  color: var(--color-fg-muted);
  font-size: var(--font-size-sm);
}

.formatOptions {
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
  margin-bottom: var(--space-2xl);
}

.formatOption {
  display: flex;
  align-items: flex-start;
  gap: var(--space-md);
  padding: var(--space-lg);
  border: 2px solid var(--border-color);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all 0.2s;

  &:hover {
    border-color: var(--color-link);
    background: rgba(100, 108, 255, 0.1);
  }

  &.selected {
    border-color: var(--color-link);
    background: rgba(100, 108, 255, 0.1);
  }

  input[type='radio'] {
    margin-top: 2px;
    cursor: pointer;
    accent-color: var(--color-link);
  }
}

.formatInfo {
  flex: 1;
}

.formatName {
  font-weight: 600;
  font-size: var(--font-size-base);
  color: var(--color-fg);
  margin-bottom: 4px;
}

.formatDescription {
  font-size: var(--font-size-sm);
  color: var(--color-fg-muted);
}

.modalActions {
  display: flex;
  gap: var(--space-md);
  justify-content: flex-end;
}

.button {
  padding: 10px 20px;
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  font-size: var(--font-size-sm);
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  background: var(--color-link);
  color: white;

  &:hover {
    background: var(--color-link-hover);
  }

  &.secondary {
    background: transparent;
    border-color: var(--border-color);
    color: var(--color-fg);

    &:hover {
      border-color: var(--border-color-hover);
      background: rgba(255, 255, 255, 0.05);
    }
  }
}
</style>
