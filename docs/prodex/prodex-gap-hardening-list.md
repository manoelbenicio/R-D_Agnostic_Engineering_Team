# prodex Gap Hardening List

Status: TARGET-MILESTONE HARDENING BACKLOG. Does not block F0 prodex-as-is
rollout unless a listed item is promoted by the tech lead.

Source scope: pinned official prodex `0.246.0`
`7750da9b6a5c91a6d429e18e6a4d422cab4bc144`, plus local
`openspec/changes/rotation-parity-polyglot/design.md` and ADR-001.

## Must Harden Before Forked L2 Broad PROD

1. Verify package pin, commit, checksum, SBOM, and Apache-2.0 attribution.
2. Keep Go as control plane and Rust/prodex as the only runtime decision owner
   per session.
3. Add sidecar health/readiness and versioned policy/account registration.
4. Enforce tenant/provider/profile kill switches before the next request.
5. Prohibit file/SQLite shared gateway state for multi-worker or multi-host
   deployment; use Postgres/Redis where state is shared.
6. Promote provider conformance so pure transforms, capability metadata, and
   fixtures agree before routing traffic through a provider.
7. Replay Smart Context before live rollout and require exact fallback on
   protocol, continuation, tool-call, JSON, mandatory-reference, or
   critical-signal risk.
8. Validate event redaction on success and error paths.
9. Validate runtime logs are diagnostics only and not required for request
   success.
10. Validate `redeem` only with gated real-account F9 evidence; do not infer
    backend efficacy from code presence.

## Native prodex Limitations To Respect

- Provider conformance is still split between `prodex-provider-core` metadata
  and app-side runtime translation modules.
- Gateway file and SQLite state are documented as single-node deployment models.
- Smart Context replay is deterministic local evidence, not proof of live model
  quality across all Multica production tasks.
- Redeem has implementation and guarded auto paths, but real-account outcomes
  remain empirical.

## Risk Classes

- upstream Codex protocol drift;
- continuation/profile affinity corruption;
- mid-stream rotation regression;
- Smart Context rewrite without exact fallback;
- provider capability overclaim or silent parameter drop;
- secret leakage in logs, evidence, events, or check-ins;
- file/SQLite lock contention in distributed deployment;
- redeem consuming scarce reset credit near natural reset or while another
  eligible profile exists;
- rollback path to raw Codex/prodex-as-is not exercised.
