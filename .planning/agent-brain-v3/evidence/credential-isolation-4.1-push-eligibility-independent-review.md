# Independent push-eligibility review — agent-credential-isolation task 4.1

- Reviewer: independent (Kiro/Opus-4.8), read-only reproduction + provenance trace. **Does not self-accept; does not authorize a push.**
- Scope: **task 4.1 only** — detector source/test candidates. Kiro TL adjudicates; root controls integration.
- Toolchain: `/home/dataops-lab/go-sdk/bin/go` (`go1.26.4`), `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`; offline.
- Basis document: `credential-current-file-acceptance-matrix.md` (2 of 27 files named as 4.1 candidates).

## Golden Rule check-IN / check-OUT

- **Check-IN** 2026-07-18T21:23:00Z — read-only; sole writable deliverable is this artifact.
- Excluded (honored): no shared docs/STATE/ledger/EVIDENCE_INDEX/OpenSpec/product/test/git/index edit; no
  credential/env value; no DB/network/service. Only this file written.
- **Check-OUT** 2026-07-18T21:41:00Z — DONE; verdict below.

## Candidates (current bytes verified at HEAD `b6571299`)

| File | Current SHA-256 | Matrix SHA-256 | Match | Git state |
|---|---|---|---|---|
| `internal/rotation/detector_discovery.go` | `bc61a46c0c700010001a49c9075b87f97e89f802b138e63f90b53193a4b45b55` | `bc61a46c…45b55` | ✅ | untracked (`??`) |
| `internal/rotation/detector_discovery_test.go` | `4e8092ff6b0f37a8416b961c89040433682e2d0d654293e192992dd3e1d55a4f` | `4e8092ff…5a4f` | ✅ | untracked (`??`) |

No drift since the matrix snapshot (same HEAD). Both files are new/untracked.

## OpenSpec checkbox + accepted EV / artifact hashes

