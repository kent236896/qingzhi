/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_APP_TITLE: string
  readonly VITE_APP_ENV: string
  readonly VITE_API_BASE_URL: string
  readonly VITE_API_TIMEOUT_MS: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
