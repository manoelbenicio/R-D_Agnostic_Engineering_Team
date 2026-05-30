# cloud-runtime-deployment — Implementation Tasks

> Owner: **IF** (Infra Dev) + **SUP** (shell + design system) for the SPA-side
> Mode toggle and rebrand. Aligns with master spec §13 deferred work.

## 1. Visible rebrand: hide "CAO" name from the SPA UI (SUP)

The brand is hidden until the cloud-runtime story is fully resolved. Internal
TypeScript symbols (`CaoClient`, `CaoApiError`, file names, query keys) stay —
they are not user-visible and refactoring them now widens blast radius.

- [x] 1.1 NavBar health pill text: `CAO ONLINE` / `CAO UNREACHABLE` /
      `CAO CONNECTING` → `RUNTIME ONLINE` / `RUNTIME UNREACHABLE` / `RUNTIME CONNECTING`
- [x] 1.2 Settings → General label `CAO Base URL` → `Runtime Base URL` with
      updated helper text mentioning local-Docker vs cloud Cloud-Run modes
- [x] 1.3 Health page section labels: `CAO Server` row label → `Runtime Engine`;
      provider row prefix `CAO Provider:` → `Provider:`
- [x] 1.4 Health page error / status strings:
      `Cannot reach CAO at` → `Cannot reach the runtime at`,
      `CAO Server is running and responding.` → `Runtime engine is running and responding.`,
      `Checking CAO connectivity…` → `Checking runtime connectivity…`
- [x] 1.5 First-Run Wizard step 1 description and FormField label
- [x] 1.6 `cao-client` user-visible network error message: `Unable to reach CAO endpoint`
      → `Unable to reach runtime endpoint`
- [x] 1.7 Page descriptions in Agent Studio, Dashboard, Memory Viewer rephrased
      to avoid the "CAO" brand
- [x] 1.8 Update affected unit tests with the new strings (HealthPage, FirstRunWizard)

## 2. Cloud-Run-compatible runtime image (IF)

- [x] 2.1 Move `infra/cao/Dockerfile` → `infra/runtime/Dockerfile` and rebrand
- [x] 2.2 Honour Cloud Run `$PORT` (with local default 9889) via a
      `runtime-entrypoint` shell wrapper
- [x] 2.3 Use `tini` as PID 1 for proper SIGTERM forwarding during graceful
      Cloud Run shutdown
- [x] 2.4 Add a state volume mount point (`/root/.cao`) — Cloud Run gen2 volume
      mounts (Cloud Storage / Filestore) plug into this directly
- [x] 2.5 Bake conservative CORS / WS allow-list defaults that the deploy
      manifest overrides per environment
- [~] 2.6 Pre-install at least one worker CLI (Claude Code or Kiro CLI) so a
      fresh deploy can actually execute deployed canvases. IaC-ready: the
      Dockerfile exposes a `WORKER_CLI` build ARG and `cloudbuild.yaml` a
      `_WORKER_CLI` substitution — empty by default so NO CLI is baked in.
      Options matrix documented in `docs/cloud-runtime-auth.md`. The actual CLI
      choice remains a USER DECISION (deferred).

## 3. Cloud Build + Cloud Run manifests (IF)

- [x] 3.1 `infra/runtime/cloudbuild.yaml` — builds the runtime image, tags it
      with `${SHORT_SHA}` + `latest`, pushes to Artifact Registry
- [x] 3.2 `infra/runtime/service.yaml` — declarative Knative spec, traffic 100 %
      to latest revision, startup + liveness probes against `/health`,
      `containerConcurrency: 1` so each tmux runtime stays isolated
- [x] 3.3 Authentication: deploy intentionally does NOT set
      `--allow-unauthenticated`; the auth-proxy / Firebase JWT enforcement is
      the next change
- [x] 3.4 Per-tenant isolation pattern — IaC implemented: one Cloud Run
      *service* per tenant from a shared, tenant-neutral image.
      `service.yaml` parametrized by `TENANT_ID` (service name + labels namespaced),
      per-tenant `CAO_TENANT_ID` / `CAO_STATE_PREFIX`, per-tenant CORS origin,
      tunable `MIN_SCALE`/`MAX_SCALE`/`CONTAINER_CONCURRENCY`/`CPU_LIMIT`/`MEMORY_LIMIT`/
      `TIMEOUT_SECONDS`. `cloudbuild.yaml` gains an optional `_TENANT` for
      tenant-pinned images. `deploy-cloud.sh` exports all vars (default tenant
      = `default`). BLOCKED on a real GCP project: per-tenant Secret Manager
      secrets (commented `secretKeyRef` in `service.yaml`).

## 4. SPA hosting on Firebase Hosting (IF)

- [x] 4.1 `firebase.json` at repo root — serves `dist/`, SPA fallback to
      `/index.html`, immutable cache headers on hashed assets, no-cache on
      HTML, baseline security headers (CSP / X-Frame / X-Content-Type)
- [x] 4.2 `.firebaserc.example` template
- [x] 4.3 Hosting ignores `mockServiceWorker.js` so the worker never reaches
      production even if a stale `dist/` accidentally contains it

## 5. Deploy automation (IF)

