# DISPATCH_QUEUE — Historical G1/G2 + current G3 dispatch

> Historical G1/G2 entries are immutable audit records and MUST NOT be replayed. Current
> planning owner: Kiro/Opus-4.8. Current operational co-lead/transport/verification owner:
> Codex#56#A. Only the latest READY entry may be dispatched. Created 2026-07-17; leadership
> and phase reconciled 2026-07-18.

## Transport instructions for Codex#56#A

1. Read this file.
2. For each entry below, identify the target pane in workspace `w3` (labels:
   `codex1-brain`, `codex2-gateway`, `codex3-runtime`, `codex4-ops`). If a pane does not
   exist yet, create it via `herdr pane split --current --direction right --no-focus`,
   label it, and start the normal `codex` executable interactively. Wait for `idle`.
3. Submit the exact `herdr pane run <pane-id> "<prompt>"` string shown below.
4. Record the resulting pane-id and timestamp back into `AGENT_LEDGER.md` under the
   respective agent row (Codex#56#A transport notes).
5. Do NOT edit GSD files, do NOT change task assignments, do NOT write product code.

## Entry 1 — Codex 1 (Brain core / integrator)

**Target pane label:** `codex1-brain`
**Task-IDs:** 1.1, 1.2, 1.6, 2.1, 2.2, 2.3, 2.4
**Files locked (exclusive):**
- `multica-auth-work/server/internal/daemon/daemon.go`
- `multica-auth-work/server/internal/daemon/config.go`
- `multica-auth-work/server/internal/daemon/health.go`
- `multica-auth-work/server/cmd/multica/cmd_daemon.go`
- `multica-auth-work/server/go.mod`
- `multica-auth-work/server/internal/daemon/execenv/execenv.go`
- `multica-auth-work/server/internal/daemon/execenv/codex_home.go`
- `multica-auth-work/server/pkg/agent/models.go`
- `multica-auth-work/server/internal/daemon/prodex.go`
- `multica-auth-work/server/internal/daemon/prodex_fs_linux.go`
- `multica-auth-work/server/internal/daemon/prodex_fs_other.go`
- `multica-auth-work/server/internal/daemon/prodex_profiles.go`
- `multica-auth-work/server/internal/daemon/l2_runtime.go`

**Evidence IDs:** EV-G1-01 (neutral contracts), EV-G1-02 (CLIKind/RouteModel/RouterOwner), EV-G1-03 (neutral config + aliases), EV-G1-04 (compatibility facade), EV-G1-05 (worktree audit + ownership)

**Exact prompt:**
```
You are Codex 1 — Lead Integrator / Brain Core for the Agent Brain program. Authorization: OMNIROUTE_ARCHITECT_RESPONSE.md §7.1 = AUTORIZADO, Waves 0-3, tier 20. PD-01 resolved: the existing dirty worktree from change persist-prodex-runtime-integration must be PRESERVED as an audited security baseline. Do NOT reset, stash, revert, or discard any uncommitted changes. Your exclusive hotspots are: daemon.go, config.go, health.go, cmd_daemon.go, go.mod, execenv/execenv.go, execenv/codex_home.go, pkg/agent/models.go, plus prodex.go, prodex_fs_linux.go, prodex_fs_other.go, prodex_profiles.go, and l2_runtime.go. No other agent may edit these. Execute tasks 1.1, 1.2, 1.6, 2.1, 2.2, 2.3, 2.4 from phases/G1/PLAN.md: freeze the Agent Brain terminology and cold/hot boundary; define CLIKind/RouteModel/RouterOwner contracts; create neutral types/packages (brain/*) without changing the active daemon path; define neutral config names, legacy alias precedence, gateway-required mode, secret-file reference, readiness policy, and tier schema; define the compatibility facade for legacy daemon API/task token/RouterOwner/env/config/CLI/brief; publish frozen interfaces+ownership to Codex 2-4. First, audit the uncommitted diff, run existing tests, and reconcile the 16 tasks of persist-prodex-runtime-integration without resetting the baseline. Report check-in/out updates to AGENT_LEDGER.md. Evidence IDs: EV-G1-01 through EV-G1-05. Never print secrets, keys, tokens, cookies, prompts, repo content, or tool payloads.
```

## Entry 2 — Codex 2 (OmniRoute gateway)

**Target pane label:** `codex2-gateway`
**Task-IDs:** 1.3 (assist), 1.5 (model-route matrix)
**Files locked:** `gateway/**` (new package)
**Evidence IDs:** EV-G1-MODELMATRIX

**Exact prompt:**
```
You are Codex 2 — OmniRoute Gateway for the Agent Brain program. Authorization: §7.1 = AUTORIZADO, Waves 0-3, tier 20. Execute the gateway-prep portion of G1: produce the per-model route matrix for Claude Code, Codex/OpenAI, Kimi, GLM, NVIDIA, and Antigravity with exact API format, account pool, tools, reasoning, structured output, context limit, rotation/affinity, and fallback chain, based on the OmniRoute architect response and acceptance checklist in openspec/changes/build-omniroute-agent-brain/. Do NOT edit central daemon/config/health entrypoints. Do NOT create the gateway implementation yet (G2B). Output the matrix as a new file under .planning/agent-brain-v3/evidence/g1-model-route-matrix.md and update AGENT_LEDGER.md. Evidence ID: EV-G1-MODELMATRIX. No secrets.
```

## Entry 3 — Codex 3 (Runtime/CLI security)

**Target pane label:** `codex3-runtime`
**Task-IDs:** 1.5 (adapter input), 5.x prep
**Files locked:** `runtimeenv/**` (new package); read-only inspect of `pkg/agent/*.go`
**Evidence IDs:** EV-G1-ADAPTERPREP

**Exact prompt:**
```
You are Codex 3 — Runtime/CLI Security for the Agent Brain program. Authorization: §7.1 = AUTORIZADO, Waves 0-3, tier 20. Execute G1 prep: read-only inspect pkg/agent/{claude,codex,kimi,nim,antigravity}.go and execenv/*.go to document the current provider credential and base-URL injection surface. Produce a contract document at .planning/agent-brain-v3/evidence/g1-runtime-adapter-prep.md listing every provider-native env var, auth file, and base URL that must be denied/removed in gateway-required mode, plus the trusted OmniRoute variables that must be applied last. Do NOT edit central daemon/config/health entrypoints. Do NOT touch secrets or secret files. Update AGENT_LEDGER.md. Evidence ID: EV-G1-ADAPTERPREP.
```

## Entry 4 — Codex 4 (Ops/parity/evidence)

**Target pane label:** `codex4-ops`
**Task-IDs:** 1.3 (checklist), 1.4 (parity matrix), 6.x prep
**Files locked:** `deploy/**`, `observability/**` (new), `EVIDENCE_INDEX.md` (coordination)
**Evidence IDs:** EV-G1-OPS-PREP

**Exact prompt:**
```
You are Codex 4 — Operations/Parity/Evidence for the Agent Brain program. Authorization: §7.1 = AUTORIZADO, Waves 0-3, tier 20. Execute G1 ops prep: (a) complete the OmniRoute architecture acceptance checklist with status Supported/Partial/Not-supported per item, version/image digest, and redacted config notes; (b) complete the Prodex-to-OmniRoute feature parity matrix P01-P34 + SC01-SC10 with target owner, acceptance evidence, and gap remediation plan or waiver flag; (c) prepare the Linux restricted secret-file reference plan and endpoint topology (host loopback 127.0.0.1:20128 vs container DNS) without copying or printing any secret value. Write outputs to .planning/agent-brain-v3/evidence/g1-omniroute-checklist.md and g1-prodex-parity-matrix.md. Update AGENT_LEDGER.md and EVIDENCE_INDEX.md. Evidence ID: EV-G1-OPS-PREP. No secrets.
```

## TL sign-off (G1)

- All four G1 check-ins recorded in `AGENT_LEDGER.md`.
- Locks are disjoint; Codex 1 owns all Prodex hotspots exclusively.
- Codex#56#A transport is authorized to relay prompts verbatim.
- No production code, canary, or cutover until G1 freeze evidence is accepted.

Ready for transport (G1).

---

# G2 — Four-stream implementation (DISPATCHED 2026-07-18 by TL, owner call)

> §7.1 authorizes Waves 0–3/tier 20. G2 is no-secret implementation against frozen G1 contracts.
> PD-08 STOP applies ONLY to credential/auth mutations, secret reads/copies/rewrites, account
> rotation, or unsafe auth tests. PD-01 baseline preserved; no reset/stash/revert/discard.
> No Prodex removal, no cutover, no tiers 50/100. G2 must NOT wire the active daemon into the
> new execution path yet (wiring is G3, Codex1 serial, hotspot-only). All four streams produce
> isolated, contract-conformant code; shared entrypoint edits are Codex1-only.

## Entry G2-A — Codex 1 (Brain core / coordinator)

**Target pane:** `w3:pD` (label `codex1-brain`)
**Task-IDs:** 3.1, 3.2, 3.3, 3.4, 3.5
**Files locked (exclusive):** the coordinator/task-executor/runtime-registry package under
`multica-auth-work/server/internal/daemon/<neutral>` (new neutral package, distinct from
`brain/**` frozen in G1; choose name consistent with frozen contract, do NOT reuse `brain/`
internals as wire) + `internal/daemon/daemon.go`, `config.go`, `health.go`, `cmd_daemon.go`,
`go.mod`, `execenv/execenv.go`, `execenv/codex_home.go`, `pkg/agent/models.go`, `prodex.go`,
`prodex_fs_linux.go`, `prodex_fs_other.go`, `prodex_profiles.go`, `l2_runtime.go`.
**Must-not-edit:** `gateway/**` (Codex2), `runtimeenv/**` (Codex3), `deploy/**`/`observability/**` (Codex4).
**Evidence IDs:** EV-G2A-01 (coordinator), EV-G2A-02 (CLIKind/RouteModel fields), EV-G2A-03 (compatibility), EV-G2A-04 (admission/fail-closed), EV-G2A-05 (preserve lifecycle).
**Exact prompt:**
```
You are Codex 1 — Lead Integrator / Brain Core for the Agent Brain program. G1 is COMPLETE and frozen. Authorization: OMNIROUTE_ARCHITECT_RESPONSE.md §7.1 = AUTORIZADO, Waves 0-3, tier 20. Execute G2A (OpenSpec tasks 3.1-3.5): create the neutral coordinator/task-executor/runtime-registry package around existing lifecycle interfaces WITHOUT moving provider credential logic and WITHOUT wiring the new execution path into the active daemon yet (wiring is G3, your serial hotspot). Add neutral task fields for CLIKind, RouteModel, RouterOwner, task/session/request correlation and approved route policy (reuse the frozen types from internal/daemon/brain). Add compatibility translations from supported legacy task/config fields into the neutral contract and emit measurable legacy-use events. Add gateway-required admission/readiness states and fail-closed task statuses WITHOUT enabling the new execution path. Preserve current workspace, repository/worktree, cancellation, watchdog, context/skills, stream batching, recovery and terminal-result behavior behind the neutral interfaces. Constraints: preserve PD-01 dirty baseline (no reset/stash/revert/discard); your hotspots are exclusive (daemon.go, config.go, health.go, cmd_daemon.go, go.mod, execenv/execenv.go, execenv/codex_home.go, pkg/agent/models.go, prodex.go, prodex_fs_linux.go, prodex_fs_other.go, prodex_profiles.go, l2_runtime.go, and the new neutral coordinator package); do NOT edit gateway/**, runtimeenv/**, deploy/**, or observability/**. PD-08 STOP applies only to credential/auth mutations and secret reads/copies/rewrites — it does NOT block this no-secret work. No Prodex removal, no cutover, no tiers 50/100. Run the Go suite and focused vet; keep tests passing. Report check-in/out to AGENT_LEDGER.md. Evidence IDs: EV-G2A-01 through EV-G2A-05. Never print secrets, keys, tokens, cookies, prompts, repo content, or tool payloads.
```

## Entry G2-B — Codex 2 (OmniRoute gateway)

**Target pane:** `w3:p8` (label `codex2-gateway`)
**Task-IDs:** 4.1, 4.2, 4.3, 4.4, 4.5, 4.6, 4.7
**Files locked (exclusive):** `multica-auth-work/server/internal/daemon/gateway/**` (new package), protocol fixtures, route-policy types, telemetry parsing types.
**Must-not-edit:** central daemon/config/health/cmd entrypoints, the neutral coordinator package (Codex1), `runtimeenv/**` (Codex3), `deploy/**`/`observability/**` (Codex4).
**Evidence IDs:** EV-G2B-01 (client), EV-G2B-02 (liveness/readiness/models), EV-G2B-03 (registry), EV-G2B-04 (profiles), EV-G2B-05 (route-policy), EV-G2B-06 (telemetry), EV-G2B-07 (fixtures).
**Exact prompt:**
```
You are Codex 2 — OmniRoute Gateway for the Agent Brain program. G1 is COMPLETE and frozen. Authorization: §7.1 = AUTORIZADO, Waves 0-3, tier 20. Execute G2B (OpenSpec tasks 4.1-4.7) in a new package gateway/** WITHOUT editing central daemon/config/health/cmd entrypoints and WITHOUT wiring into the active daemon. Tasks: (4.1) OmniRoute client with redacted authentication, configurable host/container base URL, bounded timeouts, cancellation and request/session correlation; (4.2) separate liveness/readiness checks and authenticated /v1/models retrieval with deterministic error classification; (4.3) cached versioned model/capability registry with explicit protocol, tools, reasoning, streaming, context and structured-output validation; (4.4) trusted runtime profiles for Anthropic Messages, OpenAI Responses, OpenAI Chat and the documented Antigravity-compatible route; (4.5) route-policy types for strict independent-request round-robin, continuation affinity, retry deadline, same-model fallback, approved cross-model fallback, circuit behavior and Smart Context flags; (4.6) safe OmniRoute telemetry header/event parsing for actual model/route, pseudonymous connection, retries, fallback, quota/circuit state and usage WITHOUT content or secrets; (4.7) protocol fixtures/contracts for Anthropic Messages/SSE, Responses/SSE and Chat Completions/SSE using SYNTHETIC credentials and content only. Conform to the frozen Codex1 neutral contract types (CLIKind/RouteModel/RouterOwner, gateway config names, secret-file reference). Constraints: your package gateway/** is exclusive; do NOT edit central entrypoints, the neutral coordinator package, runtimeenv/**, deploy/** or observability/**. PD-08 STOP applies only to credential/auth mutations and secret reads/copies/rewrites — use the frozen SecretFileRef reference path only, never read/print a real secret value. No Prodex removal, no cutover, no tiers 50/100. Run Go tests and focused vet in your package. Report check-in/out to AGENT_LEDGER.md. Evidence IDs: EV-G2B-01 through EV-G2B-07. Never print secrets, keys, tokens, cookies, prompts, repo content, or tool payloads.
```

## Entry G2-C — Codex 3 (Runtime/CLI security)

**Target pane:** `w3:p9` (label `codex3-runtime`)
**Task-IDs:** 5.1, 5.2, 5.5, 5.9, 5.10 (no-secret implementation scope); 5.3/5.4/5.6/5.7/5.8 are adapter/credential-bearing and must be implemented as no-secret contract + fail-closed stubs pending PD-08 lift (do NOT read/copy/rewrite a real provider credential; the dedicated Codex child-key NAME may be chosen but the VALUE must never be read/print/copied).
**Files locked (exclusive):** `multica-auth-work/server/internal/daemon/runtimeenv/**` (new package); env sanitizer; pre-launch assert. Coordinate with Codex1 only on shared hotspots (execenv, pkg/agent/models.go) — Codex1 is sole editor of those.
**Must-not-edit:** central daemon/config/health/cmd entrypoints, `gateway/**` (Codex2), `deploy/**`/`observability/**` (Codex4).
**Evidence IDs:** EV-G2C-01 (env builder), EV-G2C-02 (deny/trusted-wins), EV-G2C-03 (pre-launch assert), EV-G2C-04 (Codex config contract), EV-G2C-05 (Claude adapter), EV-G2C-09 (model/thinking gateway-aware), EV-G2C-10 (assert). EV-G2C-06/07/08 (Kimi/NIM/Agy) = contract/stub only under PD-08.
**Exact prompt:**
```
You are Codex 3 — Runtime/CLI Security for the Agent Brain program. G1 is COMPLETE and frozen. Authorization: §7.1 = AUTORIZADO, Waves 0-3, tier 20. Execute G2C (OpenSpec tasks 5.1-5.10) in a new package runtimeenv/** WITHOUT editing central daemon/config/health/cmd entrypoints. Tasks: (5.1) minimal inherited-environment builder that removes provider keys, OAuth/cookie variables, direct-provider base URLs and unsafe gateway overrides; (5.2) expand custom-environment validation to deny provider credentials and routing/auth variables in gateway-required mode, then apply trusted gateway configuration last; (5.10) pre-launch assertion that child environment/config contains only the stable OmniRoute secret and approved local task data; (5.9) make model/thinking validation gateway-aware so approved OmniRoute IDs are accepted without provider-native catalog or credential lookup; (5.5) Claude Code trusted OmniRoute root URL/token environment so internal Claude markers do not leak or override gateway policy; (5.4) a CONTROLLED Codex custom-provider configuration CONTRACT for OmniRoute Responses API, stable-key ENV-NAME lookup, HTTP/SSE transport and correlation headers WITHOUT auth.json and WITHOUT reading/printing any secret value. For credential-bearing adapters (5.3 provider-auth copy removal, 5.6 Kimi/GLM/NVIDIA, 5.7 Kimi registry, 5.8 native Agy): implement only no-secret contract + fail-closed stubs; choose the dedicated Codex child-key VARIABLE NAME (blocker resolved하세요 within no-secret scope; never fall back to OPENAI_API_KEY) but NEVER read, copy, print, or rewrite any real provider credential. Resolve the Codex child-key name explicitly now. Constraints: runtimeenv/** is exclusive; do NOT edit central entrypoints, gateway/**, deploy/** or observability/**; coordinate read-only with Codex1 on execenv and pkg/agent/models.go (Codex1 is sole editor). Preserve PD-01 baseline. PD-08 STOP applies ONLY to credential/auth mutations and secret reads/copies/rewrites — it does NOT block this no-secret work. No Prodex removal, no cutover, no tiers 50/100. Conform to G1 adapter-prep contract (g1-runtime-adapter-prep.md) and frozen Codex1 neutral names. Run Go tests and focused vet. Report check-in/out to AGENT_LEDGER.md. Evidence IDs: EV-G2C-01,02,03,04,05,09,10 as implemented; EV-G2C-06,07,08 as contract/stub. Never print secrets, keys, tokens, cookies, prompts, repo content, or tool payloads.
```

## Entry G2-D — Codex 4 (Ops / evidence / parity / secret reference)

**Target pane:** `w3:pA` (label `codex4-ops`)
**Task-IDs:** 6.1, 6.2, 6.3, 6.5 (spec), 6.7; 6.4 dashboards and 6.6 runbooks as specs/docs under §7.1 no-secret scope.
**Files locked (exclusive):** `multica-auth-work/server/internal/daemon/deploy/**`, `observability/**` (new packages), evidence harness specs, runbooks, capacity docs, `.planning/agent-brain-v3/EVIDENCE_INDEX.md` (coordination).
**Must-not-edit:** daemon, gateway, runtimeenv adapter implementation, central entrypoints.
**Evidence IDs:** EV-G2D-01 (restricted secret reference — reference only, no secret value), EV-G2D-02 (endpoint config), EV-G2D-03 (redacted events schema), EV-G2D-04 (dashboards/alerts spec), EV-G2D-05 (capacity/failure harness spec), EV-G2D-06 (runbooks), EV-G2D-07 (flags/canary/triggers).
**Exact prompt:**
```
You are Codex 4 — Operations/Parity/Evidence for the Agent Brain program. G1 is COMPLETE and frozen. Authorization: §7.1 = AUTORIZADO, Waves 0-3, tier 20. Execute G2D (OpenSpec tasks 6.1-6.7) in new packages deploy/** and observability/** WITHOUT editing daemon, gateway, runtimeenv implementation, or central entrypoints. Tasks: (6.1) Linux permission-restricted OmniRoute service secret REFERENCE — operationally derived from the existing host source WITHOUT copying its value into the repository, image, logs or screenshots (reference-only contract: path, ownership/mode, read-safety, injection, provisioning, rotation, logging/evidence, revocation/failure, backup; never read/print/hash a real secret into general telemetry); (6.2) host/WSL and future container endpoint configuration, reachability prerequisites and service start/recreate instructions WITHOUT hard-coding Docker DNS for the host daemon (host loopback 127.0.0.1:20128 for current; container DNS http://omniroute:20128 future); (6.3) structured redacted events/metrics and correlation schema for admission, gateway readiness, selection, affinity, refresh, quota, 401/403, 429/circuit, retry/fallback, cancellation, usage and overload (schema-versioned, content-off, no account identity); (6.5) synthetic capacity/failure acceptance harness SPECIFICATION for 20/50/100 tasks, protocol mix, streaming, tools, prompt/output sizes, cancellation and account distribution; (6.4) dashboards/alerts SPEC for no eligible accounts, auth refresh failure, 401/403/429/5xx spikes, circuit state, queue growth, resource pressure, latency and error SLOs; (6.6) backup/restore, account/route hot-change, key rotation, upgrade, rollback, incident classification and escalation runbooks; (6.7) staged feature flags, canary cohorts, rollback triggers and evidence locations for every protocol/provider/capacity gate. Constraints: deploy/** and observability/** are exclusive; do NOT edit daemon, gateway, runtimeenv, or central entrypoints; coordinate EVIDENCE_INDEX.md only. Conform to G1 ops artifacts (g1-omniroute-checklist.md, g1-prodex-parity-matrix.md) and frozen Codex1 names. PD-08 STOP applies ONLY to credential/auth mutations and secret reads/copies/rewrites — this secret work is REFERENCE-ONLY (no real value read/copied/printed). No Prodex removal, no cutover, no tiers 50/100 (tier report only after G4-capacity). Run any applicable Go tests/spec validation. Report check-in/out to AGENT_LEDGER.md and update EVIDENCE_INDEX.md. Evidence IDs: EV-G2D-01 through EV-G2D-07. Never print secrets, keys, tokens, cookies, prompts, repo content, or tool payloads.
```

## TL sign-off (G2)

- All locksets disjoint; Codex1 remains sole owner of all Prodex/central hotspots.
- G2 streams produce isolated code against frozen G1 contracts; no active-daemon wiring (G3).
- PD-08 honored: no credential/auth mutations; no-secret work proceeds.
- PD-01 baseline preserved; no reset/stash/revert/discard.
- No Prodex removal, cutover, or tiers 50/100.

COMPLETED (G2); do not replay.

---

# G3 — Serial central integration (READY 2026-07-18)

**Target pane:** `w3:pD` (`codex1-brain`)
**Task IDs:** OpenSpec 7.1–7.10
**Owner:** Codex1, sole editor of all central/hotspot files in `FILE_OWNERSHIP.md`
**Evidence:** EV-G3-WIRE, EV-G3-04, EV-G3-05, EV-G3-06, EV-G3-07

**Exact prompt:**

```text
You are Codex1, sole G3 integrator for Agent Brain v3. G0/G1/G2 are COMPLETE in their authorized scopes. Owner authorization covers Waves 0–3/tier-20 development validation. Execute OpenSpec tasks 7.1–7.10 and only those tasks.

Integrate the frozen brain, gateway, runtimeenv, deploy and observability packages through the central daemon/config/health/command/execenv entrypoints that you exclusively own. Required outcome: one default-off development vertical slice with CLIKind + RouteModel, RouterOwner=omniroute, trusted gateway profile, sanitized child environment/config applied last, readiness/model/protocol admission, redacted correlation through launch/result/error/cancel, and no dual router. For gateway-required tasks, disable Prodex/L2 startup, Go account rotation/retry, provider credential-home preparation and provider-account selection; keep legacy behavior isolated behind an explicit default-off migration flag. Do not enable broad admission.

Hard safety: preserve the full PD-01 dirty baseline; do not reset, stash, revert, discard, delete or rewrite unrelated changes. PD-08 STOP forbids every credential/auth read, copy, rewrite, rotation, quarantine or mutation. Never inspect or print secret values. Do not modify real auth files, provider accounts or OmniRoute credentials. Treat G2C 5.6–5.8 as fail-closed contracts only; do not invent native adapter support. No production action, production canary/soak, cutover, Prodex removal or tier 50/100. Do not dispatch Codex through the current Multica daemon during this task.

Before editing, re-read STATE.md, FILE_OWNERSHIP.md, phases/G3/PLAN.md, the latest ledger rows and the G2 evidence summaries. Check in to AGENT_LEDGER with exact files locked. Review the current dirty diff and reject hidden provider credentials or duplicate routing. Run focused package tests, the full applicable Go suite, vet and the existing credential-isolation harness where supported. Finish with a development-only isolation smoke that uses synthetic/reference-only secrets and proves no provider-native credential/auth.json/direct-provider endpoint reaches the child. If the smoke would require a real secret or auth mutation, stop and report the blocked evidence instead.

Update OpenSpec tasks 7.1–7.10 only when their acceptance is actually met. Produce `.planning/agent-brain-v3/evidence/g3-serial-integration.md` mapping every change/test to EV-G3-WIRE/04/05/06/07, include exact limitations, and check out in AGENT_LEDGER. Do not claim live provider acceptance, production readiness or native adapter completion.
```

Dispatch status: DISPATCHED ONCE to `w3:pD` at 2026-07-18T02:00:41Z after Kiro/Opus-4.8
returned APPROVE. Do not replay. Next action is status-only monitoring and evidence validation.

---

# WAVE B — Parallel foundation (AUTHORIZED — NO-SECRET foundation only; EV-ZERO-OVERLAP ACCEPTED)

> **Wave B authorized 2026-07-19** by Council+Owner and **EV-ZERO-OVERLAP ACCEPTED by
> Codex56-Principal-TL** (independently reproduced at remote-synced commit `4c67ae0`: EV hash
> `de83dc1b…b8e`, FILE_OWNERSHIP hash `763094f4…210`; both changes strict-valid; branch clean 0/0;
> ancestry includes `da42282`). Scope = **NO-SECRET foundation work only** under the frozen Wave B.0
> ownership. HELD: 9.1, PD-08 credential work, key handling/rotation, Prodex activation, cutover,
> production.
>
> **Execution hardening (Codex56-Principal-TL directives):**
> - **One branch + worktree per lane** (created by Codex56-Principal-TL); lanes work isolated.
> - **W5 publishes the correlation API contract FIRST**; dependent OBS callers (W1 OBS-4, W2 OBS-6,
>   W3 OBS-5, W6 OBS-2/8, W7 OBS-3/7) must not finalize against it until published. Other lanes may
>   begin independent acceptance/audit work in the meantime.
> - **OpenCode worker panes (W7 `w5:p1`, W8 `w5:p2`) are WITHHELD / to be replaced** until the
>   exposed `opencode.json` key safety is assured (ties to the held key-rotation work). Reassign W7/W8
>   to safe panes before dispatch; do not run them on OpenCode until cleared.

> Authored by Kiro/Opus-4.8 (planning owner) during the Wave A governance freeze on branch
> `planning/agent-brain-observability-freeze` (recovery SHA `da42282`). Wave A authorized
> planning/governance ONLY. These prompts are **READY** verbatim strings for the 8-lane topology
> (D-V3-18) but MUST NOT be dispatched until the owner authorizes Wave B implementation. Transport
> by Codex#56#A (verbatim, no product edits). Standing STOPs apply to every lane: PD-08 (no
> credential/auth read/copy/rewrite/rotation/mutation; never print secret values); no production /
> canary / soak / cutover; no tiers 50/100; no dual router (R5); preserve the PD-01 baseline (no
> reset/stash/revert/discard); Prodex stays default-OFF recovery-only (D-V3-16, never hot); all
> observability is metadata-only with structural argv redaction (AB-REQ-40); check in/out in
> `AGENT_LEDGER.md` with exact files locked; producer ≠ reviewer ≠ adjudicator. Wiring into the
> central daemon path is W1-only (Wave C). Preserve build-omniroute completed count (51 done) — OBS-* are new/OPEN; total is now 96 (85 base + 11 OBS).

