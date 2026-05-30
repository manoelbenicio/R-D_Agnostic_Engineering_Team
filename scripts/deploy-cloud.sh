#!/usr/bin/env bash
#
# AgentVerse — single-command cloud deploy.
#
# Deploys two GCP services:
#   1. Cloud Run: agentverse-runtime  (orchestration runtime)
#   2. Firebase Hosting: the SPA in dist/
#
# Prerequisites (you supply):
#   * gcloud CLI authenticated and pointed at the target project
#   * firebase CLI installed (`npm i -g firebase-tools`) and logged in
#   * .firebaserc present at repo root with your project id
#       cp .firebaserc.example .firebaserc && edit
#   * Artifact Registry repo named `agentverse` exists in $REGION:
#       gcloud artifacts repositories create agentverse \
#         --repository-format=docker --location=$REGION
#
# What it does, in order:
#   1. Resolve project id, region, image tag.
#   2. `npm run build` (Vite production), strip MSW worker.
#   3. Submit infra/runtime/cloudbuild.yaml to build + push the runtime image.
#   4. Apply infra/runtime/service.yaml against Cloud Run.
#   5. Read the runtime URL Cloud Run assigned and write it into the SPA's
#      `.env.production.local` so the next SPA build points at the cloud
#      runtime by default. (The user can still flip back to local via the
#      Settings → General "Mode" toggle.)
#   6. `firebase deploy --only hosting`.
#   7. Print the SPA + runtime URLs.
#
# Usage:
#   ./scripts/deploy-cloud.sh                     # uses defaults
#   PROJECT_ID=… REGION=us-central1 ./scripts/deploy-cloud.sh
#   IMAGE_TAG=v1.2.3 ./scripts/deploy-cloud.sh

set -euo pipefail

cd "$(dirname "$0")/.."

# Pretty output
if [[ -t 1 ]]; then
  C_BOLD="\033[1m"; C_DIM="\033[2m"; C_GRN="\033[32m"; C_YEL="\033[33m"; C_RED="\033[31m"; C_CYA="\033[36m"; C_OFF="\033[0m"
else
  C_BOLD=""; C_DIM=""; C_GRN=""; C_YEL=""; C_RED=""; C_CYA=""; C_OFF=""
fi
step() { printf "${C_CYA}${C_BOLD}»${C_OFF} %s\n" "$*"; }
ok()   { printf "  ${C_GRN}✓${C_OFF} %s\n" "$*"; }
warn() { printf "  ${C_YEL}!${C_OFF} %s\n" "$*"; }
err()  { printf "  ${C_RED}✗${C_OFF} %s\n" "$*" >&2; }
hr()   { printf "${C_DIM}%s${C_OFF}\n" "────────────────────────────────────────────────────────────"; }

# Resolve config
PROJECT_ID="${PROJECT_ID:-$(gcloud config get-value project 2>/dev/null || echo "")}"
REGION="${REGION:-us-central1}"
IMAGE_TAG="${IMAGE_TAG:-$(git rev-parse --short HEAD 2>/dev/null || echo latest)}"
ARTIFACT_REPO="${ARTIFACT_REPO:-agentverse}"
# Per-tenant model (task 3.4): one Cloud Run service per tenant from a shared
# image. Defaults deploy the `default` tenant; override TENANT_ID per tenant.
TENANT_ID="${TENANT_ID:-default}"
SERVICE_NAME="${SERVICE_NAME:-agentverse-runtime-${TENANT_ID}}"
MIN_SCALE="${MIN_SCALE:-0}"
MAX_SCALE="${MAX_SCALE:-10}"
CONTAINER_CONCURRENCY="${CONTAINER_CONCURRENCY:-1}"
CPU_LIMIT="${CPU_LIMIT:-1}"
MEMORY_LIMIT="${MEMORY_LIMIT:-1Gi}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-3600}"
RUNTIME_IMAGE="${REGION}-docker.pkg.dev/${PROJECT_ID}/${ARTIFACT_REPO}/runtime:${IMAGE_TAG}"
FIREBASE_PROJECT="${FIREBASE_PROJECT:-}"

require() {
  command -v "$1" >/dev/null 2>&1 || { err "$1 not found in PATH"; exit 2; }
}

hr
step "AgentVerse cloud deploy"
hr
echo "  PROJECT_ID    : ${PROJECT_ID:-<unset>}"
echo "  REGION        : $REGION"
echo "  TENANT_ID     : $TENANT_ID"
echo "  IMAGE_TAG     : $IMAGE_TAG"
echo "  RUNTIME_IMAGE : $RUNTIME_IMAGE"
echo "  SERVICE_NAME  : $SERVICE_NAME"
hr

[[ -z "$PROJECT_ID" ]] && { err "PROJECT_ID is unset and gcloud has no default project. Set PROJECT_ID or run gcloud config set project <id>"; exit 2; }

step "Checking required CLIs"
require gcloud
require firebase
require npm
require envsubst
ok "gcloud, firebase, npm, envsubst all present"

step "Building SPA (npm run build)"
npm run build
ok "dist/ built; MSW worker stripped by postbuild hook"

