<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import client, { extractApiError } from '@/api/client'
import type { Ticket } from '@/types'
import StatusBadge from '@/components/StatusBadge.vue'

const router = useRouter()

const tickets = ref<Ticket[]>([])
const loading = ref(false)
const error = ref('')

const search = ref('')
const statusFilter = ref('')
const priorityFilter = ref('')

const page = ref(0)
const limit = ref(20)
const total = ref(0)

async function fetchTickets() {
  loading.value = true
  error.value = ''
  try {
    const params: Record<string, string | number> = {
      limit: limit.value,
      offset: page.value * limit.value,
    }
    if (search.value) params.q = search.value
    if (statusFilter.value) params.status = statusFilter.value
    if (priorityFilter.value) params.priority = priorityFilter.value

    const { data } = await client.get<{ tickets: Ticket[] }>('/api/tickets', { params })
    tickets.value = data.tickets || []
    total.value = tickets.value.length
  } catch (err) {
    error.value = extractApiError(err).error
  } finally {
    loading.value = false
  }
}

// Debounced search.
let searchTimer: ReturnType<typeof setTimeout>
watch(search, () => {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    page.value = 0
    fetchTickets()
  }, 300)
})

watch([statusFilter, priorityFilter], () => {
  page.value = 0
  fetchTickets()
})

onMounted(fetchTickets)

function nextPage() {
  if (tickets.value.length === limit.value) {
    page.value++
    fetchTickets()
  }
}
function prevPage() {
  if (page.value > 0) {
    page.value--
    fetchTickets()
  }
}
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-4">
      <h1 class="text-2xl font-bold">Tickets</h1>
      <RouterLink to="/tickets/new" class="btn btn-primary btn-sm">New Ticket</RouterLink>
    </div>

    <!-- Filters -->
    <div class="flex flex-col sm:flex-row gap-2 mb-4">
      <input
        v-model="search"
        type="text"
        placeholder="Search title or description..."
        class="input input-bordered flex-1"
      />
      <select v-model="statusFilter" class="select select-bordered w-full sm:w-40">
        <option value="">All statuses</option>
        <option value="open">Open</option>
        <option value="in_progress">In Progress</option>
        <option value="resolved">Resolved</option>
        <option value="closed">Closed</option>
      </select>
      <select v-model="priorityFilter" class="select select-bordered w-full sm:w-40">
        <option value="">All priorities</option>
        <option value="low">Low</option>
        <option value="medium">Medium</option>
        <option value="high">High</option>
      </select>
    </div>

    <div v-if="loading" class="flex justify-center py-8">
      <span class="loading loading-spinner loading-lg"></span>
    </div>

    <div v-else-if="error" class="alert alert-error">
      <span>Failed to load tickets: {{ error }}</span>
    </div>

    <div v-else-if="tickets.length === 0" class="text-center py-8 text-base-content/60">
      No tickets found.
    </div>

    <!-- Ticket list (desktop table + mobile cards) -->
    <div v-else>
      <!-- Desktop table -->
      <div class="overflow-x-auto hidden md:block">
        <table class="table">
          <thead>
            <tr>
              <th>ID</th>
              <th>Title</th>
              <th>Status</th>
              <th>Priority</th>
              <th>Created</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="t in tickets"
              :key="t.id"
              class="hover cursor-pointer"
              @click="router.push(`/tickets/${t.id}`)"
            >
              <td class="font-mono text-sm">{{ t.id }}</td>
              <td>{{ t.title }}</td>
              <td><StatusBadge :status="t.status" /></td>
              <td><StatusBadge :priority="t.priority" /></td>
              <td class="text-sm text-base-content/60">{{ new Date(t.created_at).toLocaleDateString() }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Mobile cards -->
      <div class="space-y-2 md:hidden">
      <div
        v-for="t in tickets"
        :key="t.id"
        class="card bg-base-100 shadow compact cursor-pointer"
        @click="router.push(`/tickets/${t.id}`)"
      >
        <div class="card-body">
          <div class="flex items-center justify-between">
            <span class="font-mono text-sm text-base-content/60">#{{ t.id }}</span>
            <div class="flex gap-1">
              <StatusBadge :status="t.status" />
              <StatusBadge :priority="t.priority" />
            </div>
          </div>
          <p class="font-medium">{{ t.title }}</p>
          <p class="text-xs text-base-content/60">{{ new Date(t.created_at).toLocaleDateString() }}</p>
        </div>
      </div>
      </div>
    </div>

    <!-- Pagination -->
    <div v-if="tickets.length > 0" class="flex justify-between items-center mt-4">
      <button class="btn btn-sm btn-ghost" :disabled="page === 0" @click="prevPage">Prev</button>
      <span class="text-sm text-base-content/60">Page {{ page + 1 }}</span>
      <button class="btn btn-sm btn-ghost" :disabled="tickets.length < limit" @click="nextPage">Next</button>
    </div>
  </div>
</template>
