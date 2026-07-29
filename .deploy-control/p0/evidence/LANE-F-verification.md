# LANE F — independent verifier (read-only). Report to w5:p1.

- agent: Opus48#D · lane: F-verifier · task: P0-LANE-F-INDEP-VERIFY · pane `w8:p2`
- check-in: `.deploy-control/p0/checkins/Opus48-D__P0-LANE-F-INDEP-VERIFY__20260724T104544Z.json`
- posture: **read-only; edit/deploy NOTHING.** Verify each lane independently WITH EVIDENCE as it reports DONE to w5:p1.
- as-of (UTC): 2026-07-24T10:45Z.

## Verification criteria (per lane)
| Lane | PASS criteria | How I verify (read-only) |
|---|---|---|
| A (nim) | git diff scope is ONLY nim files (no other `.go` touched) + `go build`/`vet`/`test` green | `git diff --stat`/`--name-only` (assert every path is a nim file), then `go build ./... && go vet && go test` (targeted) with `GOCACHE=/tmp` |
| B (binaries) | both binaries built + sha recorded | `ls -l` + `sha256sum` of each binary; compare to lane B's recorded shas |
| C (frontend) | frontend `:13100` = 200 + backend/daemon undisturbed | `curl -s -o /dev/null -w %{http_code} :13100`; confirm backend/daemon health still 200 (unchanged) |
| D (daemon) | daemon running the NEW sha + health 200 | daemon `/health` = 200; running binary sha == lane B new sha (proc/exe sha or reported) |
| E (runtimes) | the 5 runtimes show online in `agent_runtime` | read-only SELECT of `agent_runtime` (status online for the 5) |

## Current status @10:45Z
- **No LANE A–E DONE signal yet.** Recent DONE check-ins are the P0 observability workstream
  (OTLP-RECV, ASSEMBLER-GUARD, P0-FINAL-OBS, realtime-delivery, AUDIT-E2E-HOPS) — NOT the A–E
  nim/binaries/frontend/daemon/runtimes lanes. → All of A–E = **PENDING (not yet reported)**. Standing by.
- Read-only baseline captured:
  - Binaries present: `/tmp/multica-auth` sha256 `bd5c38481868c46fc6a1130ffe61cf5ea0df2e49d909fab664ff274e99cacd61`;
    `/tmp/multica-auth-fixed` sha256 `b97c668f2b347dd59fa72fe4867f68fc6d96f948ed60689f680e355f14bbd63a` (Jul 24 03:15, newest).
  - **All local service ports unreachable from `w8:p2`:** `:13100`→000, `:3100`→000, `:8080/health`→000,
    `:20128/health`→000. The product stack is NOT running on this pane.

## CONCERN → w5:p1 (guard flag, raised early)
**Lanes C, D, E cannot be independently verified from `w8:p2`.** They require the live stack (frontend
`:13100`, daemon `/health`, `agent_runtime` DB), which is not running locally here, and the deploy target
(ORQ1 `100.118.244.61`) is a remote host this verifier has no authorized read-only access to (no
authorized SSH/DB path). **Requested from w5:p1:** an authorized read-only observation path for
C/D/E (e.g., a sanctioned health/DB read view, or run the verifier where the stack/ORQ1 is reachable).
Without it I will report C/D/E as `BLOCKED — no observation path`, not PASS. LANE A (git+go) and LANE B
(binary sha) ARE locally verifiable here.

## On-DONE actions (when a lane reports)
- A: run `git diff --name-only <base>..HEAD` (or working-tree) — FAIL if any non-nim `.go` appears; then
  `GOCACHE=/tmp/kgc go build ./... && go vet ./internal/... && go test <targeted>`; report file:line of any out-of-scope path or build/test failure.
- B: `sha256sum` both binaries == recorded; report exact shas.
- C/D/E: on an authorized path, curl codes / running-sha / `agent_runtime` SELECT; report exact command + output.

Report format to w5:p1: `PASS|CONCERNS|BLOCKED` per lane + file:line / command evidence. I do not edit or deploy anything.

## LANE A — VERIFIED @10:58Z: PASS (with notes)
- go build ./... EXIT 0; go vet ./... EXIT 0 (server module, GOCACHE=/tmp).
- 14 .go files changed = the NIM native-runtime REMOVAL set. nim-named: pkg/agent/nim.go, nim_test.go,
  execenv/nim_home.go, nim_home_test.go. Shared (content-verified NIM-scoped only): pkg/agent/agent.go +
  models.go (drop nim registry + nimStaticModels z-ai/meta-llama), config.go (provider-list comment),
  daemon.go (remove native-http runtimeVersion fast path), execenv/execenv.go (remove NIM cred handling),
  native_runtime_wiring_test.go, vendor_credential_fallback_test.go, agent_test.go, agent_supported_types_test.go, models_test.go.
  No unrelated Go logic in any hunk.
- Groups 2+3 PRESENT (retained): brain.CLINIM enum identity.go:19,28; runtimeenv NIM_/NVIDIA_ deny policy.go:36,37,73.
- NOTES to w5:p1: (1) "14 nim files" = NIM-removal set incl. shared files, not 14 nim-named — confirm intended.
  (2) daemon.go runtimeVersion dropped native-http path (in-scope; NIM was the native-http runtime) — confirm no other runtime needs it.
  (3) Non-Go changes also in the shared tree (frontend runtime-profile ts/json, spec.md, bin/prodex deletion) — outside LANE A Go scope, likely other lanes; cannot isolate an uncommitted LANE A commit.
