import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    proxy: {
      // Proxy REST API calls to the Go backend
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      // Proxy WebSocket connections to the Go backend.
      // Use http:// target (Vite upgrades to WS). Do NOT rewrite Origin — backend
      // CORS allows localhost/127.0.0.1 any port when CORS_ORIGINS is unset.
      '/ws': {
        target: 'http://localhost:8080',
        ws: true,
        changeOrigin: true,
      },
    },
  },
})
