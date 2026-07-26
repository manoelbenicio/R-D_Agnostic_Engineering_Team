# W1 — Integration Readiness (BLOCKED pending FLEET_SATURATED GREEN + authorization)

- agent: `Opus48#B`  ·  pane: `w6:p2`  ·  lane: `W1` (sole serial editor/integrator of frozen shared hotspots)
- task: `P0-W1-INTEGRATION`
- control lock (only mutable file): `.deploy-control/p0/handoffs/W1-integration-readiness.md`
- check-in: `.deploy-control/p0/checkins/Opus48-B__P0-W1-INTEGRATION__20260721T224642Z.json`
- posture: **BLOCKED** — product source READ-ONLY. No source edit, no test, no live run while blocked.
- repo: branch `integration/dev-transition-candidate-20260719`, HEAD `a6d50986aee7f1981a323b29c9d90f175b4b6f8b`

> This document records W1 integration readiness only. It performs no edits. The serial integration
> executes **only** after the Principal records an explicit `FLEET_SATURATED=GREEN` + implementation
> authorization; until then W1 holds and blocks (per the blocker below and PROTOCOL).

---

## 1. Tool preflight (recorded; canonical paths)

| Tool | Path / command | Version | Result |
|---|---|---|---|
| git | `git --version` | `2.50.1` | OK |
| python3 | `python3 --version` | `3.9.25` | OK |
| ripgrep | `rg --version` | `15.2.0` | OK |
| OpenSpec | `openspec --version` | `1.4.1` | OK |
| Node | `node --version` | `v22.23.1` | OK |
| pnpm | `cd multica-auth-work && pnpm --version` | `10.28.2` | OK (pin resolved) |
| **Go** | `/home/ec2-user/goroot/go/bin/go version` | `go version go1.26.1 linux/amd64` | OK — **canonical path, owner-confirmed** |
| gofmt | `/home/ec2-user/goroot/go/bin/gofmt` | present/executable | OK |
| Docker | `docker` | absent | Not required this phase |

- Go path reconciliation is closed: `control.json.fleet_saturation_gate.go_toolchain =`
  `"/home/ec2-user/goroot/go/bin/go (go1.26.1)"`. The earlier `.local/toolchains` path (superseded) is
  ignored; `$HOME` is a credential slot and was not searched.
- All W1 serial focused checks invoke Go by this absolute path with `-count=1`.

---

## 2. Exact freeze provenance (what W1 will execute once authorized)

Authoritative source-edit plan = `.deploy-control/p0/handoffs/W1-source-lock-freeze.md` (this lane's
prior deliverable, checked out DONE). Consumed lane provenance:

