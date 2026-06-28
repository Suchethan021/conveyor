/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Base URL for the backend API; empty means same-origin (proxy). */
  readonly VITE_API_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
