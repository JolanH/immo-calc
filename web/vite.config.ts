import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  server: {
    host: '127.0.0.1',
    port: 5173,
    strictPort: true,
    // make dev runs the Go server next to Vite; the API goes to it.
    proxy: { '/api': 'http://127.0.0.1:8421' },
  },
  test: {
    environment: 'jsdom',
  },
})
