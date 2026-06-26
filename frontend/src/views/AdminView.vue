<template>
  <div class="admin-layout">
    <div v-if="error" class="toast" @click="error=''">{{ error }}</div>

    <div v-if="!auth.isAdmin" class="empty-state">Access denied: admin role required.</div>
    <template v-else>
      <!-- tenants -->
      <div class="section-title">Tenants</div>
      <div class="admin-row">
        <input v-model="tName" placeholder="Tenant name" />
        <input v-model="tLabel" placeholder="Display label" />
        <button class="btn btn-primary btn-sm" @click="createTenant">Create Tenant</button>
      </div>
      <table class="admin-table">
        <thead><tr><th>ID</th><th>Name</th><th>Label</th><th>Created</th><th>Actions</th></tr></thead>
        <tbody>
          <tr v-for="t in tenants" :key="t.id">
            <td class="mono">{{ t.id.slice(0,16) }}…</td>
            <td>{{ t.name }}</td>
            <td>{{ t.label }}</td>
            <td>{{ (t.createdAt||'').slice(0,10) }}</td>
            <td>
              <button class="btn btn-ghost btn-xs" @click="showUsers(t)">Users</button>
              <button class="btn btn-ghost btn-xs" @click="editBetamap(t)">Betamap</button>
            </td>
          </tr>
        </tbody>
      </table>

      <!-- users -->
      <div v-if="selTenant">
        <div class="section-title">Users — {{ selTenant.name }}</div>
        <div class="admin-row">
          <input v-model="uName" placeholder="Username" />
          <input v-model="uPass" placeholder="Password" type="password" />
          <select v-model="uRole"><option value="user">user</option><option value="tenant_admin">tenant_admin</option></select>
          <button class="btn btn-primary btn-sm" @click="createUser">Add User</button>
        </div>
        <table class="admin-table">
          <thead><tr><th>ID</th><th>Username</th><th>Role</th><th>Actions</th></tr></thead>
          <tbody>
            <tr v-for="u in selUsers" :key="u.id">
              <td class="mono">{{ u.id.slice(0,16) }}…</td>
              <td>{{ u.username }}</td>
              <td><span class="card-badge">{{ u.role }}</span></td>
              <td><button class="btn btn-warning btn-xs" @click="resetPwd(u)">Reset Pwd</button></td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- betamap editor -->
      <div v-if="bpTenant" class="betamap-editor">
        <div class="section-title">Betamap — {{ bpTenant.name }}</div>
        <textarea v-model="bpJson"></textarea>
        <div class="actions">
          <button class="btn btn-primary btn-sm" @click="saveBetamap">Save</button>
          <button class="btn btn-ghost btn-sm" @click="bpTenant=null">Cancel</button>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useAuthStore } from '../stores/auth'
import api from '../utils/api'

const auth = useAuthStore()
const tenants = ref([])
const tName = ref(''); const tLabel = ref('')
const selTenant = ref(null); const selUsers = ref([])
const uName = ref(''); const uPass = ref(''); const uRole = ref('user')
const bpTenant = ref(null); const bpJson = ref('')
const error = ref('')

onMounted(async () => {
  try { const r = await api.get('/api/admin/tenants'); tenants.value = r.data } catch {}
})

async function createTenant() {
  if (!tName.value) return
  await api.post('/api/admin/tenants', { name: tName.value, label: tLabel.value })
  tName.value = ''; tLabel.value = ''
  const r = await api.get('/api/admin/tenants'); tenants.value = r.data
}

async function showUsers(t) {
  selTenant.value = t; bpTenant.value = null
  const r = await api.get(`/api/admin/tenants/${t.id}/users`); selUsers.value = r.data
}

async function createUser() {
  if (!uName.value || !uPass.value) return
  await api.post(`/api/admin/tenants/${selTenant.value.id}/users`, { username: uName.value, password: uPass.value, role: uRole.value })
  uName.value = ''; uPass.value = ''
  showUsers(selTenant.value)
}

async function resetPwd(u) {
  const p = prompt('New password for ' + u.username)
  if (!p) return
  await api.put(`/api/admin/users/${u.id}/password`, { password: p })
}

async function editBetamap(t) {
  const r = await api.get(`/api/admin/tenants/${t.id}/betamap`)
  bpTenant.value = t; selTenant.value = null
  bpJson.value = JSON.stringify(typeof r.data === 'string' ? JSON.parse(r.data) : r.data, null, 2)
}

async function saveBetamap() {
  try { JSON.parse(bpJson.value) } catch { error.value = 'Invalid JSON'; return }
  await api.put(`/api/admin/tenants/${bpTenant.value.id}/betamap`, JSON.parse(bpJson.value))
  bpTenant.value = null; error.value = ''
}
</script>
