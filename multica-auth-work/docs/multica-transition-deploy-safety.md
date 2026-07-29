# Multica transition backend deploy safety

For the `multica-dev-transition` Compose project, the only approved backend deploy/recreate command is:

```bash
# Safe inspection only; does not recreate anything.
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
- zero tasks in `queued`, `dispatched`, `running`, or `waiting_local_directory`;
- a reachable `/api/me` baseline (HTTP 200 or the expected unauthenticated 401).

Execution preserves the current image reference/ID, an `images.yml` backup, and a final rollback override under `~/.config/multica-transition/rollback/`. It then runs only `up -d --force-recreate --no-deps backend`. `/health`, `/readyz`, and `/api/me` must pass after recreation. Any failure after mutation automatically recreates the backend with the preserved previous image.

The base self-host Compose file also requires explicit PostgreSQL identity/password, backend port, JWT secret, and frontend origin. This removes the unsafe fallback behavior that caused the ORQ-26 outage when `--env-file` was omitted.

## Validation without deployment

```bash
bash -n scripts/deploy/recreate-transition-backend.sh
bash scripts/deploy/recreate-transition-backend.test.sh
bash scripts/selfhost-config.test.sh
```

The focused harness uses stubbed `docker` and `curl`; it does not contact or mutate the product deployment.
