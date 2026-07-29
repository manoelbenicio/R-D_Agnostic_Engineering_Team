# A9-OPUS48 — Reserved single Opus48 live acceptance (evidence lock)

- agent: Codex56#B · lane: A9-OPUS48 · task: P0-OPUS48-LIVE-ACCEPTANCE · pane: w7:p4 (HERDR_ENV=1)
- covers: OpenSpec 5.8 / 8.1 / 8.2 (one accepted run closes all overlapping requirements it proves)
- evidence lock (only mutable artifact): `.deploy-control/p0/evidence/R2-opus48-live-acceptance.md`
- status: **BLOCKED** — no execution now. This lane owns **exactly ONE** future integrated Opus48
  execution and reserves it; it does not run GLM/Kimi/Antigravity and edits no product source.
- repo: HEAD `a6d5098` · check-in: `.deploy-control/p0/checkins/Codex56-B__P0-OPUS48-LIVE-ACCEPTANCE__20260721T224402Z.json`
- consumes: `.deploy-control/p0/handoffs/A4-opus48.md`, `.deploy-control/p0/handoffs/R2-opus48-source-delta.md`

## 1. Tool preflight (exact, recorded 2026-07-21T22:44Z)

| Tool | Path | Result |
|---|---|---|
| git | `/usr/bin/git` | `git version 2.50.1` |
| python3 | `/usr/bin/python3` | `Python 3.9.25` |
| rg | `/usr/local/bin/rg` | `ripgrep 15.2.0 (rev e89fff89ac)` |
| Go 1.26.1 | `/home/ec2-user/goroot/go/bin/go` | `go version go1.26.1 linux/amd64` |
| gofmt | `/home/ec2-user/goroot/go/bin/gofmt` | present/executable |
| Registry (no secret) | `curl http://127.0.0.1:20128/v1/models` | `http_code=000` connection refused — no local gateway |
| repo HEAD | — | `a6d5098` (`integration/dev-transition-candidate-20260719`) |

No secret/credential/account value read or written. No compile/live run performed.

## 2. Scope of this lane

- **Owns exactly ONE** future live non-prod Kanban→terminal execution of the **Kiro/Opus48** route
  (`CLIKind=claude-code` + the certified exact Opus48 AWS `RouteModel` via OmniRoute), per
  `control.json.live_runs.opus48`. That single run closes overlapping 5.8/8.1/8.2.
- **Does NOT**: run now; run GLM / Kimi / Antigravity (Antigravity is evidence-reuse via A5); execute a
  second/duplicate run; perform broad regression or QA-A/B/C; edit product source, OpenSpec, or GSD;
  test or implement any OmniRoute-owned auth/credential/quota/retry/failover behavior.

## 3. Preconditions before the single run may be authorized/executed

All must be simultaneously true (per `PROTOCOL.md` live-run token + D-V3-27 + D-V3-25(B)):

1. Exact Opus48 AWS `RouteModel` **certified/published** by the OmniRoute registry owner (anthropic-messages
   capability row + provenance) — currently absent. Do not invent it.
2. `control.json.live_runs.opus48.authorized == true` with the exact RouteModel + integrated
   commit/build/config provenance and no prior accepted run for the same family/build/scenario.
   (Currently `authorized=false`, `accepted_run=null`.)
3. W1 serial integration of the Opus48 route complete on a known build/config.
4. D-V3-25(B) security-stop lifted: Owner confirms revocation of the exposed UI key (live-provider tests
   are SECURITY-STOPPED until then).
5. Explicit authorization from the Principal Orchestrator (final authority; issues the live-run token).

## 4. Reserved run plan (to execute only after §3 all-green)

1. Record the run ID before launch in `control.json.live_runs.opus48.accepted_run`.
2. Create/use squad + project + Kanban task; assign the Opus48 (claude-code) route.
3. Observe launch via Agent Brain; exercise **tools, reasoning, usage**; when applicable **cancel** and
   confirm termination/cleanup; confirm **deterministic error** surfaces; confirm **terminal result**
   persisted and delivered to UI.
4. Capture terminal evidence with full provenance (command, host, OmniRoute version/digest, UTC, runner)
   per `EVIDENCE_CONTRACT.md`; no secrets/prompts/payloads. `5.8`, `8.1`, `8.2` reuse this one run.
5. One run only. No rerun if equivalent evidence already proves a sub-requirement.

## 5. Blocker (recorded via p0_control.py block)

- **Blocker:** exact Opus48 `RouteModel` not certified by the external OmniRoute owner;
  `control.json.live_runs.opus48.authorized=false`.
- **Owner:** OmniRoute registry owner + Principal Orchestrator.
- **Next action:** consume the certified exact ID without inventing it, then — after explicit
  authorization — execute exactly one accepted run.
