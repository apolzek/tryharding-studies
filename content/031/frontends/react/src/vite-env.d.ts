/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_OTLP_ENDPOINT?: string;
  readonly VITE_FARO_URL?: string;
  readonly VITE_REPLAY_ENDPOINT?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
