# Runtime Manager evaluations

This self-contained SPE-7 fixture module encodes deterministic acceptance and fail-closed cases for the canonical `credential-account-home-restoration` OpenSpec frozen at commit `a57c12e8424cd88cda178499563b69569d244d84`.

## Contract

`evaluations.json` contains exactly five golden paths:

1. reuse an owner-global accountless session with existing agent, runtime, and daemon rows;
2. attach a healthy exclusive opaque subscription/home without recreating persistent identities;
3. capability-validate and hot-apply a per-runtime model/reasoning version while preserving running-task snapshots;
4. rollback by auditable activation of a prior immutable version; and
5. inherit the parent snapshot, binding, and home for a subagent without persistent row creation.

It also contains fail-closed cases for raw-path exposure, account-home sharing, source credential copy/move/delete, static slot allowlists, cross-binding fallback, custom infrastructure, and SharePoint work. Forbidden cases must reject with zero mutation and zero side effects.

The fixture uses only opaque synthetic identifiers. It contains no credential value, account identity, provider-native secret, or source-home filesystem path. It is evaluation data only: it does not perform inference, contact a runner, mutate production, deploy, push, or access SharePoint.

## Digest form

Two digest fields exist, and they are **not interchangeable**:

| Field | Meaning |
| --- | --- |
| `configuration_digest` | the immutable configuration/version payload digest. An active version is represented by `active_version_id` (or a snapshot's `configuration_version_id`) **plus that version record's** `configuration_digest` |
| `capability_digest` | the capability snapshot/provider digest, covering values input, validation and activation result |

`active_digest` and `effective_configuration_digest` are **forbidden aliases** and appear nowhere in
this module; so is `schema_version`, the canonical runtime-configuration document discriminator being
JSON `version` with value `v1`. `contract-binding.test.mjs` asserts the absence of every forbidden
name, so an alias cannot creep back in.

Every digest — both fields plus `canonical_spec.digest` — is **raw lowercase, exactly 64 hexadecimal
characters**. The `sha256:` prefix is forbidden, per clause 6 of the unified authority baseline
(`.deploy-control/p0/evidence/spe4-spe5-unified-authority-baseline.md`, pinned at commit `c35c200`):
*"The effective configuration digest wire/storage form is raw lowercase 64 hexadecimal characters.
The `sha256:` prefix is forbidden."* That quotation is verbatim and must not be reworded — its prose
phrase "effective configuration digest" names a concept, not a JSON field.

`$defs.digest` enforces `^[0-9a-f]{64}$`. `validate.test.mjs` rejects any `sha256:` occurrence in
fixture data, rejects uppercase and over-length hex, and carries an explicit forbidden negative
fixture proving the schema rejects a prefixed digest — the only place the prefix may appear.
Synthetic digests must stay single-nibble (`aaaa…`) so they can never be mistaken for a real hash;
the one real digest is `canonical_spec.digest`, which pins the canonical spec file.

## Real contract bindings

This module is **Git-free by owner ruling**: consolidated tests must not shell out to Git. Contracts
are resolved with ordinary filesystem reads of the consolidated source tree and anchored by pinned
digests. The integrity property that mattered is preserved by hashing rather than by object
immutability — a contract cannot be edited without changing its hash, and a mismatch is a hard
failure. `frozen-contracts.mjs` is the single accessor, so `contract-binding.test.mjs` and
`hardening.test.mjs` provably consume the same pinned contracts. A missing contract file is a hard
failure, never a skip.

The repository root is located by walking up for the `openspec/` + `multica-auth-work/` marker pair.
`authority-fixture.json` carries the pinned provenance, the verbatim raw-64 clause, the canonical
digest/discriminator vocabulary, and the canonical spec digest.

| Binding | Source | Effect |
| --- | --- | --- |
| `requirement_refs` | `canonical_spec.path` on disk | every referenced `REQ-NN` must be a declared requirement, and the declared count must equal `declared_requirements` |
| spec freeze | same file | its sha256 must equal `canonical_spec.digest`; a spec edit fails the gate and forces re-review |
| raw-64 digest form | `authority-fixture.json`, pinned from `c35c200:.deploy-control/p0/evidence/spe4-spe5-unified-authority-baseline.md` | the clause is enforced against every fixture digest. Where the authority document **is** present under the resolved root, its sha256 is verified and the clause is relocated inside it, upgrading the pin to a real read |
| forbidden vocabulary | `authority-fixture.json` | `active_digest`, `effective_configuration_digest` and `schema_version` must appear nowhere in the fixture |
| event-name shape | `multica-auth-work/server/pkg/protocol/events.go` | fixture events must obey the product `subject:verb` contract, derived from the real declarations (verbs may contain `-` and `_`, as in `inbox:batch-read` and `task:waiting_local_directory`) |
| event registry gap | same file | `pending_contracts.unregistered_events` must equal the recomputed gap, in both directions |
| adapter gap | recursive `.go` scan of `adapter_scan_root`, minus `adapter_scan_excludes` | every `pending_contracts.absent_source_symbols` entry must still have no hand-written adapter |
| scan integrity | same scan | the exclusion list must name only generated paths under the scan root, and the scan must still find real source |

Only `evaluations.json`, `evaluation.schema.json` and `authority-fixture.json` are module-local; the
rest are real repository files.

## Adapter bindings and remaining pending contracts

The consolidated Runtime Manager now has hand-written adapters for `apply_class`,
`capability_digest`, `configuration_digest`, `runtime_session`, `runtime_binding`
and `runtime_configuration`. `implemented_source_symbols` binds that vocabulary to
real non-generated Go source and fails if any implementation disappears.
`absent_source_symbols` is the exact unresolved adapter ledger and is currently
empty. The six fixture events remain in `pending_contracts.unregistered_events`
until the product event registry declares them; that gap is recomputed in both
directions.

Generated sqlc storage code under `multica-auth-work/server/pkg/db/generated` remains
excluded from the adapter scan, and that exclusion is itself asserted. A generated
column proves storage only; the implemented ledger must resolve in hand-written
source.

## Validation

Run the focused dependency-free tests from `multica-auth-work`, naming the three files explicitly.
Directory mode (`node --test testdata/runtime-manager-evaluations/`) is **not** supported and reports
a module-resolution error:

```bash
node --test testdata/runtime-manager-evaluations/validate.test.mjs \
             testdata/runtime-manager-evaluations/contract-binding.test.mjs \
             testdata/runtime-manager-evaluations/hardening.test.mjs
```

No Git binary is required; the suite passes with `git` absent from `PATH`.

`validate.test.mjs` validates the fixture against `evaluation.schema.json`, verifies exact
golden/forbidden operation coverage, digest normalization, and checks operation-specific identity,
snapshot, exclusivity, hot-apply, rollback, inheritance, and fail-closed invariants.
`contract-binding.test.mjs` enforces the bindings above. `hardening.test.mjs` adds the
exclusivity-race, rollback, digest, pathless and prohibition-coverage checks, each bound to clause
text in the canonical spec.

## Prohibition coverage, and a reserved term

The forbidden set is pinned by its **nine declared operations**, in order, and every forbidden case
must bind at least one explicit `MUST NOT` / `MUST be forbidden` clause located in the canonical
spec. No count invariant is asserted over the prohibition requirements: how many distinct
requirements the nine cases happen to touch is an observation, not a contract, and asserting it
would freeze an incidental number.

The term **exact-eight** is deliberately not used in this module. It is reserved to the SPE-18
capacity envelope (`3 Kiro + 5 Codex`), whose frozen-ceiling reading is BLOCKED, and it has no
bearing on prohibition coverage.
