# DESIGN-ONLY — persist-prodex-runtime-integration task 3.5 (legacy credential + obsolete account purge)

- Author: independent design reviewer (Kiro/Opus-4.8) — 2026-07-18. **Design only; NO code, NO acceptance claim.**
- Task 3.5 (confirmed MISSING implementation): "Purge unreferenced legacy credential files and obsolete
  unassigned account records while preserving non-credential agent state."
- Governing spec requirement: `prodex-runtime-continuity` → "Obsolete credentials are removed" (2 scenarios)
  + "One slot per credential identity" (no global/cross-slot fallback).
- Role: independent; TL/owners adjudicate. This artifact proposes; it does not implement or self-accept.

## Golden Rule check-in / check-out

- **Check-IN** 2026-07-18T20:56:00Z — claimed: read-only source study + one design artifact (this file).
- Excluded (honored): no credential/auth-home/env **contents** read; no deletion/quarantine performed; no
  product/test/spec/`tasks.md`/`design.md` edits; no DB/network/live-provider; no git stage/commit/push.
- **Check-OUT** 2026-07-18T21:12:00Z — DONE; design below; no product or OpenSpec artifact modified.

## Provenance — current source hashes (SHA-256, read-only)

| File | SHA-256 |
|---|---|
| `openspec/changes/persist-prodex-runtime-integration/tasks.md` | `de661603af6b0ec1aece2ffe442f446ece86cc9dd91aa596fa81fc1553b3eaea` |
| `openspec/changes/persist-prodex-runtime-integration/design.md` | `2bf341e8323467f1fc8235190c6f42711ffae392d081849a7c7127063ffb7feb` |
| `.../specs/prodex-runtime-continuity/spec.md` | `4a47a6cfc92b5c63acdd8baf83556063f84d47dbd9a9e8270b38e6988375c251` |
| `server/internal/daemon/prodex_profiles.go` | `afe8b75f024e825eaea301c22566e28e653ccb57727e596905858a385489e4c8` |
| `server/internal/daemon/execenv/codex_home.go` | `aa3f8bbd82ffb3fe70ed72da037eaae3028af4b558a7b1d0c207ccaeb4cb3fe0` |
| `server/cmd/server/runtime_sweeper.go` | `f654797b20d2bef03b6492a8ac982409f5aef67d0a5038ff740107f137f7c418` |

Grounding facts (verified in source): `accounts` live in Postgres via `rotation.Store` — interface has
**no delete** (`ListAccounts/GetAccount/UpdateAccountStatus/RecordUsage/Assign/CurrentAssignment/RecordRotation`
only). Slot layout `<slotsRoot>/slot-*/codex/auth.json` (dir 0700, file 0600); `slotsRoot` =
`MULTICA_AGENT_CREDENTIAL_SLOTS_ROOT` (default `~/.agent-cred-homes/slots`). `PRODEX_HOME/state.json` holds
`profiles:{name:{codex_home}}`. `validateApprovedPOSIXFilesystem` (prodex_fs_linux.go:16) accepts only
ext4/xfs; the `!linux` variant (prodex_fs_other.go:10) **always errors** (fails closed off-Linux).

## 1. Proposed files / functions (all NEW; additive)

### 1.1 `server/internal/daemon/prodex_credential_purge.go` (package `daemon`, co-located with reconcile)

