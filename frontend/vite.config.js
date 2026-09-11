import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5175,
    proxy: {
      // Proxy API calls to the Go backend during dev so requests are
      // same-origin (avoids CORS entirely, including for custom headers
      // like X-Filename). Production is unaffected: the Go binary serves
      // the built frontend and /api from the same origin already.
      '/api': {
        target: 'http://localhost:8085',
        changeOrigin: true,
      },
    },
  },
})
