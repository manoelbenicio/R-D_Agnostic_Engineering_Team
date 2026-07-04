# PROJECT PLAN — Multica Account Rotation + Auth + Observability
**Consolidated planning (root copy).** Orchestrator/SME: Opus 4.8.
Last updated: 2026-07-02. Sources of truth: `.deploy-control/` (board) + `docs/project/`.

> Rule of engagement: Opus (orchestrator) writes NO product code — it prepares the
> environment, designs the agentic plan, coordinates agents, and validates every DONE in
> the container (never trusts the tail). Coders work disjoint streams; hotspots are serial.

---

## PART 1 — WHERE WE ARE (proven, verified by Opus)
- **Rotation engine:** state machine, reactive detector, proactive (banner + ledger) —
  unit tests + real-Postgres E2E + live-daemon realtime rotation.
  Evidence: `rotation_events` ...001→...002 `quota_forecast_proactive`
  (`docs/project/realtime-rotation-evidence.md`).
- **Per-account isolation:** `CODEX_HOME` / `XDG_DATA_HOME` / `HOME` via execenv — tested,
  and independently confirmed correct by prior art (subswap uses the same envs).
- **Credential switch:** `CredentialAuthenticator` is REAL — restores per-vendor credential
  files (codex `auth.json`, kiro `data.sqlite3`, antigravity `.gemini/antigravity-cli`).
  It is NOT OAuth/device-login; it restores PRE-PROVISIONED credentials.
- **Stack operative (staging lab):** backend + Postgres 17 (pgvector) up from our checkout;
  `/metrics` enabled. Observability stack UP: Prometheus :9090 (all targets UP),
  Grafana :3000, Alertmanager :9093, postgres-exporter :9187, 4 dashboards provisioned.

## Vendor fleet (7 vendors, 3 detection classes) — see docs/project/BACKLOG-detection.md
- Class A (CLI %, proactive): **Codex** (5h), **Kiro** (credits), **Antigravity** (per-model), **Kimi** (5h).
- Class B (meta-agent, reactive 429): **OpenCode**, **Cline**.
- Class C (spend budget): **Kimchi** (kimchi.dev — spend console, likely A-like per operator).
- Golden rule: never invent a vendor string/endpoint/flag — primary source only.

---

