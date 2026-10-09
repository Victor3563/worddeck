import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'

// В разработке Vite проксирует /api на backend:
// VITE_API_PROXY=http://localhost:8080 npm run dev
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const target = env.VITE_API_PROXY || 'http://localhost:8080'
  return {
    plugins: [vue()],
    server: {
      port: 5173,
      proxy: {
        '/api': { target, changeOrigin: true },
      },
    },
    build: {
      target: 'es2020',
      sourcemap: false,
    },
  }
})
