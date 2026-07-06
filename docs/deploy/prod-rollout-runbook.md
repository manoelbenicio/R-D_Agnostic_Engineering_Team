# PROD Rollout Runbook - prodex AS-IS under Multica

Status: DRAFT FOR OWNER REVIEW - PROD DEPLOY NO-GO

This runbook is the owner review artifact for F7. It intentionally contains no executed PROD action. Operators must not run any command that changes PROD behavior until the owner approval record below is complete.

## 1. Owner Approval Gate

Required record in `.deploy-control/evidence/status-board.md`:

```text
deploy_owner_approved: true
owner:
timestamp:
artifact_hash:
prodex_version:
prodex_commit:
rollback_command_ref:
kill_switch_command_ref:
accepted_risk:
approval_notes:
```

Current state is `deploy_owner_approved: false`; therefore real PROD deploy is NO-GO.

## 2. Pre-Deploy Freeze

Before any change window:

1. Announce deploy window and owner approver.
2. Confirm no active critical incident.
3. Confirm all active agents have no conflicting locks on deploy/runtime docs or daemon launch code.
4. Snapshot current raw Codex launch configuration.
5. Snapshot current Go daemon version, config hash, and health.
6. Snapshot current Postgres migration version.
7. Record exact rollback and kill switch command references.
8. Confirm deploy evidence directory is writable and redaction-enabled.

## 3. Mandatory Pre-Checks

All checks must pass or the rollout is canceled.

```text
approval gate recorded with deploy_owner_approved=true
prodex version/commit/artifact sha256 matches approved pin
artifact attestation/SBOM/dependency audit reviewed
Postgres reachable with expected migration version
shared SQLite/file backend absent
Redis reachable if required
kill switch store reachable and writable
event stream can write durable audit rows
PRODEX_HOME on real ext4 or approved Linux fs, not 9p
per-profile CODEX_HOME/auth stores on real ext4 or approved Linux fs, not 9p
credential files mode 0600, credential directories mode 0700
profile homes resolve inside approved managed roots
logs directory writable by daemon runtime user
redaction scrubber enabled for Go logs, prodex logs, events, and evidence
raw Codex rollback path preserved
Smart Context starts in shadow with canary percent 0
auto-redeem disabled unless separately approved
```

Do not paste command output containing secrets. Any failed check involving filesystem, credentials, redaction, Postgres, kill switch, or rollback readiness is a hard stop.

## 4. Initial PROD Runtime Settings

Initial settings:

```text
PRODEX_SMART_CONTEXT_SHADOW=1
PRODEX_SMART_CONTEXT_CANARY_PERCENT=0
PRODEX_AUTO_REDEEM_ENABLED=0
MULTICA_L2_KILL_SWITCH_DEFAULT=enabled
MULTICA_L2_EVENT_STREAM_REQUIRED=1
MULTICA_L2_LOG_REDACTION=1
```

Expected behavior:

- prodex is used for controlled runtime sessions only after readiness;
- Smart Context observes and emits shadow metrics but sends exact original requests upstream;
- auto-redeem remains off;
- Go ingests events but never re-routes committed requests;
- kill switch can disable Smart Context, gateway, auto-redeem, or provider bridge by scope.

## 5. Rollout Steps

These steps are executable only after the approval gate is open.

1. Record pre-deploy evidence with scrubbed config hashes and current health.
2. Apply approved config pointing Multica to the pinned prodex launch path.
3. Restart or reload only the required Multica runtime component.
4. Verify Go daemon health.
5. Verify prodex liveness.
6. Verify prodex readiness, including Postgres, no shared SQLite, log dir, event stream, and kill switch checks.
7. Apply policy with Smart Context shadow and auto-redeem disabled.
8. Register approved profile/account references; do not send OAuth tokens, API keys, cookies, or `auth.json`.
9. Start one controlled owner-approved smoke session.
10. Confirm `router_owner=rust_l2` or AS-IS equivalent evidence that prodex owns runtime routing.
11. Confirm profile isolation and fail-closed behavior for invalid profile metadata.
12. Confirm runtime event stream emits scrubbed lifecycle and selection/affinity events.
13. Run redaction smoke using fake markers only.
14. Apply Smart Context kill switch for the next request and confirm disabled state.
15. Restore Smart Context to shadow only if the owner-approved smoke requires continued observation.
16. Keep observation window open and record metrics.

## 5.1 Deployment Surfaces

Operators must select exactly one deployment surface for the Multica app, then attach observability before any owner-approved smoke session.

### Helm

Use `multica-auth-work/deploy/helm/multica` for Kubernetes installs:

