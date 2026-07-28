# ORQ-43A — peer-review follow-up

New local commit (no amend): `7d2f36fc485d0218f9e9e72bdcef3bb955a9bd88`.

Implemented corrections:

- `RequireHumanActor` is attached to issuance and single-token revocation;
  strict JSON decoding rejects unknown fields.
- `daemon_id` and workspace binding are mandatory.
- Expiry purge runs on issuance; a secure 0700/0600 client-side token-file
  writer was added with symlink refusal, fsync and atomic rename.
- Revocation deletes exactly one `(token id, workspace, daemon_id)` row and
  invalidates that hash in `DaemonTokenCache`; it cannot delete the overlap
  sibling. Responses use `Cache-Control: no-store`.
- Router carve-out is the formally granted ORQ-43A hunk; W3 must not edit it.

Focused handler/client tests, vet, gofmt and diff-check passed using private
caches. No secret, AWS, production, DB, rotation, board, push or PR action was
performed.

## Remaining hard gate for re-review

The existing `daemon_token` schema stores only the hash, so a durable
`Idempotency-Key` cannot yet replay the same raw token after a lost response.
Adding that key requires a migration-number reservation and SQLC ownership
decision; it was intentionally not invented in this commit. The independent
review must therefore decide whether an approved equivalent retry contract is
acceptable or reserve a migration before ORQ-43A can be marked complete.

The required ephemeral-DB matrix (HashToken-only, raw-once, overlap/replay,
single revocation/cache invalidation and expiry) remains a release gate and has
not been run against production or any shared DB.
