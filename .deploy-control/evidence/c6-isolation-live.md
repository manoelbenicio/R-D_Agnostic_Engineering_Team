# C6 Triple Isolation Live Evidence

- status: LIVE_GREEN
- captured_at_utc: 2026-07-05T04:33:48Z
- executor: Codex#5.5#C
- sidecar: `multica-auth-work/prodex-sidecar/target/release/prodex-sidecar`
- bind: `127.0.0.1:43117`
- contract: `rpp.l2.v1`

## Account Registration

The live sidecar accepted a managed profile reference and rejected an invalid
profile home:

```text
valid profile under /tmp/rpp-smoke/profiles/profile-live-a:
  HTTP 200 registered_count=1

invalid profile_home=/tmp/rpp-smoke-outside-managed-root:
  HTTP 200 rejected_count=1
```

Only profile references were sent. No real credential material was read or
written.

## Synthetic Triple Isolation

Fake credential files were created for:

```text
CODEX_HOME A
CODEX_HOME B
Herdr pane CODEX_HOME
PRODEX_HOME/profiles/profile-live-a
PRODEX_HOME/profiles/profile-live-b
```

After simulating a refresh in `CODEX_HOME A`, the hash diff was:

```text
filesystem_changed=["auth.json@codex-home-a"]
filesystem_unchanged_count=4
filesystem_pass=true
```

This proves the C6 no-clobber model for the live sidecar registration path plus
synthetic fake credential homes. The probe used fake auth markers only and did
not expose secrets.
