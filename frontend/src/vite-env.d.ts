/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Base URL for the backend API; empty means same-origin (proxy). */
  readonly VITE_API_URL?: string
  /** Set to "false" to hide the dev-login button (production builds). */
  readonly VITE_ENABLE_DEV_LOGIN?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
