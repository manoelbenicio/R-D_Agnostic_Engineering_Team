# G3 Serial Integration Evidence

Status: complete for the authorized default-off development scope of OpenSpec tasks 7.1–7.10. This is not live-provider acceptance, production readiness, cutover, or native-adapter completion.

## EV-G3-WIRE — Central integration and single-router ownership

- The sole-owner daemon/config/health/command/execenv entrypoints now connect the frozen `brain`, `gateway`, `runtimeenv`, `deploy`, and `observability` contracts.
- `AGENT_BRAIN_DEVELOPMENT_ENABLED` is default off. The gateway slice additionally requires explicit gateway-required, CLI kind, exact route model, secret-file reference, strict readiness, and tier-20 schema settings.
- Development gateway mode registers only the configured Claude Code or Codex frontend and caps active admission at one task; it does not enable broad tier-20 admission.
- Gateway-required tasks translate the legacy task facade into `CLIKind` plus exact `RouteModel`, force `RouterOwner=omniroute`, and resolve one trusted protocol profile.
- Prodex/L2 configuration and startup, rotation-store initialization, provider-account selection, provider credential-home resolution, proactive rotation, rotation retry, and session-resume retry are bypassed for the gateway slice.
- OmniRoute-owned tasks explicitly block every legacy Go rotation path. The gateway and legacy migration modes are mutually exclusive, so one task cannot have two router owners.
- The unchanged legacy runtime remains available only in an alternate development migration mode enabled by the explicit default-off legacy flag. When the Agent Brain development master flag is off, the preserved PD-01 baseline remains inertly unchanged.

## EV-G3-04 — Apply-last credentialless child environment

- The gateway slice starts from the minimal inherited environment, removes provider credentials/direct endpoints/credential roots, validates local and custom keys, prevents overrides of control-token and correlation fields, and applies trusted gateway values last.
- Claude receives the trusted OmniRoute root and one synthetic/reference-scoped credential slot. Codex receives a task-local credentialless home and controlled Responses configuration with an environment-key reference; no shared home, provider auth, or `auth.json` is copied.
- Gateway execution refuses reuse of an unclassified prior workdir and refuses to inspect or rewrite an existing credentialless task root.
- Pre-launch validation checks trusted provenance, the one-secret invariant, controlled home manifest, and CLI configuration before process creation.

## EV-G3-05 — Fail-closed readiness and capability admission

- Admission performs distinct liveness and authenticated readiness probes, authenticated model-registry retrieval, exact model lookup, selected protocol validation, and streaming/tools capability validation.
- Missing credential source, gateway/auth/readiness failure, model mismatch, profile mismatch, unaccepted adapter, custom environment violation, and capability mismatch fail closed before process launch.
- G2C tasks 5.6–5.8 remain fail-closed contracts. No native Kimi, NIM, OpenAI-compatible, or Antigravity adapter support was invented or enabled.

## EV-G3-06 — Redacted correlation and diagnostics

- Task/session/request correlation is derived from bounded digests of identifiers and accompanies admission, readiness, launch, terminal result, error, and cancellation records.
- Structured events use the content-off observability schema. Logs and diagnostics contain only safe correlation, route/profile/status/classification fields.
- Health exposes development enablement, gateway/legacy mode, secret-reference presence as a boolean, one-task admission limit, readiness state, CLI kind, route model, router owner, protocol, trusted profile, last outcome, and measured legacy-use count. It does not expose a secret path or value, authorization, prompt, tool payload, response body, or repository content.

## EV-G3-07 — Development isolation smoke

- A synthetic local gateway exercises liveness, stable-key authentication, exact `/v1/models` capability admission, the trusted Anthropic Messages profile, and the frozen `agy/claude-opus-4-6-thinking` route ID.
- The admitted plan builds the controlled environment and launches a real helper child process without dispatching Codex or the current Multica daemon.
- The child verifies that provider-native keys, direct-provider endpoints, Codex home/auth state, Prodex variables, and L2 variables are absent; the trusted gateway root, one synthetic credential slot, router owner, and correlation are present; and no `auth.json` exists in the synthetic controlled home.
- A separate regression proves the credentialless Codex preparer creates no `auth.json`.

## Verification

- Focused G3 smoke and fail-closed tests: pass.
- Focused daemon/execenv/brain/gateway/runtimeenv/deploy/observability/command tests: pass.
- Focused race test for G3 integration: pass.
- Full Go suite (`go test ./...`): pass.
- Full Go vet (`go vet ./...`): pass.
- Existing synthetic credential-isolation harness: pass.
- Formatting and `git diff --check`: pass.
- Scope audit: zero G3 credential-file readers, zero edits to G2-owned gateway/runtimeenv/deploy/observability packages, and zero worktree deletions.

## Exact limitations

- PD-08 remains active. No file-backed or real-secret credential source was added or exercised. The ordinary command entrypoint supplies no credential source, so an explicitly enabled gateway slice fails closed until a separately authorized service-layer source is injected.
- The smoke uses a synthetic gateway, synthetic/reference-only credential, synthetic capability document, and helper child. It does not execute Claude Code or Codex against a live OmniRoute/provider route and does not claim protocol/provider acceptance.
- Only the frozen Claude Code and Codex contracts are accepted by configuration. The smoke covers Claude Code/Anthropic Messages only.
- Gateway-mode prior-workdir/session reuse is held closed in G3 to avoid inheriting unclassified legacy credential state. Fresh workspace/task lifecycle, cancellation, watchdog, context/skills, stream, recovery interfaces, and terminal reporting remain preserved; credentialless reuse requires a later accepted provenance marker design.
- Managed MCP configuration and non-empty thinking overrides are held closed in this first slice.
- Readiness diagnostics report the most recent admission probe; no independent periodic live-gateway monitor was enabled.
- No production action, canary/soak, cutover, Prodex removal, provider account operation, tier-20 broad admission, or tier 50/100 work occurred.
