# Main Brain / OmniRoute rollout runbook

Status: owner approval required for production changes.

## Invariant

OmniRoute is the only router owner. Rollout never enables direct-provider, native CLI routing, provider credential injection, or any alternate runtime. Unready OmniRoute means admission is closed.

## Preconditions

1. Record owner, window, immutable Main Brain and OmniRoute revisions, configuration hashes and rollback target.
2. Confirm Postgres health/migration state and preserve durable Kanban, project, squad, issue, task, session and terminal data.
3. Validate the restricted OmniRoute secret-file metadata without printing content.
4. Validate topology: host daemon uses the configured loopback endpoint; containerized execution uses its explicit private-network endpoint.
5. Confirm authenticated readiness for the approved protocol/model and capacity limit.
6. Confirm dashboards, cancellation, terminal persistence and rollback controls.

## Rollout

1. Hold new task admission.
2. Deploy the approved OmniRoute revision and wait for strict readiness.
3. Deploy/restart Main Brain with gateway-required configuration.
4. Verify health diagnostics report `router_owner=omniroute` and the expected route/protocol.
5. Verify, without inference, that provider-native secrets/direct endpoints are absent from child-environment plans and that no alternate-router startup/config exists.
6. Open the approved cohort only after all prechecks pass; retain bounded capacity.
7. Record redacted correlation IDs, revisions, status/counters and terminal-persistence results. Never record prompts, tool payloads, credentials, account identities or connection strings.

## Stop conditions

Immediately close admission and invoke rollback for: readiness failure, dual/unknown router owner, provider credential/direct-route exposure, terminal persistence failure, cancellation/resource leak, Postgres integrity failure, secret/content leakage, or owner request.

## Product stack

The self-host backend/frontend/Postgres stack remains:

```bash
cd multica-auth-work
docker compose -f docker-compose.selfhost.yml up -d
```

This command does not deploy OmniRoute and does not authorize inference. Production values and secret references are provisioned out of band.
