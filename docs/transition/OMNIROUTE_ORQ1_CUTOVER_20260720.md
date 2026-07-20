# OmniRoute ORQ1 Cutover Record

Status date: 2026-07-20

## Decision

OmniRoute runs on `orq1` with the DEV application infrastructure. `orq2` is reserved for Herdr and coding agents. Agents reach OmniRoute over the private Tailscale network.

## Final topology

| Component | Node | Address |
|---|---|---|
| OmniRoute | `orq1` | `http://100.118.244.61:20128` |
| Multica backend | `orq1` | `http://127.0.0.1:18080` |
| Multica frontend | `orq1` | `http://127.0.0.1:13100` |
| Multica PostgreSQL | `orq1` | `127.0.0.1:15433` |
| Herdr and agents | `orq2` | Connect to OmniRoute through Tailscale |

OmniRoute is published only on the `orq1` Tailscale address. It is not published on `0.0.0.0` or the public EC2 interface.

## Container identity

```text
name=omniroute
image=diegosouzapw/omniroute@sha256:badb560971fdc23c2fb84b3e8695116239ff215b4cca4b07076201a8efae7f0d
image_id=sha256:d5500a44f56948044f5951cff1c867d901398fd4b66cda987e03e5381edfbd6d
restart=unless-stopped
memory_limit=1536 MiB
node_heap=1024 MiB
volume=omniroute-data:/app/data
binding=100.118.244.61:20128->20128/tcp
```

The container uses its image-provided `node healthcheck.mjs` healthcheck.

## Cutover procedure executed

1. Confirmed identical OmniRoute image IDs and identical initial volume byte counts on both nodes.
2. Stopped the `orq2` OmniRoute writer before recreating the `orq1` service.
3. Preserved the `orq2` volume during initial cutover validation.
4. Recreated `orq1` with the digest-pinned image, Tailscale-only binding, memory limit, persistent volume, and `unless-stopped` restart policy.
5. Waited for Docker health `healthy`.
6. Verified TCP and HTTP access from `orq2` to `orq1`.
7. Performed a controlled OmniRoute restart and waited for health to return.
8. Verified no error, warning, fatal, or panic lines in the post-cutover log window by count only.
9. Removed the stopped `orq2` OmniRoute container, volume, and image after acceptance.
10. Pruned remaining unused Docker objects on `orq2`; no agent source, worktree, or credential path was removed.

## Acceptance evidence

```text
orq1_omniroute_running=true
orq1_omniroute_health=healthy
orq1_restart_policy=unless-stopped
orq1_memory_limit=1610612736
orq2_to_orq1_tcp_20128=reachable
orq2_to_orq1_http_after_redirect=200
post_restart_health=healthy
post_restart_log_error=0
post_restart_log_warn=0
post_restart_log_fatal=0
post_restart_log_panic=0
backend_health_http=200
frontend_login_http=200
orq2_docker_containers=0
orq2_docker_volumes=0
orq2_docker_images=0
orq2_free_space_after_cleanup=6.3G
```

No paid/live-provider inference request was executed during this infrastructure validation. Docker health, UI HTTP access, cross-node connectivity, restart recovery, persistence attachment, and scrubbed log counts are proven; live account routing remains a separate owner-authorized canary gate.

## Agent configuration

The non-secret gateway base URL for agents on `orq2` is:

```text
AGENT_BRAIN_GATEWAY_BASE_URL=http://100.118.244.61:20128
```

This does not authorize or supply the gateway secret. `AGENT_BRAIN_GATEWAY_SECRET_FILE` remains owner/security provisioned outside Git.

## Host names and SSH aliases

The Tailscale node names are:

```text
orq1
orq2
```

The owner WSL SSH configuration contains a managed block that uses Tailscale name resolution through `tailscale nc`. The supported commands are:

```bash
ssh orq1
ssh orq2
```

The SSH aliases do not depend on Linux system MagicDNS integration or on memorizing the current Tailscale IPs.

Name durability conditions:

- the devices must remain registered in the same tailnet;
- the Tailscale hostnames must remain `orq1` and `orq2`;
- the local Tailscale client must be running and authenticated;
- tailnet ACL/SSH policy must continue allowing access;
- deleting and recreating a Tailscale device can require host-key or alias reconciliation.

Therefore the names are stable operational identities, but no network name should be described as unconditional or permanent across device deletion, tailnet migration, or administrator renaming.

## Safe verification

From the owner WSL:

```bash
tailscale ping orq1
tailscale ping orq2
ssh orq1 'hostname'
ssh orq2 'hostname'
```

From `orq2`:

```bash
curl -fsSL -o /dev/null -w '%{http_code}\n' http://100.118.244.61:20128/
```

Expected final HTTP status after redirects: `200`.

On `orq1`:

```bash
docker inspect -f '{{.State.Health.Status}} {{.HostConfig.RestartPolicy.Name}}' omniroute
docker stats --no-stream omniroute
curl -fsS http://127.0.0.1:18080/health
curl -fsS -o /dev/null -w '%{http_code}\n' http://127.0.0.1:13100/login
```

## Rollback

The verified full transition backup remains the recovery authority and contains the earlier OmniRoute volume archive and SQLite snapshot. Do not reintroduce an active `orq2` OmniRoute instance while the `orq1` service is running.

If rollback is required:

1. stop `orq1` OmniRoute;
2. restore the approved backup to a new volume;
3. start exactly one instance;
4. verify health and client connectivity;
5. update the gateway base URL;
6. retain a single writer.
