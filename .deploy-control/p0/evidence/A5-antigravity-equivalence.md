# A5 — Antigravity evidence equivalence audit (REUSE vs STALE)

- agent / identity: **Opus48#C** (immutable)
- lane / task: **A5 / P0-ANTIGRAVITY-EQUIVALENCE**
- pane: `w8:p1` (`$HERDR_PANE_ID`)
- output lock (sole mutable file): `.deploy-control/p0/evidence/A5-antigravity-equivalence.md`
- covers OpenSpec: **5.8**, **8.1**, **8.2**
- check-in: `.deploy-control/p0/checkins/Opus48-C__P0-ANTIGRAVITY-EQUIVALENCE__20260721T222949Z.json`
- audit timestamp (UTC): `2026-07-21T22:33:17Z`
- classification: read-only on product source; **no live run executed**; no Antigravity rerun; no auth/credential/account/quota/failover work; Prodex untouched.

## Decision

> **STALE.** The Antigravity route’s **runtime launch/environment adapter changed
> after all accepted provenance and is unproven on the candidate build.** The
> route-identity / protocol sub-layer is equivalent and reusable, but a material
> field (the child launch environment that determines OmniRoute reachability for
> `agy`) diverged, so net equivalence fails. Per the A5 contract, REUSE requires
> *all* material fields equivalent; one is not → **STALE**, with the single
> minimal scenario named below. The scenario is **not executed here**.

## 0. Preflight (exact versions/paths)

| Tool | Path | Version |
|---|---|---|
| git | `/usr/bin/git` | `2.50.1` |
| python3 | `/usr/bin/python3` | `3.9.25` |
| rg | `/usr/local/bin/rg` | `ripgrep 15.2.0` |
| sha256sum | `/usr/bin/sha256sum` | `coreutils 8.32` |

- Repo root: `/home/ec2-user/workspace/R-D_Agnostic_Engineering_Team`
- **Candidate revision (HEAD):** `a6d50986aee7f1981a323b29c9d90f175b4b6f8b`
- **Accepted-evidence revision (all G-manifests pinned here):** `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`
- `git merge-base --is-ancestor b6571299… HEAD` → **YES** (linear history; candidate is strictly ahead of the accepted revision).
- The five audited source paths are **clean/committed** at HEAD (`git status --porcelain` empty for them); candidate == committed HEAD bytes.

## 1. What "accepted Antigravity evidence" actually is (scope + provenance)

The frozen mapping (`authoritative-route-matrix-D-V3-27.md`, FROZEN 2026-07-19,
row 1) states: *Antigravity ↔ OmniRoute — “Already fully tested with OmniRoute →
treat as FULLY OPERATIONAL. Revalidate provenance/hashes/evidence — NOT redundant
reimplementation.”* and *“Antigravity provenance revalidation (hashes/evidence) is
offline and may proceed now.”* The same matrix binds: **“No task accepted solely
from prose.”** So the operational claim must be pinned to concrete accepted artifacts:

| Dimension | Accepted artifact | Pinned value / hash | Nature of acceptance |
|---|---|---|---|
| Route identity | `EV-G1-MODELMATRIX` (`g1-model-route-matrix.md`) | route `agy/claude-opus-4-6-thinking`; 4 `agy` accounts, **4×200 connectivity CONFIRMED**; capability **PENDING** | connectivity/selection only — **not capability-accepted** |
| Protocol contract | `EV-G4-01` (`g4-gateway-tests.md`) | exact route `agy/claude-opus-4-6-thinking`, `CLIKind=claude-code`, profile `omniroute-anthropic-messages`, endpoint `/v1/messages`; synthetic non-streaming + SSE | **synthetic/offline only**; text: *“does not prove live/native Antigravity behavior”* |
| Protocol build/provenance | `g4-provenance-manifest.md` @ `b657` | `internal/daemon/gateway/profiles.go` = `4824ef0562e8ffdb9ecebc9ac78534ae339dfab52e8faa3adc24eb44fc430ef8` (4,058 B); gateway set digest `3a32c737541fd03b227ef3c7057f82f0bb38bcd68edc419d78545e2ae66e7497`; image `sha256:ae5a2316d12f3e78fd99177dad452e6ad4f240af2d71d57b480c3477f250fec6`; Go `1.26.5`; coverage 82.5% | synthetic package coverage; *“neither proves a live route”* |
| Route projection | `active-accepted-push-candidate-matrix.md` (HOLD-NATIVE) @ `b657` | `pkg/agent/models.go` = `a6957e3e0b4a05050da6dc198049581d6402103d474185d0912f3360e8a7b313` | accepted-manifest hash pin |
| Runtime adapter surface | `EV-G1-ADAPTERPREP` (`g1-runtime-adapter-prep.md`) @ `b657` | cites `antigravity.go:78` = `cmd.Env = buildEnv(b.cfg.Env)`; *“native Agy exposes no source-proven endpoint override; native Agy is not approved”* | read-only prep contract |
| CLI reachability | `antigravity-1.1.4-ipv4-resolver.md` (2026-07-18) @ `b657` | 1.1.4 IPv6 eligibility-check regression; fix = **per-process** `GODEBUG=netdns=cgo agy`; **explicitly “Not implemented: No automatic GODEBUG injection into product code / runtime adapter”** and *“scan of antigravity.go … found no ipv6/netdns/GODEBUG … handling”* | external-CLI diagnosis; workaround NOT in product |

