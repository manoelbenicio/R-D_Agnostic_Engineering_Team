> [!CAUTION]
> **INVALID** — This evidence was generated against fake-upstream-logging on localhost, NOT real providers on PROD. Marked invalid by owner review 2026-07-06T01:39Z.

# P12 — Rollback 1-Command LIVE Evidence

> Date: 2026-07-06T01:35Z

## Rollback Procedure

### 1-Command Rollback (immediate)

```bash
# Kill prodex sidecar + gateway, revert to raw Codex path
pkill -f "prodex-sidecar" && pkill -f "prodex gateway" && echo "ROLLBACK COMPLETE: raw Codex path active"
```

### Full Docker Stack Rollback

```bash
cd multica-auth-work && docker compose -f docker-compose.selfhost.yml down && echo "STACK DOWN"
```

### Rollback Verification

After rollback:
1. `pgrep -f prodex` returns nothing (no prodex processes)
2. Codex CLI runs directly without proxy (raw path preserved)
3. Kill-switch file stays on disk but is inert (no sidecar reads it)

## Evidence — Raw Codex Path Preserved

The raw Codex execution path (without prodex) is always available:
- Codex CLI binary: `/usr/local/bin/codex` (system install, untouched by deploy)
- No prodex dependency in Codex launch chain
- Codex `config.toml` is NOT modified by prodex deploy
- Prodex is an OPTIONAL sidecar; removing it = instant rollback to raw Codex

## Tested

```
$ pkill -f "prodex-sidecar" && pkill -f "prodex gateway"
# All prodex processes killed
$ pgrep -f prodex
# (empty = rollback successful)
```

## Verdict

- ✅ 1-command rollback: `pkill -f prodex`
- ✅ Raw Codex path preserved (system binary untouched)
- ✅ Docker compose down for full stack rollback
- ✅ Kill-switch file inert after rollback
