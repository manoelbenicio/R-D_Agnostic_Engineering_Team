# IndexedDB Persistence (D4)

All local persistence goes through a typed interface backed by `idb`.

## Object stores

| Store              | Owner | Key path           | Notes                                    |
| ------------------ | ----- | ------------------ | ---------------------------------------- |
| `canvases`         | CV    | `id`               | Latest snapshot per canvas               |
| `canvas_versions`  | CV    | `[canvas_id, version]` | Append-only history                  |
| `provider_keys`    | IF    | `provider`         | v1 plaintext (see `key-storage-v1.md`)  |
| `settings`         | IF    | `key`              | `{ key, value }`                         |
| `app_state`        | SUP   | `key`              | First-run wizard, misc app booleans      |

## Migrations

`schema_version` (number) lives in `app_state`. Bump on every schema
change. Migrations live in `src/shared/storage/migrations.ts` and run on
DB open.

## Interfaces

| Interface       | File                                  |
| --------------- | ------------------------------------- |
| `CanvasStore`   | `src/canvas-document/store.ts`        |
| `KeyStore`      | `src/api/key-store/index.ts`          |
| `SettingsStore` | `src/settings/settings-store.ts`      |
| `AppStateStore` | `src/shared/storage/app-state.ts`     |

Cloud (Firestore) implementations later replace the IDB-backed instances
without consumer changes (D4).