### Excluded from the accepted OmniRoute set (with reason)
- `.deploy-control/evidence/V1-antigravity-gemini-validation.md` — this run is the
  **Prodex L2 sidecar** (`router_owner=rust_l2`, `contract_version=rpp.l2.v1`,
  `127.0.0.1:43292`, provider `google`/`gemini-2.5-pro`). It is the RPP/Prodex
  Phase-3/HOLD path, **not** the OmniRoute Antigravity route. Not part of the
  OmniRoute provenance and not comparable to the P0 candidate. Cited only to be
  explicitly ruled out.

## 2. Candidate vs accepted — exact hash comparison

| Artifact | Accepted (`b657`) SHA-256 | Candidate (`a6d5`) SHA-256 | Verdict |
|---|---|---|---|
| `internal/daemon/gateway/profiles.go` | `4824ef05…c430ef8` | `4824ef0562e8ffdb9ecebc9ac78534ae339dfab52e8faa3adc24eb44fc430ef8` | **EQUIVALENT (byte-identical)** |
| `pkg/agent/models.go` (route projection) | `a6957e3e…60e8a7b313` | `a6957e3e0b4a05050da6dc198049581d6402103d474185d0912f3360e8a7b313` | **EQUIVALENT (byte-identical)** |
| `pkg/agent/antigravity.go` (runtime adapter) | `96ee0c982cab104cd5690eba71b59536f4bef2306c184bf52471198dd36887a1` | `1196c6f4f11b4aff5f8f2b26cf9f602a5738141e38bb945a79813447071b22de` (13,543 B) | **CHANGED → STALE delta** |
| `pkg/agent/antigravity_test.go` | (b657 baseline) | `5154877c088671d548d0ce09531810c4866fb4792bafaec26ea38f0a01a44acb` (12,427 B) | CHANGED (adds `TestAntigravityResolverEnvAddsCgoResolverWhenUnset`) |
| `internal/daemon/execenv/antigravity_home.go` | (aggregate only; not isolable) | `9f116ab69d399e275bae47a0f8daf788a579bb7c4dd2c194748dc977de31cb04` (3,489 B) | not on accepted OmniRoute path (native-token copy; disabled in gateway-required mode) — see §4 |

### The exact material difference (the STALE cause)
`antigravity.go:78`
- accepted `b657`: `cmd.Env = buildEnv(b.cfg.Env)`
- candidate `a6d5`: `cmd.Env = antigravityResolverEnv(buildEnv(b.cfg.Env))`

Plus a **new** `antigravityResolverEnv(...)` function (candidate lines ~305–350)
injecting `GODEBUG=netdns=cgo` (non-overriding) so the native `agy` child uses the
cgo/system resolver and can reach the loopback OmniRoute endpoint on the
IPv6-disabled host. Introduced by commit `7735bdc — "fix(agent): make agy 1.1.4
reachable via OmniRoute with cgo resolver"` (post-`b657`; the only `antigravity.go`
commit after the `aa62401` checkpoint).

This is a **material change to the exact child launch environment that governs
OmniRoute reachability for the Antigravity route**, and it directly contradicts
the accepted `antigravity-1.1.4-ipv4-resolver.md` (which states the injection was
NOT in product code). No accepted manifest pins the new `antigravity.go`
(`1196c6f4…`); all acceptance is at `b657` where the injection is absent.
Therefore the "already fully tested / FULLY OPERATIONAL" claim was established on a
build (`b657`, per-process workaround) that is **not** the candidate build (`a6d5`,
in-product injection). Connectivity (`4×200`) and the resolver evidence both
pre-date the in-product change.

