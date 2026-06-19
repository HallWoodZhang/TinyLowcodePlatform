<template>
  <div class="form-page">
    <div class="form-box">
      <h2>Tiny Lowcode Platform</h2>
      <div class="form-group">
        <label>Tenant</label>
        <input v-model="tenant" placeholder="admin" @keyup.enter="handleLogin" />
      </div>
      <div class="form-group">
        <label>Username</label>
        <input v-model="username" placeholder="admin" @keyup.enter="handleLogin" />
      </div>
      <div class="form-group">
        <label>Password</label>
        <input v-model="password" type="password" placeholder="Password" @keyup.enter="handleLogin" />
      </div>
      <div class="form-error">{{ error }}</div>
      <button class="btn btn-primary" :disabled="loading" @click="handleLogin">
        {{ loading ? 'Logging in...' : 'Login' }}
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()
const tenant = ref('admin')
const username = ref('admin')
const password = ref('admin123')
const error = ref('')
const loading = ref(false)

async function handleLogin() {
  if (!tenant.value || !username.value || !password.value) {
    error.value = 'All fields required'
    return
  }
  loading.value = true
  error.value = ''
  try {
    await auth.login(tenant.value, username.value, password.value)
    router.push('/')
  } catch (e) {
    error.value = e.response?.data?.error || 'Login failed'
  } finally {
    loading.value = false
  }
}
</script>
