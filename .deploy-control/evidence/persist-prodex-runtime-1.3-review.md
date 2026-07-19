# Independent Review: persist-prodex-runtime-integration — Task 1.3 ONLY

**Reviewer:** Kiro (independent; distinct from producer Opus48#A and from Gemini,
who reviewed 1.1-1.3 jointly as a contract-incomplete pass on 2026-07-18)
**Date:** 2026-07-18T17:52:00-03:00
**Scope:** Task 1.3 only, per current on-disk task text. Tasks 1.1/1.2, product code,
tests, specs, task checkboxes, DB/network/live services, credentials, and env
contents are explicitly OUT OF SCOPE and were not touched.
**Adjudication authority:** Kiro TL adjudicates final ACCEPT/REJECT for the
OpenSpec checkbox. This artifact is evidence only and does not self-accept.

## Task 1.3 — exact current text (verified from disk)

`openspec/changes/persist-prodex-runtime-integration/tasks.md` line 15:

> `- [ ] 1.3 Add MULTICA_PRODEX_REQUIRED startup enforcement so a required
> Prodex/L2 configuration cannot silently downgrade`

Checkbox state on disk: **unchecked** (`[ ]`). Confirmed consistent with the
AGENT_LEDGER "REOPENED — PENDING independent QA (3→0/16)" and the subsequent
Gemini/Opus48#A joint review row ("CONTRACT INCOMPLETE — NOT RECLOSED", 0/16).

## AB-REQ / EV mapping

- **Requirement (AB-REQ-PP-1.3):** "a required Prodex/L2 configuration cannot
  silently downgrade" — i.e. if the operator sets `MULTICA_PRODEX_REQUIRED=1`,
  any missing/invalid dependent configuration (Prodex enablement, L2 tenant)
  must produce a hard failure (fail-closed), never a silent fallback to a
  disabled/default state.
- **Evidence (EV-PP-1.3-KIRO):** this artifact + the three named tests below,
  re-executed independently on 2026-07-18T17:5x.

## Source & test provenance (SHA-256, current disk state)

```
312fef692eeca53b84413f429b2cb19136974f7326a133105379d18263bdf78e  internal/daemon/prodex_runtime_integration_test.go
a54fb79dcb55e4f5928261a4a7cb200dd6377c57c528168f0983053e561139de  internal/daemon/l2_runtime.go
82035719e5a3b3a1472af973412f431e0c14ad11e04939885141893187ee38f7  internal/daemon/prodex.go
```

These three hashes are byte-identical to the hashes cited in the prior
`evidence/persist-prodex-runtime-1.2-review.md` (Antigravity, task 1.2 only),
confirming **no source drift** since that review. No stop condition triggered.

## Conflict / concurrency scan (Golden Rule)

