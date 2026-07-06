# Triple-Interaction Coexistence — CODEX_HOME x prodex x Herdr-codex-integration

Status: PLAN + ACCEPTANCE CRITERIA. LIVE PROOF F0-GATED (no live execution in this pass).
Owner: GLM#52#CLINE#B (G1), opus-4.8-orchestrator Tech-Lead approval required to run live.
Stream: G1-TRIPLE. Disjoint from docs/qa/{prod-redeem-validation-checklist,runtime-conformance-plan,smart-context-shadow-canary-plan}.md.

## 0. Purpose

Prove, with a gated test plan, that three distinct code paths that all touch the
Codex `CODEX_HOME` directory (and its `hooks` surface) can coexist without
clobbering the Multica account pool or breaking per-account auth isolation.

This is a plan/criteria deliverable. No live proof is run here. Live execution is
gated behind F0 prodex-as-is readiness, owner approval, and a green redaction
smoke (see §6). All test cases are PLANNED / `not-validated` until that gate opens.

Labeling convention (matches docs/prodex/prodex-runtime-invariants.md):

- `verified`: present in pinned source/docs read for this pass.
- `inferred`: required behavior derived from verified sources.
- `not-validated`: requires live/load validation (gated).

## 1. The Three CODEX_HOME Touchers

### Toucher 1 — Multica Go daemon execenv

`verified` from `multica-auth-work/server/internal/daemon/execenv/`:

- `execenv.go:254-263` — for provider `codex`, the daemon creates a per-task
  `CODEX_HOME` at `envRoot/codex-home` and sets `env.CodexHome`. This is the
  daemon's native isolation lever for Codex (`execenv.go:68`:
  "Codex: CODEX_HOME source").
- `codex_home.go` `prepareCodexHomeWithOpts`:
  - creates the per-task dir;
  - **symlinks** the `sessions/` directory from the shared home
    (`codexSymlinkedDirs`) so session logs stay in the global home;
  - **symlinks** `auth.json` from the shared home by default
    (`codexSymlinkedFiles`);
  - **copies** `config.json`, `config.toml`, `instructions.md` from the shared
    home (isolated copies, refresh-on-reuse via `syncCopiedFile`);
  - when `CodexHomeOptions.AccountHome` is set (the daemon's
    `CredentialAccountHome`), `seedAccountAuth` **copies** `auth.json` from
    `AccountHome/auth.json` instead of symlinking. Comment (`codex_home.go:53-56`):
    "OAuth clients rewrite auth.json on token refresh; a symlink would push one
    account's refresh onto the shared file and clobber the others. A copy keeps
    each account's refresh contained to its own per-task home."
- `resolveSharedCodexHome` reads the `CODEX_HOME` env var first, falls back to
  `~/.codex`.
- Managed blocks injected into the per-task `config.toml` only (never the user's
  global `~/.codex/config.toml`), each idempotent and marker-delimited
  (`# BEGIN multica-managed … / # END multica-managed …`):
  - sandbox block — `codex_sandbox.go` (`sandbox_mode`,
    `[sandbox_workspace_write] network_access`);
  - multi-agent block — `codex_multi_agent.go`
    (`features.multi_agent = false` unless `MULTICA_CODEX_MULTI_AGENT=1`),
    with explicit TOML `[features]` table-redefinition handling for `toml-rs`;
  - memory block — `codex_memory.go`.
- `hydrateCodexSkills` wipes and rewrites `CODEX_HOME/skills/` from the shared
  `~/.codex/skills/` plus workspace-assigned skills (workspace skills win).
- `codex_memory.go` writes agent turn summaries to `CODEX_HOME/memories/raw_memories.md`.
- `logCodexAuthState` records auth file **kind** only, never contents
  (redaction-compliant).

`not-validated`: the daemon's symlink/copy lists do NOT include any codex
`hooks` file or directory (confirmed: no `hooks.json`, `[hooks]`, or `hooks/`
reference in `daemon/execenv/`). Hooks are an **unmanaged surface** for Toucher 1.

### Toucher 2 — prodex (Rust L2 runtime plane)

`verified` from `multica-auth-work/server/internal/daemon/prodex.go` and
`docs/prodex/prodex-{l2-facade,runtime-invariants}.md`:

