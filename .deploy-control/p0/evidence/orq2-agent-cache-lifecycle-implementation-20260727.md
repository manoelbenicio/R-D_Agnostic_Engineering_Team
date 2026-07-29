# ORQ2 agent Go cache lifecycle — implementation, user-scope cutover, and versioning

- **Owner authorization:** bounded host lifecycle authorization received 2026-07-27; owner cutover authority granted 2026-07-29 (`[GTL-OWNER-CUTOVER-20260729]`).
- **Host:** `ip-172-31-30-9.sa-east-1.compute.internal`
- **Instance:** `i-0af937456e125143d`
- **Mode:** installed as `systemd --user` unit on ORQ2 (`ec2-user`); single-scope cutover complete.
- **Repository Versioning:** Canonical script versioned under `scripts/ops/orq2-agent-cache-lifecycle` (SHA-256 `c8f1e1852995d9c00c509413bf7f42eb83b45a385231758303a2f0d0a61c4411`).

## Baseline and policy

- Effective systemd-tmpfiles policy remains `/usr/lib/tmpfiles.d/tmp.conf`: `q /tmp 1777 root root 10d`.
- Exact writable roots: `/tmp` and `/home/ec2-user/.cache` only.
- Recognized directory names: `gocache`, `go-build`, `gomod`/`gomodcache`, `gotmp`, and `kgc` token patterns.
- `gotmp` is audit-only because Go exposes no semantic gotmp cleanup primitive.
- Build caches require the canonical Go cache `README`; module caches use the module-cache semantic operation.
- Cleanup commands are exclusively pinned Go `clean -cache` and `clean -modcache`. The implementation contains no recursive shell removal, `find -delete`, or truncation.
- Pinned Go binary: `/home/ec2-user/goroot/go/bin/go`, Go `1.26.1`, SHA-256 `548e61b2d08ae52043be2f1924ed3c1d2b2c41967e360f3e317667f6fa912fc2`.

## Fail-closed gates and summary reporting

Every run verifies exact hostname, instance ID, machine ID, UID/user, Go binary hash, allow roots, and non-overlap lock. Every candidate is skipped if it:

- is outside the exact roots or resolves through a symlink;
- has any symlink, wrong-owner inode, nested mount, protected entry, or file modified within 24 hours;
- has any open file/process cwd reference (`lsof +D`);
- is empty, has an unrecognized Go-cache layout, or lacks a supported semantic cleanup primitive.

Summary outputs (`plan` and `complete` lines) report sorted deterministic `skip_summary` reason counts.
When root disk space usage exceeds 92% and zero candidates are eligible or reclaimed, the script emits `event=critical_no_reclaim` and exits with status `2` (nonzero) so systemd and monitoring signal critical failure.

Protected tokens include worktrees/workspace/repository, `.git`, `.deploy-control`, evidence/checkins/backups, `.agent-cred-homes`/slots/logins, `sqlcbuild`, ORQ38/ORQ41/Kiro, journals, databases/product data, Docker/containers, and Multica workspaces.

## Installed files, repository tracking, and hashes

| Location | Path | Owner/mode | SHA-256 |
|---|---|---|---|
| Repository (Canonical Source) | `scripts/ops/orq2-agent-cache-lifecycle` | `ec2-user:ec2-user 0755` | `c8f1e1852995d9c00c509413bf7f42eb83b45a385231758303a2f0d0a61c4411` |
| Host Executable | `/home/ec2-user/.local/libexec/orq2-agent-cache-lifecycle` | `ec2-user:ec2-user 0700` | `c8f1e1852995d9c00c509413bf7f42eb83b45a385231758303a2f0d0a61c4411` |
| User Service Unit | `/home/ec2-user/.config/systemd/user/orq2-agent-cache-lifecycle.service` | `ec2-user:ec2-user 0644` | `d29798b01d7c2dd745f0e75a1fb7c88f16504368783e21d0097ec72bdc891460` |
| User Timer Unit | `/home/ec2-user/.config/systemd/user/orq2-agent-cache-lifecycle.timer` | `ec2-user:ec2-user 0644` | `f24de7cb9d1ef973081b17ad050db9c82b9c86f35be612148c853d32f3fc06c1` |

Legacy system units (`/etc/systemd/system/orq2-agent-cache-lifecycle.timer`) were disabled and stopped (`systemctl disable --now`). Single-scope assertion: system timer `disabled`/`inactive`, user timer `enabled`/`active`.

## Service/timer controls

- Runs under `systemctl --user`, oneshot, `UMask=0077`, `NoNewPrivileges=true`.
- `ProtectSystem=strict`, `ProtectHome=read-only`; only `/tmp`, `/home/ec2-user/.cache`, and `/run/user/1000/orq2-agent-cache-lifecycle` are writable.
- Workspace and credential homes are inaccessible to the service (`InaccessiblePaths`).
- I/O class idle, priority 7; Nice 19; CPU/IO weights 10; memory limit 1 GiB; TasksMax 64.
- Schedule: daily `04:15 UTC`, `Persistent=true`, randomized delay 90 minutes, accuracy 5 minutes.

## Validation and executions

- Syntax checks: `bash -n` PASS; `systemd-analyze --user verify` PASS (0 errors / 0 warnings).
- Synthetic fixture validation (`/tmp/test-fixture.sh`): PASS.
- Live `--dry-run` output:
  - `skip_summary=file_newer_than_24h:1 no_files:25 no_semantic_go_gotmp_clean_primitive:1 open_reference_check_ambiguous:11`
  - `orq2_cache_lifecycle level=error event=critical_no_reclaim root_pcent=100% threshold=92% discovered=38 eligible=0 reclaimed_bytes=0 reason=root_disk_critical_with_zero_reclaimable_bytes`
  - Exit status `2`.
- Active timer: `systemctl --user is-enabled` -> `enabled`; `systemctl --user is-active` -> `active`.

## Rollback

To tear down user scope and restore system scope (if authorized):

```bash
systemctl --user disable --now orq2-agent-cache-lifecycle.timer
rm -f /home/ec2-user/.config/systemd/user/orq2-agent-cache-lifecycle.service
rm -f /home/ec2-user/.config/systemd/user/orq2-agent-cache-lifecycle.timer
rm -f /home/ec2-user/.local/libexec/orq2-agent-cache-lifecycle
systemctl --user daemon-reload
systemctl --user reset-failed orq2-agent-cache-lifecycle.service orq2-agent-cache-lifecycle.timer
sudo systemctl enable --now orq2-agent-cache-lifecycle.timer
```
