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
      // Proxy WebSocket connections to the Go backend
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true,
        changeOrigin: true,
        // Rewrites the Origin header to match the target host (localhost:8080)
        // so Go's websocket.Upgrader default CheckOrigin accepts the connection.
        // Without this, the proxy forwards Origin: localhost:5174 → Go rejects it → EPIPE
        rewriteWsOrigin: true,
      },
    },
  },
})
