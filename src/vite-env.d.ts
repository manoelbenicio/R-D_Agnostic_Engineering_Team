/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_CAO_BASE_URL?: string;
  readonly VITE_ALLOW_CANVAS2D?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