```
type PurgeReasonCode string   // opaque, value-free classification
const (
    ReasonUnreferencedSlotAuth PurgeReasonCode = "unreferenced_slot_auth"
    ReasonObsoleteLegacyAccount PurgeReasonCode = "obsolete_legacy_account"
    ReasonSkippedActiveLease    PurgeReasonCode = "skipped_active_lease"
    ReasonSkippedCrossTenant    PurgeReasonCode = "skipped_cross_tenant"
    ReasonRejectedUnsafePath    PurgeReasonCode = "rejected_unsafe_path"
    ReasonRejectedNonPOSIX      PurgeReasonCode = "rejected_non_posix_fs"
)

type PurgeOptions struct {
    DryRun         bool          // DEFAULT true — report-first
    QuarantineRoot string        // MUST be same-device sibling of slotsRoot
    Retention      time.Duration // quarantine dwell before permanent delete (default 7d, mirrors offlineRuntimeTTLSeconds)
    MaxItemsPerRun int           // batch cap (mirrors queuedExpireBatchSize)
}

type PurgeItem struct {          // NEVER carries secret bytes or full credential paths
    Kind       string            // "slot_auth" | "account_record"
    OpaqueID   string            // opaque profile/slot identifier (prodexProfileName) — never a raw path
    Reason     PurgeReasonCode
    Action     string            // "would_quarantine" | "quarantined" | "would_soft_delete" | "soft_deleted" | "skipped" | "rejected"
}

type PurgePlan struct {
    RunID        string
    TenantID     string
    Referenced   int              // authoritative homes kept
    Candidates   []PurgeItem
    Preserved    int              // non-credential state left intact
}

// planCredentialPurge is READ-ONLY: builds the reference graph and returns the plan. Writes nothing.
func (d *Daemon) planCredentialPurge(ctx context.Context) (PurgePlan, error)

// applyCredentialPurge executes quarantine-before-delete for a plan. Refuses unless opts.DryRun==false
// AND the operator gate is set. Two-phase; never os.Remove in the hot path.
func (d *Daemon) applyCredentialPurge(ctx context.Context, plan PurgePlan, opts PurgeOptions) (PurgeResult, error)

buildReferenceGraph(accounts []rotation.Account, state prodexState, slotsRoot string) (referenceGraph, error) // pure
quarantineCredentialFile(src, quarantineRunDir string) (manifestEntry, error) // atomic os.Rename, same-device only
writeQuarantineManifest / rollbackQuarantine(runID) / commitQuarantine(runID, retention)
```

Reuses existing helpers **unchanged**: `validateApprovedPOSIXFilesystem`, `validateCodexSlotHome`,
`validateDirectoryMode`, `prodexProfileName`, `loadProdexState`, `redactCommandOutput`,
and `sha256.Sum256` for identity (computed, never logged).

### 1.2 `rotation.Store` extension (design proposal — OWNER-GATED, see §11)

```
ListLegacyUnassignedAccounts(ctx, vendor, tenantID) ([]Account, error) // read-only classifier
SoftDeleteAccount(ctx, accountID string) error                          // sets legacy=true, purged_at=now(), home_dir=NULL
HardDeleteAccount(ctx, accountID string, olderThan time.Time) error     // only after retention; single tx
```

An account is **obsolete** iff: explicitly classified legacy (new `legacy` column) AND
`CurrentAssignment` returns none for every agent referencing it. Never inferred from `StatusDegraded`
alone (degraded is transient).

### 1.3 Wiring

Called **after** a successful `reconcileProdexProfiles` (so the authoritative referenced set +
`l2ProfileByHome` are known). Gated behind `MULTICA_PRODEX_CREDENTIAL_PURGE` (default **off** →
report-only). Optionally scheduled on the existing sweeper cadence (`runtime_sweeper.go`) in
**report-only** mode; destructive apply is a separate operator-invoked command.

## 2. Reference graph

- Nodes: (A) validated `accounts` rows for `(vendor=codex, tenantID)` — **authoritative**; (B) on-disk slot
  codex homes under `slotsRoot`; (C) `state.json` profiles; (D) credential identity = `sha256(auth.json bytes)`.
- Edges: `account.HomeDir → slot codex home`; `state.profile → codex_home`; `identity → owning profile`.
- **Referenced set** = homes referenced by validated rows that PASS `validateCodexSlotHome`.
- **Unreferenced legacy credential file** = an `auth.json` under `slotsRoot` whose home ∉ referenced set.
- **Obsolete account record** = legacy-classified row with no current assignment (via `CurrentAssignment`).
- Purge candidate granularity = the **credential file only** (`auth.json`), never the slot directory.

