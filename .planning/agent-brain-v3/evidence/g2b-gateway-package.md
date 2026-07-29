# G2B OmniRoute gateway package — EV-G2B-01..07

- Status: IMPLEMENTED and package-verified; not wired, canaried, or route-accepted
- Owner: Codex 2 — OmniRoute Gateway
- OpenSpec: `build-omniroute-agent-brain` tasks 4.1–4.7
- Authorization: architect response §7.1, Waves 0–3, tier 20
- Implementation: `multica-auth-work/server/internal/daemon/gateway/**`

## Provenance

- Host: `manoelneto-laptop`, Linux/amd64
- Repository commit at verification: `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`
- Verification completed: `2026-07-18T01:28:37Z`
- Toolchain: Go `1.26.1` Linux/amd64, matching `server/go.mod` and CI
- Toolchain archive checksum verified before use: SHA-256
  `031f088e5d955bab8657ede27ad4e3bc5b7c1ba281f05f245bcc304f327c987a`
- Sorted gateway file-manifest aggregate SHA-256:
  `14be18029668c4f4d5ff1b1881570b6f54c129f087ff37107d96011a3611410f`

The WSL/host environment had no Go binary on PATH. The exact repository-pinned official
toolchain was staged under `/tmp` only; no project or system toolchain configuration changed.

## Evidence mapping

| Evidence | OpenSpec | Implemented contract | Primary files |
|---|---:|---|---|
| EV-G2B-01 | 4.1 | Configurable base URL, injected reference-scoped authentication, bounded HTTP/body timeouts, cancellation, safe errors, correlation headers, redirect denial | `client.go`, `errors.go` |
| EV-G2B-02 | 4.2 | Separate configurable liveness/readiness probes, authenticated `/v1/models`, deterministic status/transport classification, frozen fail-closed readiness interface | `client.go`, `health_models.go` |
| EV-G2B-03 | 4.3 | Concurrency-safe TTL cache; versioned exact-model registry; explicit protocol/stream/tools/reasoning/structured/context/pool/rotation/affinity/fallback validation | `registry.go` |
| EV-G2B-04 | 4.4 | Evidence-gated trusted profiles for Anthropic Messages, OpenAI Responses, OpenAI Chat Completions, and Antigravity-compatible `/v1/antigravity` | `profiles.go` |
| EV-G2B-05 | 4.5 | OmniRoute-only route policy; independent-request RR/failure-only modes; continuation affinity; bounded pre-commit retry; same-model-first and approved equivalent cross-model fallback; scoped circuits; Smart Context safety flags | `policy.go` |
| EV-G2B-06 | 4.6 | Allowlisted headers/events for actual model/route, request, retry/fallback, quota/circuit and usage; irreversible connection pseudonymization; unknown event fields rejected | `telemetry.go` |
| EV-G2B-07 | 4.7 | SSE contracts and synthetic-only JSON/SSE fixtures for Messages, Responses and Chat Completions | `protocol.go`, `testdata/**` |

The package imports and implements the frozen `agent-brain.v1` contracts directly:
`brain.GatewayConfig`, `brain.SecretFileRef`, `brain.Correlation`, `brain.CLIKind`,
`brain.RouteModel`, `brain.RouterOwner`, `brain.ProtocolFamily`,
`brain.GatewayReadinessChecker`, and `brain.ModelCapabilityRegistry`.

## Verification

Commands ran from `multica-auth-work/server` using the staged Go 1.26.1 binaries:

```text
go test -count=20 ./internal/daemon/gateway
PASS — ok, 1.147s

go test -race ./internal/daemon/gateway
PASS — ok, 1.095s

go vet ./internal/daemon/gateway
PASS — exit 0, no findings

go test ./internal/daemon/brain ./internal/daemon/gateway
PASS — brain cached; gateway 0.074s

gofmt -d internal/daemon/gateway/*.go
PASS — no diff
```

The first repeated run exposed a timeout-classification race between `http.Client.Timeout`
and the request context. Classification was corrected to inspect cancellation/deadline and
`net.Error.Timeout` categories without emitting the underlying error text; the 20-run,
race, vet, and contract-pair matrix then passed.

Additional hygiene checks:

- credential-pattern scan: zero matches;
- production `os.ReadFile`/`os.Open` scan: zero matches;
- forbidden `runtimeenv`/`deploy`/`observability`/Prodex dependency scan: zero matches;
- all fixture model IDs and content are explicitly synthetic;
- no live OmniRoute endpoint, provider account, auth slot, or secret file was contacted.

## Security and scope boundary

`gateway` has no secret-file reader. It retains only the frozen `SecretFileRef` and invokes an
injected `CredentialSource.WithCredential` callback for authenticated requests. Authentication
values are applied only during `http.Client.Do`, removed from the request immediately afterward,
never included in public structs/strings/errors, and redirects are denied.

No central daemon/config/health/cmd entrypoint, frozen `brain/**` file, `runtimeenv/**`,
`deploy/**`, `observability/**`, or Prodex implementation was edited by Codex 2. The active
daemon remains unwired. No cutover, provider traffic, secret mutation/read, Prodex removal,
or tier 50/100 behavior is claimed.

G3 integration and Wave-3 live protocol/failure/tier-20 acceptance remain separate gated work.
