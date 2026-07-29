#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
WRAPPER="$ROOT_DIR/scripts/deploy/recreate-transition-backend.sh"
PASS_COUNT=0

fail() { echo "FAIL: $*" >&2; exit 1; }
pass() { PASS_COUNT=$((PASS_COUNT + 1)); echo "ok $PASS_COUNT - $*"; }

setup_case() {
  CASE_ROOT="$(mktemp -d)"
  STUB_BIN="$CASE_ROOT/bin"
  mkdir -p "$STUB_BIN" "$CASE_ROOT/log"
  cat >"$CASE_ROOT/dev.env" <<'EOF'
POSTGRES_DB=multica_transition
POSTGRES_USER=multica_transition
POSTGRES_PASSWORD=test-only-password
POSTGRES_PORT=15432
BACKEND_PORT=18080
FRONTEND_ORIGIN=https://dev.multica.example
JWT_SECRET=test-only-jwt
EOF
  cat >"$CASE_ROOT/images.yml" <<'EOF'
services:
  backend:
    image: multica-backend:candidate-test
EOF
  cat >"$CASE_ROOT/backend-env.override.yml" <<'EOF'
services:
  backend:
    environment:
      TEST_ONLY: "true"
EOF
  cat >"$CASE_ROOT/config.json" <<EOF
{
  "name": "multica-dev-transition",
  "services": {
    "postgres": {
      "environment": {"POSTGRES_DB": "multica_transition", "POSTGRES_USER": "multica_transition"}
    },
    "backend": {
      "image": "multica-backend:candidate-test",
      "environment": {
        "DATABASE_URL": "postgres://multica_transition:redacted@postgres:5432/multica_transition?sslmode=disable",
        "FRONTEND_ORIGIN": "https://dev.multica.example"
      },
      "ports": [{"host_ip": "127.0.0.1", "published": "18080", "target": 8080}],
      "labels": {
        "com.multica.deploy.approved-wrapper": "scripts/deploy/recreate-transition-backend.sh",
        "com.multica.deploy.env-file": "$CASE_ROOT/dev.env"
      }
    }
  }
}
EOF

  cat >"$STUB_BIN/docker" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"$STUB_LOG_DIR/docker.log"
if [[ " $* " == *" compose version "* ]]; then exit 0; fi
if [[ " $* " == *" config --format json "* ]]; then cat "$STUB_CONFIG"; exit 0; fi
if [[ " $* " == *" exec -T postgres sh -c "* ]]; then
  IFS= read -r password
  [[ "$password" == "test-only-password" ]] || { echo "authenticated child env missing" >&2; exit 3; }
  printf 'AUTH_OK\n' >>"$STUB_LOG_DIR/events.log"
  while IFS= read -r line; do
    if [[ "$line" == "LOCK TABLE agent_task_queue IN SHARE MODE;" ]]; then
      printf 'LOCK_HELD\n' >>"$STUB_LOG_DIR/events.log"
    elif [[ "$line" == *"THEN 'ACTIVE_QUEUE_PRESENT'"* ]]; then
      if [[ "${STUB_QUEUE_COUNT:-0}" == "0" ]]; then echo ADMISSION_FREEZE_HELD; else echo ACTIVE_QUEUE_PRESENT; fi
    elif [[ "$line" == "COMMIT;" ]]; then
      printf 'RELEASE_COMMIT\n' >>"$STUB_LOG_DIR/events.log"
    elif [[ "$line" == "ROLLBACK;" ]]; then
      printf 'RELEASE_ROLLBACK\n' >>"$STUB_LOG_DIR/events.log"
    elif [[ "$line" == "\\q" ]]; then
      exit 0
    fi
  done
  exit 0
fi
if [[ " $* " == *" ps -q backend "* ]]; then echo backend-container; exit 0; fi
if [[ "${1:-}" == "inspect" ]]; then
  if [[ " $* " == *"{{.Config.Image}}|{{.Image}}"* ]]; then
    echo 'multica-backend:previous-safe|sha256:previous'
  else
    echo 'multica-backend:candidate-test'
  fi
  exit 0
fi
if [[ "${1:-}" == "image" && "${2:-}" == "inspect" ]]; then exit 0; fi
if [[ " $* " == *" up -d --force-recreate --no-deps backend "* ]]; then
  count=0
  [[ ! -f "$STUB_LOG_DIR/up.count" ]] || count="$(cat "$STUB_LOG_DIR/up.count")"
  count=$((count + 1))
  printf '%s' "$count" >"$STUB_LOG_DIR/up.count"
  printf 'UP_%s\n' "$count" >>"$STUB_LOG_DIR/events.log"
  exit 0
fi
echo "unexpected docker invocation: $*" >&2
exit 2
STUB
  chmod +x "$STUB_BIN/docker"

  cat >"$STUB_BIN/curl" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail
url="${*: -1}"
case "$url" in
  */api/me) printf '401' ;;
  */readyz) printf '200' ;;
  */health)
    count=0
    [[ ! -f "$STUB_LOG_DIR/up.count" ]] || count="$(cat "$STUB_LOG_DIR/up.count")"
    if [[ "${STUB_SCENARIO:-ok}" == "post-health-fail" && "$count" == "1" ]]; then printf '503'; else printf '200'; fi
    ;;
  *) printf '000'; exit 1 ;;
