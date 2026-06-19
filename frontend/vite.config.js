import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    proxy: {
      '/api/auth':    'http://127.0.0.1:9722',
      '/api/admin':   'http://127.0.0.1:9723',
      '/api/scripts': 'http://127.0.0.1:9720',
      '/api/sql':     'http://127.0.0.1:9721',
      '/api/bff':     'http://127.0.0.1:9724',
    }
  }
})
