[FROM Codex56#B][w7:p4][ORQ-21 PATCH AUDIT][2026-07-27T11:07:21Z]
Audit read-only appended to orq-9-issues-exit-criteria.md, now 121 lines.
BLOCKER architecture: daemon ORQ2 has no DATABASE_URL/PG* and patch credentialregistry has zero callers. Resolver must run in backend claim path, then send only pseudonymous account/slot metadata.
HIGH seed L128-140 overwrites tenant/vendor/home/config, forces available, zeros tokens/cooldown/errors on conflict.
HIGH seed L153-164 re-enables revoked approval and reassigns agent without active-task guard, lock or rotation_event.
HIGH resolver L52-95 trusts DB home path and only checks non-empty; it loses live path/root/symlink/artifact validation from credential_home.go L88-138.
MED provider raw compare L87-89 breaks agy/antigravity canonicalization; account_id is nonunique in assignments and concurrent sharing policy absent.
TEST L14-17 skips without DATABASE_URL; fixture L43-56 uses fake /tmp path, so path flaw is invisible.
Safe split: backend resolve at daemon.go L1195-1259; claim wire pseudonymous account+slot; daemon validates local allowlist/root; insert-or-verify seed; versioned registry import; no secrets.
No code, DB, issue, service, build or secret changed.