## Wave B.0 — Freeze (TL + Codex#56#A, before any lane edits)
- TL freezes the exact file paths W6 owns (ingress HTTP middleware + WS transport) and W7 owns
  (task-queue repo + terminal-result store), publishes them into `FILE_OWNERSHIP.md`, and removes
  them from every other lane glob.
- Codex#56#A runs the glob-intersection check (each owned path matches exactly one lane) and records
  `EV-ZERO-OVERLAP`. No lane starts until `EV-ZERO-OVERLAP` is recorded.

## Entry W1 — Lead Integrator (pane `codex1-brain`)
**Task-IDs:** OBS-4 (span hooks in central path, calling W5 lib); recovery-mode state-machine design stub (AB-REQ-41, default-OFF, not enabled); prep for Wave C wiring. **Evidence:** EV-OBS-04, EV-REC-MODE (design only).
```
You are W1 — Lead Integrator for Agent Brain v3, sole editor of the central daemon hotspots. Authorization: Wave B foundation only; no cutover, no tier 50/100, no production. Do NOT wire the new execution path yet (central wiring is Wave C). In your exclusive hotspots only (daemon.go, config.go, health.go, cmd_daemon.go, go.mod, execenv/**, pkg/agent/models.go, prodex*.go, l2_runtime.go, brain/**): (1) add the daemon admission/lifecycle observability span (OBS-4) by CALLING the W5 observability/e2e library — emit task_id/session_id/launch_id, admission decision, readiness-gate result, CLIKind/RouteModel labels and fail-closed classification, metadata-only; (2) author the platform recovery-mode state-machine DESIGN and default-OFF scaffolding at the single runtime-authority select point (health.go:177-184) per AB-REQ-41 — states NORMAL/DEGRADED/RECOVERY, Prodex default-OFF, mutually exclusive, operator-gated, single router owner, session-boundary transitions, DEGRADED fail-closed and NEVER auto-promoting Prodex; do NOT enable recovery mode and do NOT make Prodex hot. Preserve the PD-01 baseline (no reset/stash/revert). PD-08 STOP: no credential/auth read/copy/rewrite/rotation; never print secret values. Prodex is retained as default-OFF cold recovery mode only (D-V3-16) — never delete, never hot. Run focused package tests + vet; keep tests passing. Check in/out to AGENT_LEDGER.md with exact files locked. Evidence: EV-OBS-04 (span) and EV-REC-MODE (state-machine design/scaffold only). Never print secrets, keys, tokens, cookies, prompts, repo content, or tool payloads.
```