| Artifact | Lane | Status | SHA-256 (provenance pin) |
|---|---|---|---|
| `handoffs/W1-source-lock-freeze.md` | W1 (this) | frozen | `2edcbdbbd1ec4c5cf83272d908605c0f23a0b0ab6402e21961be371d52fed29f` |
| `handoffs/R1-cline-route-source-delta.md` | R1-DESIGN (Codex56#A) | DONE | `b5ddb9dc2c4668a915ad9c7d692c27a842d0d01d397af94e534f47af69ad7712` |
| `handoffs/R2-opus48-source-delta.md` | R2-DESIGN (Codex56#B) | DONE | `effea13fcf27178b729c40f3dd582fc84bda369541ec5dd8416f229c61320d46` |
| `handoffs/A3-route-freeze.md` | A3 (Codex56#A) | DONE | `36c78a2e4dbdfc475285f8fbab3cd77176c599b7c64138b275cca4125d9d0713` |
| `handoffs/A4-opus48.md` | A4 (Codex56#B) | DONE | `5aabbf06d68f9ee57a8db162005459d56226dc3351e74ebd7a406660c5e3fadc` |
| `handoffs/A6-lifecycle-gap-matrix.md` | A6 (Opus48#D) | DONE | `a0244022a76337867a37dbb83ab009d8c1d418bbfb00da14765e7fd8694c5314` |
| `evidence/A5-antigravity-equivalence.md` | A5 (Opus48#C) | DONE | `335a73e168901765e1424cdd7d261805a2fa606a50b2edb9556135b6af14debb` |
| `handoffs/A1-A2-cline-foundation.md` | A1+A2 (Opus48#A) | DONE | `920a41beaad3eacb6c63ac268ba543103e37374064f725673d58a77300ff4d3c` |

> Provenance note: the `W1-source-lock-freeze.md` hash above is its content digest at authoring time of
> this readiness record; W1 re-verifies each digest at authorization time before acquiring locks.

### W1 exclusive source locks to acquire (from the freeze, only after GREEN)
`pkg/agent/{models.go,models_test.go,nim.go,nim_test.go}`, `internal/handler/agent_thinking_test.go`
(coupled), `internal/daemon/{daemon,config,health,brain_integration}.go`, `cmd/multica/cmd_daemon.go`,
`go.mod`, `internal/daemon/brain/**` (only on proven gap), the escalated shared `runtimeenv/{env.go,
adapter.go,policy.go,home.go}`, `internal/daemon/execenv/cline_home.go` (add
`WriteCredentiallessClineConfig` — file EXISTS, reconcile), and the L7b reclamation test. R1/A1 retain
Cline-exact `runtimeenv/{cline.go,cline_test.go, new cline_*.go}`. Pairwise overlap = ∅ (freeze §4).

### Frozen serial order (freeze §3.6 / R1 §8) — execute one edit at a time, focused check after each
1. **D1** `runtimeenv/adapter.go` `CredentiallessAdapterContract(CLIOpenAICompatible)` fail-closed→`AdapterReady`/`ProtocolOpenAIChat`.
2. **D2** `runtimeenv/env.go` `trustedAdapterEntries` add Cline case + `AdapterEnvironment.ClineDataDir` field.
3. **D5** `internal/daemon/execenv/cline_home.go` add `WriteCredentiallessClineConfig` (mirror `codex_home.go:87`).
4. **D6** `brain_integration.go` `buildLaunch` add `CLIOpenAICompatible` branch (+ `updatedAt` format decision).
5. **D3+D4** `config.go` `agentBrainBuiltInCLIFor` (`:141`) + `Validate` (`:179/:196`) accept `CLIOpenAICompatible`.
6. **D7** `pkg/agent/models.go:562` GLM `cp/` prefix (A3 SUB-1) + `models_test.go:172` lockstep.
7. **GLM live run** (single reserved token) → closes 5.6/8.1/8.2 for the GLM family.
8. On **BLK-KIMI** clear: apply K2/K3 one-line Kimi fixtures → single Kimi live run (5.7).

Focused checks (per delta, `/home/ec2-user/goroot/go/bin/go`, `-count=1`):
`./internal/daemon/runtimeenv/...`, `./internal/daemon/execenv/...`, `./internal/daemon/...`,
`./pkg/agent/...`, `./internal/handler/...` (only the affected package per edit) + `gofmt -l`.

- **Opus48 (5.8):** R2 proved **config-only** — `AGENT_BRAIN_CLI_KIND=claude-code` +
  `AGENT_BRAIN_ROUTE_MODEL=<OPUS48_AWS_ID>`; **zero source edit**; blocked on the external RouteModel.
- **Antigravity:** A5 = STALE; one live scenario only; no source edit; blocked by D-V3-25(B).

---

## 3. Gate state at check-in (verified provenance, 2026-07-21T22:44–22:46Z)

| Signal | Value (verified) | Source |
|---|---|---|
| `implementation_authorized` | **false** | `control.json` |
| `fleet_saturation_gate.status` | **RED** | `control.json` |
| gate `last_red_reason` | "Awaiting A7/A8 artifact preflight evidence and current successor check-in/block convergence; source locks remain held." | `control.json` |
| `assignment_phase` | `PREFLIGHT_AND_GAP_MATRIX` | `control.json` |
| latest monitor snapshot | `2026-07-21T22:44:46Z` sev **RED**, active 9, detected 10 | `monitor.jsonl` |
| monitor findings | RED `NOT_WORKING` Agy-P0-A7; RED `NOT_WORKING` Agy-P0-A8; AMBER `BLOCKED` Codex56#B | `monitor.jsonl` |
| A7 (`Agy-P0-A7`) | checked in `22:42:33Z`; **handoff body absent**; Herdr `NOT_WORKING` | `checkins/`, monitor |
| A8 (`Agy-P0-A8`) | checked in `22:38:22Z`; **handoff body absent**; Herdr `NOT_WORKING` | `checkins/`, monitor |

⇒ Gate is RED; A7/A8 prod-integrity handoffs have not landed and Herdr shows them not working; the
Principal has not authorized implementation. **W1 must not acquire source locks or edit.**

---

## 4. BLOCKED

- **Blocker:** `FLEET_SATURATED RED pending A7/A8 handoffs and Kiro audit; implementation_authorized=false`.
- **Owner:** `Principal Orchestrator + A7/A8 + Opus48-Kiro audit`.
- **Next action:** after explicit `FLEET_SATURATED=GREEN` / authorization, acquire **only** the W1 source
  locks from `W1-source-lock-freeze.md` and execute the serial **D1–D7** order (§2) with focused tests;
  reserve one live run per changed/unproven family.
- **While blocked:** no source edit, no test execution, no live run; W1 holds the frozen shared hotspots
  (sole serial editor) and takes no action until GREEN + authorization.

No product source edited. No test executed. No live run. Read-only readiness record only.
