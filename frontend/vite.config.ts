/// <reference types="vitest/config" />
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { defineConfig } from 'vite'

// Dev-only proxy so the browser sees one origin (frontend + API), matching
// prod where atlas-server serves both. See docs/plans/atlas-frontend/02-architecture.md.
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      '/traces': 'http://127.0.0.1:8080',
      '/discovery': 'http://127.0.0.1:8080',
      '/healthz': 'http://127.0.0.1:8080',
    },
  },
  build: {
    outDir: 'dist',
  },
  test: {
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    globals: true,
  },
})