- [x] 5.1 `scripts/deploy-cloud.sh` — single command: build SPA → submit
      Cloud Build → apply Cloud Run service → read deployed runtime URL →
      bake it into `.env.production.local` → rebuild SPA against the cloud
      URL → `firebase deploy --only hosting`
- [x] 5.2 `start.sh` gains `cloud-deploy` and `cloud` subcommands; `local`
      stays the default
- [x] 5.3 Help text and command dispatcher updated; environment overrides
      documented inline (`PROJECT_ID`, `REGION`, `IMAGE_TAG`, etc.)

## 6. Auth scaffolding (SUP)

The SPA is wired for Firebase Auth but the integration is **disabled by
default** until the user provides a Firebase project. Local-mode bundles
contain none of the Firebase SDK (tree-shaken via `import.meta.env.PROD` and
`VITE_AUTH_PROVIDER` static checks).

- [x] 6.1 `src/shell/auth.ts` — `getAuthProviderName`, `isAuthEnabled`,
      `getAuthToken`, `getAuthSession` with a clean `none | firebase` switch
- [x] 6.2 `src/shell/auth.firebase.ts` — lazy-loaded Firebase Auth helper
      reading `VITE_FIREBASE_*` env vars; never imported eagerly
- [x] 6.3 `src/shell/app-fetch.ts` rewritten to attach
      `Authorization: Bearer <jwt>` when auth is enabled and a token is
      available, otherwise pass-through
- [x] 6.4 Login UI — `LoginButton` (NavBar), `LoginScreen` + `LoginPanel`,
      and `RequireAuthGate` (wraps content in `AppLayout`) backed by
      `auth-store.ts`. Inert in local mode (`isAuthEnabled()` false); renders
      the login flow only when `VITE_AUTH_PROVIDER=firebase`.
- [~] 6.5 Server-side enforcement — DECISION POINT documented in
      `docs/cloud-runtime-auth.md` (Cloud Run IAM vs custom proxy vs hybrid,
      with tradeoffs). Credential-free skeleton added: `service.yaml` keeps the
      no-`--allow-unauthenticated` network boundary + commented per-tenant
      secret wiring, and `infra/runtime/auth-proxy/README.md` holds a
      non-functional Option B reference (no deps, never built into the image).
      The enforcement choice remains a human decision (deferred); E2E verify
      needs GCP creds (task 8.6).

## 7. SPA Mode toggle (SUP)

- [x] 7.1 Settings → General: new "Runtime Mode" form field above the URL
      input with two preset buttons:
        * **Local (Docker)** → fills `http://127.0.0.1:9889`
        * **Cloud (Run)** → fills the URL baked at build time via
          `VITE_CLOUD_RUNTIME_URL`, or warns if not set
- [x] 7.2 Helper text explains that the URL is the source of truth and the
      preset buttons are conveniences

## 8. Verification (IF)

- [x] 8.1 `npm run lint` clean (0 errors; 3 pre-existing warnings unrelated to this change)
- [x] 8.2 `npm run typecheck` clean
- [x] 8.3 `npm test` green (389 passed / 8 skipped) with the rebrand string updates
- [~] 8.4 `docker build -t agentverse-runtime:latest -f infra/runtime/Dockerfile .`
      builds and the container responds with HTTP 200 on `/health`.
      NOT RUN (no Docker daemon available / no CAO source context). Static
      checks done instead: Dockerfile reviewed; `WORKER_CLI` ARG added; would
      run as written once a daemon + build context are present.
- [ ] 8.5 `./start.sh` (local mode) brings up the runtime + SPA end-to-end
      (deferred — needs a local CAO + Docker)
- [ ] 8.6 `./start.sh cloud-deploy` runs cleanly end-to-end against a real GCP
      project (deferred — requires user GCP creds; documented as a runbook)

### 8.x Syntax-only validations performed (no Docker/GCP, this change)

- [x] `bash -n scripts/deploy-cloud.sh` → syntax valid.
- [x] `python3 yaml.safe_load_all` on `infra/runtime/cloudbuild.yaml` → parses.
- [x] `python3 yaml.safe_load` on `infra/runtime/service.yaml` (raw) → parses.
- [x] `envsubst` render of `service.yaml` with sample tenant vars → parses;
      `metadata.name = agentverse-runtime-acme`, `containerConcurrency = 1`.
- Would run with Docker: `docker build ... -f infra/runtime/Dockerfile .`
  (optionally `--build-arg WORKER_CLI=…`).
- Would run with GCP: `gcloud builds submit --config infra/runtime/cloudbuild.yaml`
  then `envsubst < service.yaml | gcloud run services replace -`.

## Out of scope

- Per-tenant CAO isolation: IaC model now landed (task 3.4). Still out of
  scope: provisioning real per-tenant Secret Manager secrets + IAM (needs a
  GCP project).
- Auth-proxy implementation choice (Cloud Run IAM vs custom FastAPI proxy):
  tradeoffs documented in `docs/cloud-runtime-auth.md` (task 6.5); the choice
  itself is still a human decision and unimplemented.
- Cost monitoring of Cloud Run + Firebase Hosting (folds into FinOps Tier 2)
- Replacing internal `CaoClient` / `caoQueryKeys` / file paths with neutral
  names (cosmetic-only refactor, deferred until brand decision is final)
