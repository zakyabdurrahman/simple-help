<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import client, { extractApiError } from '@/api/client'
import FormField from '@/components/FormField.vue'

const router = useRouter()

const loading = ref(false)
const generalError = ref('')
const success = ref(false)

const form = reactive({
  title: '',
  description: '',
  category: '',
  priority: 'medium',
})

const errors = reactive<Record<string, string>>({})

async function submit() {
  loading.value = true
  generalError.value = ''
  success.value = false
  Object.keys(errors).forEach((k) => delete errors[k])

  // Client-side validation.
  if (!form.title.trim()) errors.title = 'Title is required'
  if (!form.description.trim()) errors.description = 'Description is required'
  if (Object.keys(errors).length > 0) {
    loading.value = false
    return
  }

  try {
    const { data } = await client.post('/api/tickets', {
      title: form.title,
      description: form.description,
      category: form.category || undefined,
      priority: form.priority,
    })
    success.value = true
    setTimeout(() => router.push(`/tickets/${data.id}`), 800)
  } catch (err) {
    const apiErr = extractApiError(err)
    if (apiErr.fields) {
      Object.assign(errors, apiErr.fields)
    } else {
      generalError.value = 'Failed to create ticket. Please try again.'
    }
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="max-w-2xl">
    <h1 class="text-2xl font-bold mb-4">Create Ticket</h1>

    <div v-if="success" class="alert alert-success mb-4">
      <span>Ticket created! Redirecting...</span>
    </div>
    <div v-if="generalError" class="alert alert-error mb-4">
      <span>{{ generalError }}</span>
    </div>

    <form @submit.prevent="submit" class="space-y-4">
      <FormField label="Title" :error="errors.title">
        <input
          v-model="form.title"
          type="text"
          placeholder="Brief summary of the issue"
          :class="['input input-bordered w-full', errors.title ? 'input-error' : '']"
        />
      </FormField>

      <FormField label="Description" :error="errors.description">
        <textarea
          v-model="form.description"
          placeholder="Describe the issue in detail..."
          :class="['textarea textarea-bordered w-full min-h-32', errors.description ? 'textarea-error' : '']"
        ></textarea>
      </FormField>

      <FormField label="Category (optional)">
        <input
          v-model="form.category"
          type="text"
          placeholder="e.g. hardware, software, network"
          class="input input-bordered w-full"
        />
      </FormField>

      <FormField label="Priority">
        <select v-model="form.priority" class="select select-bordered w-full">
          <option value="low">Low</option>
          <option value="medium">Medium</option>
          <option value="high">High</option>
        </select>
      </FormField>

      <div class="flex gap-2">
        <button type="submit" class="btn btn-primary" :disabled="loading">
          <span v-if="loading" class="loading loading-spinner loading-sm"></span>
          Create Ticket
        </button>
        <button type="button" class="btn btn-ghost" @click="router.push('/tickets')">Cancel</button>
      </div>
    </form>
  </div>
</template>
