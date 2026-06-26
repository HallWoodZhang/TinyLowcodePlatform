<template>
  <div class="login-page">
    <div class="login-card">
      <div class="login-logo">
        <div class="icon">⚡</div>
        <h1>Tiny Lowcode Platform</h1>
        <div class="subtitle">Sign in to your workspace</div>
      </div>
      <div class="form-group">
        <label>Tenant</label>
        <input v-model="tenant" placeholder="Enter tenant name" @keyup.enter="handleLogin" />
      </div>
      <div class="form-group">
        <label>Username</label>
        <input v-model="username" placeholder="Enter username" @keyup.enter="handleLogin" />
      </div>
      <div class="form-group">
        <label>Password</label>
        <input v-model="password" type="password" placeholder="Enter password" @keyup.enter="handleLogin" />
      </div>
      <div class="form-error">{{ error }}</div>
      <button class="btn btn-primary" :disabled="loading" @click="handleLogin">
        {{ loading ? 'Signing in...' : 'Sign In' }}
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
  if (!tenant.value || !username.value || !password.value) { error.value = 'All fields required'; return }
  loading.value = true; error.value = ''
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