## 3. Per-requirement disposition (5.8 / 8.1 / 8.2)

- **5.8 (Antigravity revalidation half):** **STALE.** Route identity + protocol
  contract (`profiles.go`, `models.go`) **REUSE** (byte-identical to accepted
  pins). Runtime launch adapter (`antigravity.go`) changed and is unproven →
  revalidation of the launch/reachability path is required (one scenario, §5).
  (The Kiro/Opus48 half of 5.8 is A4’s lane, not audited here.)
- **8.1 (model/protocol/availability on changed routes):** protocol/registry
  contract **REUSE** (unchanged `profiles.go`/`models.go`; EV-G4-01 synthetic
  conformance still valid for the route tuple). Live availability on the candidate
  build is **unproven** and folds into the same §5 scenario.
- **8.2 (tools/reasoning/usage/cancel/terminal/error):** accepted state is
  *“Agy remained deterministic fail-closed contracts”* (synthetic). No prior live
  tools/reasoning/usage/cancel proof exists to reuse for Antigravity; the §5
  scenario is the single run that would produce it. No separate campaign.

## 4. Notes / bounded limitations
- `execenv/antigravity_home.go` (native `.gemini/antigravity-cli/**` token copy) is
  **not** on the accepted OmniRoute route: `EV-G1-ADAPTERPREP` disables native-Agy
  token copy in gateway-required mode and names the approved frontend as
  Claude/Codex `CLIKind` with an `agy/...` model. I could not isolate its accepted
  per-file hash because `EV-G1-ADAPTERPREP` pins only an **aggregate** 40-file
  `execenv` manifest (`ad85fe4c5770214a0e248503a681bd2b2351d1c4e8234279409594fc1c538805`),
  not individual files. It is immaterial to this decision.
- **Ambiguity flagged (not resolved by guessing):** planning artifacts
  (`EV-G1-ADAPTERPREP`) treat *native `agy`* as disabled (route = claude-code
  frontend → Anthropic Messages → OmniRoute), whereas the shipped `antigravity.go`
  + resolver fix + `authoritative-route-matrix-D-V3-27` treat *native `agy` ↔
  OmniRoute* as the operational, "already fully tested" route. Under **either**
  reading the decision is STALE: if native `agy` is the route, its adapter changed;
  if the claude-code frontend is the route, the "operational" native-`agy`
  connectivity evidence no longer matches the shipped build. Owner to confirm which
  frontend is authoritative for `agy/claude-opus-4-6-thinking` (owner: OmniRoute
  architect / Codex1-W1; action: freeze the authoritative CLIKind for the Agy row).

## 5. The one minimal scenario required to clear STALE (DO NOT RUN here)

**Exactly one** non-streaming launch, reused to close overlapping 5.8/8.1/8.2:

> On the candidate build (HEAD `a6d5`, with `antigravityResolverEnv` active), run a
> single Kanban→terminal task on route `agy/claude-opus-4-6-thinking` and confirm
> the child process **reaches the OmniRoute endpoint and returns a terminal
> result** — i.e., prove the in-product `GODEBUG=netdns=cgo` injection delivers the
> same reachability the accepted per-process workaround delivered. One run;
> non-streaming sufficient; captures tools/reasoning/usage/cancel/terminal-error to
> also satisfy 8.2.

- **Owner / integration:** W1 serial live-run token, route family R2 (Kiro/Agy) per
  D-V3-27; producer ≠ reviewer ≠ adjudicator.
- **Blocker (external):** `D-V3-25(B)` — live-provider tests are **SECURITY-STOPPED
  until the Owner confirms UI invalidation/revocation of the exposed key.** Until
  then this scenario is `BLOCKED_EXTERNAL` (owner: Product/OmniRoute owner; action:
  confirm key revocation). Offline hash/provenance revalidation (this file) was
  permitted and is complete.
- Not executed by A5. No Antigravity rerun performed.

## 6. Commands used (read-only)
```text
git rev-parse HEAD
git merge-base --is-ancestor b6571299… HEAD
git status --porcelain=v1 -- <5 paths>
git show b6571299…:multica-auth-work/server/pkg/agent/antigravity.go | sha256sum
git log --oneline -- …/pkg/agent/antigravity.go
sha256sum <5 candidate paths>; wc -c <paths>
rg -n 'antigravityResolverEnv|netdns|GODEBUG|cmd.Env' <antigravity.go / _test.go>
```
No secret/token/cookie/prompt/provider payload/account identity was read, printed, or hashed. No product code edited. No live/daemon/provider/network call. Prodex untouched.
