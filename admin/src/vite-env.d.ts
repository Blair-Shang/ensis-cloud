/// <reference types="vite/client" />

import 'vue-router'

declare module 'vue-router' {
  interface RouteMeta {
    public?: boolean
    title?: string
    subtitle?: string
  }
}

interface ImportMetaEnv {
  readonly VITE_CLOUD_API_BASE?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