## Entry W2 — OmniRoute Gateway (pane `codex2-gateway`)
**Task-IDs:** 8.1, 8.4, 8.5, 8.6, 8.7 (gateway side); OBS-6. **Files:** `gateway/**`. **Evidence:** EV-G4-01/04/05, EV-OBS-06.
```
You are W2 — OmniRoute Gateway for Agent Brain v3. Authorization: Wave B foundation only; no cutover/tiers/production. In gateway/** only (do NOT edit central entrypoints, coordinator, runtimeenv, deploy, observability/e2e): advance the protocol/failure acceptance from the gateway side — 8.1 authenticated models/capabilities + one non-streaming and one streaming completion per approved protocol family; 8.4 strict concurrent round-robin for independent requests + affinity for Responses continuation/prompt-cache/tool turns; 8.5 expired/revoked/quota/401/403/account-scoped 429/provider-global 429/5xx/timeout/malformed handling; 8.6 pre-commit safe retry, no replay after partial output/tool action, dedup, cancellation slot release; 8.7 account add/remove/quarantine/re-entry + OmniRoute restart/config rollback under load. Then implement OBS-6: the OmniRoute/provider span from SAFE telemetry only (extends task 4.6) — actual route/model, pseudonymous account/connection, selection reason, retries/fallback, quota/circuit state, safe usage, joined on request_id↔omni_request_id — by CALLING the W5 observability/e2e library; metadata-only, no content or secrets. Use synthetic/reference credentials only; PD-08 STOP; never make Prodex hot. Run gateway tests + vet. Check in/out to AGENT_LEDGER.md. Evidence: EV-G4-01/04/05 and EV-OBS-06. Never print secrets, keys, tokens, cookies, prompts, repo content, or tool payloads.
```

