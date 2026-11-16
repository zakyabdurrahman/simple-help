<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import client, { extractApiError } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import type { Ticket, TicketDetail, Comment } from '@/types'
import StatusBadge from '@/components/StatusBadge.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const ticket = ref<TicketDetail | null>(null)
const loading = ref(true)
const error = ref('')

// Comment form.
const commentBody = ref('')
const commentInternal = ref(false)
const commentError = ref('')
const commentLoading = ref(false)

// Status update form.
const updateLoading = ref(false)
const updateError = ref('')

const ticketId = computed(() => Number(route.params.id))

// Transition table: which transitions are allowed for the current role.
interface Transition {
  to: string
  label: string
  class: string
}

const allowedTransitions = computed<Transition[]>(() => {
  if (!ticket.value) return []
  const role = auth.role
  const status = ticket.value.status
  const transitions: Transition[] = []

  // open -> in_progress (agent, admin)
  if (status === 'open' && (role === 'agent' || role === 'admin')) {
    transitions.push({ to: 'in_progress', label: 'Move to In Progress', class: 'btn-warning' })
  }
  // open -> closed (requester, admin)
  if (status === 'open' && (role === 'requester' || role === 'admin')) {
    transitions.push({ to: 'closed', label: 'Cancel Ticket', class: 'btn-ghost' })
  }
  // in_progress -> resolved (agent, admin)
  if (status === 'in_progress' && (role === 'agent' || role === 'admin')) {
    transitions.push({ to: 'resolved', label: 'Mark as Resolved', class: 'btn-success' })
  }
  // resolved -> closed (requester, agent, admin)
  if (status === 'resolved') {
    transitions.push({ to: 'closed', label: 'Mark as Closed', class: 'btn-success' })
  }
  // resolved -> open (requester)
  if (status === 'resolved' && role === 'requester') {
    transitions.push({ to: 'open', label: 'Reopen Ticket', class: 'btn-warning' })
  }
  return transitions
})

// Can change priority/assignee: agent or admin only.
const canManage = computed(() => auth.role === 'agent' || auth.role === 'admin')

// Priority edit.
const showPrioritySelect = ref(false)
const newPriority = ref('')

async function fetchTicket() {
  loading.value = true
  error.value = ''
  try {
    const { data } = await client.get<TicketDetail>(`/api/tickets/${ticketId.value}`)
    ticket.value = data
  } catch (err) {
    error.value = extractApiError(err).error
  } finally {
    loading.value = false
  }
}

async function changeStatus(to: string) {
  if (!ticket.value) return
  updateLoading.value = true
  updateError.value = ''
  try {
    // PATCH returns a plain Ticket (no comments); merge the existing
    // comments onto the response so the template keeps rendering them
    // instead of going blank on `ticket.comments` being undefined.
    const payload: Record<string, unknown> = { status: to }
    // Starting work also claims the ticket for the acting agent/admin,
    // so "Move to In Progress" is a single action.
    if (to === 'in_progress' && auth.user) {
      payload.assignee_id = auth.user.id
    }
    const { data } = await client.patch<Ticket>(`/api/tickets/${ticket.value.id}`, payload)
    ticket.value = { ...data, comments: ticket.value.comments }
  } catch (err) {
    updateError.value = extractApiError(err).error
  } finally {
    updateLoading.value = false
  }
}

async function changePriority() {
  if (!ticket.value || !newPriority.value) return
  updateLoading.value = true
  updateError.value = ''
  try {
    const { data } = await client.patch<Ticket>(`/api/tickets/${ticket.value.id}`, { priority: newPriority.value })
    ticket.value = { ...data, comments: ticket.value.comments }
    showPrioritySelect.value = false
    newPriority.value = ''
  } catch (err) {
    updateError.value = extractApiError(err).error
  } finally {
    updateLoading.value = false
  }
}

async function addComment() {
  if (!ticket.value || !commentBody.value.trim()) return
  commentLoading.value = true
  commentError.value = ''
  try {
    const { data } = await client.post<Comment>(`/api/tickets/${ticket.value.id}/comments`, {
      body: commentBody.value,
      is_internal: commentInternal.value,
    })
    ticket.value.comments.push(data)
    commentBody.value = ''
    commentInternal.value = false
  } catch (err) {
    commentError.value = extractApiError(err).error
  } finally {
    commentLoading.value = false
  }
}

onMounted(fetchTicket)
</script>

