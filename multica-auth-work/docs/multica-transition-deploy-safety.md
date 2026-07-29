# Multica transition backend deploy safety

For the `multica-dev-transition` Compose project, the only approved backend deploy/recreate command is:

```bash
# Deployment-non-mutating: does not recreate containers or write deploy config,
# but briefly blocks agent_task_queue writers while queue-zero is checked.
scripts/deploy/recreate-transition-backend.sh --dry-run

# Only during an approved deploy window.
DEPLOY_ALLOW_EXECUTE=1 scripts/deploy/recreate-transition-backend.sh --execute
```

Do not invoke `docker compose up`, `restart`, or `force-recreate` directly for this project. The wrapper fixes the project/service/Compose files and always supplies `/home/ec2-user/.config/multica-transition/dev.env`. Operator-shell `JWT_SECRET`, `POSTGRES_*`, port, origin, image, and Compose overrides are rejected because shell values take precedence over `--env-file` interpolation.

Before mutation, the wrapper requires all of the following:

- the transition database identity (`multica_transition` user and database);
- loopback backend publication on host port `18080`;
- the exact non-fallback frontend origin from the mandatory env file;
- the approved-wrapper and env-file labels in a content-free rendered-config check;
- a pinned backend image and a locally available current image for rollback;
- a reachable `/api/me` baseline (HTTP 200 or the expected unauthenticated 401);
- an authenticated PostgreSQL connection over the bridge-network SCRAM path.

Queue-zero is evaluated inside a long-lived PostgreSQL transaction only after it takes `LOCK TABLE agent_task_queue IN SHARE MODE`. That lock conflicts with task inserts and claim/status writes, so no new task can be admitted or started between queue-zero and backend recreation. The transaction remains open through post-checks or automatic rollback. Success commits it; dry-run and every failure path roll it back. Process termination closes the database connection and releases the lock fail-closed. The database password is read from the fixed env file into the container child's stdin, exported as `PGPASSWORD` only in that child, and never appears in argv, Compose flags, logs, or wrapper output.

`--no-deps backend` is a lock-lifetime invariant, not only a blast-radius optimization: the PostgreSQL container that owns the lock-holder session is never recreated. Candidate and rollback backend startup both use that same uninterrupted database session. The isolated integration harness starts the actual candidate and rollback backend images against a real PostgreSQL 17 schema while the lock is held, verifies `/readyz` and `/health`, proves a queue writer times out under the lock, then proves the writer succeeds after release.

Execution persists `~/.config/multica-transition/last-known-good.yml` and requires it to match the running image on later runs. It archives the requested candidate under `rollback/*.roll-forward`, then runs only `up -d --force-recreate --no-deps backend`. `/health`, `/readyz`, and `/api/me` must pass, and the running image must equal the rendered candidate before it is promoted to last-known-good.

On failure, the wrapper atomically rewrites both durable last-known-good and the normal `images.yml` backend image to the previous reference before recreating. Therefore a later ordinary Compose command cannot silently redeploy the failed candidate. Retrying that candidate requires an explicit roll-forward from the preserved archive and a new successful wrapper execution.

The base self-host Compose file also requires explicit PostgreSQL identity/password, backend port, JWT secret, and frontend origin. This removes the unsafe fallback behavior that caused the ORQ-26 outage when `--env-file` was omitted.

## Validation without deployment

```bash
bash -n scripts/deploy/recreate-transition-backend.sh
bash scripts/deploy/recreate-transition-backend.test.sh
CANDIDATE_IMAGE=<candidate> ROLLBACK_IMAGE=<lkg> \
  bash scripts/deploy/recreate-transition-backend.integration.test.sh
bash scripts/selfhost-config.test.sh
```

The focused harness uses stubbed `docker` and `curl`; it does not contact or mutate the product deployment. The integration harness requires pre-existing local images, creates only uniquely named temporary containers/network with an ephemeral PostgreSQL volume, never attaches to the product Compose project, and cleans all resources on exit.
