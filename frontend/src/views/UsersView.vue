<script setup lang="ts">
import { ref, onMounted } from 'vue'
import client, { extractApiError } from '@/api/client'
import type { User, Role } from '@/types'

const users = ref<User[]>([])
const loading = ref(true)
const error = ref('')
const updatingId = ref<number | null>(null)

async function fetchUsers() {
  loading.value = true
  error.value = ''
  try {
    const { data } = await client.get<{ users: User[] }>('/api/users')
    users.value = data.users || []
  } catch (err) {
    error.value = extractApiError(err).error
  } finally {
    loading.value = false
  }
}

async function changeRole(user: User, event: Event) {
  const role = (event.target as HTMLSelectElement).value as Role
  if (user.role === role) return
  updatingId.value = user.id
  try {
    const { data } = await client.patch<User>(`/api/users/${user.id}/role`, { role })
    const idx = users.value.findIndex((u) => u.id === user.id)
    if (idx !== -1) users.value[idx] = data
  } catch (err) {
    error.value = extractApiError(err).error
  } finally {
    updatingId.value = null
  }
}

onMounted(fetchUsers)
</script>

<template>
  <div>
    <h1 class="text-2xl font-bold mb-4">Manage Users</h1>

    <div v-if="loading" class="flex justify-center py-8">
      <span class="loading loading-spinner loading-lg"></span>
    </div>

    <div v-else-if="error" class="alert alert-error">
      <span>{{ error }}</span>
    </div>

    <div v-else class="overflow-x-auto">
      <table class="table">
        <thead>
          <tr>
            <th>ID</th>
            <th>Name</th>
            <th>Email</th>
            <th>Role</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="u in users" :key="u.id">
            <td class="font-mono text-sm">{{ u.id }}</td>
            <td>{{ u.name }}</td>
            <td class="text-sm">{{ u.email }}</td>
            <td>
              <select
                :value="u.role"
                class="select select-bordered select-sm"
                :disabled="updatingId === u.id"
                @change="changeRole(u, $event)"
              >
                <option value="requester">requester</option>
                <option value="agent">agent</option>
                <option value="admin">admin</option>
              </select>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