## Entry W3 — Runtime/CLI Security (pane `codex3-runtime`)
**Task-IDs:** 8.2, 8.3 (accepted-path + child-env isolation); OBS-5; 5.6–5.8 remain fail-closed. **Files:** `runtimeenv/**`, `pkg/agent/{claude,codex,kimi,nim,antigravity}.go` (coordinated). **Evidence:** EV-G4-ADP, EV-G4-03, EV-OBS-05.
```
You are W3 — Runtime/CLI Security for Agent Brain v3. Authorization: Wave B foundation only; no cutover/tiers/production. In runtimeenv/** and the coordinated pkg/agent adapter files only (do NOT edit central entrypoints, gateway, deploy, observability/e2e): advance 8.2 (Claude/Codex accepted paths with tools/reasoning/cancellation/usage/deterministic errors; Kimi/GLM/NVIDIA/NIM/Agy stay deterministic fail-closed — do NOT invent native support, 5.6–5.8 remain fail-closed contracts) and 8.3 (child environments/task homes/process trees/logs/diagnostics contain no provider-native credentials, auth files or direct-provider endpoints). Then implement OBS-5: the CLI-process span — launch/exit, exit code, cancellation, and STRUCTURALLY-redacted argv (shape only, never values) joined on launch_id/proc_id — by CALLING the W5 observability/e2e library; metadata-only. PD-08 STOP: never read/copy/print a real provider credential; the OmniRoute child-key name may be referenced but its value never read/printed. Preserve PD-01 baseline. Never make Prodex hot. Run tests + vet. Check in/out to AGENT_LEDGER.md. Evidence: EV-G4-ADP, EV-G4-03, EV-OBS-05. Never print secrets, keys, tokens, cookies, prompts, repo content, or tool payloads.
```