- `openspec/changes/agent-credential-isolation/tasks.md`: **4.1 `[x]`** ("Detectar sessão esgotada/`expired`
  (status + expires_at do discovery)"). (4.3/4.4/5.3/5.4 remain `[ ]`.)
- `EVIDENCE_INDEX.md:130`: `EV-CREDISO-4.1 | task 4.1 | ACCEPT | AGENT_LEDGER row | …focused ×20/race/vet/gofmt/diff`.
- `AGENT_LEDGER.md:196`: `independent reviewer | cred-iso-4.1-accept | … ACCEPT | read-only | EV-CREDISO-4.1 |
  Detect exhausted/expired … focused ×20/race/vet/gofmt/diff all accepted … no checkbox set by TL.`
- Detector hash `bc61a46c…` is also pinned in independently reproduced **EV-CREDISO-4.2**, artifact
  `credential-isolation-next-account-selection.md`, **verified SHA-256 `d5a8022873bd5ae359e7d9cb1fda09563d909e3a753fff8f69c8e50ab60f804f`** (matches matrix). EV-4.2 explicitly states it does **not** re-grade 4.1 —
  EV-CREDISO-4.1 remains the detection acceptance of record.

## Producer / reviewer separation

- Acceptance is attributed to an **"independent reviewer"** (AGENT_LEDGER row `cred-iso-4.1-accept`), not the
  source author; task checkbox "not set by TL" per the row. Separation is **asserted and directionally sound**.
- **Provenance thinness (documented, not fatal):** EV-CREDISO-4.1 exists only as an EVIDENCE_INDEX/AGENT_LEDGER
  row — there is **no standalone EV-4.1 artifact file** with a named producer and a pinned source manifest. The
  sibling `active-accepted-push-candidate-matrix.md:201/225` independently flags this as a **HOLD** ("only an
  AGENT_LEDGER summary is indexed; no file-level provenance"). This review corroborates the *technical* basis by
  first-hand reproduction (below) but cannot elevate ledger-row provenance to artifact-grade provenance.

## Independent reproduction (this session, offline)

Focused run over the five named detector tests
(`TestDetectDiscoverySessionStatusExhaustedOrExpired`, `…ExpiresAtBoundary`,
`…MissingOrMalformedExpiryPreservesFallback`, `…ProviderBoundaries`, `…DoesNotLogSecrets`):

- `go test ./internal/rotation -run <5 tests> -v -count=20`: **100 `--- PASS`** (5 × 20), **0 FAIL**, `ok`.
  Named assertions are non-vacuous (exhaustion/status/expiry-boundary/provider-family/no-secret-log).
- `-race -count=20`: `ok`, **no data races** (1.076s).
- `gofmt -l` on both files: **clean** (empty).
- `go vet ./internal/rotation`: **exit 0**.

## Dependency completeness / atomic-push build analysis

- `detector_discovery.go` imports only stdlib (`strings`, `time`) + package types `DetectionResult`/
  `ExhaustionSignal`, which live in **`contract.go` — committed and clean at HEAD** (`git status` empty;
  `git grep HEAD` confirms the types). It defines its own new symbols (`SignalDiscovery`, `DiscoverySession`,
  `DetectDiscoverySession`, `sameDiscoveryProvider`, `canonicalDiscoveryProvider`).
- `detector_discovery_test.go` imports only stdlib + in-package symbols and its own same-file helper
  `assertDiscoveryExhausted`; **no cross-file untracked test-helper dependency**.
- The detector is **absent at HEAD** (new); nothing at HEAD references its new symbols → additive.
- Its consumer `discovery_reassignment.go` is **untracked and NOT in this push** (and is excluded 4.3 scope), so
  the pushed tree has no dangling reference. `service.go` is modified but its HEAD-committed form does not
  reference the new symbols.
- **Conclusion:** a 2-file atomic push (`HEAD` + these two files) is **statically build- and dependency-complete**.
  Caveat: an isolated clean-worktree build was **not executed** (would require staging/git manipulation, out of
  read-only scope); the reproduction above ran in the full working tree, which also compiles.

## Ownership / shared-scope conflicts

- `internal/rotation/**` is **not** a listed exclusive hotspot in `FILE_OWNERSHIP.md` (hotspots are daemon/config/
  health/prodex/execenv/models). No single-owner lock blocks these files.
- The detector is a **shared leaf** referenced across 4.1/4.2/4.3/5.3/session-api evidence lanes. Pushing it under
  4.1 must be scoped to **exactly these two files** and must **not** be construed as accepting the excluded 4.3/5.3
  consumers. Functionally the detector is **inert** until 4.2/4.3 consumers also land (build-safe, no runtime effect
  on HEAD).

## Verdict — three distinct levels

1. **Technically passing: YES (first-hand).** 5 named tests ×20 green, race-clean, gofmt-clean, vet-0; source is
   self-contained and statically build/dependency-complete for a 2-file push.
2. **Independently accepted: QUALIFIED YES.** EV-CREDISO-4.1 is recorded ACCEPT by an independent reviewer in
   `EVIDENCE_INDEX`/`AGENT_LEDGER`, task 4.1 `[x]`, hashes match; corroborated by this reproduction. **Qualifier:**
   acceptance provenance is a ledger row only (no standalone artifact, producer unnamed) — a real thinness that a
   sibling matrix treats as a HOLD. Recommend a named EV-4.1 artifact with a source manifest before relying on it
   as artifact-grade acceptance.
3. **Atomic push-ready: CONDITIONAL — NOT self-authorized.** Build/dependency-complete and hash-stable, but
   conditioned on: (a) an isolated clean-worktree build executed by the integrator; (b) push scoped to exactly the
   two files, explicitly not implying 4.2/4.3/5.3 acceptance; (c) **Kiro TL authorization** and **root integration
   control**. This review authorizes nothing.

## Clean-room isolated two-file build — 2026-07-18T21:50:00Z (CLOSES the clean-room condition)

Method (no repository git/index mutation): `mktemp -d /tmp/credroom.4.1.XXXXXX`; materialized committed HEAD
subtree with `git archive HEAD multica-auth-work/server | tar -x`; overlaid **only** the two current files at
their paths; ran the pinned offline toolchain; removed only that exact temp dir afterward (confirmed gone). No
untracked files, auth/config/env files, credentials, or working-tree changes were copied.

- **Baseline check**: the HEAD archive did **not** contain `detector_discovery.go` or `detector_discovery_test.go`
  (both reported absent) — confirms they are new/untracked. The untracked consumer `discovery_reassignment.go`
  did **not** leak into the clean room (only the two overlaid files match in `internal/rotation/`).
- **Overlay hashes (in clean room, match §Candidates exactly):**
  - `internal/rotation/detector_discovery.go` → `bc61a46c0c700010001a49c9075b87f97e89f802b138e63f90b53193a4b45b55`
  - `internal/rotation/detector_discovery_test.go` → `4e8092ff6b0f37a8416b961c89040433682e2d0d654293e192992dd3e1d55a4f`
- **Results (HEAD + only these 2 files):**
  - `gofmt -l` both files: clean (exit 0).
  - `go build ./internal/rotation/`: **exit 0** — dependency-complete (only stdlib + committed `contract.go`).
  - `go vet ./internal/rotation`: exit 0.
  - focused `-v -count=20` over the 5 named tests: **100 `--- PASS`**, `ok`.
  - focused `-race -count=20`: `ok`, no data races.
  - **full package** `go test ./internal/rotation`: **`ok`** — the entire committed HEAD rotation package plus the
    two overlaid files compiles and passes offline (DB `store_pg_test` / `!offline` E2E self-skip without
    `DATABASE_URL`). This also proves no committed HEAD test depends on an untracked helper.

**Clean-room condition: CLOSED (PASS).** The two-file atomic patch is empirically dependency-complete and
self-sufficient on a pristine committed HEAD, not merely in the full working tree. This upgrades verdict item 3's
condition (a) "isolated clean-worktree build" from *pending* to *satisfied*. Conditions (b) exact 2-file scope /
no implied 4.2/4.3/5.3 acceptance and (c) **Kiro TL authorization + root push control** remain.

## Non-claims
- Read-only w.r.t. the repository. No product/test/spec/task/shared-doc/git/index change; no credential/env value;
  no DB/network/service. No checkbox set. The clean-room used a private `/tmp` copy of committed HEAD (removed
  after validation) and did not touch the working tree or index. The final push decision remains with the
  integrator/TL.
