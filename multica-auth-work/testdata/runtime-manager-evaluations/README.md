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

Every digest — `active_digest`, `capability_digest`, `effective_configuration_digest` and
`canonical_spec.digest` — is **raw lowercase, exactly 64 hexadecimal characters**. The `sha256:`
prefix is forbidden, per clause 6 of the unified authority baseline frozen in commit `c35c200`
(`.deploy-control/p0/evidence/spe4-spe5-unified-authority-baseline.md`): *"The effective
configuration digest wire/storage form is raw lowercase 64 hexadecimal characters. The `sha256:`
prefix is forbidden."*

`$defs.digest` enforces `^[0-9a-f]{64}$`. `validate.test.mjs` rejects any `sha256:` occurrence in
fixture data, rejects uppercase and over-length hex, and carries an explicit forbidden negative
fixture proving the schema rejects a prefixed digest — the only place the prefix may appear.
Synthetic digests must stay single-nibble (`aaaa…`) so they can never be mistaken for a real hash;
the one real digest is `canonical_spec.digest`, which pins the frozen spec file.

## Real contract bindings

Every external contract is read as an **immutable Git object at a pinned revision**, never from the mutable working tree or another checkout: a working-tree read would let an uncommitted edit silently satisfy a gate. `frozen-contracts.mjs` is the single accessor, so `contract-binding.test.mjs` and `hardening.test.mjs` provably consume the same frozen blob. A missing object is a hard failure, never a skip.

| Binding | Pinned source | Effect |
| --- | --- | --- |
| `requirement_refs` | `suite.frozen_commit:canonical_spec.path` | every referenced `REQ-NN` must be a declared requirement of the frozen blob |
| spec freeze | same blob | its sha256 must equal `canonical_spec.digest`; a spec edit forces re-review |
| raw-64 digest form | `c35c200be62d63cb654638d5b142b2a8105c1484:.deploy-control/p0/evidence/spe4-spe5-unified-authority-baseline.md` | the authority clause is located in the pinned blob and enforced against every fixture digest. Read as a Git object precisely so the producer branch never merges the authority branch; K1 handles ancestry later |
| event-name shape | `HEAD:multica-auth-work/server/pkg/protocol/events.go` | fixture events must obey the product `subject:verb` contract, derived from the real declarations (verbs may contain `-`, as in `inbox:batch-read`) |
| event registry gap | same blob | `pending_contracts.unregistered_events` must equal the gap recomputed from the committed HEAD tree |
| source gap | `git grep` over `HEAD:multica-auth-work/server` | every `pending_contracts.absent_source_symbols` entry must still be absent in that exact committed tree |

Only `evaluations.json` and `evaluation.schema.json` are read from disk, because they are the artifacts under validation.

## Pending contracts, and why they are declared

The Runtime Manager adapters these scenarios describe are **not implemented yet**. `apply_class`, `effective_configuration_digest`, `runtime_session`, `runtime_binding` and `runtime_configuration` appear nowhere in the Go source, and none of the six fixture events is in the product event registry. Binding those scenarios to an adapter would require inventing one, so the gap is declared in `pending_contracts` and machine-checked instead: the lists must shrink as real contracts land, and the suite fails when they drift in either direction.

## Validation

Run the focused dependency-free tests from `multica-auth-work`:

```bash
node --test testdata/runtime-manager-evaluations/validate.test.mjs \
             testdata/runtime-manager-evaluations/contract-binding.test.mjs \
             testdata/runtime-manager-evaluations/hardening.test.mjs
```

`validate.test.mjs` validates the fixture against `evaluation.schema.json`, verifies exact golden/forbidden operation coverage, digest normalization, and checks operation-specific identity, snapshot, exclusivity, hot-apply, rollback, inheritance, and fail-closed invariants. `contract-binding.test.mjs` enforces the bindings above. `hardening.test.mjs` adds the exclusivity-race, rollback, digest, pathless and prohibition-coverage checks, each bound to clause text in the frozen spec blob.

## Prohibition coverage, and a reserved term

The forbidden set is pinned by its **nine declared operations**, in order, and every forbidden case
must bind at least one explicit `MUST NOT` / `MUST be forbidden` clause located in the frozen spec
blob. No count invariant is asserted over the prohibition requirements: how many distinct
requirements the nine cases happen to touch is an observation, not a contract, and asserting it
would freeze an incidental number.

The term **exact-eight** is deliberately not used in this module. It is reserved to the SPE-18
capacity envelope (`3 Kiro + 5 Codex`), whose frozen-ceiling reading is BLOCKED, and it has no
bearing on prohibition coverage.
