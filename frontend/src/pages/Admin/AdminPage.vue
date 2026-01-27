<script setup lang="ts">
import { ref, computed, onMounted, watchEffect } from 'vue'
import { useRouter } from 'vue-router'
import { adminApi, type AdminUser } from '../../api'
import { useUser } from '../../composables/useUser'

const router = useRouter()
const { isAuthed, isLoaded, isAdmin } = useUser()

// Redirect non-admin users
watchEffect(() => {
  if (!isLoaded.value) return
  if (!isAuthed.value || !isAdmin.value) {
    void router.replace({ name: 'home' })
  }
})

const adminKey = ref('')
const users = ref<AdminUser[]>([])
const isLoading = ref(false)
const errorMessage = ref<string | null>(null)
const successMessage = ref<string | null>(null)
const searchQuery = ref('')

const editingUser = ref<AdminUser | null>(null)
const editUserType = ref('')

const filteredUsers = computed(() => {
  if (!searchQuery.value) return users.value
  const query = searchQuery.value.toLowerCase()
  return users.value.filter(
    (u) =>
      u.email?.toLowerCase().includes(query) ||
      u.userType?.toLowerCase().includes(query) ||
      u.id?.toLowerCase().includes(query)
  )
})

async function loadUsers() {
  if (!adminKey.value.trim()) {
    errorMessage.value = 'Admin key required'
    return
  }

  isLoading.value = true
  errorMessage.value = null

  try {
    const result = await adminApi.listUsers(adminKey.value.trim())
    users.value = result
    successMessage.value = `Loaded ${result.length} users`
    setTimeout(() => {
      successMessage.value = null
    }, 3000)
  } catch (err) {
    errorMessage.value = 'Failed to load users. Check your admin key.'
    users.value = []
  } finally {
    isLoading.value = false
  }
}

function startEdit(user: AdminUser) {
  editingUser.value = user
  editUserType.value = user.userType || 'free'
}

function cancelEdit() {
  editingUser.value = null
  editUserType.value = ''
  errorMessage.value = null
}

async function saveUser() {
  if (!editingUser.value) return
  if (!adminKey.value.trim()) {
    errorMessage.value = 'Admin key required'
    return
  }

  isLoading.value = true
  errorMessage.value = null
  successMessage.value = null

  try {
    const updated = await adminApi.updateUser(
      editingUser.value.id,
      { userType: editUserType.value },
      adminKey.value.trim()
    )
    
    // Update in list
    const index = users.value.findIndex((u) => u.id === updated.id)
    if (index !== -1) {
      users.value[index] = updated
    }

    successMessage.value = `Updated ${updated.email}`
    cancelEdit()
    setTimeout(() => {
      successMessage.value = null
    }, 3000)
  } catch (err) {
    errorMessage.value = 'Failed to update user'
  } finally {
    isLoading.value = false
  }
}

onMounted(() => {
  // Try to load admin key from localStorage
  const saved = localStorage.getItem('adminKey')
  if (saved) {
    adminKey.value = saved
  }
})

function saveAdminKey() {
  localStorage.setItem('adminKey', adminKey.value.trim())
}
</script>

<template>
  <main class="page">
    <header class="header">
      <h1 class="title">Admin - User Management</h1>
      <p class="subtitle">Manage user subscriptions and entitlements</p>
    </header>

    <section class="card">
      <div class="adminKeySection">
        <label class="field">
          <span class="label">Admin API Key</span>
          <div class="keyInput">
            <input
              v-model="adminKey"
              type="password"
              class="input"
              placeholder="Enter admin API key"
              @blur="saveAdminKey"
            />
            <button class="button" type="button" @click="loadUsers" :disabled="isLoading || !adminKey.trim()">
              {{ isLoading ? 'Loading...' : 'Load Users' }}
            </button>
          </div>
        </label>
      </div>

      <p v-if="successMessage" class="success">{{ successMessage }}</p>
      <p v-if="errorMessage" class="error">{{ errorMessage }}</p>

      <div v-if="users.length > 0" class="searchSection">
        <input
          v-model="searchQuery"
          type="text"
          class="input"
          placeholder="Search by email, user type, or ID..."
        />
      </div>

      <div v-if="users.length > 0" class="usersTable">
        <table class="table">
          <thead>
            <tr>
              <th>Email</th>
              <th>User Type</th>
              <th>Created</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="user in filteredUsers" :key="user.id">
              <td>{{ user.email }}</td>
              <td>
                <span class="badge" :class="`badge-${user.userType || 'free'}`">
                  {{ user.userType || 'free' }}
                </span>
              </td>
              <td class="dateCell">{{ user.createdAt ? new Date(user.createdAt).toLocaleDateString() : '-' }}</td>
              <td>
                <button class="buttonSmall" type="button" @click="startEdit(user)">
                  Edit
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <p v-else-if="!isLoading" class="muted">No users loaded. Enter your admin key and click "Load Users".</p>
    </section>

    <!-- Edit User Modal -->
    <dialog v-if="editingUser" :open="!!editingUser" class="dialog" @cancel.prevent="cancelEdit">
      <div class="dialogInner">
        <div class="dialogHeader">
          <h2 class="dialogTitle">Edit User</h2>
          <button class="iconButton" type="button" aria-label="Close" @click="cancelEdit">×</button>
        </div>

        <div class="dialogContent">
          <p class="userEmail">{{ editingUser.email }}</p>

          <label class="field">
            <span class="label">User Type / Subscription</span>
            <select v-model="editUserType" class="select">
              <option value="free">Free</option>
              <option value="basic">Basic</option>
              <option value="enterprise">Enterprise</option>
              <option value="admin">Admin</option>
            </select>
          </label>

          <div class="hint">
            <strong>Note:</strong> Changing the user type here updates Cognito. For Stripe subscription cancellations, 
            use the Stripe Dashboard or Customer Portal - webhooks will automatically sync the entitlement.
          </div>
        </div>

        <div class="actions">
          <button class="buttonSmall" type="button" @click="saveUser" :disabled="isLoading">
            {{ isLoading ? 'Saving...' : 'Save Changes' }}
          </button>
          <button class="buttonSmall secondary" type="button" @click="cancelEdit">
            Cancel
          </button>
        </div>

        <p v-if="errorMessage" class="error">{{ errorMessage }}</p>
      </div>
    </dialog>
  </main>
</template>

<style scoped src="../HomePage/HomePage.scss" lang="scss"></style>
<style scoped src="./AdminPage.scss" lang="scss"></style>
