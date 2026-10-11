import { defineConfig, type Plugin } from 'vite'
import vue from '@vitejs/plugin-vue'
import { ensisUiHost } from '@niuma/ui/vite-plugins/ensis-ui-host'
import { createRequire } from 'node:module'
import { dirname, resolve } from 'node:path'
import { existsSync, readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

const __dirname = fileURLToPath(new URL('.', import.meta.url))
const root = resolve(__dirname, '..')
const require = createRequire(import.meta.url)

/** 本机有同级源码则联调；CI / 打包从 npm 的 ensis-ui 解析。 */
function resolveNiumaUiRoot(): string {
  const sibling = resolve(root, '../ensis-ui')
  if (existsSync(resolve(sibling, 'package.json'))) {
    return sibling
  }
  try {
    return dirname(require.resolve('@niuma/ui/package.json'))
  } catch {
    return dirname(require.resolve('ensis-ui/package.json'))
  }
}

const niumaUiRoot = resolveNiumaUiRoot()

/** 浏览器默认请求 /favicon.ico（不带 /ops 前缀），开发时回落到 svg */
function faviconFallback(): Plugin {
  return {
    name: 'ops-favicon-fallback',
    configureServer(server) {
      server.middlewares.use((req, res, next) => {
        const raw = req.url ?? ''
        // Vite 会用 ?import / ?url 把 svg 转成 JS 模块；这些请求必须交给 Vite，不能回原始图片
        if (/[?&](?:import|url|raw)(?:=|&|$)/.test(raw)) {
          next()
          return
        }
        const url = raw.split('?')[0] ?? ''
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

/**
 * ensisUiHost：只服务 pnpm dev（源码 HMR）。vite build 走包主入口。
 * 不要把 @niuma/ui 别名到 src/index.ts。
 */
export default defineConfig({
  root: __dirname,
  base: '/niuma/ops/',
  plugins: [vue(), ...ensisUiHost(), faviconFallback()],
  resolve: {
    alias: [
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