## Entry W4 — Ops/Capacity/Evidence (pane `codex4-ops`)

> **AMENDMENT 2026-07-19:** W4 exclusive scope now ALSO includes the real observability stack
> `multica-auth-work/deploy/observability/**` (Grafana/Prometheus/Alertmanager). OBS-11 acceptance
> requires this stack. **HOLD:** until the amended EV-ZERO-OVERLAP is RE-ACCEPTED by
> Codex56-Principal-TL, W4 must NOT edit `multica-auth-work/deploy/observability/**` and must NOT
> claim OBS-11. W4 commit `2c5f4d4` = PRODUCED-NOT-ACCEPTED (insufficient; stack was omitted).
> NO real secret values in `secrets/*.example`.
**Task-IDs:** 8.8; 9.x harness prep (NOT run — gated on G4-OBS PASS); OBS-11. **Files:** `deploy/**`, `observability/dashboards/**`, harness specs, `EVIDENCE_INDEX.md`. **Evidence:** EV-G4-08, EV-OBS-10 (co), EV-OBS-11.
```
You are W4 — Ops/Capacity/Evidence for Agent Brain v3. Authorization: Wave B foundation only; no cutover/tiers/production; do NOT run any capacity tier (9.x is gated on G4-OBS PASS). In deploy/**, observability/dashboards/** and evidence artifacts only (do NOT edit daemon/gateway/runtime impl, central entrypoints, or observability/e2e): finish 8.8 (record evidence against every OmniRoute checklist and Prodex parity ID; stop cutover for unsupported blocker rows without an approved waiver); prepare (do not execute) the 20-task capacity/failure harness so it runs WITH observability instrumentation enabled and measures span overhead (R30); implement OBS-11 — per-hop latency/error/drop/gap dashboards and alerts using pseudonymous identifiers only, plus the consolidated G4-OBS acceptance bundle that declares PASS only when OBS-1..OBS-10 are each independently accepted, OBS-9 shows a continuous trace per synthetic task, and OBS-10 is clean; and co-own OBS-10 with W5 (structural leak scan). PD-08 STOP; secret work is reference-only. Never make Prodex hot. Check in/out to AGENT_LEDGER.md and update EVIDENCE_INDEX.md. Evidence: EV-G4-08, EV-OBS-10 (co), EV-OBS-11. Never print secrets, keys, tokens, cookies, prompts, repo content, or tool payloads.
```