esac
STUB
  chmod +x "$STUB_BIN/curl"
}

cleanup_case() { rm -rf "$CASE_ROOT"; }

run_wrapper() {
  env -i \
    PATH="$STUB_BIN:/usr/bin:/bin" \
    DEPLOY_WRAPPER_TEST_MODE=1 \
    DEPLOY_WRAPPER_TEST_ROOT="$CASE_ROOT" \
    STUB_LOG_DIR="$CASE_ROOT/log" \
    STUB_CONFIG="$CASE_ROOT/config.json" \
    STUB_QUEUE_COUNT="${STUB_QUEUE_COUNT:-0}" \
    STUB_SCENARIO="${STUB_SCENARIO:-ok}" \
    ${DEPLOY_ALLOW_EXECUTE:+DEPLOY_ALLOW_EXECUTE="$DEPLOY_ALLOW_EXECUTE"} \
    bash "$WRAPPER" "$@"
}

test_dry_run_is_content_free_and_non_mutating() {
  setup_case
  local out
  out="$(run_wrapper --dry-run 2>&1)" || fail "valid dry-run should pass: $out"
  [[ "$out" == *"DRY-RUN PASS"* ]] || fail "dry-run evidence missing"
  [[ "$out" != *"test-only-password"* && "$out" != *"test-only-jwt"* ]] || fail "dry-run leaked rendered secret content"
  [[ ! -f "$CASE_ROOT/log/up.count" ]] || fail "dry-run attempted a recreate"
  grep -Fq -- "--env-file $CASE_ROOT/dev.env" "$CASE_ROOT/log/docker.log" || fail "mandatory env file absent from compose calls"
  grep -Fq 'AUTH_OK' "$CASE_ROOT/log/events.log" || fail "authenticated database child was not established"
  grep -Fq 'LOCK_HELD' "$CASE_ROOT/log/events.log" || fail "dry-run did not hold the admission lock"
  grep -Fq 'RELEASE_ROLLBACK' "$CASE_ROOT/log/events.log" || fail "dry-run did not release the admission lock"
  ! grep -R -Fq 'test-only-password' "$CASE_ROOT/log" || fail "database secret leaked into command/event logs"
  cleanup_case
  pass "dry-run validates authenticated queue-zero under a released admission freeze without mutation or secret output"
}

test_operator_override_is_rejected_before_docker() {
  setup_case
  local out rc=0
  out="$(env -i PATH="$STUB_BIN:/usr/bin:/bin" DEPLOY_WRAPPER_TEST_MODE=1 DEPLOY_WRAPPER_TEST_ROOT="$CASE_ROOT" POSTGRES_PASSWORD=operator-override bash "$WRAPPER" --dry-run 2>&1)" || rc=$?
  [[ "$rc" -ne 0 && "$out" == *"operator shell override is prohibited: POSTGRES_PASSWORD"* ]] || fail "POSTGRES override was not rejected"
  [[ ! -f "$CASE_ROOT/log/docker.log" ]] || fail "docker ran before override rejection"
  cleanup_case
  pass "operator-shell JWT/POSTGRES precedence is fail-closed"
}

test_active_queue_blocks_recreate() {
  setup_case
  STUB_QUEUE_COUNT=2
  local out rc=0
  out="$(run_wrapper --dry-run 2>&1)" || rc=$?
  [[ "$rc" -ne 0 && "$out" == *"active task queue is not zero"* ]] || fail "active queue did not block"
  [[ ! -f "$CASE_ROOT/log/up.count" ]] || fail "queue-blocked run attempted recreate"
  grep -Fq 'LOCK_HELD' "$CASE_ROOT/log/events.log" || fail "queue was checked without holding admission lock"
  grep -Fq 'RELEASE_ROLLBACK' "$CASE_ROOT/log/events.log" || fail "active-queue failure did not release admission lock"
  cleanup_case
  unset STUB_QUEUE_COUNT
  pass "nonzero active task queue blocks disruptive recreate"
}

test_rendered_identity_mismatch_blocks() {
  setup_case
  sed -i 's/"published": "18080"/"published": "8080"/' "$CASE_ROOT/config.json"
  local out rc=0
  out="$(run_wrapper --dry-run 2>&1)" || rc=$?
  [[ "$rc" -ne 0 && "$out" == *"rendered config assertion failed: backend host port"* ]] || fail "bad rendered port did not block: $out"
  [[ "$out" != *"test-only-password"* ]] || fail "failed render assertion leaked content"
  cleanup_case
  pass "content-free rendered-config identity assertions fail closed"
}

