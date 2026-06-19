<template>
  <div style="padding-top: 20px;">
    <div class="muted" style="text-align:center" v-if="loading">Loading...</div>
    <div class="card-grid" v-else>
      <a v-for="e in entries" :key="e.id" class="card" :class="{ disabled: !e.enabled }" :href="e.enabled ? e.url : '#'" @click.prevent="e.enabled && go(e.url)">
        <div class="icon">{{ icon(e.id) }}</div>
        <div class="label">{{ e.label }}</div>
      </a>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import api from '../utils/api'

const router = useRouter()
const entries = ref([])
const loading = ref(true)

onMounted(async () => {
  try {
    const res = await api.get('/api/bff/entries')
    entries.value = res.data.entries || res.data
  } catch {} finally { loading.value = false }
})

function icon(id) {
  return { scripts: '\u{1F4DD}', sql_runner: '\u{1F50D}', admin: '\u{2699}\u{FE0F}' }[id] || '\u{1F4E6}'
}

function go(url) {
  // map external URLs to Vue routes
  if (url.includes('ts-quickjs')) router.push('/scripts')
  else if (url.includes('sql-runner')) router.push('/sql')
  else if (url.includes('admin-server')) router.push('/admin')
  else window.open(url, '_blank')
}
</script>
