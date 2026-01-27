<template>
  <div class="clientPage">
    <h1>Client Debug Info</h1>
    <div class="jsonContainer">
      <pre>{{ userJson }}</pre>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useUser } from '../../composables/useUser'

const { user } = useUser()

const userJson = computed(() => {
  if (!user.value) {
    return 'No user data available'
  }
  return JSON.stringify(user.value, null, 2)
})
</script>

<style scoped>
.clientPage {
  max-width: 800px;
  margin: 0 auto;
  padding: 2rem;
}

h1 {
  color: var(--color-text);
  margin-bottom: 2rem;
}

.jsonContainer {
  background: #1e1e1e;
  border: 1px solid #333;
  border-radius: 8px;
  padding: 1.5rem;
  overflow-x: auto;
}

pre {
  color: #d4d4d4;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 14px;
  line-height: 1.6;
  margin: 0;
  white-space: pre;
}
</style>