test_execute_uses_backend_only_forced_recreate() {
  setup_case
  DEPLOY_ALLOW_EXECUTE=1
  local out
  out="$(run_wrapper --execute 2>&1)" || fail "valid execute simulation should pass: $out"
  [[ "$(cat "$CASE_ROOT/log/up.count")" == "1" ]] || fail "expected exactly one recreate"
  grep -Fq -- "up -d --force-recreate --no-deps backend" "$CASE_ROOT/log/docker.log" || fail "bounded recreate flags missing"
  grep -R -Fq 'deploy_status=healthy' "$CASE_ROOT/state" || fail "healthy state was not preserved"
  grep -Fq 'image: multica-backend:candidate-test' "$CASE_ROOT/last-known-good.yml" || fail "successful candidate was not promoted to durable last-known-good"
  local events
  events="$(tr '\n' ' ' <"$CASE_ROOT/log/events.log")"
  [[ "$events" == *"LOCK_HELD"*"UP_1"*"RELEASE_COMMIT"* ]] || fail "admission lock did not span recreate through success: $events"
  cleanup_case
  unset DEPLOY_ALLOW_EXECUTE
  pass "execute path holds admission freeze through backend-only recreate, releases it, and promotes durable last-known-good"
}

test_failed_health_automatically_rolls_back_previous_image() {
  setup_case
  DEPLOY_ALLOW_EXECUTE=1
  STUB_SCENARIO=post-health-fail
  local out rc=0
  out="$(run_wrapper --execute 2>&1)" || rc=$?
  [[ "$rc" -ne 0 ]] || fail "failed post-health should keep original failure status"
  [[ "$out" == *"automatic rollback completed"* ]] || fail "automatic rollback evidence missing: $out"
  [[ "$(cat "$CASE_ROOT/log/up.count")" == "2" ]] || fail "expected candidate recreate plus rollback recreate"
  local rollback_file roll_forward_file events
  rollback_file="$(find "$CASE_ROOT/state" -name '*.rollback.yml' -print -quit)"
  roll_forward_file="$(find "$CASE_ROOT/state" -name '*.roll-forward' -print -quit)"
  grep -Fq 'image: multica-backend:previous-safe' "$rollback_file" || fail "rollback did not pin previous image ref"
  grep -Fq 'image: multica-backend:previous-safe' "$CASE_ROOT/images.yml" || fail "normal images.yml was not atomically restored to last-known-good"
  grep -Fq 'image: multica-backend:previous-safe' "$CASE_ROOT/last-known-good.yml" || fail "durable last-known-good was not restored"
  grep -Fq 'image: multica-backend:candidate-test' "$roll_forward_file" || fail "failed candidate was not preserved for explicit roll-forward"
  grep -Fq -- "-f $rollback_file up -d --force-recreate --no-deps backend" "$CASE_ROOT/log/docker.log" || fail "rollback override was not applied last"
  events="$(tr '\n' ' ' <"$CASE_ROOT/log/events.log")"
  [[ "$events" == *"LOCK_HELD"*"UP_1"*"UP_2"*"RELEASE_ROLLBACK"* ]] || fail "admission lock did not span failure and rollback: $events"
  cleanup_case
  unset DEPLOY_ALLOW_EXECUTE STUB_SCENARIO
  pass "failed post-check restores durable operational config, rolls back under freeze, then releases admission"
}

test_mismatched_last_known_good_blocks_before_freeze() {
  setup_case
  cat >"$CASE_ROOT/last-known-good.yml" <<'EOF'
services:
  backend:
    image: multica-backend:unexpected
EOF
  local out rc=0
  out="$(run_wrapper --dry-run 2>&1)" || rc=$?
  [[ "$rc" -ne 0 && "$out" == *"running backend image differs from durable last-known-good config"* ]] || fail "last-known-good drift did not block"
  [[ ! -f "$CASE_ROOT/log/events.log" ]] || fail "admission freeze started despite last-known-good drift"
  cleanup_case
  pass "running image drift from durable last-known-good fails closed"
}

test_direct_paths_are_fail_closed_in_compose_sources() {
  grep -Fq "\${POSTGRES_PASSWORD:?POSTGRES_PASSWORD must be set in .env or --env-file}" "$ROOT_DIR/docker-compose.selfhost.yml" || fail "database fallback still present"
  grep -Fq "\${JWT_SECRET:?JWT_SECRET must be set in .env or --env-file}" "$ROOT_DIR/docker-compose.selfhost.yml" || fail "JWT fallback still present"
  grep -Fq "\${BACKEND_PORT:?BACKEND_PORT must be set in .env or --env-file}" "$ROOT_DIR/docker-compose.selfhost.yml" || fail "backend port fallback still present"
  grep -Fq "\${MULTICA_DEPLOY_ENV_FILE_LABEL:?Use scripts/deploy/recreate-transition-backend.sh; direct recreate is prohibited}" "$ROOT_DIR/deploy/multica-transition.guard.yml" || fail "guard label is not fail-closed"
  pass "unsafe fallback interpolation and unlabelled guarded recreates are rejected"
}

test_dry_run_is_content_free_and_non_mutating
test_operator_override_is_rejected_before_docker
test_active_queue_blocks_recreate
test_rendered_identity_mismatch_blocks
test_execute_uses_backend_only_forced_recreate
test_failed_health_automatically_rolls_back_previous_image
test_mismatched_last_known_good_blocks_before_freeze
test_direct_paths_are_fail_closed_in_compose_sources

echo "deploy safety tests passed ($PASS_COUNT)"
