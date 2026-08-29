/// <reference types="vitest/config" />
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { defineConfig } from 'vite'

// Dev-only proxy so the browser sees one origin (frontend + API), matching
// prod where atlas-server serves both. See docs/plans/atlas-frontend/02-architecture.md.
//
// /traces and /discovery are both API paths AND SPA route paths (React
// Router pages at those same URLs). A browser hard-navigation (page load,
// reload) sends Accept: text/html and must fall through to the SPA's
// index.html; only actual API calls (fetch/XHR, Accept: application/json)
// go to the backend. bypass() returning the request's own URL tells the
// proxy to skip and let Vite's own SPA-fallback middleware serve it.
const htmlNavBypass = (req: { headers: { accept?: string }; url?: string }) => {
  if (req.headers.accept?.includes('text/html')) return req.url
}

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      '/traces': { target: 'http://127.0.0.1:8080', bypass: htmlNavBypass },
      '/discovery': { target: 'http://127.0.0.1:8080', bypass: htmlNavBypass },
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