## Entry W5 — E2E Correlation library + leak-scan (new pane `codex5-obs-e2e`)
**Task-IDs:** OBS-1, OBS-9, OBS-10. **Files:** `internal/daemon/observability/e2e/**` (new). **Evidence:** EV-OBS-01/09/10.
```
You are W5 — End-to-End Observability library owner for Agent Brain v3. Authorization: Wave B foundation only; no cutover/tiers/production. Create the new package internal/daemon/observability/e2e/** ONLY (do NOT edit any other package; other lanes CALL your library, they do not co-edit it). Implement OBS-1: the versioned metadata-only correlation schema and propagation contract — identifiers request_id, queue_msg_id, task_id, session_id, launch_id, proc_id, omni_request_id, result_id, delivery_id; their join relationships; header/metadata carriers; contract_version; and the secrets_present=false invariant. Implement OBS-9: the trace assembler that joins all eight hops per task, detects gaps/orphans, and proves one continuous trace per synthetic task. Implement OBS-10: the STRUCTURAL (not pattern-only) secret/content leak scanner over all spans/labels/logs that fails closed on any leak. Provide a clean API for the hop owners (W1/W2/W3/W4/W6/W7). Everything metadata-only; no prompts/tool payloads/repo content/reasoning/secrets/cookies/account emails/connection strings. Run package tests + vet. Check in/out to AGENT_LEDGER.md. Evidence: EV-OBS-01, EV-OBS-09, EV-OBS-10. Never print secrets, keys, tokens, cookies, prompts, repo content, or tool payloads.
```

