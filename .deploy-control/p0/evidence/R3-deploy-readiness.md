# R3 — Deploy Readiness + Task 6.1 Revision/Readiness Evidence (REC-DEPLOY-READINESS)

- agent: `Codex56#A` · lane: `R3` · task: `REC-DEPLOY-READINESS` · pane: `w7:p3`
- lock: `.deploy-control/p0/evidence/R3-deploy-readiness.md` (only mutable artifact)
- read-only verification scope: `multica-auth-work/server/internal/daemon/deploy/**`; read-only 6.1 source: `gateway/health_models.go`, `gateway/registry.go`
- covers OpenSpec: **5.2** (targeted tests) · **6.1** (immutable revision + protocol/model readiness, evidence-only)
- MODE: **VERIFICATION-ONLY / EVIDENCE-ONLY**. No product edits, **zero inference, zero secret, zero deploy** (`implementation_authorized=false`, `live_runs.*=false`).

> STATUS: **VERIFIED**. Deploy package is green (fmt/vet/tests). Task 6.1 readiness/revision
> contract is present and fail-closed by construction (evidence below). No checkbox closed (Principal adjudicates).

---

## 0. Preflight

| Field | Value |
|---|---|
| cwd | `/home/ec2-user/workspace/R-D_Agnostic_Engineering_Team` |
| git HEAD | `a6d50986aee7f1981a323b29c9d90f175b4b6f8b` |
| git status count | 300 |
| go | `go1.26.1` (`/home/ec2-user/goroot/go/bin/go`) |
| gofmt | present (`/home/ec2-user/goroot/go/bin/gofmt`) |
| node | `v22.23.1` |
| disk_free | 866M (97% used) — no clone/install performed |

---

## 1. Deploy package — targeted verification (read-only)

Package files: `README.md`, `doc.go`, `rollout.go`, `runbooks.go`, `secret_reference.go`, `topology.go`,
`deploy_test.go`, `runbook_trigger_test.go`. (`secret_reference.go` handles secret **references**, not values — consistent with the no-secret invariant.)

| Command | Exit code | Result |
|---|---:|---|
| `/home/ec2-user/goroot/go/bin/gofmt -l internal/daemon/deploy` | 0 | empty (all formatted) |
| `/home/ec2-user/goroot/go/bin/go vet ./internal/daemon/deploy/...` | 0 | PASS (clean) |
| `/home/ec2-user/goroot/go/bin/go test ./internal/daemon/deploy/...` | 0 | `ok` 0.002s |
| `/home/ec2-user/goroot/go/bin/go test -count=1 ./internal/daemon/deploy/...` | 0 | `ok` 0.008s (uncached) |
| `... test -count=1 -v ... \| grep -c '^=== RUN'` | 0 | **6** top-level tests ran |

(cwd for the above = `multica-auth-work/server`.) Deploy package is fmt-clean, vet-clean, and its 6
targeted tests pass uncached. No source edited.

---

## 2. Task 6.1 — immutable OmniRoute revision + protocol/model readiness (evidence-only)

Source of truth (read-only): `server/internal/daemon/gateway/health_models.go`
`ReadinessChecker.CheckGatewayReadiness` + `registry.go` immutable snapshot.

**Distinct, ordered, fail-closed readiness gates** (each a separate boolean on `brain.ReadinessSnapshot`,
not collapsed into a single "up" check):
1. **Liveness** — `client.CheckLiveness` → `snapshot.Live=true`. (Liveness probe is unauthenticated per `gateway_test.go`.)
2. **Catalog/auth readiness** — `client.CheckReadiness` → `snapshot.Authenticated=true`.
3. **Model registry ready (immutable revision)** — `registry.Snapshot(ctx)`; `snapshot.ModelRegistryReady = registrySnapshot.Version != ""`. The `Version` is the OmniRoute registry revision carried by header `X-OmniRoute-Registry-Version` (`HeaderRegistryVersion`) / body `registry_version`.
4. **Selected model ready** — only if the exact `request.RouteModel` exists in the snapshot **and** `model.Available` → `snapshot.SelectedModelReady=true`.
5. **Selected protocol ready** — `snapshot.SelectedProtocolReady = model.Capability.Protocol == request.Protocol` (per-model protocol match, not catalog-wide).
Then `policy.Evaluate(snapshot)` under the **strict, fail-closed** policy (constructor rejects any
non-`ReadinessStrict`/non-`FailClosed` policy).

**Immutable revision integrity:** the registry snapshot is built immutably (`registry.go`: "building
this immutable registry snapshot"); a revision mismatch between the `X-OmniRoute-Registry-Version`
header and the body `registry_version` is rejected as `ErrorProtocol` and does **not** echo bounded
upstream details (evidenced by `g4_priority0_boundaries_test.go` — header `synthetic-header-v2` vs body
`synthetic-body-v1` → class `ErrorProtocol`, no echo).

**6.1 acceptance mapping (per plan §9 L2 / §11):**
- readiness distinguishes liveness / catalog / model / protocol → **YES** (five distinct booleans, ordered).
- `/v1/models` is not treated as proof of inference/protocol fidelity → **YES** (readiness/catalog is a
  separate gate from `SelectedProtocolReady`; per-model `Available`+`Protocol` required).
- selected-model capability required, unsupported rejected → **YES** (`SelectedModelReady` requires
  `model.Available`; `SelectedProtocolReady` requires exact protocol match; else strict policy fails closed).
- Brain implements no credential/account lifecycle here → **YES** (readiness consumes revision/readiness
  metadata only; no auth/rotation/fallback in this path).
- immutable revision captured → **YES** (`registrySnapshot.Version`, header/body cross-check → `ErrorProtocol` on mismatch).

**Non-inference note:** this is source/code evidence read read-only. No liveness/readiness HTTP probe,
no `/v1/models` call, and no inference were executed. OmniRoute runtime endpoint is not reachable and
is not required for this evidence.

---

## 3. Verification commands + exit codes (summary)

```
cd multica-auth-work/server
gofmt -l internal/daemon/deploy                     -> exit 0 (empty)
go vet ./internal/daemon/deploy/...                 -> exit 0
go test ./internal/daemon/deploy/...                -> exit 0 (ok)
go test -count=1 ./internal/daemon/deploy/...        -> exit 0 (ok, 0.008s)
go test -count=1 -v ... | grep -c '^=== RUN'         -> 6 tests
git diff --check -- .../internal/daemon/deploy/      -> CLEAN (no edits)
```

---

## 4. Blockers / non-claims

- No product edits made or authorized (`implementation_authorized=false`, `authorized_lanes=[]`).
- 6.1 is **evidence-only**; the checkbox is NOT closed — Principal adjudicates from disk evidence.
- No inference, no secret read, no deploy/restart/Docker/systemd, no commit/push, no destructive git.
- Live readiness against a running OmniRoute (actual revision string, live protocol conformance) remains
  gated by `live_runs.*=false` + BLK-AVAIL + D-V3-25(B); out of scope this window.
- Server-wide build/test is L1/L8 scope, not run here.
- Prior assignment P0-GLM-LIVE-ACCEPTANCE remains BLOCKED; no lock overlap with `deploy/**` or this file.
