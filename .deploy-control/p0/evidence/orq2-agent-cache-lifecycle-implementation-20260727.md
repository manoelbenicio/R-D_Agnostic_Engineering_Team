# ORQ2 agent Go cache lifecycle — implementation and verification

- **Owner authorization:** bounded host lifecycle authorization received 2026-07-27.
- **Host:** `ip-172-31-30-9.sa-east-1.compute.internal`
- **Instance:** `i-0af937456e125143d`
- **Mode:** installed on ORQ2; no AWS, board, product-code, branch, or worktree mutation.
- **Board sync:** `PENDING_AUTHENTICATED_REGISTRAR`; local evidence is authoritative until A8 posts through the authenticated idempotent flow.

## Baseline and policy

- Effective systemd-tmpfiles policy remains `/usr/lib/tmpfiles.d/tmp.conf`: `q /tmp 1777 root root 10d`.
- Exact writable roots: `/tmp` and `/home/ec2-user/.cache` only.
- Recognized directory names: `gocache`, `go-build`, `gomod`/`gomodcache`, `gotmp`, and `kgc` token patterns.
- `gotmp` is audit-only because Go exposes no semantic gotmp cleanup primitive.
- Build caches require the canonical Go cache `README`; module caches use the module-cache semantic operation.
- Cleanup commands are exclusively pinned Go `clean -cache` and `clean -modcache`. The implementation contains no recursive shell removal, `find -delete`, or truncation.
- Pinned Go binary: `/home/ec2-user/goroot/go/bin/go`, Go `1.26.1`, SHA-256 `548e61b2d08ae52043be2f1924ed3c1d2b2c41967e360f3e317667f6fa912fc2`.

## Fail-closed gates

Every run verifies exact hostname, instance ID, machine ID, UID/user, Go binary hash, allow roots, and non-overlap lock. Every candidate is skipped if it:

- is outside the exact roots or resolves through a symlink;
- has any symlink, wrong-owner inode, nested mount, protected entry, or file modified within 24 hours;
- has any open file/process cwd reference (`lsof +D`);
- is empty, has an unrecognized Go-cache layout, or lacks a supported semantic cleanup primitive.

Protected tokens include worktrees/workspace/repository, `.git`, `.deploy-control`, evidence/checkins/backups, `.agent-cred-homes`/slots/logins, `sqlcbuild`, ORQ38/ORQ41/Kiro, journals, databases/product data, Docker/containers, and Multica workspaces.

## Installed files and hashes

| File | Owner/mode | SHA-256 |
|---|---|---|
| `/usr/local/libexec/orq2-agent-cache-lifecycle` | `root:root 0755` | `8b7a43a5b0f1dcc19b73eaec6a19325748fbe5de605feefe658d3284f7e6c1ee` |
| `/etc/systemd/system/orq2-agent-cache-lifecycle.service` | `root:root 0644` | `5d3bcdd60fc9f0fd789321714a8f7e000882aa917b2074accd31ec9511be2e5a` |
| `/etc/systemd/system/orq2-agent-cache-lifecycle.timer` | `root:root 0644` | `f24de7cb9d1ef973081b17ad050db9c82b9c86f35be612148c853d32f3fc06c1` |

No collision existed, so no backup file was required. SELinux contexts are `usr_t` for the script and `systemd_unit_file_t` for units.

## Service/timer controls

- Runs as `ec2-user`, oneshot, `UMask=0077`, `NoNewPrivileges=true`, empty capability bounding set.
- `ProtectSystem=strict`, `ProtectHome=read-only`; only the two cache roots and private runtime lock are writable.
- Workspace and credential homes are inaccessible to the service.
- Network denied; I/O class idle, priority 7; Nice 19; CPU/IO weights 10; memory limit 1 GiB.
- Journal output is rate-limited and candidate count is capped at 256.
- Schedule: daily `04:15 UTC`, `Persistent=true`, randomized delay 90 minutes, accuracy 5 minutes.

## Validation and executions

- Bash syntax: PASS. `systemd-analyze verify`: PASS (only an unrelated existing `acpid.socket` legacy-path warning).
- Unit hardening score: exposure `2.8 OK`.
- Dry-run log SHA-256: `3631f4e03a3be6d91194dbdf8e8e9dde5d0d674c0cba00ba206b038aeee2d4f0`.
- Dry-run: 69 recognized directories, 17 eligible, reclaimed 0 by design.
- Exactly one real lifecycle run executed. Journal SHA-256: `8df323a361f4fb7c47d4a6109777564e0775486b062e91d9d68325275fcb94b4`.
- Real result: exit 0; 70 recognized, 17 eligible/cleaned, `reclaimed_bytes=0` because prior containment had already reduced them to Go metadata-only 8-KiB caches.
- Logs explicitly show ORQ38, ORQ41, Kiro and current protected caches as `reason=protected_path`.
- Git status fingerprint remained unchanged. Workspace and credential-home metadata changed concurrently, but the service could not access those paths (`InaccessiblePaths`) and did not log them as candidates.
- Timer: `enabled`, `active`, `waiting`; next run measured as `2026-07-28T05:15:51Z`.

## Rollback

Rollback stops future execution and removes only the three exact installed files; it does not restore already reclaimed cache bytes (caches are reconstructible):

```bash
sudo systemctl disable --now orq2-agent-cache-lifecycle.timer
sudo unlink /etc/systemd/system/orq2-agent-cache-lifecycle.timer
sudo unlink /etc/systemd/system/orq2-agent-cache-lifecycle.service
sudo unlink /usr/local/libexec/orq2-agent-cache-lifecycle
sudo systemctl daemon-reload
sudo systemctl reset-failed orq2-agent-cache-lifecycle.service orq2-agent-cache-lifecycle.timer
```

No backup restore command is needed because collision detection returned `collision=0`.
