<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const auth = useAuthStore()

function logout() {
  auth.logout()
  router.push('/login')
}
</script>

<template>
  <div class="min-h-screen bg-base-200">
    <!-- Navbar -->
    <div v-if="auth.isLoggedIn" class="navbar bg-base-100 shadow-sm">
      <div class="navbar-start">
        <div class="dropdown">
          <div tabindex="0" role="button" class="btn btn-ghost lg:hidden">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
            </svg>
          </div>
          <ul tabindex="0" class="menu menu-sm dropdown-content mt-3 z-[1] p-2 shadow bg-base-100 rounded-box w-52">
            <li><RouterLink to="/tickets">Tickets</RouterLink></li>
            <li><RouterLink to="/tickets/new">New Ticket</RouterLink></li>
            <li v-if="auth.isAdmin"><RouterLink to="/users">Users</RouterLink></li>
          </ul>
        </div>
        <RouterLink to="/tickets" class="btn btn-ghost text-xl">SimpleHelp</RouterLink>
      </div>
      <div class="navbar-center hidden lg:flex">
        <ul class="menu menu-horizontal px-1">
          <li><RouterLink to="/tickets">Tickets</RouterLink></li>
          <li><RouterLink to="/tickets/new">New Ticket</RouterLink></li>
          <li v-if="auth.isAdmin"><RouterLink to="/users">Users</RouterLink></li>
        </ul>
      </div>
      <div class="navbar-end">
        <div class="dropdown dropdown-end">
          <div tabindex="0" role="button" class="btn btn-ghost btn-circle avatar placeholder">
            <div class="bg-neutral text-neutral-content rounded-full w-10">
              <span class="text-sm">{{ auth.user?.name?.charAt(0).toUpperCase() || '?' }}</span>
            </div>
          </div>
          <ul tabindex="0" class="menu menu-sm dropdown-content mt-3 z-[1] p-2 shadow bg-base-100 rounded-box w-52">
            <li class="menu-title">
              {{ auth.user?.name }}
              <span class="badge badge-sm badge-ghost ml-1">{{ auth.user?.role }}</span>
            </li>
            <li><a @click="logout">Logout</a></li>
          </ul>
        </div>
      </div>
    </div>

    <!-- Page content -->
    <main class="container mx-auto px-4 py-6 max-w-5xl">
      <RouterView />
    </main>
  </div>
</template>