- Checked `AGENT_LEDGER.md`, `FILE_OWNERSHIP.md`, and the prior check-in file
  `.deploy-control/Kiro__PRODEX-RUNTIME-1.1-1.3__20260718T200600Z.md`
  (producer Opus48#A's original self-check, filed under a `Kiro__` name but
  attributed to Opus48#A in the ledger).
- No `files_locked` entry currently claims `prodex.go`, `l2_runtime.go`, or
  `prodex_runtime_integration_test.go` as being actively edited. Ledger notes
  Opus48#A has since moved to unrelated read-only architecture review.
- No ownership conflict, no credential need, no DB/network requirement
  encountered. Nothing required stopping.

## Execution proof (bounded, offline, synthetic-only)

Environment: `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`, pinned local
`/home/dataops-lab/go-sdk/bin/go` (go1.26.4, linux/amd64).

```
go build ./internal/daemon   => exit 0
go vet   ./internal/daemon   => exit 0 (no findings)

go test -v -count=20 -race ./internal/daemon -run \
  'TestLoadProdexLaunchConfigRequiredButDisabledFailsClosed|TestLoadL2RuntimeConfigRequiredMissingTenantFailsClosed|TestLoadL2RuntimeConfigNotRequiredDefaultsTenant'

=== 3 distinct named tests, count=20 each => 60 === RUN, 60 --- PASS, 0 --- FAIL
PASS
ok  	github.com/multica-ai/multica/server/internal/daemon	3.5-3.7s (two runs)
```

No data races reported. No test required a database, network call, or live
service; all three tests use only `t.Setenv` (auto-restored) and one synthetic
temp executable fixture (`writeFakeExecutable`), consistent with the file's own
header comment.

## Restart / recovery authority semantics (task 1.3 focus)

This is the substantive check for 1.3: does the "required" flag actually make
Prodex/L2 startup fail-closed on restart, rather than silently downgrading to
a disabled or default-tenant state?

- `loadProdexLaunchConfig()` (`prodex.go:16-36`): reads
  `MULTICA_PRODEX_REQUIRED` first. If Prodex is **not enabled** and required
  is true, it returns a hard error (`"prodex is required but
  MULTICA_PRODEX_ENABLED is disabled"`) — no fallback path exists in this
  branch. Verified directly by
  `TestLoadProdexLaunchConfigRequiredButDisabledFailsClosed`.
- `loadL2RuntimeConfig()` (`prodex.go:62-105`): the tenant-ID resolution block
  (lines 94-101) only defaults `tenantID` to `"default"` when
  `!envBool("MULTICA_PRODEX_REQUIRED")`. When required is true and
  `MULTICA_L2_TENANT_ID` is empty, it returns a hard error instead of
  defaulting — verified by `TestLoadL2RuntimeConfigRequiredMissingTenantFailsClosed`.
- The inverse (not required, no tenant) legitimately defaults to `"default"`
  rather than erroring — this is correct non-required behavior, not a
  "silent downgrade" of a *required* configuration, and is what
  `TestLoadL2RuntimeConfigNotRequiredDefaultsTenant` asserts.
- **No duplicate-state divergence observed:** both enforcement points key off
  the *same* `MULTICA_PRODEX_REQUIRED` env var read via the same `envBool`
  helper (`prodex.go:186-192`), so there is a single source of truth for the
  "required" authority — not two independently-drifting required flags. I
  found no second/shadow required-flag path in `prodex.go` or `l2_runtime.go`.
- **Restart authority:** `loadProdexLaunchConfig` and `loadL2RuntimeConfig`
  are the config-loading entry points invoked at daemon startup (not
  re-derived per-session), so a restart with `MULTICA_PRODEX_REQUIRED=1` and
  an incomplete configuration will fail at startup every time — consistent
  with "cannot silently downgrade" on restart. I did not execute an actual
  daemon restart (out of scope: no live service), so this is a static-code
  + unit-test confirmation only, not a live-restart observation.

## Explicit non-claims

- Did not modify prodex.go, l2_runtime.go, the test file, tasks.md, or any
  spec.
- Did not run task 1.1 or 1.2 tests as part of grading 1.3 (they were run
  incidentally as part of the shared `-run` regex in the prior artifact, not
  here); this review's `-run` filter is scoped to the 3 tests that exercise
  1.3 behavior only.
- Did not touch DB, network, credentials, or environment contents beyond
  `t.Setenv` inside the test binary's own process.
- Did not run `git add`/`commit`/`push`.
- Did not set or unset any OpenSpec checkbox.

## Verdict

**ACCEPT (task 1.3 technical scope only).**

Rationale: the exact current task-1.3 text ("MULTICA_PRODEX_REQUIRED startup
enforcement so a required Prodex/L2 configuration cannot silently downgrade")
is satisfied by the existing baseline code, independently re-verified with
fresh hashes, build, vet, and a 60/60 race-clean test run. Single source of
truth for the required flag; no duplicate/divergent state found.

**This ACCEPT does not itself close the checkbox.** Per the standing
adjudication pattern in this change (see the Gemini/Opus48#A 1.1-1.3 review,
which found the *evidence contract* — not the technical behavior — incomplete:
missing source-hash manifest, reviewer-identity/provenance, AB-REQ/EV mapping),
Kiro TL must still adjudicate whether this artifact's contract fields (now
included above: hashes, provenance, AB-REQ/EV mapping, distinct reviewer
identity) are sufficient to reclose 1.3, and whether 1.1/1.2 (out of this
review's scope) are handled separately. **I do not self-accept the checkbox.**
