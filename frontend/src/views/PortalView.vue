<template>
  <div class="portal-page">
    <div class="portal-hero">
      <h2>Welcome, {{ auth.user?.username }}</h2>
      <p>{{ auth.tenantName }} · {{ auth.user?.role === 'admin' ? 'Platform Admin' : auth.user?.role }}</p>
    </div>
    <div v-if="loading" class="loading-state"><div class="spinner"></div> Loading...</div>
    <div class="card-grid" v-else>
      <template v-for="e in entries" :key="e.id">
        <div class="entry-card" :class="{ disabled: !e.enabled }" @click="e.enabled && go(e)">
          <div class="card-icon">{{ icons[e.id] || '📦' }}</div>
          <div class="card-label">{{ e.label }}</div>
          <div class="card-badge" v-if="!e.enabled">Disabled</div>
          <div class="card-badge" v-else-if="e.id==='admin'">Admin</div>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import api from '../utils/api'

const router = useRouter()
const auth = useAuthStore()
const entries = ref([])
const loading = ref(true)
const icons = { scripts: '📝', sql_runner: '🔍', admin: '⚙️' }

onMounted(async () => {
  try {
    const res = await api.get('/api/bff/entries')
    entries.value = res.data.entries || res.data
  } catch {} finally { loading.value = false }
})

function go(e) {
  if (e.url?.includes('ts-quickjs')) router.push('/scripts')
  else if (e.url?.includes('sql-runner')) router.push('/sql')
  else if (e.url?.includes('admin-server')) router.push('/admin')
  else window.open(e.url, '_blank')
}
</script>