- `prodex.go` `applyProdexEnv` sets `PRODEX_HOME`, `MULTICA_PRODEX_*`,
  `PRODEX_SMART_CONTEXT_*`, and `PRODEX_KILL_SWITCH_DEFAULT_ON` for the codex
  provider. It does **NOT set `CODEX_HOME`**. Therefore the codex process that
  prodex launches **inherits the daemon's per-task `CODEX_HOME`** — there is a
  single `CODEX_HOME` owner on the F0/prodex-as-is path. This is the desired
  coexistence property and is asserted in test T2.
- prodex has its **own** profile/auth system under `PRODEX_HOME/profiles/<name>`
  (`prodex-runtime-invariants.md` §State Backends rule 3: "Profile auth
  isolation under `PRODEX_HOME/profiles/<name>` must remain stronger than
  convenience."). Relevant crates: `prodex-shared-codex-fs`,
  `prodex-profile-identity`, `prodex-profile-export`, `prodex-secret-store`.
- `prodex-l2-facade.md` `accounts.register`: "The fork must define how Go
  approved accounts map onto prodex profiles without mutating profile auth
  unexpectedly."
- Invariant `prodex-runtime-invariants.md` §State Backends rule 4: "Shared Codex
  state remains upstream-compatible and must not become profile authority."

`inferred` conflict surface: dual auth authority — the daemon's per-task
`CODEX_HOME/auth.json` (account-isolated copy) versus prodex's
`PRODEX_HOME/profiles/<name>` auth. A refresh through one path that does not
propagate to the other yields stale credentials or a cross-clobber.

`not-validated`: whether `prodex-shared-codex-fs` writes `CODEX_HOME` directly
(sessions, logs, config) concurrently with the daemon's symlink/copy model.

### Toucher 3 — Herdr-codex-integration

`verified` from `.deploy-control/HERDR_COMMS_GUIDE.md` and the operational fleet
model: Herdr is a terminal-native agent multiplexer; each pane is a real terminal
with its own shell and environment; fleet agents (e.g. Codex#5.5#A/B/C/D, GLM
agents) run inside Herdr panes; `HERDR_ENV=1` marks a Herdr-managed pane.

`inferred`: a codex agent running in a Herdr pane executes the `codex` CLI, which
reads the `CODEX_HOME` env var from the pane environment. If the pane does not
export `CODEX_HOME`, codex defaults to the shared `~/.codex` — the **same**
shared home the daemon symlinks `sessions/` and (by default) `auth.json` from,
and copies `config.json/config.toml/instructions.md` from.

`not-validated`: the Herdr skill files referenced by the codebase search index
(`.kiro/skills/herdr/SKILL.md`, `.kiro/skills/herdr-guide/SKILL.md`) are not
present on disk in this checkout (`.kiro/skills/` contains only openspec
skills). Herdr behavior here is grounded in the on-disk
`.deploy-control/HERDR_COMMS_GUIDE.md` and the observed fleet operation, not in a
checked-in Herdr skill.

## 2. The Hooks Surface (cross-cutting)

`inferred`: Codex supports a hooks mechanism (a legacy `hooks.json` is referenced
by the codebase tooling manifest as `002-codex-legacy-hooks-json.cjs`; modern
Codex may encode hooks as a `[hooks]` table in `config.toml` or a `hooks/`
directory inside `CODEX_HOME`).

`verified` gap: the daemon's `codex_home.go` symlink/copy lists include only
`sessions/`, `auth.json`, `config.json`, `config.toml`, `instructions.md`. They
do **not** include any hooks file or directory. So across all three touchers,
hooks are an **unmanaged surface**:

- a hook installed in the shared `~/.codex` (e.g. by a Herdr-pane codex or an
  operator) can leak into a daemon per-task `CODEX_HOME` via the copied
  `config.toml` if hooks live in `config.toml`, OR be silently absent if hooks
  live in an unmanaged `hooks.json`/`hooks/` dir;
- a hook expected by prodex may be missing from the per-task `CODEX_HOME`;
- a hook from one account/context may fire inside another account's isolated
  session, breaking the account-pool boundary.

This is the most subtle coexistence risk because it is invisible to the daemon's
current isolation model. It must be explicitly managed or blocked before
coexistence is declared proven (test T7).

## 3. Conflict Risks

`R1 — auth.json clobber / account-pool contamination (HIGHEST).`
If any toucher writes the shared `~/.codex/auth.json` (the daemon's default
symlink target when `AccountHome` is empty, or a Herdr-pane codex that defaults
to `~/.codex`, or prodex refreshing into the wrong store), an OAuth refresh
overwrites the shared file and contaminates every account. The daemon's
copy-based isolation holds only when `AccountHome` is set AND no other toucher
writes the shared file. prodex's separate `PRODEX_HOME/profiles/<name>` store
adds a second auth authority that can drift.

`R2 — config.toml managed-block collision.`
The daemon injects sandbox/multi-agent/memory managed blocks into the per-task
`config.toml`. If prodex (policy apply) or a Herdr-pane codex/user also writes
`config.toml` — or if the shared `~/.codex/config.toml` the daemon copies on the
next prepare has drifted — the managed blocks can be overwritten/lost, or a
`[features]` table redefinition can make `toml-rs` reject the file.
`codex_multi_agent.go` handles table redefinition only for the daemon's own
injection, not for external writers.

`R3 — sessions/ symlink contention.`
The daemon symlinks `sessions/` to the shared `~/.codex/sessions`. A Herdr-pane
codex, a daemon task, and prodex all writing session logs to the same symlinked
directory can interleave/corrupt logs and let continuation affinity
(`previous_response_id`, `session_id`) bind to the wrong session.

`R4 — skills/ wipe race.`
`hydrateCodexSkills` does `RemoveAll` + rewrite of `CODEX_HOME/skills/`. A
concurrent reader/writer (prodex, Herdr-pane codex) can see a partial skills dir.

`R5 — hooks non-management.`
Hooks installed by any toucher leak into or drop from per-task homes
non-deterministically (see §2). A hook from one account/context firing inside
another account's session breaks isolation.

`R6 — CODEX_HOME ownership ambiguity.`
`prodex.go` does not set `CODEX_HOME` today (inherits the daemon's — good). But
prodex's profile system implies a future build could set
`CODEX_HOME=PRODEX_HOME/profiles/<name>`, which would override the daemon's
per-task `CODEX_HOME`, break the daemon's isolation contract, and discard the
managed blocks. This must be codified as an invariant (T2) before coexistence is
trusted.

`R7 — shared-home authority tension.`
prodex invariant: "Shared Codex state remains upstream-compatible and must not
become profile authority." The daemon's model treats the shared `~/.codex` as the
seed authority for config/sessions/skills. The fork must reconcile which store is
authoritative for what, or the three touchers will silently fight over the shared
home.

`R8 — memories/ concurrent writes.`
`codex_memory.go` writes `CODEX_HOME/memories/raw_memories.md`. A concurrent
Herdr-pane codex or prodex writing the same file can corrupt it.

## 4. Test Plan (PLANNED — LIVE PROOF F0-GATED)

All tests are `not-validated` until the §6 gate opens. Each test states setup,
action, PASS criteria, and FAIL-CLOSED criteria. No live execution in this pass.
Fake markers only for redaction tests; no real credentials in evidence.

### T1 — Daemon-only baseline isolation (account pool not clobbered)

Setup: two daemon tasks with two distinct `AccountHome` credential dirs
(account A, account B), both provider `codex`, no prodex, no Herdr-pane codex.
Action: run both tasks; trigger an OAuth refresh inside task A.
PASS: per-task `CODEX_HOME/auth.json` for A and B are independent copies whose
sha256 match their respective `AccountHome/auth.json` sources and differ from
each other; the shared `~/.codex/auth.json` is unchanged; `logCodexAuthState`
records kind `copy` (not `symlink`) for both; no raw token in logs.
FAIL-CLOSED: any per-task auth.json equals the shared file, or A's refresh
mutates B's auth.json → session aborts, readiness fails, redacted error.

### T2 — prodex inherits daemon CODEX_HOME (single owner invariant)

Setup: one daemon task, provider `codex`, prodex enabled (`MULTICA_PRODEX_ENABLED=1`,
pinned version/commit).
Action: inspect the env of the codex process that prodex launches.
PASS: `CODEX_HOME` equals the daemon's per-task `envRoot/codex-home`; the
`prodex.go` `applyProdexEnv` env delta contains no `CODEX_HOME` key; prodex
profile dir `PRODEX_HOME/profiles/<name>` is NOT used as `CODEX_HOME`.
FAIL-CLOSED: `CODEX_HOME` is unset, set to the shared `~/.codex`, or set to a
prodex profile dir → prodex launch refused; the single-owner invariant is
violated.

### T3 — prodex profile auth vs daemon account auth non-clobber

Setup: T2 task with `AccountHome` set; prodex profile under
`PRODEX_HOME/profiles/<name>` populated.
Action: trigger an auth refresh through prodex's profile path.
PASS: the daemon per-task `CODEX_HOME/auth.json` sha256 is stable unless the
daemon's `seedAccountAuth` refresh-on-reuse runs; prodex writes only under
`PRODEX_HOME/profiles/<name>`; no cross-write to `AccountHome` or shared
`~/.codex/auth.json`; no raw credential in any log/event.
FAIL-CLOSED: prodex writes `CODEX_HOME/auth.json` or `AccountHome/auth.json` or
the shared `~/.codex/auth.json` → session aborts; dual-authority clobber
detected.

### T4 — Herdr-pane codex isolation from daemon tasks

Setup: a Herdr-managed codex pane (`HERDR_ENV=1`) running `codex` with
`CODEX_HOME` unset (defaults to `~/.codex`); one in-flight daemon task with
`AccountHome` set; one in-flight daemon task WITHOUT `AccountHome` (legacy
symlink path).
Action: perform an OAuth refresh inside the Herdr-pane codex.
PASS: the daemon task WITH `AccountHome` is insulated (its per-task auth.json
sha256 unchanged); the shared `~/.codex/auth.json` changes (Herdr-pane codex
refreshed it); the daemon task WITHOUT `AccountHome` is contaminated (its
symlinked auth.json now matches the shared file) → this is the documented
hard-gate result requiring `AccountHome` for any task co-located with Herdr codex
panes.
FAIL-CLOSED: a task WITH `AccountHome` is contaminated → isolation contract
broken; abort.

### T5 — config.toml managed-block survival under external write

Setup: a daemon per-task `CODEX_HOME/config.toml` with the daemon's sandbox +
multi-agent + memory managed blocks; simulate an external write (prodex policy
apply, or a user/Herdr-pane edit to the shared `~/.codex/config.toml` that the
daemon copies on the next prepare) adding a `[features]` table and extra keys.
Action: re-prepare the env (`Reuse`) so the daemon re-injects its managed blocks.
PASS: all three managed blocks are present and idempotent (marker-delimited, no
duplicate blocks); `toml-rs` accepts the file (no "table 'features' already
exists"); the external keys are preserved unless they conflict with a managed
key; codex parses and starts.
FAIL-CLOSED: a managed block is missing/duplicated, or `toml-rs` rejects the
file → prepare fails, no session starts.

### T6 — sessions/ symlink non-interleaving

Setup: one daemon task (per-task `CODEX_HOME` with `sessions/` symlinked to
shared `~/.codex/sessions`) and one Herdr-pane codex (using `~/.codex`
directly); prodex hard-affinity binding present.
Action: both start sessions concurrently.
PASS: each session's `session_id` is scoped to its own `CODEX_HOME`/shared dir
path and does not cross-bind; prodex continuation affinity
(`previous_response_id`, `session_id`) binds to the correct profile only; no
session log file is corrupted or truncated.
FAIL-CLOSED: a session_id or continuation binding crosses homes, or a log file
is corrupted → sessions abort; affinity invariant violated.

### T7 — hooks non-management containment

Setup: install a codex hook in the shared `~/.codex` (in `config.toml` as a
`[hooks]` table AND/OR as a `hooks/` dir / `hooks.json`, whichever Codex uses);
prepare a daemon per-task `CODEX_HOME`.
Action: start the daemon task and observe whether the hook fires.
PASS (one of): (a) the hook is **deterministically excluded** from the per-task
`CODEX_HOME` and does NOT fire (hooks are blocked/isolated), OR (b) the hook is
**deterministically included** and documented as an intentional shared surface
with no account-specific payload. Either way the behavior is declared, not
silent.
FAIL-CLOSED: the hook leaks non-deterministically, or a hook from one
account/context fires inside another account's isolated session → isolation
boundary broken; abort. This test MUST close before coexistence is declared
proven (hooks are currently an unmanaged surface — see §2).

### T8 — skills/ no race

Setup: a daemon per-task `CODEX_HOME` with workspace + user skills; a concurrent
reader (a Herdr-pane codex or prodex) polling `CODEX_HOME/skills/`.
Action: trigger `hydrateCodexSkills` (which does `RemoveAll` + rewrite) while the
reader is active.
PASS: the final `CODEX_HOME/skills/` set is deterministic and matches the
expected union (workspace skills win on name conflict); no partial/missing skill
files are observed by the reader at rest; no stale user skill lingers.
FAIL-CLOSED: the reader observes a partial skills dir, or the final set does not
match expected → prepare fails.

### T9 — kill-switch + fail-closed on coexistence violation

Setup: a daemon task with prodex enabled and a Herdr-pane codex present; inject
one coexistence violation (e.g. simulate prodex setting `CODEX_HOME` to a profile
dir, or an auth-contamination signal, or a missing managed block).
Action: attempt to start/continue the session.
PASS: readiness fails closed; the session does NOT start; the error is redacted
(no raw token/credential); the kill switch can disable the offending feature
scope; durable audit row written.
FAIL-CLOSED: the session starts despite the violation, or a raw secret appears
in the error/log → critical rollback trigger (see rollback-runbook.md §2).

### T10 — redaction smoke across all three touchers

Setup: inject fake markers (`sk-test-secret`, `Bearer test-secret`,
`postgres://user:pass@example/db`, `redis://:pass@example:6379`) into the outputs
of each toucher (daemon logs, prodex runtime logs, a Herdr-pane codex transcript).
Action: run the coexistence scenario and collect logs/events/evidence.
PASS: zero unredacted markers in Go logs, prodex logs, runtime events, evidence,
or pasted command output; all markers appear only as
`[REDACTED:<kind>:sha256:<first12>]` per `docs/security/secrets-redaction-policy.md`;
runtime events carry `secrets_present=false`.
FAIL-CLOSED: any marker appears unredacted → hard stop; redaction scrubber
treated as disabled (secrets-redaction-policy.md §6).

## 5. Isolation Invariants to Assert (coexistence contract)

The following must hold for the duration of any coexistence scenario; each is
asserted by the test(s) in parentheses:

1. **Single `CODEX_HOME` owner for daemon-managed codex tasks.** The daemon sets
   per-task `CODEX_HOME`; prodex must NOT override it (T2). A Herdr-pane codex is
   a separate, non-daemon-managed process and must NOT share a per-task
   `CODEX_HOME` with a daemon task (T4, T6).
2. **Per-account auth isolation via copy, not symlink, whenever `AccountHome` is
   set.** A refresh in one account never mutates another account's credential or
   the shared `~/.codex/auth.json` (T1, T3, T4).
3. **`AccountHome` is MANDATORY** for any codex task co-located with Herdr codex
   panes or prodex in the same host/workspace. The legacy shared-symlink auth
   fallback is forbidden in coexistence topologies (T4).
4. **prodex profile auth stays under `PRODEX_HOME/profiles/<name>`** and does not
   write `CODEX_HOME/auth.json`, `AccountHome/auth.json`, or the shared
   `~/.codex/auth.json` (T3).
5. **Daemon managed `config.toml` blocks survive external writes** and remain
   idempotent and `toml-rs`-parseable (T5).
6. **`sessions/` and continuation affinity do not cross homes.** A session_id
   and `previous_response_id` binding belong to exactly one `CODEX_HOME`/profile
   (T6).
7. **Hooks are a declared surface** — either deterministically isolated or
   explicitly shared — never silent (T7).
8. **`skills/` hydration is race-free at rest** (T8).
9. **Any coexistence violation fails closed** with a redacted error and a durable
   audit row; kill switch can scope-disable the offender (T9).
10. **No raw secret in any log/event/evidence/command output** from any toucher
    (T10).

## 6. Acceptance / Gating Criteria to Unlock Live Proof

Live execution of T1–T10 is F0-GATED. All must be true before any live run:

- F0 prodex-as-is readiness is established and the prodex pin (version + commit +
  artifact sha256) is verified per `docs/deploy/l2-sidecar-deploy-plan.md` §2.
- Owner approval is recorded with `deploy_owner_approved: true` in
  `.deploy-control/evidence/status-board.md` per `docs/deploy/prod-rollout-runbook.md` §1.
- Redaction smoke is green (`docs/security/secrets-redaction-policy.md` §5;
  fake markers only) for Go logs, prodex logs, the runtime event stream, and
  evidence.
- `AccountHome` (per-account credential dir) is provisioned for every codex
  account used in the scenario; the legacy shared-symlink auth fallback is
  disabled for coexistence topologies (Invariant 3).
- The single-`CODEX_HOME`-owner invariant is codified for prodex (prodex must
  NOT set `CODEX_HOME`) — assert via T2 before any prodex launch.
- The hooks surface is explicitly managed or blocked (T7 closes) before
  coexistence is declared proven.
- ext4 / mode 0600 (files) / 0700 (dirs) credential invariants hold for the
  daemon per-task `CODEX_HOME`, `AccountHome`, and `PRODEX_HOME/profiles`
  roots; none resolves onto 9p or a shared host mount
  (`docs/deploy/l2-sidecar-deploy-plan.md` §3).
- Kill switch store is reachable and the fail-closed path (T9) is exercised in a
  non-destructive path first.
- No live run is authorized by this document alone; this pass delivers plan +
  criteria only.

## 7. Non-Goals

- No live execution / no real PROD deploy / no real redeem (F0-GATED).
- No change to the daemon's isolation model or to prodex in this pass
  (recommendations only; implementation is owned by the F2/F3 streams).
- No redefinition of the Go/Rust authority boundary
  (`docs/contracts/l2-runtime-contract.md` §1 stands).
- No claim that the Herdr-codex-integration is a checked-in artifact; it is an
  operational fleet reality grounded in `.deploy-control/HERDR_COMMS_GUIDE.md`.

## 8. References

- `multica-auth-work/server/internal/daemon/execenv/codex_home.go` — per-task
  `CODEX_HOME` preparation, symlink/copy lists, `seedAccountAuth`, `resolveSharedCodexHome`.
- `multica-auth-work/server/internal/daemon/execenv/codex_multi_agent.go` —
  managed `features.multi_agent` block + TOML table-redefinition handling.
- `multica-auth-work/server/internal/daemon/execenv/codex_sandbox.go` — managed
  sandbox block.
- `multica-auth-work/server/internal/daemon/execenv/codex_memory.go` —
  `CODEX_HOME/memories` writes.
- `multica-auth-work/server/internal/daemon/execenv/execenv.go` — `CODEX_HOME`
  ownership and `CredentialAccountHome` contract.
- `multica-auth-work/server/internal/daemon/prodex.go` — `applyProdexEnv` env
  delta (sets `PRODEX_HOME`, not `CODEX_HOME`).
- `docs/prodex/prodex-l2-facade.md` — `accounts.register` mapping rule.
- `docs/prodex/prodex-runtime-invariants.md` — `PRODEX_HOME/profiles/<name>`
  isolation and shared-Codex-state authority invariant.
- `docs/contracts/l2-runtime-contract.md` — Go/Rust authority boundary.
- `docs/deploy/l2-sidecar-deploy-plan.md` — artifact pinning, ext4/permission
  invariants.
- `docs/deploy/prod-rollout-runbook.md` — owner approval gate, redaction smoke.
- `docs/deploy/rollback-runbook.md` — rollback triggers.
- `docs/security/secrets-redaction-policy.md` — redaction format and fail-closed
  conditions.
- `.deploy-control/HERDR_COMMS_GUIDE.md` — Herdr pane/env model (`HERDR_ENV=1`).
- `.deploy-control/STATUS_REPORTING_STANDARD.md` — check-in front-matter.

## 9. Conclusion

The three CODEX_HOME touchers can coexist without clobbering the account pool or
isolation **iff** the §5 invariants hold and the §6 gate opens before live proof.
The dominant risk is R1 (auth.json clobber via the shared `~/.codex` symlink
target when `AccountHome` is unset, or via a Herdr-pane codex defaulting to
`~/.codex`); it is mitigated by making `AccountHome` mandatory in coexistence
topologies. The most subtle risk is R5 (hooks are currently an unmanaged
surface); T7 must close before coexistence is declared proven. prodex inheriting
the daemon's per-task `CODEX_HOME` (T2) is the key single-owner property that
keeps the daemon and prodex from fighting over `CODEX_HOME` on the F0 path.






