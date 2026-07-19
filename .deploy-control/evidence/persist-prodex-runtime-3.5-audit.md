# Independent Read-Only Audit: persist-prodex-runtime-integration — Task 3.5 ONLY

**Reviewer:** Kiro (independent; read-only auditor role for this task)
**Date:** 2026-07-18T17:58:00-03:00
**Scope:** Task 3.5 only, per current on-disk task text. Tasks 3.1-3.4, 1.x,
2.x, 4.x, product/test/spec edits, DB/network/live providers, credential/env
values, and delete operations are explicitly OUT OF SCOPE for this audit and
were not performed/touched.
**Adjudication authority:** Kiro TL adjudicates the final status for the
OpenSpec checkbox. This artifact reports a factual finding only.

## Task 3.5 — exact current text (verified from disk)

`openspec/changes/persist-prodex-runtime-integration/tasks.md` line 19:

> `- [ ] 3.5 Purge unreferenced legacy credential files and obsolete
> unassigned account records while preserving non-credential agent state`

Checkbox state on disk: **unchecked** (`[ ]`).

## AB-REQ / EV mapping

- **Requirement (AB-REQ-PP-3.5)**, per
  `specs/prodex-runtime-continuity/spec.md` ("Requirement: Obsolete
  credentials are removed"):
  - *Scenario: Unreferenced Codex slot* — WHEN a Codex slot is not
    referenced by any current validated Codex account row, THEN its legacy
    `auth.json` is removed without deleting non-credential agent state.
  - *Scenario: Obsolete account record* — WHEN an account is explicitly
    classified as legacy and has no current assignment, THEN its account and
    credential references are removed so it cannot participate in selection
    or fallback.
  - Design elaboration (`design.md`, "Legacy credential purge" section):
    purging is scoped to *provider credential material only* — agent
    skills, sessions, prompts, caches, and workspaces must never be deleted
    by this mechanism.
- **Evidence (EV-PP-3.5-KIRO):** this artifact. No test/implementation
  evidence exists to cite because none exists on disk (see below).

## Source & spec provenance (SHA-256, current disk state)

```
afe8b75f024e825eaea301c22566e28e653ccb57727e596905858a385489e4c8  internal/daemon/prodex_profiles.go
de661603af6b0ec1aece2ffe442f446ece86cc9dd91aa596fa81fc1553b3eaea  openspec/changes/persist-prodex-runtime-integration/tasks.md
2bf341e8323467f1fc8235190c6f42711ffae392d081849a7c7127063ffb7feb  openspec/changes/persist-prodex-runtime-integration/design.md
4a47a6cfc92b5c63acdd8baf83556063f84d47dbd9a9e8270b38e6988375c251  openspec/changes/persist-prodex-runtime-integration/specs/prodex-runtime-continuity/spec.md
```

## Implementation trace (what actually exists today)

Full read of `internal/daemon/prodex_profiles.go` (the file that owns
profile reconciliation, tasks 3.1-3.4) plus targeted greps across
`internal/daemon/**` for any purge-adjacent logic:

- `reconcileProdexProfiles` (`prodex_profiles.go:26-119`) — builds the
  reconciled/approved profile set from the validated account inventory.
  It reads `state.json`, validates each account's slot home, rejects
  duplicate credential identities, and calls `addProdexProfileReference`
  (`prodex profile add ... --codex-home ...`) for new profiles. **It never
  deletes anything.** Homes/profiles that exist on disk but are *not* in
  the current account inventory are simply never added to `reconciled` —
  they are silently excluded from the in-memory registry, not purged from
  disk or from Prodex's own `state.json`/profile store.
- No call to `os.Remove`, `os.RemoveAll`, `prodex profile remove`, or any
  equivalent destructive/cleanup operation exists in `prodex_profiles.go`,
  `prodex.go`, or `l2_runtime.go`.
- A repo-wide grep for `Purge`, `PurgeUnreferenced`, `PurgeLegacy`,
  `purgeProdex`, `purgeCredential` (case-sensitive, whole codebase) returned
  **zero matches**.
- A grep for `purge|unreferenced|legacy.*credential|obsolete` inside
  `internal/daemon/**` matched only two unrelated files:
  - `prompt.go` — unrelated prompt-template language, not credential logic.
  - `runtime_isolation_test.go` — three comments referencing the *naming*
    of legacy `rotation_*`-prefixed DB tables (an unrelated rotation-schema
    migration concern), not credential-file purging.
- `internal/daemon/gc.go` implements a *different* garbage collector
  (`gcLoop`/`runGC`/`cleanTaskDir`/`cleanTaskArtifacts`) that reclaims
  **task workspace directories** (issue/chat/autopilot-run workdirs) based
  on their parent record's lifecycle status. This is a legitimate, tested
  subsystem, but it purges task working directories, not provider
  credential files or account rows, and its doc comments make no reference
  to Prodex, Codex slots, or account records. **It is not an implementation
  of task 3.5** and must not be conflated with it.
- No "obsolete account record" deletion path exists anywhere in the
  `rotation` package call sites reachable from `daemon` (`rotationStore`
  is only read via `ListAccounts` in `reconcileProdexProfiles`; no write/
  delete call on the account store was found in this scope).

## Test coverage trace

Repo-wide grep for test functions matching `Purge`, `Legacy` (in a
credential/account-removal sense), `Obsolete`, `prodexProfile*`, or
`Reconcile*` across the whole `multica-auth-work/server` tree. Findings:

- **Zero** test functions reference credential/account purging in the
  sense required by 3.5.
- The only tests actually covering `prodex_profiles.go` behavior would be
  reconciliation tests for tasks 3.1-3.4 (duplicate identity, POSIX mode
  enforcement, slot-home validation) — out of this audit's scope to grade,
  and in any case no dedicated `*_test.go` file for `prodex_profiles.go`
  was found in the daemon package at all (no `prodex_profiles_test.go` on
  disk).
- Every `Legacy`-named test found (`TestAgentBrainDevelopmentSkipsLegacyStartup`,
  `TestLegacyDaemonUUIDs_ScansProfileDirs`, `TestEnsureCodexSandboxConfigStripsLegacyInlineDirectives`,
  `TestCreateWorktreeRemovesLegacyCoAuthoredByHook`, etc.) belongs to
  unrelated subsystems (Agent Brain gateway bypass, daemon UUID legacy
  scan, sandbox config directives, repo-cache git hooks) — none purge
  provider credential files or account records per the 3.5 requirement.

## Deletion targets, reference checks, tenant/account boundaries (design intent vs. code)

Per `design.md`'s "Legacy credential purge" section, the *intended* design
is:
1. Only Codex slot homes referenced by the current validated Codex account
   inventory retain `auth.json`; unreferenced slot-local `auth.json` files
   should be removed.
2. Obsolete account rows should be removed only after confirming no current
   assignment references them.
3. Purge scope is strictly provider credential material — never agent
   skills/sessions/prompts/caches/workspaces.

None of these three behaviors have a corresponding code path today. The
current `reconcileProdexProfiles` function computes the reconciled *allow-list*
in memory each run but performs no filesystem removal of unreferenced
`auth.json` files and no account-row deletion. There is consequently:

- **No deletion-target enumeration logic** to trace (nothing walks slot
  homes looking for ones absent from the account inventory).
- **No reference-check-before-delete logic** to trace.
- **No tenant/account boundary enforcement specific to purging** (the
  existing tenant boundary in `reconcileProdexProfiles` — `d.cfg.L2Runtime.TenantID`
  passed to `ListAccounts` — governs which accounts are reconciled, but has
  no bearing on removal since removal doesn't exist).
- **No rollback/recovery mechanism** to trace, since there is no forward
  destructive operation to roll back from.
- **No test coverage**, synthetic or otherwise, of any purge behavior.

## Conflict / concurrency scan (Golden Rule)

- Checked `AGENT_LEDGER.md` and `FILE_OWNERSHIP.md`: no active lock or
  in-progress claim on `prodex_profiles.go`, the 3.5 task, or any
  purge-related file. `FILE_OWNERSHIP.md`'s Prodex hotspot table (owner
  Codex1, `build-omniroute-agent-brain` change) lists `prodex_profiles.go`
  as "preserved PD-01 baseline," consistent with it being read-only
  baseline code rather than actively edited for 3.5.
- No stop condition triggered: no source drift concern (nothing to drift
  from — feature absent), no ownership conflict, no credential need, no
  destructive action attempted, no DB/network/live-provider touch.

## Explicit non-claims

- Did not modify `prodex_profiles.go`, `tasks.md`, `design.md`, or the spec.
- Did not read any real `auth.json`, credential file, or auth-home content;
  all inspection was of Go source, Markdown spec/design text, and grep
  output only.
- Did not run any delete/remove operation, staged, committed, or pushed
  anything to git.
- Did not touch any database, network endpoint, or live provider.
- Did not set or unset any OpenSpec checkbox.

## Verdict

**MISSING.**

Task 3.5 ("Purge unreferenced legacy credential files and obsolete
unassigned account records while preserving non-credential agent state")
has **no implementation and no test coverage** anywhere in the current
`multica-auth-work/server` tree. The only two candidate matches on a
purge/legacy/obsolete grep are semantically unrelated (`prompt.go` template
text; `runtime_isolation_test.go` comments about legacy DB table-name
prefixes) and the one real garbage-collection subsystem that exists
(`gc.go`) purges task workspace directories, not provider credentials or
account records, and does not reference Prodex/Codex/account inventory
anywhere in its logic or comments.

This is not a partial-credit or contract-completeness gap like several
other tasks in this change (e.g. 1.1-1.3's evidence-contract-incomplete
pattern) — the underlying behavior itself does not exist yet. Recommend
recording this as **MISSING** in the evidence index rather than PARTIAL,
and routing to a producer (not a reviewer) for implementation. I do not
self-accept or self-reject the checkbox; Kiro TL adjudicates.
