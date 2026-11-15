import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { guestOnly: true },
    },
    {
      path: '/tickets',
      name: 'tickets',
      component: () => import('@/views/TicketListView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/tickets/new',
      name: 'ticket-new',
      component: () => import('@/views/TicketCreateView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/tickets/:id',
      name: 'ticket-detail',
      component: () => import('@/views/TicketDetailView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/users',
      name: 'users',
      component: () => import('@/views/UsersView.vue'),
      meta: { requiresAuth: true, requiresAdmin: true },
    },
    {
      path: '/',
      redirect: '/tickets',
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/tickets',
    },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()

  // If we have a token but no user yet, fetch the profile.
  if (auth.token && !auth.user) {
    try {
      await auth.fetchMe()
    } catch {
      auth.clear()
      return '/login'
    }
  }

  if (to.meta.requiresAuth && !auth.isLoggedIn) {
    return '/login'
  }
  if (to.meta.guestOnly && auth.isLoggedIn) {
    return '/tickets'
  }
  if (to.meta.requiresAdmin && !auth.isAdmin) {
    return '/tickets'
  }
})

export default router
