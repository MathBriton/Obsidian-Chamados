/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Base URL da API. Vazio em dev (usa o proxy do Vite). */
  readonly VITE_API_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