## 3. Approved roots

Operate strictly within `slotsRoot` (abs + `validateApprovedPOSIXFilesystem`) and `PRODEX_HOME`. Every
candidate is re-validated with `filepath.Rel`/`..`/prefix checks (same guard as `validateCodexSlotHome`).
QuarantineRoot MUST be a sibling **on the same device** as `slotsRoot` (so `os.Rename` is atomic; EXDEV →
fail closed). Any path escaping the approved root, or reached via symlink, is `rejected`, never followed.

## 4. Dry-run / report-first mode

`DryRun=true` is the default and the only mode enabled without the operator gate. It returns a `PurgePlan`
(counts + opaque IDs + reason codes) and **writes nothing** — verifiable by asserting the filesystem and
`accounts` table are byte/row-identical before and after (test §10.1). Apply requires explicit opt-in.

## 5. Atomic quarantine-before-purge

Two phases, never a direct delete on the hot path:
1. **Quarantine**: `os.Rename(authPath, <QuarantineRoot>/<RunID>/<opaqueID>/auth.json)` — atomic, same-device.
   A manifest (`<QuarantineRoot>/<RunID>/manifest.json`) records original path, mode, mtime, size, and an
   opaque identity **prefix only** (never the full hash, never contents).
2. **Commit**: permanent delete happens only in a later, separately-gated step after `Retention` elapses
   (`commitQuarantine`). Account rows: `SoftDeleteAccount` first; `HardDeleteAccount` only after retention.

## 6. Rollback / recovery

`rollbackQuarantine(runID)` renames each manifest entry back to its original path (mode/mtime restored),
restoring byte-identical credential files while within the retention window. Account rollback = clear
`legacy`/`purged_at`, restore `home_dir` (soft-delete is reversible; hard-delete is not, hence retention).
Recovery is idempotent and logs only counts + opaque IDs.

## 7. Tenant / account isolation

All account queries scoped by `(vendor, tenantID)` from `d.cfg.L2Runtime.TenantID`; never cross-tenant. The
reference graph, quarantine namespace (`<QuarantineRoot>/<tenantID>/<RunID>`), and audit records are
per-tenant. A slot home not resolvable under the configured tenant's `slotsRoot` is `skipped_cross_tenant`,
never purged — prevents deleting another tenant's material.

## 8. Preservation of non-credential agent state

Purge-eligible = a documented **allowlist of credential filenames only** (`auth.json` for codex; per-vendor
credential-store names added only behind declared capability support). Everything else in the slot —
`config.toml`, skills, memory, caches, logs, sessions, prompts, workspaces — is preserved. Implementation
operates at **file granularity** (`os.Rename` of the single credential file); it MUST NOT `os.RemoveAll` a
slot directory. This mirrors the existing "no full envRoot RemoveAll" rule (gc.go:200) and the
"Cleanup MUST NEVER delete the user's own path" precedent (execenv.go:804). Removing an unreferenced slot's
`auth.json` leaves the slot and its non-credential contents intact.

## 9. Windows / POSIX hazards

- **Primary anchor**: `validateApprovedPOSIXFilesystem` accepts only ext4/xfs on Linux and **always errors
  off-Linux** (prodex_fs_other.go). On Windows/WSL-drvfs/9p/CIFS the purge **no-ops with `rejected_non_posix_fs`**
  — no deletion is ever attempted on a non-approved filesystem.
- Cross-device rename (EXDEV): enforce same-device QuarantineRoot; on EXDEV fail closed (no copy-then-delete
  fallback, which would risk a partial/leaked copy).
- Symlink/reparse: resolve and **reject** symlinked credential files (never follow into another slot).
- Case-insensitive collisions (`auth.json` vs `Auth.json`): allowlist match is case-sensitive; on a
  case-insensitive FS the non-POSIX guard already blocks the run.
