<template>
  <div class="app-shell">
    <header class="app-header">
      <router-link to="/" class="brand">
        <div class="logo-icon">⚡</div>
        <span>Tiny Lowcode</span>
      </router-link>
      <nav v-if="auth.token">
        <router-link to="/scripts">Scripts</router-link>
        <router-link to="/sql">SQL</router-link>
        <router-link v-if="auth.isAdmin" to="/admin">Admin</router-link>
      </nav>
      <div class="user-area" v-if="auth.token">
        <div class="user-tag">
          <div class="user-avatar">{{ (auth.user?.username || '?')[0].toUpperCase() }}</div>
          <span>{{ auth.tenantName }}/{{ auth.user?.username }}</span>
          <span style="color:var(--text-muted)">· {{ auth.user?.role }}</span>
        </div>
        <button class="btn btn-ghost btn-sm" @click="handleLogout">Logout</button>
      </div>
    </header>
    <router-view />
  </div>
</template>

<script setup>
import { useAuthStore } from './stores/auth'
import { useRouter } from 'vue-router'
const auth = useAuthStore()
const router = useRouter()
async function handleLogout() {
  await auth.logout()
  router.push('/login')
}
</script>
