#!/usr/bin/env bash
set -euo pipefail

ROOT="/mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work"
SERVER_DIR="$ROOT/server"
SQL_FILE="$ROOT/scripts/staging/set_active_account_ledger.sql"
PROFILE="${PROFILE:-staging}"
EMAIL="${RT_EMAIL:-staging-rotation@example.com}"
CONFIG_PATH="$HOME/.multica/profiles/$PROFILE/config.json"

need() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing required command: $1" >&2
    exit 1
  }
}

redact() {
  sed -E 's/[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+/[email-redacted]/g; s/(token[=: ][[:space:]]*)[A-Za-z0-9._-]+/\1<redacted>/Ig'
}

mask_token() {
  sed -E 's/^(.{6}).*/\1***/'
}

multica() {
  PATH="$HOME/.local/bin:$HOME/.local/go/bin:$PATH" \
  GOCACHE=/tmp/gocache-host \
  GOMODCACHE=/tmp/gomod-host \
  MULTICA_CONFIG_DIR=/tmp/multica-staging \
  "$SERVER_DIR/bin/multica" --profile "$PROFILE" "$@"
}

psql_db() {
  docker exec -i multica-postgres-1 psql -U multica -d multica -P pager=off "$@"
}

rotation_database_url() {
  local pg_ip
  pg_ip="$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' multica-postgres-1)"
  if [[ -z "$pg_ip" ]]; then
    echo "could not resolve multica-postgres-1 Docker IP" >&2
    exit 1
  fi
  printf 'postgres://multica:multica@%s:5432/multica?sslmode=disable' "$pg_ip"
}

multica_with_rotation_db() {
  PATH="$HOME/.local/bin:$HOME/.local/go/bin:$PATH" \
  GOCACHE=/tmp/gocache-host \
  GOMODCACHE=/tmp/gomod-host \
  MULTICA_CONFIG_DIR=/tmp/multica-staging \
  DATABASE_URL="$ROTATION_DATABASE_URL" \
  "$SERVER_DIR/bin/multica" --profile "$PROFILE" "$@"
}

need curl
need jq
need docker

cd "$SERVER_DIR"

mkdir -p "$(dirname "$CONFIG_PATH")"
if [[ ! -f "$CONFIG_PATH" ]]; then
  install -m 600 /dev/null "$CONFIG_PATH"
  printf '{\n  "server_url": "http://localhost:8080",\n  "app_url": "http://localhost:3000"\n}\n' >"$CONFIG_PATH"
fi

echo "P1 auth: requesting one-shot code"
curl -fsS -X POST http://localhost:8080/auth/send-code \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\"}" >/dev/null

VERIFY_JSON="$(curl -fsS -X POST http://localhost:8080/auth/verify-code \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"code\":\"888888\"}")"
TOKEN="$(printf '%s' "$VERIFY_JSON" | jq -r '.token')"
USER_ID="$(printf '%s' "$VERIFY_JSON" | jq -r '.user.id // .user.ID // empty')"
if [[ -z "$TOKEN" || "$TOKEN" == "null" ]]; then
  echo "verify-code did not return a token" >&2
  exit 1
fi

tmp="$(mktemp)"
jq --arg token "$TOKEN" --arg server_url "http://localhost:8080" --arg app_url "http://localhost:3000" \
  '.server_url=$server_url | .app_url=$app_url | .token=$token' "$CONFIG_PATH" >"$tmp"
install -m 600 "$tmp" "$CONFIG_PATH"
rm -f "$tmp"
echo "P1 auth: token=$(printf '%s' "$TOKEN" | mask_token) user_id=$USER_ID"
multica auth status 2>&1 | redact

if [[ -n "${RT_WORKSPACE_ID:-}" ]]; then
  WORKSPACE_ID="$RT_WORKSPACE_ID"
  echo "P2 workspace: using RT_WORKSPACE_ID=$WORKSPACE_ID"
