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

## Validation

Run the focused dependency-free test from `multica-auth-work`:

```bash
node --test testdata/runtime-manager-evaluations/validate.test.mjs
```

The test validates the fixture against `evaluation.schema.json`, verifies exact golden/forbidden operation coverage, and checks operation-specific identity, snapshot, exclusivity, hot-apply, rollback, inheritance, and fail-closed invariants.
