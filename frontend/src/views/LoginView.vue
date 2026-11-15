<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { extractApiError } from '@/api/client'
import FormField from '@/components/FormField.vue'

const router = useRouter()
const auth = useAuthStore()

const mode = ref<'login' | 'register'>('login')
const loading = ref(false)
const generalError = ref('')

const form = reactive({
  name: '',
  email: '',
  password: '',
})

const errors = reactive<Record<string, string>>({})

function validateClient(): boolean {
  Object.keys(errors).forEach((k) => delete errors[k])
  if (mode.value === 'register' && !form.name.trim()) {
    errors.name = 'Name is required'
  }
  if (!form.email.trim()) {
    errors.email = 'Email is required'
  } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email)) {
    errors.email = 'Invalid email format'
  }
  if (!form.password) {
    errors.password = 'Password is required'
  } else if (mode.value === 'register' && form.password.length < 8) {
    errors.password = 'Password must be at least 8 characters'
  } else if (mode.value === 'register' && !/(?=.*[a-zA-Z])(?=.*\d)/.test(form.password)) {
    errors.password = 'Password must contain at least one letter and one digit'
  }
  return Object.keys(errors).length === 0
}

async function submit() {
  generalError.value = ''
  if (!validateClient()) return

  loading.value = true
  Object.keys(errors).forEach((k) => delete errors[k])

  try {
    if (mode.value === 'login') {
      await auth.login(form.email, form.password)
      router.push('/tickets')
    } else {
      await auth.register(form.name, form.email, form.password)
      // Auto-login after register.
      await auth.login(form.email, form.password)
      router.push('/tickets')
    }
  } catch (err) {
    const apiErr = extractApiError(err)
    if (apiErr.fields) {
      Object.assign(errors, apiErr.fields)
    } else if (apiErr.error === 'invalid_credentials') {
      generalError.value = 'Invalid email or password'
    } else if (apiErr.error === 'email_exists') {
      errors.email = 'Email already registered'
    } else {
      generalError.value = 'Something went wrong. Please try again.'
    }
  } finally {
    loading.value = false
  }
}

function switchMode() {
  mode.value = mode.value === 'login' ? 'register' : 'login'
  generalError.value = ''
  Object.keys(errors).forEach((k) => delete errors[k])
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-base-200 px-4">
    <div class="card w-full max-w-md bg-base-100 shadow-xl">
      <div class="card-body">
        <h2 class="text-2xl font-bold text-center mb-4">
          {{ mode === 'login' ? 'Login' : 'Register' }}
        </h2>

        <div role="alert" v-if="generalError" class="alert alert-error mb-4">
          <span>{{ generalError }}</span>
        </div>

        <form @submit.prevent="submit" class="space-y-4">
          <FormField v-if="mode === 'register'" label="Name" :error="errors.name">
            <input
              v-model="form.name"
              type="text"
              placeholder="Jane Doe"
              :class="['input input-bordered w-full', errors.name ? 'input-error' : '']"
            />
          </FormField>

          <FormField label="Email" :error="errors.email">
            <input
              v-model="form.email"
              type="email"
              placeholder="jane@example.com"
              :class="['input input-bordered w-full', errors.email ? 'input-error' : '']"
            />
          </FormField>

          <FormField label="Password" :error="errors.password">
            <input
              v-model="form.password"
              type="password"
              placeholder="********"
              :class="['input input-bordered w-full', errors.password ? 'input-error' : '']"
            />
          </FormField>

          <button type="submit" class="btn btn-primary w-full" :disabled="loading">
            <span v-if="loading" class="loading loading-spinner loading-sm"></span>
            {{ mode === 'login' ? 'Login' : 'Register' }}
          </button>
        </form>

        <div class="text-center mt-4">
          <button @click="switchMode" class="link link-hover text-sm">
            {{ mode === 'login' ? "Don't have an account? Register" : 'Already have an account? Login' }}
          </button>
        </div>

        <div class="text-center mt-2">
          <p class="text-xs text-base-content/60">
            Seed accounts: requester/agent/admin @example.com / password
          </p>
        </div>
      </div>
    </div>
  </div>
</template>