## PART 2 — BLOCKERS BEFORE PROD (must resolve for GA)
| # | Item | Why it blocks | Owner | Gate |
|---|------|---------------|-------|------|
| B1 ✅ DONE (Codex#2, Opus-verified 2026-07-02: token_lifecycle.go — stale-heuristic + login-status checker, full pkg green) — Token lifecycle | Switch only restores files; an EXPIRED token isn't refreshed → rotating to a stale account = dead account. subswap has keepalive. | Opus (arch) → Codex | Rotation to expired-token account is detected/refreshed; test proves it. |
| B2 ✅ DONE (Codex#1, Opus-verified 2026-07-02: daemon.go:274 loud WARN; daemon gate green) — Daemon w/o DATABASE_URL = rotation SILENTLY off | rotationStore nil, no error (Codex#d finding). | Codex | Startup emits loud WARN/error when rotation configured but DATABASE_URL absent. |
| B3 ✅ DONE (Codex#2, Opus-verified 2026-07-02: detector_reactive_ext.go, 6 tests green, false-positive guard) — Reactive detector false-positive | "usage limit reached" fires even with quota left (ChatGPT-seat monthly cap; openai/codex#23994). | Codex | Detector cross-checks /status before treating as exhausted; test. |
| B4 ✅ DONE (GLM-5.2, Opus-verified 2026-07-02: enroll_account.sh/.sql idempotent, real 600 creds on ext4, negative tests) — Real N-account provisioning | staging used cloned auth.json; PROD needs real isolated enrollment. | Opus (runbook) + Codex (script) | Runbook + script enroll ≥2 real accounts/vendor, isolated. |
| B5 | Real AUTH test (pending) | Never exercised login/switch with a REAL vendor account end-to-end. | Opus (env) + Codex (harness) | Real switch proves valid credential swaps and task runs on new account. |

## PART 3 — HARDENING (should-have)
| # | Item | Owner | Deliverable |
|---|------|-------|-------------|
| H1 | Observability operative | — | ✅ DONE (WAVE A): stack up, scrape real, dashboards live. |
| H2 | Expose DAEMON metrics | Codex | Daemon serves its own /metrics (or push) → rotation_total increments live. |
| H3 | Cooldown-return | Codex | Test: exhausted account returns selectable after window reset. |
| H4 | Pool concurrency | Codex | Stress test: K agents, 2 accounts, no lease/ref-count race. |
| H5 | Robustness (subswap patterns) | Codex | Manual-swap-independent-of-quota; atomic swap + rollback snapshot; quota cache stale-fallback. |
| H6 ✅ DONE (GLM-5.2, Opus-verified 2026-07-02: 10 rules live, PostgresDown firing on real signal, no faked metric) — | Alerts validated | Codex/infra | all_accounts_exhausted alert fires in a real empty-pool scenario. |
| H7 ✅ DONE (Codex#3, Opus-verified 2026-07-02: gen_dashboards.py, Grafana loads it, unknown-metric fails loud) — | **Dashboards-as-code generator** — INPUT FORMAT DECIDED: **YAML component spec** (Opus, 2026-07-02: deterministic parse + JSON-Schema-validatable + matches our all-YAML stack; markdown front-matter needs md+yaml parsing and ambiguous body). Optional one-way markdown summary emitted FROM the YAML for humans. | Codex | `scripts/observability/gen_dashboards.py` (or Go cmd): reads a YAML component spec (components → metrics → panels) → emits Grafana dashboard JSON into `deploy/observability/grafana/dashboards/` (already provisioned/auto-loaded). NOT a native Grafana feature — a thin generator over Grafana's real as-code (provisioning). |

## PART 4 — OPS RUNBOOKS
| # | Item | Owner | Output |
|---|------|-------|--------|
| O1 | PROD deploy runbook (stack + daemon + DATABASE_URL + METRICS_ADDR) | Opus | docs/project/prod-deploy-runbook.md |
| O2 | Account enrollment runbook (per vendor, isolated, secure secret) | Opus | docs/project/account-enrollment-runbook.md |
| O3 ✅ DONE (GLM-5.2: observability-runbook.md, H2 gap documented) — | Observability runbook (bring up, dashboards, alerts, what to watch) | Opus | docs/project/observability-runbook.md |
| O4 | Secrets at rest (file 600 vs keyring/KMS) — decision | Opus | docs/project/secrets-at-rest.md |

## PART 5 — FUTURE SCOPE (doesn't block 3-vendor GA)
| # | Item | State |
|---|------|-------|
| F1 | Phase 3 vendors: Kimchi / OpenCode / Cline | Planned (docs/project/SPRINT-NEXT-vendors.md); waiting real Kimchi usage layout. |
| F2 | Detector: "usage limit reached / wait until HH:MM" + weekly window | Researched; implement with B3. |

---

## PART 6 — AGENTIC EXECUTION (MAX PARALLELISM, ZERO COLLISION)
### Hotspots = single-owner, SERIAL (never two agents at once)
`daemon/daemon.go`, `daemon/execenv/execenv.go`, `rotation/contract.go` (frozen),
`metrics/credential_metrics.go`. A hotspot change = its own serial stream, exclusive lock.

### Parallel groups (NEW files, disjoint — run together, no collision)
- **G1 rotation robustness (new files in rotation/):** PR-TOKEN-LIFECYCLE (token_refresh.go, B1),
  PR-DETECT-HARDEN (detector_reactive_ext.go, B3/F2), PR-COOLDOWN (cooldown_return_test.go, H3),
  PR-CONCURRENCY (pool_concurrency_test.go, H4), PR-ROBUST-SUBSWAP (swap_snapshot.go, H5).
- **G2 auth + provisioning:** PR-ENROLL-SCRIPT (scripts/staging/enroll_*, B4),
  PR-AUTH-HARNESS (real_auth_switch_test.go //go:build staging, B5),
  PR-ENROLL-RUNBOOK (O2).
- **G3 observability (config/docs, no product Go):** PR-OBS-ALERTS (alerts.yml, H6),
  PR-OBS-RUNBOOK (O3), PR-DASH-GEN (dashboards-as-code generator, H7 — pending format).
- **G4 ops docs:** PR-DEPLOY-RUNBOOK (O1), PR-SECRETS-DECISION (O4).

### Serial daemon queue (one at a time on daemon.go; parallel with G1–G4)
1. S-DAEMON-DBURL-GUARD (B2) → 2. S-DAEMON-METRICS (H2) → 3. S-INT-TOKEN (wire B1).

### Collision guarantee
G1–G4 are disjoint new files → all parallel. Serial daemon streams share daemon.go →
queued, but run in parallel with G1–G4 (different files). No stream edits a hotspot it
doesn't own; if it must, it STOPS and becomes a serial request.

### Capacity NOW (no hard dependency): 5+ agents simultaneously
- S-DAEMON-DBURL-GUARD [B2] · PR-DETECT-HARDEN [B3] · PR-ENROLL-SCRIPT [B4] ·
  PR-OBS-ALERTS+RUNBOOK [H6,O3] · PR-DEPLOY-RUNBOOK/PR-SECRETS [O1,O4].
- B1 (token) + B5 (auth harness) unlock when a 2nd REAL account exists (credential, not code).

---

## PART 7 — EXECUTION WAVES (recommended order)
- **WAVE A — environment operative:** ✅ DONE (observability up, scrape fixed).
- **WAVE B — real auth + provisioning (B4, B5, O2):** enroll real accounts (start Codex),
  run real login/switch end-to-end, task on new account. Needs 2nd real account.
- **WAVE C — robustness blockers (B1, B2, B3):** token lifecycle, DATABASE_URL guard,
  detector cross-check. 3 green tests + evidence.
- **WAVE D — hardening (H2–H7):** daemon metrics, cooldown-return, concurrency, subswap
  patterns, alerts, dashboards-as-code.
- **WAVE E — final runbooks (O1, O3, O4) + PROD acceptance.**
- **WAVE F — Phase 3 vendors (when prioritized).**

## PART 8 — OPEN DECISIONS FOR THE OWNER
1. **Dashboards-as-code format:** ✅ DECIDED by Opus — YAML component spec (programmatic
   reliability; owner approved "whatever works programmatically"). PR-DASH-GEN specced.
2. **2nd real vendor account** for B1/B5 (a credential, not a decision) — unlocks WAVE B.
3. **Agent count now** — to size the first parallel wave (plan supports 5+).

## PART 9 — DISCIPLINE (non-negotiable)
0. **MANDATORY SIGN-IN / SIGN-OUT (hard gate — not optional):**
   - **BEFORE any work:** the agent MUST create its check-in file in `.deploy-control/`
     named `<AGENT>__<STREAM>__<START_UTC>.md` with `agent:`, `stream:`,
     `started_at:` (UTC ISO 8601, from `date -u +%Y%m%dT%H%M%SZ`), `status: IN_PROGRESS`,
     and `files_locked:`. No editing any file before this exists.
   - **IMMEDIATELY AFTER finishing:** the agent MUST update the SAME file with
     `finished_at:` (UTC timestamp) + `agent:` name confirmed, `status: DONE` (or BLOCKED),
     and paste `build_result`. A stream is NOT complete without the sign-out timestamp + agent name.
   - Opus rejects any DONE that lacks started_at, finished_at, and agent name.
1. Check-in/out in `.deploy-control/` with files_locked BEFORE editing.
2. Never edit a file locked by another; hotspots = exclusive lock + serial.
3. Green in container BEFORE DONE; Opus re-runs and validates.
4. Nothing invented; vendor behavior from primary source only (early adopters).
5. No secrets in logs/commits; credentials by reference; tokens masked.

---
### Index of detailed docs
- Board: `.deploy-control/MASTER_PROD_READINESS.md`, `PARALLELIZATION_PLAN_PROD.md`,
  `REALTIME_E2E_RUNBOOK.md`, `STATUS.md`, `README.md`, per-stream check-ins + prompts.
- Specs: `docs/project/00..07` (overview→test guide), `BACKLOG-detection.md` (vendor bible),
  `SPRINT-NEXT-vendors.md` (Phase 3), `observability-rotation-staging.md`,
  `multica-auth-work/docs/project/realtime-rotation-evidence.md` (realtime proof).