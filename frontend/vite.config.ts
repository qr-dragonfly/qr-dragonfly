import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],

  css: {
    preprocessorOptions: {
      scss: {
        additionalData: '@use "./src/styles/variables" as *;\n',
      },
    },
  },

  // Local dev convenience: forward API calls to the Go backends.
  // This avoids CORS and prevents 404s from Vite for /api/* routes.
  server: {
    proxy: {
      '/api/qr-codes': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/api/settings': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/api/admin': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/api/dev': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/api/clicks': {
        target: 'http://localhost:8082',
        changeOrigin: true,
      },
      '/api/users': {
        target: 'http://localhost:8081',
        changeOrigin: true,
      },
      '/api/stripe': {
        target: 'http://localhost:8081',
        changeOrigin: true,
      },
    },
  },
})
