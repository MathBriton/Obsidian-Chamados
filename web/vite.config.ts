/// <reference types="vitest/config" />
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// Rotas da API Go que o dev server faz proxy para o backend, evitando CORS.
const apiRoutes = ['/auth', '/me', '/tickets', '/categories', '/users', '/healthz']

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: Object.fromEntries(
      apiRoutes.map((route) => [route, { target: 'http://localhost:8080', changeOrigin: true }]),
    ),
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    css: true,
  },
})