```bash
helm lint multica-auth-work/deploy/helm/multica
helm template multica multica-auth-work/deploy/helm/multica --namespace multica >/tmp/multica-rendered.yaml
helm upgrade --install multica multica-auth-work/deploy/helm/multica --namespace multica --create-namespace
```

The chart expects sensitive values in a pre-created Secret named by `existingSecret`; do not template real secrets into values files. Backend startup runs migrations before serving traffic and uses the Postgres advisory-lock migration path, so reversible SQL must be present before rollout.

### Self-host Docker Compose

Use `multica-auth-work/docker-compose.selfhost.yml` for the self-hosted app stack:

```bash
cd multica-auth-work
cp .env.example .env
# Edit .env out of band; rotate JWT_SECRET and Postgres password before production.
docker compose -f docker-compose.selfhost.yml up -d
```

The self-host compose binds frontend/backend ports to `127.0.0.1` by design. Put TLS and public exposure behind an explicit reverse proxy instead of changing those bindings to `0.0.0.0`.

### Observability Compose

Use `multica-auth-work/deploy/observability/docker-compose.yml` for Prometheus, Grafana, Alertmanager, and postgres-exporter:

```bash
cd multica-auth-work/deploy/observability
cp secrets/grafana_admin_password.example secrets/grafana_admin_password
cp secrets/pg_user.example secrets/pg_user
cp secrets/pg_pass.example secrets/pg_pass
chmod 600 secrets/grafana_admin_password secrets/pg_user secrets/pg_pass
docker compose config >/dev/null
docker compose up -d
```

Expected host ports are Prometheus `9090`, Grafana `3000`, Alertmanager `9093`, and postgres-exporter `9187`. If `3000` is already owned by a local frontend or WSL relay, free the port before production deploy; a temporary alternate Grafana port is acceptable only for local validation evidence.

### Reversible Migrations

Before deploy, verify migration reversibility:

```bash
cd multica-auth-work/server/migrations
comm -3 \
  <(find . -maxdepth 1 -name '*.up.sql' -printf '%f\n' | sed 's/\.up\.sql$//' | sort) \
  <(find . -maxdepth 1 -name '*.down.sql' -printf '%f\n' | sed 's/\.down\.sql$//' | sort)
```

The command must print no unmatched migration stems. Any missing `down` migration blocks production rollout.

### Account Enrollment

Enroll prodex profiles using `docs/deploy/prodex-account-enrollment-runbook.md`. Enrollment is a pre-traffic operation: profiles can be registered while disabled, then enabled only after kill-switch, rollback, observability, and redaction checks pass.

## 6. Success Criteria

All must pass:

- controlled real session launches through pinned prodex path;
- Go records one runtime router owner for the session and does not call legacy routing for that session;
- profile auth isolation remains per profile and fail-closed;
- event stream writes durable audit rows;
- Smart Context remains shadow-only unless separately approved;
- kill switch disables Smart Context for the target scope;
- no raw secret appears in Go logs, prodex logs, events, traces, evidence, screenshots, or command output;
- Postgres remains healthy and no shared SQLite path is used;
- raw Codex rollback remains immediately available;
- owner and Opus 4.8 receive scrubbed status.

## 7. Immediate Rollback Triggers

Rollback immediately if any occurs:

- raw secret appears anywhere;
- credential/profile path is on 9p or has unsafe permissions;
- profile switch does not fail closed;
- session affinity or continuation binding breaks;
- Smart Context causes protocol, tool-call, continuation, JSON, or mandatory-context failure without exact fallback;
- kill switch cannot be applied or confirmed;
- sidecar/readiness fails after deploy;
- event ingest cannot write durable audit rows;
- Postgres is unavailable for required state;
- Go daemon unhealthy;
- owner requests rollback.

## 8. Observation Window

Minimum fields to record in scrubbed evidence:

```text
deploy_id
owner_approval_ref
prodex_version
prodex_commit
artifact_sha256
session_id
runtime_session_id
tenant_id
provider
profile_alias_or_hash
Smart Context mode
canary percent
event ids
kill switch smoke result
redaction smoke result
rollback readiness result
errors and mitigations
```

Do not record raw prompts, raw tool outputs, OAuth material, cookies, API keys, bearer tokens, database URLs, Redis URLs, or `auth.json`.

## 9. Post-Deploy Evidence

Store scrubbed evidence under `.deploy-control/evidence/`:

- approval record;
- artifact verification record;
- precheck result;
- ext4/permission check result;
- health/readiness result;
- controlled session smoke result;
- event stream snippet with `secrets_present=false`;
- redaction smoke result;
- kill switch smoke result;
- rollback readiness result;
- metrics snapshot.

If the rollout is canceled before execution, record the cancel reason and leave `deploy_owner_approved:false` unchanged.
