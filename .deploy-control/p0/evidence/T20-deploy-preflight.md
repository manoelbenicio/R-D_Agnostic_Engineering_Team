# T20 deploy PREFLIGHT — STANDBY (ORQ1). CORRECTED: loopback-only OTLP, no OTEL creds. NOTHING deployed.

- agent: Opus48#D · lane: deploy-preflight · task: T20-DEPLOY-PREFLIGHT(-CORR) · pane `w8:p2`
- check-in: `.deploy-control/p0/checkins/Opus48-D__T20-DEPLOY-PREFLIGHT-CORR__20260723T235213Z.json`
- as-of (UTC): 2026-07-23T23:52Z · target: orq1 `/tmp/multica-auth`
- **STATUS: STANDBY / DRAFT. Nothing executed — no build, copy, restart, `at`, SSH, env-set, chmod, or observation. No secrets read/printed.**
- **HARD GATE: DO NOT deploy until Kiro signals green build.** `control.json` now: `green_build`/`build_signal`=unset, `deploy_authorized`=unset. Standby holds.
- canonical runbook (single source, reused): `.deploy-control/p0/evidence/FIX-deploy-prep.md`; tier-20 env: `T20-deploy.md`.

> CORRECTION (2026-07-23T23:51Z): the OTLP receiver is **loopback-only `127.0.0.1`** with a **fixed,
> daemon-injected endpoint** (`internal/daemon/observability/otlpreceiver` — loopback-only OTLP logs,
> body not modeled → content-off by design). There are **NO OTEL auth headers, NO external OTEL
> credentials, NO mTLS, and NO OTEL owner/blocker.** My earlier "OTEL owner/credential/mTLS BLOCKED"
> item was incorrect and is withdrawn.

## Preflight verifies ONLY these five items (nothing else):

### 1. Artifact checksum
`/tmp/multica-auth-fixed` sha256 `1abda82d8dc70abe91a068a4bb7519bdc3226001d79384014ac8cb3cfe59d6c8`
(ELF 64-bit, mtime 2026-07-23T22:22). Re-verify against Kiro's green-build record and after transfer to
`orq1:/tmp/multica-auth.new` before any swap:
```bash
sha256sum /tmp/multica-auth-fixed            # expect 1abda82d...
ssh orq1 'sha256sum /tmp/multica-auth.new'   # MUST match
```

### 2. Local export-file path exists
The dedicated JSONL span export file(s) the collector consumes (`observability/e2e` JSONLSink,
daemon-configured path). Verify presence only (no content read):
```bash
ssh orq1 'test -f <JSONL_EXPORT_FILE_PATH> && echo present || echo missing'
```
`<JSONL_EXPORT_FILE_PATH>` is the daemon-configured dedicated export path (operator confirms from the
daemon config; not invented here).

### 3. Perms 0600 on the local export file
```bash
ssh orq1 'stat -c "%n %U:%G %a" <JSONL_EXPORT_FILE_PATH>'   # expect owner-only, 600
```
Remediate to `0600`/owner-only before deploy if permissive. No content is ever read.

### 4. Content-off flags present
Confirm telemetry stays content-off / metadata-only (no bodies, prompts, results, identities):
the daemon enforces this by design (otlpreceiver ignores the log-record body; telemetry schema
requires versioned + content-off; e2e spans carry `secrets_present=false`). Preflight confirms **no
env/config override disables content-off** — i.e., the content-off telemetry flags remain ON and no
raw-content/debug-body export flag is set.

### 5. Rollback plan
Reuse canonical `FIX-deploy-prep.md` §2–§3: keep `/tmp/multica-auth.prev` (copy of the current live
binary) before swap; rollback = swap `.prev` back + relaunch with the PRESERVED launcher (runtimes
`claude-code` + `claude_code_kimi_2.7_Code` + `AGENT_BRAIN_*` + tier-20 env) + health gate (`:20128` + `:8080`).

## OmniRoute secret-file — UNCHANGED, values NEVER read
`/etc/agent-brain/secrets/omniroute-inference-key` handling is unchanged (owner-provisioned, required
`0600`). This lane does not read, print, hash, copy, or modify its value — only the daemon consumes it.

## Deploy sequence (on green build + authorization; nothing now)
Canonical `FIX-deploy-prep.md` §1 preserve launcher → §2 stage `/tmp/multica-auth-fixed` (sha256
`1abda82d…`) + checksum + `.prev` rollback → §3 `at` restart + health gate. Tier-20 concurrency stays
fail-closed at the dev admission limit until the separate 9.1/9.2 capacity acceptance.

## Go/No-go gate
- [ ] Kiro **green-build signal** (unset now); artifact digest `1abda82d…` matches green-build record.
- [ ] `deploy_authorized=true` + authorized orq1 operator session.
- [ ] §2 local export-file present, §3 `0600`, §4 content-off flags ON.
- [ ] OmniRoute health OK; gateway secret file present + `0600` (handling unchanged; value never read).

## Non-claims / scope
- Nothing executed: no build/scp/ssh/env-set/chmod/swap/`at`/restart/observe/inference/secret-read.
- OTLP is loopback-only, daemon-fixed; no OTEL auth/headers/creds/mTLS/owner (prior blocker withdrawn).
- OmniRoute secret values never read. OmniRoute internals never probed (readiness only).
- No product/OpenSpec edit; no commit. Single canonical runbook preserved (no divergent copy).