## Entry W6 — Ingress + WS/UI delivery instrumentation (new pane `codex6-ingress-ws`)
**Task-IDs:** OBS-2, OBS-8. **Files:** frozen ingress HTTP middleware + WS transport file(s) from Wave B.0. **Evidence:** EV-OBS-02/08.
```
You are W6 — Ingress and WS/UI delivery instrumentation for Agent Brain v3. Authorization: Wave B foundation only; no cutover/tiers/production. Edit ONLY the exact ingress HTTP middleware file(s) and WebSocket transport file(s) frozen for you in FILE_OWNERSHIP.md during Wave B.0 (do NOT touch squad_briefing*.go, daemon hotspots, or any other lane's files; if you need a file outside your frozen set, stop and escalate to W1). Implement OBS-2: the ingress-API span (hop 1) — method/route, pseudonymous principal, status, latency, joined request_id→task_id, no request/response bodies — by CALLING the W5 observability/e2e library. Implement OBS-8: the WS/UI-delivery span (hop 7) — delivery latency, backpressure/drops, reconnects, joined on session_id/delivery_id, no delivered payload content — by CALLING the W5 library. Metadata-only. PD-08 STOP. Never make Prodex hot. Run tests + vet. Check in/out to AGENT_LEDGER.md. Evidence: EV-OBS-02, EV-OBS-08. Never print secrets, keys, tokens, cookies, prompts, repo content, or tool payloads.
```

## Entry W7 — Queue + terminal-persistence instrumentation (new pane `codex7-queue-persist`)
**Task-IDs:** OBS-3, OBS-7. **Files:** frozen task-queue repo + terminal-result store file(s) from Wave B.0. **Evidence:** EV-OBS-03/07.
```
You are W7 — Queue and terminal-persistence instrumentation for Agent Brain v3. Authorization: Wave B foundation only; no cutover/tiers/production. Edit ONLY the exact task-queue repository file(s) and terminal-result store file(s) frozen for you in FILE_OWNERSHIP.md during Wave B.0 (do NOT touch daemon hotspots, handler, or any other lane's files; escalate to W1 if you need a file outside your frozen set). Implement OBS-3: the DB-queue span (hop 2) — enqueue/dequeue timestamps, queue depth, wait time, joined queue_msg_id↔task_id, no task payload content — by CALLING the W5 observability/e2e library. Implement OBS-7: the terminal-persistence span (hop 6) — persist latency, byte/token counts, terminal status, joined task_id/result_id, no result content — by CALLING the W5 library. Metadata-only. PD-08 STOP. Never make Prodex hot. Run tests + vet. Check in/out to AGENT_LEDGER.md. Evidence: EV-OBS-03, EV-OBS-07. Never print secrets, keys, tokens, cookies, prompts, repo content, or tool payloads.
```

## Entry W8 — Governance + Prodex cold-recovery disposition + sibling closure (pane `codex8-governance`)
**Task-IDs:** sibling reopened-task evidence (chat 1.2/1.3, cred-iso 4.3/4.4/5.4, native 1.5/1.6/1.7); recovery-mode disposition support; zero-overlap proof support. **Files:** OpenSpec change docs + evidence artifacts only (GSD authored by Kiro TL). **Evidence:** per sibling task.
```
You are W8 — Governance and sibling-change closure for Agent Brain v3. Authorization: Wave B foundation only; documentation/spec/evidence only — do NOT edit product code and do NOT author GSD .planning files (Kiro TL authors those). Produce independent-reviewer evidence and disposition drafts for the reopened sibling work that gates push readiness: chat-orchestration 1.2/1.3, agent-credential-isolation 4.3/4.4/5.4, native-runtimes-onboarding 1.5/1.6/1.7. For each, provide truthful producer≠reviewer≠adjudicator provenance, exact file/hash manifests, and honest reviewed/implemented/verified/accepted classification — never fabricate acceptance. Support the Prodex cold-recovery disposition (D-V3-16) and the zero-overlap proof (EV-ZERO-OVERLAP) with any spec/doc reconciliation needed. PD-08 STOP; no secrets. Never make Prodex hot. Check in/out to AGENT_LEDGER.md. Never print secrets, keys, tokens, cookies, prompts, repo content, or tool payloads.
```

## Promotion order (gates)
Wave B.0 freeze + `EV-ZERO-OVERLAP` → Wave B lanes (W1–W8 parallel) → Wave C (W1 serial central wiring of OBS spans + recovery-mode scaffold) → **G4-OBS stop-gate (OBS-1..OBS-11 PASS, D-V3-17)** → tier-20 capacity (9.1/9.2; needs owner 9.1 A1–F3) → G5 parity → G6 cutover + Prodex quiesced to cold recovery mode. Each gate: TL adjudication + independent review.

Dispatch status: **AUTHORIZED — NO-SECRET foundation only (EV-ZERO-OVERLAP ACCEPTED @ 4c67ae0).** Order: W5 publishes the correlation API contract first; all other lanes may start independent acceptance/audit immediately; OBS callers finalize against the W5 contract once published. Central wiring of spans + recovery-mode scaffold remain Wave C (W1 serial). W7/W8 held off OpenCode panes pending key safety. Still HELD: 9.1, PD-08, key rotation, Prodex activation, cutover, production.


---

# WAVE B.1 — D-V3-19 minimal TEST-ownership (Council-unanimous 2026-07-19) + Owner P0 priority amendment (D-V3-21)

