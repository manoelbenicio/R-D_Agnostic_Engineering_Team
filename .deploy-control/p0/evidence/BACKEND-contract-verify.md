# BACKEND contract verify — native-vs-gateway branch + gateway-required fail-closed

- agent `Opus48#B` · lane `BACKEND-VERIFY` · pane `w6:p2` · task `BACKEND-CONTRACT-VERIFY`
- check-in receipt: `.deploy-control/p0/checkins/Opus48-B__BACKEND-CONTRACT-VERIFY__20260722T215632Z.json`
- posture: **READ-ONLY** (no edits). Source inspected at HEAD `a6d50986…`.
- verdict: **VERIFIED** — when `AGENT_BRAIN_DEVELOPMENT_ENABLED` + gateway-required, no native account path is reachable.

## Verified logic — `internal/daemon/daemon.go` `runTask` admission gate (~3240–3290)

1. `entry, provider, err := d.resolveTaskAgentEntry(task, provider)` (`:3240`) — in gateway mode this
   resolves the **immutable built-in CLI** (per `agentBrainBuiltInCLIFor`); it does not select a native
   provider account. Launch-identity errors are classified + observed and returned (fail closed).
2. `:3260` `if d.cfg.AgentBrain.DevelopmentEnabled && d.cfg.AgentBrain.Neutral.Gateway.Required && d.agentBrainInitErr != nil`
   → **fail closed** `integration_initialization_failed` (returns before any launch).
3. `:3263` `agentBrainPlan, err := d.agentBrain.admitTask(ctx, task, provider, legacyModel)` — on `err`
   → return `TaskResult{}, err` (fail closed; admission error observed).
4. `:3274` `if agentBrainPlan != nil` → `task.RuntimeRouterOwner = RouterOwnerOmniRoute` (`:3276`),
   `entry.Model = RouteModel`, `defer recordTerminal`. **Gateway path.** Downstream, legacy
   rotation/credential-account work is gated by `agentBrainPlan != nil` (`:3440`, `:3706`), i.e.
   disabled on the gateway path.
5. `else if d.cfg.AgentBrain.DevelopmentEnabled && d.cfg.AgentBrain.Neutral.Gateway.Required` (~`:3282`)
   → **fail closed** `gateway_required` **before any workdir/StartTask**; inline comment: *"No provider
   credential or account rotation is reintroduced here."*
6. Only when **NOT** gateway-required (or dev disabled) does a nil plan fall through to the legacy/native
   execution path — the intended native branch, outside the gateway contract.

## admitTask fail-closed classes — `internal/daemon/brain_integration.go`

`enabled()` (`:142`) = `DevelopmentEnabled && Gateway.Required`. In gateway-required mode `admitTask`
returns **either a valid plan (`:266` `admitted`) or a typed error with nil plan**, covering:
`adapter_fail_closed` (`:164`), `trusted_profile_unavailable` (`:169`), `route_model_not_approved`
(`:176`), `legacy_contract_rejected` (`:191`), **`credential_source_unavailable` (`:198–199`, returns
nil,err — no native credential fallback)**, `gateway_client_invalid` (`:207`), `readiness_checker_invalid`
(`:236`), `readiness_cancelled` (`:244`), `capability_rejected` (`:254`). Every non-admitted outcome is a
fail-closed error; a nil-plan-without-error in gateway-required mode is caught by daemon.go step 5.

## Conclusion (contract holds)
- Gateway-required + dev-enabled: init-error → fail closed; admitTask → valid OmniRoute plan (native
  rotation/credential-home disabled via `plan != nil` gates) **or** typed fail-closed error; nil plan →
  fail closed `gateway_required` before any launch. **No native provider/account path is reachable.**
- Native/legacy path is reachable **only** when gateway-required is false (or dev disabled) — as designed.

## UI smoke — PENDING daemon readiness (not executed)
Real web UI smoke (chat/Kanban/status) against the ORQ frontend could NOT run: readiness probes at
`2026-07-22T21:56Z` returned **daemon `http://127.0.0.1:8080/health` = 000 (unreachable)** and
**frontend `http://127.0.0.1:3100` = 000 (unreachable)**. UI smoke is BLOCKED until the daemon/frontend
are up. Next: on daemon ready, run read-only HTTP/UI assertions (health 200; chat send omits/【picker】
sends agent; Kanban task create/list; status/terminal) and capture concrete HTTP codes — no source edits,
no capacity tests, no secrets.

## Non-claims
Static source verification only (no runtime/live proof; daemon down). No edit. No inference/secret/deploy/
capacity/commit/push. UI smoke not executed (daemon/frontend unreachable).
