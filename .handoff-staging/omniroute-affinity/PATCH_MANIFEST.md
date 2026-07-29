# OmniRoute Responses Affinity DEV Patch

Base image: `diegosouzapw/omniroute@sha256:badb560971fdc23c2fb84b3e8695116239ff215b4cca4b07076201a8efae7f0d`

This is a reversible, non-upstream DEV overlay. It was prepared from a code-only snapshot; `/app/data`, `node_modules`, environment files, and secrets are excluded. The live container was not modified.

## Files

- `src/lib/db/sessionAccountAffinity.ts`: hashed `responses:` handles, transactional SQLite bind/read/touch, TTL expiration, deterministic maximum-size eviction, and restart-persistent ownership.
- `src/sse/services/auth.ts`: internal `requiredConnectionId` constraint which precedes ordinary session affinity and prevents rotation/fallback of a continuation.
- `src/sse/handlers/chat.ts`: resolves `previous_response_id` before provider dispatch; constrains combo/single-provider selection; fails closed for missing, expired, stale, ineligible, or database-unavailable ownership.
- `open-sse/handlers/chatCore.ts`: binds only structurally valid successful non-streaming and streaming Responses IDs; touches the prior handle only after success; fails closed on persistence errors.
- `tests/unit/db/responseConnectionAffinity.test.ts`: origin bind/lookup, raw-handle non-persistence, missing/expired behavior, transactional cap eviction, concurrent inserts, restart persistence, and exact-owner touch.
- `/tmp/omniroute-affinity-patch.js`: unified patch artifact for the code-only snapshot.

## Test commands

From an isolated overlay checkout with the pinned image dependencies available:

```sh
DATA_DIR="$(mktemp -d)" DISABLE_SQLITE_AUTO_BACKUP=true \
  node --import tsx/esm --import ./open-sse/utils/setupPolyfill.ts \
  --test --test-concurrency=1 \
  tests/unit/db/responseConnectionAffinity.test.ts

npm run lint -- \
  src/lib/db/sessionAccountAffinity.ts \
  src/sse/services/auth.ts \
  src/sse/handlers/chat.ts \
  open-sse/handlers/chatCore.ts \
  tests/unit/db/responseConnectionAffinity.test.ts

npm run build:backend
```

The live A/B/A acceptance harness remains frozen until its separate authorization; no paid request is part of this artifact.

## Integrity

See `CHECKSUMS.sha256`. Patch SHA256 is the entry for `/tmp/omniroute-affinity-patch.js`.