else
  slug="rt-setup-$(date -u +%Y%m%d%H%M%S)"
  name="RT Setup $(date -u +%Y%m%dT%H%M%SZ)"
  response="$(curl -fsS -X POST http://localhost:8080/api/workspaces \
    -H "Authorization: Bearer $TOKEN" \
    -H 'Content-Type: application/json' \
    -d "{\"name\":\"$name\",\"slug\":\"$slug\",\"issue_prefix\":\"RTS\"}")"
  WORKSPACE_ID="$(printf '%s' "$response" | jq -r '.id')"
  echo "P2 workspace: created id=$WORKSPACE_ID slug=$slug"
fi

tmp="$(mktemp)"
jq --arg ws "$WORKSPACE_ID" '.workspace_id=$ws' "$CONFIG_PATH" >"$tmp"
install -m 600 "$tmp" "$CONFIG_PATH"
rm -f "$tmp"

echo "P3 daemon: status before start"
ROTATION_DATABASE_URL="$(rotation_database_url)"
echo "P3 daemon: rotation database url=postgres://multica:***@$(printf '%s' "$ROTATION_DATABASE_URL" | sed -E 's#^postgres://multica:multica@([^:/]+):.*#\1#'):5432/multica?sslmode=disable"
status_json="$(multica daemon status --output json 2>&1 | redact || true)"
printf '%s\n' "$status_json"
if printf '%s' "$status_json" | jq -e '.status == "running"' >/dev/null 2>&1; then
  echo "P3 daemon: restarting with DATABASE_URL so rotation is enabled"
  multica daemon stop 2>&1 | redact || true
  sleep 2
fi
multica_with_rotation_db daemon start 2>&1 | redact
sleep 3

echo "P3 daemon: status after start"
multica daemon status --output json 2>&1 | redact

RUNTIME_ID="$(multica runtime list --output json | jq -r '.[] | select(.provider=="codex") | .id' | head -1)"
if [[ -z "$RUNTIME_ID" ]]; then
  echo "no codex runtime detected" >&2
  exit 1
fi
echo "P3 runtime: codex_runtime_id=$RUNTIME_ID"

if [[ -n "${RT_AGENT_ID:-}" ]]; then
  AGENT_ID="$RT_AGENT_ID"
  echo "P4 agent: using RT_AGENT_ID=$AGENT_ID"
else
  agent_json="$(multica agent create \
    --name "RT Codex $(date -u +%H%M%S)" \
    --runtime-id "$RUNTIME_ID" \
    --max-concurrent-tasks 1 \
    --visibility private \
    --output json)"
  AGENT_ID="$(printf '%s' "$agent_json" | jq -r '.id')"
  printf 'P4 agent: %s\n' "$(printf '%s' "$agent_json" | jq -c '{id,name,runtime_id,model,status,visibility}')"
fi

echo "P5 ledger: applying SQL"
psql_db -v workspace_id="$WORKSPACE_ID" -v agent_id="$AGENT_ID" <"$SQL_FILE"

echo "P6 issue: dispatching one real task"
issue_json="$(multica issue create \
  --title "RT ledger rotation smoke $(date -u +%Y%m%dT%H%M%SZ)" \
  --description "Trivial staging task for proactive ledger rotation smoke." \
  --assignee-id "$AGENT_ID" \
  --allow-duplicate \
  --output json)"
ISSUE_ID="$(printf '%s' "$issue_json" | jq -r '.id')"
printf 'P6 issue: %s\n' "$(printf '%s' "$issue_json" | jq -c '{id,title,status,assignee_type,assignee_id}')"

echo "P6 task: polling agent_task_queue"
for _ in $(seq 1 30); do
  task_rows="$(psql_db -c "SELECT atq.id::text AS task_id, atq.status, atq.runtime_id::text, atq.agent_id::text, atq.issue_id::text, atq.created_at, atq.started_at, atq.completed_at FROM agent_task_queue atq WHERE atq.agent_id = '$AGENT_ID'::uuid ORDER BY atq.created_at DESC LIMIT 3;")"
  printf '%s\n' "$task_rows"
  if printf '%s' "$task_rows" | grep -Eq 'running|completed|failed|cancelled'; then
    break
  fi
  sleep 2
done

echo "summary: workspace_id=$WORKSPACE_ID agent_id=$AGENT_ID runtime_id=$RUNTIME_ID issue_id=$ISSUE_ID"
