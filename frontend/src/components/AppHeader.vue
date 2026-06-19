<template>
  <header class="app-header">
    <router-link to="/" class="logo">Tiny Lowcode Platform</router-link>
    <nav class="nav">
      <router-link to="/scripts">Scripts</router-link>
      <router-link to="/sql">SQL</router-link>
      <router-link v-if="auth.isAdmin" to="/admin">Admin</router-link>
    </nav>
    <div class="right">
      <span>{{ auth.tenantName }}/{{ auth.user?.username }} ({{ auth.user?.role }})</span>
      <button @click="handleLogout">Logout</button>
    </div>
  </header>
</template>

<script setup>
import { useAuthStore } from '../stores/auth'
import { useRouter } from 'vue-router'
const auth = useAuthStore()
const router = useRouter()
async function handleLogout() {
  await auth.logout()
  router.push('/login')
}
</script>