step "Submitting Cloud Build (runtime image)"
gcloud builds submit \
  --config infra/runtime/cloudbuild.yaml \
  --substitutions=_REGION="${REGION}",_REPO="${ARTIFACT_REPO}",_TAG="${IMAGE_TAG}" \
  --project "${PROJECT_ID}" \
  .
ok "image pushed to Artifact Registry"

step "Applying Cloud Run service manifest"
SPA_ORIGIN_PLACEHOLDER="https://${PROJECT_ID}.web.app"
TENANT_ID="${TENANT_ID}" PROJECT_ID="${PROJECT_ID}" REGION="${REGION}" IMAGE_TAG="${IMAGE_TAG}" \
  SPA_ORIGIN="${SPA_ORIGIN_PLACEHOLDER}" \
  MIN_SCALE="${MIN_SCALE}" MAX_SCALE="${MAX_SCALE}" \
  CONTAINER_CONCURRENCY="${CONTAINER_CONCURRENCY}" \
  CPU_LIMIT="${CPU_LIMIT}" MEMORY_LIMIT="${MEMORY_LIMIT}" TIMEOUT_SECONDS="${TIMEOUT_SECONDS}" \
  envsubst < infra/runtime/service.yaml \
  | gcloud run services replace - --region "${REGION}" --project "${PROJECT_ID}"
ok "Cloud Run service applied"

step "Reading deployed runtime URL"
RUNTIME_URL=$(gcloud run services describe "${SERVICE_NAME}" --region "${REGION}" --project "${PROJECT_ID}" --format='value(status.url)')
[[ -z "$RUNTIME_URL" ]] && { err "could not read Cloud Run service URL"; exit 1; }
ok "runtime URL: $RUNTIME_URL"

step "Writing .env.production.local with cloud runtime URL + auth"
# Auth env vars are sourced from your existing .env.local so the same Firebase
# project is used in both modes. We surface them in .env.production.local so
# Vite bakes them into the cloud bundle — and we flip VITE_AUTH_REQUIRED=true
# so the cloud SPA forces sign-in.
LOCAL_FIREBASE_API_KEY=$(grep -E '^VITE_FIREBASE_API_KEY=' .env.local 2>/dev/null | cut -d= -f2- | tr -d '"' | tr -d "'" || true)
LOCAL_FIREBASE_AUTH_DOMAIN=$(grep -E '^VITE_FIREBASE_AUTH_DOMAIN=' .env.local 2>/dev/null | cut -d= -f2- | tr -d '"' | tr -d "'" || true)
LOCAL_FIREBASE_PROJECT_ID=$(grep -E '^VITE_FIREBASE_PROJECT_ID=' .env.local 2>/dev/null | cut -d= -f2- | tr -d '"' | tr -d "'" || true)
LOCAL_FIREBASE_APP_ID=$(grep -E '^VITE_FIREBASE_APP_ID=' .env.local 2>/dev/null | cut -d= -f2- | tr -d '"' | tr -d "'" || true)

cat > .env.production.local <<EOF
# Generated by scripts/deploy-cloud.sh on $(date -u +%Y-%m-%dT%H:%M:%SZ)
# Cloud-mode defaults baked into the SPA bundle. Users can still flip back
# to a local runtime via Settings → General.

VITE_USE_MSW=false
VITE_ALLOW_CANVAS2D=false
VITE_CAO_BASE_URL=${RUNTIME_URL}
VITE_CLOUD_RUNTIME_URL=${RUNTIME_URL}
VITE_RUNTIME_MODE=cloud

# Auth in cloud mode: REQUIRED. Users must sign in with Google.
VITE_AUTH_PROVIDER=firebase
VITE_AUTH_REQUIRED=true
VITE_FIREBASE_API_KEY=${LOCAL_FIREBASE_API_KEY}
VITE_FIREBASE_AUTH_DOMAIN=${LOCAL_FIREBASE_AUTH_DOMAIN}
VITE_FIREBASE_PROJECT_ID=${LOCAL_FIREBASE_PROJECT_ID}
VITE_FIREBASE_APP_ID=${LOCAL_FIREBASE_APP_ID}
EOF
ok ".env.production.local written (auth: required; provider: firebase)"

# Sanity check: warn loudly if the Firebase config is empty — the cloud bundle
# will load but every sign-in attempt will fail.
if [[ -z "$LOCAL_FIREBASE_API_KEY" || -z "$LOCAL_FIREBASE_PROJECT_ID" || -z "$LOCAL_FIREBASE_APP_ID" ]]; then
  warn "VITE_FIREBASE_* values are missing from .env.local — the cloud SPA will not be able to sign users in."
  warn "Populate .env.local with your Firebase web app config before re-running this deploy."
fi

step "Re-building SPA against cloud runtime URL"
npm run build
ok "rebuilt"

step "Deploying SPA to Firebase Hosting"
if [[ -n "$FIREBASE_PROJECT" ]]; then
  firebase deploy --only hosting --project "$FIREBASE_PROJECT"
else
  firebase deploy --only hosting
fi
ok "Firebase Hosting deploy complete"

hr
ok "AgentVerse cloud deploy complete"
echo "    SPA          : ${SPA_ORIGIN_PLACEHOLDER}  (or your configured Firebase Hosting domain)"
echo "    Runtime API  : ${RUNTIME_URL}"
echo "    Health probe : ${RUNTIME_URL}/health"
hr