- Open-handle / lock: check no active lease/assignment (`CurrentAssignment`) and no `StatusLeased` before
  quarantining a slot; `skipped_active_lease` otherwise.
- Reject UNC/`\\`, path-length, and ADS concerns are moot under the POSIX-only gate but documented for review.

## 10. Pure synthetic destructive-safety tests (design; all offline, `t.TempDir()`, fake bytes)

1. Dry-run writes nothing (FS + rows unchanged; plan lists candidates).
2. **Safety invariant**: a referenced slot's `auth.json` is NEVER quarantined/deleted.
3. Non-credential files in an unreferenced slot are preserved; only `auth.json` is quarantined.
4. Quarantine is atomic + same-device; manifest records original→quarantined; `rollback` restores
   byte-identical.
5. Cross-tenant slot is `skipped_cross_tenant` (isolation).
6. Path-escape/symlink candidate is `rejected`, not followed.
7. EXDEV / cross-device QuarantineRoot → fail closed, zero deletions.
8. Account with any current assignment is NEVER deleted even if legacy-classified.
9. Soft-delete precedes hard-delete; retention window enforced (no hard-delete before `Retention`).
10. **No secret bytes in logs**: capture the redacted slog buffer; assert synthetic sentinel absent
    (ties to the accepted `pkg/redact` mechanism / task 5.4).
11. Idempotency: a second run after apply is a no-op.
12. Malformed/zero-byte `auth.json` handled without panic and treated as NOT referenced (never selected).
13. Batch cap `MaxItemsPerRun` bounds a single run.
14. Non-POSIX filesystem (simulated guard failure) → `rejected_non_posix_fs`, zero deletions.

## 11. AB-REQ / EV mapping · conflict scan · owner gates

### AB-REQ / EV
- **AB-REQ-3.5-A** ← spec "Obsolete credentials are removed / Unreferenced Codex slot" → §2, §5, §8, tests 2/3.
- **AB-REQ-3.5-B** ← spec "Obsolete credentials are removed / Obsolete account record" → §1.2, §5, §6, tests 8/9.
- **AB-REQ-3.5-C** ← spec "One slot per credential identity" (no global/cross-slot fallback) → §2 identity graph, test 5/6.
- **EV-PPRI-3.5-DESIGN** = this artifact (design only; not acceptance evidence).

### Conflict scan
- `prodex_profiles.go` is owned by reconciliation tasks 3.1–3.4; the new `prodex_credential_purge.go` is
  **additive** and only **reads** shared `Daemon` fields (`l2ProfileByHome`, `reconciledL2Profiles`) under the
  existing `l2ProfilesMu`. Coordinate merge ordering with the 3.1–3.4 owner.
- `rotation.Store` interface + `store_pg.go` are owned by the rotation/W-PGSTORE stream; the delete/soft-delete
  additions and the `accounts` schema migration (`legacy bool`, `purged_at timestamptz`, nullable `home_dir`)
  are **cross-owner** changes.
- Depends on the `agent-credential-isolation` Phase-1 slot mechanism (same `accounts`/slots surface) — align
  before implementing; no overlap with `build-omniroute-agent-brain`.

### Owner gates (all must clear before any implementation)
1. **DB/migration owner + operator**: `accounts` soft-delete schema migration (destructive-capable).
2. **Rotation owner**: `rotation.Store` interface extension (delete/soft-delete/legacy classifier).
3. **Operator gate**: destructive apply behind `MULTICA_PRODEX_CREDENTIAL_PURGE=1` + explicit invocation;
   default is report-only. Permanent delete (post-retention `commit`) is a separate operator-gated action.
4. **Kiro TL**: adjudicates this design. **No self-acceptance; no `tasks.md`/`design.md`/`spec.md` edits made.**

## Non-claims
- Design only. No code produced; no credential/auth-home/env contents inspected; no deletion/quarantine run;
  no DB/network/live-provider access; no spec/task/product edits; no git operations; no acceptance asserted.
