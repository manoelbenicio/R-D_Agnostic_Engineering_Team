# ORQ-42 V8 independent adversarial review

- Reviewed artifact SHA-256: `21d820de2c071152dbbe6fae7ddc3f41d68900f426902a6d108ecbee0d8b893f`
- Mode: READ-ONLY; no SSH, asm-exec, AWS, Docker, DB, secret, login, or runtime action.
- Verdict: **BLOCK**.

## Findings

- **B1: BLOCK.** The absolute allowlist, five compose files, project/service, `env -u` precedence guard, and source-integral assertions are well specified. However the approved editor is explicitly nonexistent and has no source, hash, or `O_NOFOLLOW` implementation. Q-H is therefore a hard precondition, not a documentation detail.
- **B2: BLOCK.** The boolean HMAC comparison is secret-safe in principle and avoids `.Config.Env`; the AWSCURRENT/dev.env/effective-container chain is clearly described. But ARN, region, JSON key and initial AWSCURRENT `VersionId` remain placeholders, and Q-C is explicitly blocked by prior `AccessDenied`. No equality gate may run until the version identity is captured and remains stable.
- **B3: BLOCK.** REST cookie/Bearer gates, WS cookie pre-upgrade behavior, daemon/PAT/cloud/fallback distinctions, and the no-ORQ2-mutation rule are concrete. The first-frame `auth_ack` gate (C2b) has no available WebSocket client; Q-I is open. This leaves a required consumer unverified and requires an owner-approved pinned client or browser procedure before rotation.
- **B4: CONDITIONAL PASS.** Stdin-only login, builtin `printf`, `umask`, `ulimit`, private 0700/0600 custody, non-symlink checks, and retention are executable. The claimed `O_NOFOLLOW` equivalence is valid only for the curl custody path; the editor still must implement actual `O_NOFOLLOW`.
- **B5: BLOCK.** Prefix/suffix construction, `cmp -s`, fsync/rename rollback, and fixed labels are directionally sound. The editor absence also blocks byte-preserving proof. Version-stage/version-id pins are unresolved; rollback cannot be authorized while Q-C is open.

## Additional gate status

- The ten authorization rows are explicit, but several are still proposals/open (Q-A, Q-B, Q-C, Q-D, Q-E, Q-F, Q-G, Q-H, Q-I); they do not constitute completed approvals.
- `AWSPREVIOUS` handling is metadata-only and appropriately conditional, but the runbook must record the actual `VersionIdsToStages` result and stop on any AWSCURRENT VersionId drift.
- No secret value is requested or exposed by this review.

## Exact next actions

1. Build, independently review, and pin the editor source/hash with real `O_NOFOLLOW`, byte-preserving prefix/suffix logic, fsync/rename, and fixed output labels; close Q-H.
2. Obtain owner-authorized metadata-only ARN/region/key/stage/VersionId evidence and a stability check; close Q-B/Q-C.
3. Provide an independently pinned WebSocket client/browser procedure that proves first-frame `auth_ack` and old-token rejection; close Q-I.
4. Re-review all B1-B5 gates. Until then, no rotation, edit, recreate, login, or rollback window is authorized.