> Planning-only ownership amendment EXECUTED and PASS (see `evidence/ev-zero-overlap-wave-b0.md` Wave B.1
> section + `FILE_OWNERSHIP.md` Wave B.1 amendment). Four NEW `*_test.go` paths added to W6/W7; no
> source/schema/migration/shared-anchor transfer. **Owner priority (D-V3-21):** W6/W7 observability
> implementation is **Priority 2 / DEFERRED** — must NOT consume Priority-0 Main Brain capacity until P0 is
> functionally complete/integrated/tested/running OR explicitly reauthorized. Holds intact.

## Entry W6.T — Ingress/Delivery span TESTS (PREP-ONLY; deferred behind P0)
**Target lane:** W6 (`codex6-ingress-ws`). **NEW owned files:** `internal/middleware/obs_ingress_test.go`,
`internal/daemonws/obs_delivery_test.go`. **Covers:** `obs_ingress.go`/`obs_delivery.go` (`a715b0a`).
**Status:** **PREPARE-ONLY.** May be prepared now that EV-ZERO-OVERLAP PASSED; **DO NOT dispatch/implement**
against Priority-0 Main Brain capacity until P0 complete or explicit reauth (D-V3-21). Safeguards: no
`TestMain`/no side-effecting `init()`; `obs_delivery_test.go` is package-coupled to W1 anchor `hub.go`
(Wave C) → must not edit/force-change `hub.go`; target the frozen span-helper contract only. Producer ≠
reviewer ≠ adjudicator. NOT DISPATCHED.

## Entry W7.T — Queue/Persist span TESTS (HELD; design accepted, deferred behind P0)
**Target lane:** W7 (`codex7-queue-persist`). **NEW owned files:** `internal/service/obs_queue_test.go`,
`internal/service/obs_persist_test.go`. **Covers:** `obs_queue.go`/`obs_persist.go` (**not yet produced**).
**Status:** **HELD.** Zero-schema W7 design ACCEPTED as architecture (D-V3-21) but implementation is
**Priority 2 / DEFERRED behind P0 Main Brain**. Test authoring blocked until the source helper contract
exists and is frozen AND P0 reauthorization. Package-coupled to W1 anchor `task.go` (Wave C) → subordinate
to W1 serial integration; no `TestMain`/no side-effecting `init()`; target frozen helper contract only.
NOT DISPATCHED.

## Wave B.1 sign-off
- EV-ZERO-OVERLAP re-run at CURRENT planning HEAD: file-glob PASS (∩=0, 4 paths absent + single-lane) +
  Go-package/TestMain coupling identified/safeguarded (W1-serial). Both OpenSpec changes strict-valid.
- **Coordination focus per Owner: non-observability Main Brain gaps (Priority 0).** D-V3-20 functional
  tests proceed pre-G4-OBS.
- Holds intact: 9.1/capacity/PD-08/keys/Prodex activation/cutover/production/canary/soak/tier 50/100.


---

# P0 INTEGRATION QUEUE — D-V3-26 (PREPARED 2026-07-19; NOT DISPATCHED)

> Written per Owner Decision 7 (council-unanimous). **PREPARE-ONLY.** Authoritative integration is **HELD**
> until active W1/W3/W4 commits finish AND independent reviews pass. **W1 = sole serial integrator.**
> `main b657129` remains UNTOUCHED (no main merge). Deferred **W6/W7/promexport EXCLUDED from P0**.

## Foundation
- **W5 `fd4aa4d`** = canonical foundation (technical integration baseline; Principal gofmt/test/race/vet PASS
  + independent Gemini structural PASS). All OBS callers cherry-picked this chain.

## Integration order (integrate the LATEST INDEPENDENTLY-REVIEWED commit per lane)

| # | Lane | Latest INDEPENDENTLY-REVIEWED (integrate) | Current tip (PENDING review → gate) | Gate |
|---|---|---|---|---|
| I0 | W5 | `fd4aa4d` (canonical foundation) | `fd4aa4d` | ready as foundation |
| I1 | W1 | `9745eaf` (Gemini static + Principal/Codex daemon test+vet PASS; cross-lane-clean) | **`3711eb4`** "cover credential isolation offline" | HELD — `3711eb4` needs independent review (synthetic/no-secret per D-V3-25C) |
| I2 | W2 | `7a2a808` (independent Gemini review PASS) | **`528d1bb`** "cover priority-zero failure boundaries" | HELD — `528d1bb` needs independent review |
| I3 | W3 | *none integrable* (`0ba88da` blocked on W1 Wave C; no accepted OBS-5) | origin **`1716186`** | HELD — no reviewed integrable commit; OBS-5 blocked |
| I4 | W4 | `47c693c` (promtool 14-rule + Codex review PASS; **OBS-11 PRODUCED-NOT-ACCEPTED** — functional/ops portion only) | **`0a291d9`** "RolloutPlan triggers→OperationsCatalog runbooks" | HELD — `0a291d9` needs independent review; OBS-11 acceptance still gated on promexport (D-V3-23) |

> "Required W3/W4" = only the independently-reviewed **functional/P0** portions; OBS-11 acceptance and any
> exporter/dashboards remain gated (D-V3-17/23/24) and are NOT closed by integration.

## Mandatory safeguards (before any branch update)
1. **Disposable latest-tip dry-run FIRST** (throwaway worktree; never on `main` or the protected branch).
2. **Duplicate W5 patches deduped by `git patch-id`** (the lanes cherry-picked the W5 chain — collapse duplicates).
3. **File-by-file ownership conflict resolution — NO `ours`/`theirs` bulk strategy.**
4. **No force push.** No history rewrite, no `gc`/`prune`.
5. **Independent reviewer ≠ adjudicator ≠ producer** (truthful distinct provenance).
6. **Full offline build + test + race + vet + smoke + provenance PASS BEFORE updating the branch.**
7. W1 is the sole serial integrator; lanes do not self-integrate.

## Explicit non-authorizations (D-V3-26)
- Does **NOT** authorize: main merge · live credentials · 9.1/capacity · Prodex activation · cutover ·
  production/canary/soak · tiers 50/100. All holds intact.

## Dispatch status
**PREPARED / NOT DISPATCHED.** Branch `integration/agent-brain-p0` NOT yet created. Authoritative
integration begins only after the active W1 (`3711eb4`)/W3 (`1716186`)/W4 (`0a291d9`) commits finish and
their independent reviews pass, then TL/Principal adjudication.
