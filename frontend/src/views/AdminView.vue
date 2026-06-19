<template>
  <div class="admin-layout">
    <div v-if="error" class="error-text" style="margin-bottom:12px">{{ error }}</div>
    <div v-if="!auth.isAdmin" class="error-text">Access denied: admin role required.</div>
    <template v-else>
      <h2>Tenants</h2>
      <div class="admin-row">
        <input v-model="newTenantName" placeholder="Tenant name" />
        <input v-model="newTenantLabel" placeholder="Display label" />
        <button class="btn-primary" @click="createTenant">Create</button>
      </div>
      <table class="admin-table">
        <thead><tr><th>ID</th><th>Name</th><th>Label</th><th>Created</th><th></th></tr></thead>
        <tbody>
          <tr v-for="t in tenants" :key="t.id">
            <td class="muted">{{ t.id.substring(0,16) }}...</td>
            <td>{{ t.name }}</td>
            <td>{{ t.label }}</td>
            <td class="muted">{{ t.createdAt?.substring(0,10) }}</td>
            <td>
              <button class="btn-green" style="padding:2px 8px;font-size:11px" @click="selectTenant(t)">Users</button>
              <button class="btn-yellow" style="padding:2px 8px;font-size:11px;margin-left:4px" @click="editBetamap(t)">Betamap</button>
            </td>
          </tr>
        </tbody>
      </table>

      <div v-if="selectedTenant">
        <h2>Users — {{ selectedTenant.name }}</h2>
        <div class="admin-row">
          <input v-model="newUsername" placeholder="Username" />
          <input v-model="newPassword" placeholder="Password" type="password" />
          <select v-model="newRole">
            <option value="user">user</option>
            <option value="tenant_admin">tenant_admin</option>
          </select>
          <button class="btn-primary" @click="createUser">Create User</button>
        </div>
        <table class="admin-table">
          <thead><tr><th>ID</th><th>Username</th><th>Role</th><th>Created</th><th></th></tr></thead>
          <tbody>
            <tr v-for="u in selectedUsers" :key="u.id">
              <td class="muted">{{ u.id.substring(0,16) }}...</td>
              <td>{{ u.username }}</td>
              <td>{{ u.role }}</td>
              <td class="muted">{{ u.createdAt?.substring(0,10) }}</td>
              <td>
                <button class="btn-yellow" style="padding:2px 8px;font-size:11px" @click="resetPwd(u)">Reset Pwd</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="betamapTenant">
        <h2>Betamap — {{ betamapTenant.name }}</h2>
        <textarea v-model="betamapJson" style="width:100%;height:200px;font-family:monospace;font-size:12px"></textarea>
        <br/><br/>
        <button class="btn-primary" @click="saveBetamap">Save Betamap</button>
        <button class="btn-blue" style="margin-left:8px" @click="closeBetamap">Cancel</button>
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
const newTenantName = ref('')
const newTenantLabel = ref('')
const selectedTenant = ref(null)
const selectedUsers = ref([])
const newUsername = ref('')
const newPassword = ref('')
const newRole = ref('user')
const betamapTenant = ref(null)
const betamapJson = ref('')
const error = ref('')

onMounted(loadTenants)

async function loadTenants() {
  const res = await api.get('/api/admin/tenants')
  tenants.value = res.data
}

async function createTenant() {
  if (!newTenantName.value) return
  await api.post('/api/admin/tenants', { name: newTenantName.value, label: newTenantLabel.value })
  newTenantName.value = ''; newTenantLabel.value = ''
  loadTenants()
}

async function selectTenant(t) {
  selectedTenant.value = t
  const res = await api.get(`/api/admin/tenants/${t.id}/users`)
  selectedUsers.value = res.data
}

async function createUser() {
  if (!newUsername.value || !newPassword.value) return
  await api.post(`/api/admin/tenants/${selectedTenant.value.id}/users`, {
    username: newUsername.value, password: newPassword.value, role: newRole.value
  })
  newUsername.value = ''; newPassword.value = ''
  selectTenant(selectedTenant.value)
}

async function resetPwd(u) {
  const pwd = prompt('New password for ' + u.username)
  if (!pwd) return
  await api.put(`/api/admin/users/${u.id}/password`, { password: pwd })
}

async function editBetamap(t) {
  const res = await api.get(`/api/admin/tenants/${t.id}/betamap`)
  betamapTenant.value = t
  betamapJson.value = typeof res.data === 'string' ? JSON.stringify(JSON.parse(res.data), null, 2) : JSON.stringify(res.data, null, 2)
}

async function saveBetamap() {
  try {
    JSON.parse(betamapJson.value)
  } catch { error.value = 'Invalid JSON'; return }
  await api.put(`/api/admin/tenants/${betamapTenant.value.id}/betamap`, JSON.parse(betamapJson.value))
  betamapTenant.value = null
  error.value = ''
}

function closeBetamap() { betamapTenant.value = null; error.value = '' }
</script>
