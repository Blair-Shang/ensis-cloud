import { defineConfig, type Plugin } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { resolve } from 'node:path'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

const __dirname = fileURLToPath(new URL('.', import.meta.url))
const root = resolve(__dirname, '..')
const niumaUiRoot = resolve(root, '../niuma-ui')
const niumaUiSrc = resolve(niumaUiRoot, 'src')

/** 浏览器默认请求 /favicon.ico（不带 /ops 前缀），开发时回落到 svg */
function faviconFallback(): Plugin {
  return {
    name: 'ops-favicon-fallback',
    configureServer(server) {
      server.middlewares.use((req, res, next) => {
        const url = req.url?.split('?')[0] ?? ''
        if (
          url !== '/favicon.ico' &&
          url !== '/favicon.svg' &&
          url !== '/niuma/ops/favicon.ico' &&
          url !== '/niuma/ops/favicon.svg'
        ) {
          next()
          return
        }
        try {
          const svg = readFileSync(resolve(__dirname, 'public/favicon.svg'))
          res.statusCode = 200
          res.setHeader('Content-Type', 'image/svg+xml')
          res.setHeader('Cache-Control', 'public, max-age=86400')
          res.end(svg)
        } catch {
          next()
        }
      })
    },
  }
}

export default defineConfig({
  root: __dirname,
  base: '/niuma/ops/',
  plugins: [vue(), tailwindcss(), faviconFallback()],
  resolve: {
    alias: [
      {
        find: '@niuma/ui/styles.css',
        replacement: resolve(niumaUiSrc, 'styles.css'),
      },
      {
        find: '@niuma-ui-src',
        replacement: niumaUiSrc,
      },
      {
        find: '@',
        replacement: resolve(__dirname, 'src'),
      },
    ],
  },
  optimizeDeps: {
    include: ['vue', 'vue-router'],
  },
  server: {
    port: 5174,
    warmup: {
      clientFiles: [
        './src/main.ts',
        './src/App.vue',
        './src/router/index.ts',
        './src/views/login/LoginView.vue',
        './src/components/ui.ts',
      ],
    },
    fs: {
      allow: [root, niumaUiRoot],
    },
    proxy: {
      // 只转发 Cloud API；/niuma/ops/ 必须由本机 Vite 出，否则会打到未启动的 :8090
      '/niuma/cloud': { target: 'http://127.0.0.1:8090', changeOrigin: true },
    },
  },
  build: {
    outDir: resolve(root, 'server/internal/web/dist'),
    emptyOutDir: true,
    target: 'es2022',
  },
})
