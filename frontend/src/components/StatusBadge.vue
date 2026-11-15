<script setup lang="ts">
import type { TicketStatus, Priority } from '@/types'

const props = defineProps<{
  status?: TicketStatus
  priority?: Priority
}>()

function statusClass(s: string): string {
  switch (s) {
    case 'open': return 'badge-info'
    case 'in_progress': return 'badge-warning'
    case 'resolved': return 'badge-success'
    case 'closed': return 'badge-ghost'
    default: return 'badge-ghost'
  }
}

function priorityClass(p: string): string {
  switch (p) {
    case 'low': return 'badge-ghost'
    case 'medium': return 'badge-warning'
    case 'high': return 'badge-error'
    default: return 'badge-ghost'
  }
}

function label(v: string): string {
  return v.replace(/_/g, ' ')
}
</script>

<template>
  <span v-if="props.status" :class="['badge', statusClass(props.status), 'badge-sm']">
    {{ label(props.status) }}
  </span>
  <span v-else-if="props.priority" :class="['badge', priorityClass(props.priority), 'badge-sm']">
    {{ label(props.priority) }}
  </span>
</template>
