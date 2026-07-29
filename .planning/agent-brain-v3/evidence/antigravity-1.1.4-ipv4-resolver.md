# Antigravity CLI 1.1.4 — IPv6 eligibility-check regression and IPv4 resolver workaround

Status: **RESOLVED via per-process workaround.** Classification: **external CLI regression (not an Agent Brain / repo defect).**
Date: 2026-07-18. Provenance: WSL host `manoelneto-laptop`, Linux; read-only diagnosis + one per-process env workaround. No secrets, tokens, cookies, credential files, or login state were read, copied, printed, or modified. TLS and the eligibility check itself were not weakened or bypassed.

## Root cause

Antigravity CLI **1.1.4** regressed the Eligibility Check's network address selection. During startup it fetches the Google account avatar (`https://lh3.googleusercontent.com/...`) and `oauth2/v2/userinfo` using Go's `net/http`. On a host with **no IPv6 egress**, 1.1.4's Go **built-in (pure-Go) resolver** returns and attempts the `AAAA` (IPv6) record and hard-fails with:

```
dial tcp [2800:3f0:4001:816::2001]:443: connect: cannot assign requested address
```

instead of falling back to the working IPv4 address. The system (cgo/`getaddrinfo`) resolver does **not** exhibit this because `AI_ADDRCONFIG` suppresses `AAAA` when no global IPv6 address exists — so only the pure-Go resolver path trips.

This is internal to the third-party CLI's eligibility/startup code. It is independent of the Agent Brain integration: `multica-auth-work/server/pkg/agent/antigravity.go` only sets the configured launch env (`cmd.Env = buildEnv(b.cfg.Env)`); a scan of `antigravity.go` and `internal/daemon/runtimeenv/**` found **no** ipv6/netdns/GODEBUG/dialer/tcp6/eligibility/avatar handling. Our routing does not cause or influence the avatar fetch.

## Evidence

- **Version differential on the same host (decisive):** pane `w3:pE` runs **1.1.3** and accepts work; pane `w3:pF` runs **1.1.4** and every prompt fails during the Eligibility Check on the IPv6 avatar fetch. Same host, same broken IPv6 egress → the differentiator is the CLI version → 1.1.4 regression.
- **Installed binary version (non-secret metadata, not executed for version):** `~/.local/bin/agy` embeds `data:1.1.4`. (`func1.1.3` strings are Go compiler closure names, not the app version.)
- **Host IPv6 egress absent:** 0 global IPv6 addresses, no IPv6 default route; direct IPv6 connect to the avatar host fails immediately; IPv4 connects (TLS handshake reaches the host, `http=400` = no path only).
- **Resolver split:** `getent ahostsv6 lh3.googleusercontent.com` returns empty (IPv4-only via `getaddrinfo`), while the observed failure shows the pure-Go resolver attempting an IPv6 address.

## Confirmed fix (per-process workaround)

Forcing the CLI to use the system (cgo) resolver makes it receive **IPv4-only** for the avatar host and complete the eligibility check. This was **used successfully to start pF and run Sonnet 4.6**.

Exact safe launch command:

```bash
GODEBUG=netdns=cgo agy
```

- Per-process, no root, reversible (affects only that invocation).
- Does not alter auth/tokens/TLS/eligibility — the eligibility check still runs, over IPv4.

## Limitations

- Requires the 1.1.4 binary to be cgo-enabled (it is dynamically linked, so the cgo resolver is available); if a build were `CGO_ENABLED=0`, `netdns=cgo` silently falls back to the Go resolver and would not help.
- Per-invocation only: it must be present in each shell/launch environment that starts `agy` (it is not baked into product code — see "Not implemented").
- Addresses the symptom (resolver family), not the upstream regression itself.

## Rollback

- Nothing to roll back for the workaround: simply omit `GODEBUG=netdns=cgo` to return to default resolver behavior.

## Optional owner action (NOT required)

A host-wide IPv6 disable (`net.ipv6.conf.{all,default,lo}.disable_ipv6=1`, persisted in `/etc/sysctl.d/99-disable-ipv6.conf`) also resolves it by making Go's IPv6 probe report unavailable. This is an **optional owner/system action, not required**, since the per-process `GODEBUG=netdns=cgo` workaround already resolves the issue without any system-network mutation. Undo: remove the sysctl.d file and set the three keys back to 0.

## Not implemented (per scope)

- No automatic `GODEBUG` injection into product code / runtime adapter was implemented (defect is external; a repo change is not warranted here).
- No `sudo`/`sysctl` executed by this documentation task; no credentials inspected; no other files edited.

## Recommended durable options (owner decision)

1. Pin/downgrade the failing pane to **1.1.3** (proven working) until Antigravity ships a 1.1.4+ that restores IPv4 fallback in the eligibility check; or
2. Keep the per-process `GODEBUG=netdns=cgo` launch convention for 1.1.4; or
3. (Optional) host-wide IPv6 disable as above.
