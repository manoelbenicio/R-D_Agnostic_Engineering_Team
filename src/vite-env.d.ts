/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** GO Core Server base URL. Default: http://127.0.0.1:8080 */
  readonly VITE_GO_CORE_BASE_URL?: string;
  readonly VITE_ALLOW_CANVAS2D?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