<template>
  <div>
    <button class="btn btn-ghost btn-sm mb-4" @click="router.push('/tickets')">← Back</button>

    <div v-if="loading" class="flex justify-center py-8">
      <span class="loading loading-spinner loading-lg"></span>
    </div>

    <div v-else-if="error" class="alert alert-error">
      <span>Failed to load ticket: {{ error }}</span>
    </div>

    <div v-else-if="ticket">
      <!-- Ticket info card -->
      <div class="card bg-base-100 shadow-xl mb-6">
        <div class="card-body">
          <div class="flex flex-wrap items-start justify-between gap-2">
            <div>
              <h1 class="text-xl font-bold">{{ ticket.title }}</h1>
              <p class="text-sm text-base-content/60">#{{ ticket.id }} · Created {{ new Date(ticket.created_at).toLocaleString() }}</p>
            </div>
            <div class="flex flex-wrap gap-1">
              <StatusBadge :status="ticket.status" />
              <StatusBadge :priority="ticket.priority" />
              <span v-if="ticket.category" class="badge badge-outline badge-sm">{{ ticket.category }}</span>
            </div>
          </div>

          <p class="mt-2 whitespace-pre-wrap">{{ ticket.description }}</p>

          <div class="text-sm text-base-content/60 mt-2">
            <span>Requester: {{ ticket.requester_name }} &lt;{{ ticket.requester_email }}&gt;</span>
            <span v-if="ticket.assignee_id">
              · Assignee: {{ ticket.assignee_name }} &lt;{{ ticket.assignee_email }}&gt;
            </span>
            <span v-else> · Unassigned</span>
          </div>

          <!-- Role-based actions -->
          <div v-if="updateError" class="alert alert-error mt-3">
            <span>{{ updateError }}</span>
          </div>

          <div class="flex flex-wrap gap-2 mt-4">
            <button
              v-for="t in allowedTransitions"
              :key="t.to"
              :class="['btn btn-sm', t.class]"
              :disabled="updateLoading"
              @click="changeStatus(t.to)"
            >
              {{ t.label }}
            </button>

            <!-- Priority change (agent/admin) -->
            <button v-if="canManage && !showPrioritySelect" class="btn btn-sm btn-outline" @click="showPrioritySelect = true; newPriority = ticket.priority">
              Change Priority
            </button>
            <div v-if="showPrioritySelect" class="flex gap-1 items-center">
              <select v-model="newPriority" class="select select-sm select-bordered">
                <option value="low">Low</option>
                <option value="medium">Medium</option>
                <option value="high">High</option>
              </select>
              <button class="btn btn-sm btn-primary" @click="changePriority" :disabled="updateLoading">Save</button>
              <button class="btn btn-sm btn-ghost" @click="showPrioritySelect = false">Cancel</button>
            </div>
          </div>
        </div>
      </div>

      <!-- Comments -->
      <div class="card bg-base-100 shadow-xl">
        <div class="card-body">
          <h2 class="text-lg font-bold mb-2">Comments ({{ ticket.comments.length }})</h2>

          <div v-if="ticket.comments.length === 0" class="text-sm text-base-content/60 py-4">
            No comments yet.
          </div>

          <div class="space-y-3">
            <div
              v-for="c in ticket.comments"
              :key="c.id"
              :class="['chat', 'chat-start']"
            >
              <div class="chat-header">
                {{ c.author_name }}
                <span v-if="c.is_internal" class="badge badge-error badge-xs ml-1">internal</span>
                <time class="text-xs opacity-50 ml-1">{{ new Date(c.created_at).toLocaleString() }}</time>
              </div>
              <div :class="['chat-bubble', c.is_internal ? 'chat-bubble-error' : '']">{{ c.body }}</div>
            </div>
          </div>

          <!-- Comment form -->
          <div class="mt-4 pt-4 border-t border-base-300">
            <div v-if="commentError" class="alert alert-error mb-2">
              <span>{{ commentError }}</span>
            </div>
            <textarea
              v-model="commentBody"
              placeholder="Write a comment..."
              class="textarea textarea-bordered w-full"
            ></textarea>
            <div class="flex items-center justify-between mt-2">
              <label v-if="canManage" class="label cursor-pointer gap-2">
                <span class="label-text">Internal note</span>
                <input v-model="commentInternal" type="checkbox" class="toggle toggle-sm toggle-error" />
              </label>
              <button
                class="btn btn-primary btn-sm"
                :disabled="!commentBody.trim() || commentLoading"
                @click="addComment"
              >
                <span v-if="commentLoading" class="loading loading-spinner loading-xs"></span>
                Post Comment
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
