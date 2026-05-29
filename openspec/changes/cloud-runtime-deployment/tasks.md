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
- [ ] 2.6 Pre-install at least one worker CLI (Claude Code or Kiro CLI) so a
      fresh deploy can actually execute deployed canvases. Deferred until the
      user picks which CLIs to bundle by default.

## 3. Cloud Build + Cloud Run manifests (IF)

- [x] 3.1 `infra/runtime/cloudbuild.yaml` — builds the runtime image, tags it
      with `${SHORT_SHA}` + `latest`, pushes to Artifact Registry
- [x] 3.2 `infra/runtime/service.yaml` — declarative Knative spec, traffic 100 %
      to latest revision, startup + liveness probes against `/health`,
      `containerConcurrency: 1` so each tmux runtime stays isolated
- [x] 3.3 Authentication: deploy intentionally does NOT set
      `--allow-unauthenticated`; the auth-proxy / Firebase JWT enforcement is
      the next change
- [ ] 3.4 Per-tenant isolation pattern (out of scope for this change)

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
- [ ] 6.4 Login UI (deferred until enforcement). When the user wants auth on,
      a follow-up wires a login button + protected-route guard.
- [ ] 6.5 Server-side enforcement (deferred — owned by `validation-proxy` /
      this change's auth-proxy section once we pick Cloud Run IAM vs custom
      proxy)

## 7. SPA Mode toggle (SUP)

- [x] 7.1 Settings → General: new "Runtime Mode" form field above the URL
      input with two preset buttons:
        * **Local (Docker)** → fills `http://127.0.0.1:9889`
        * **Cloud (Run)** → fills the URL baked at build time via
          `VITE_CLOUD_RUNTIME_URL`, or warns if not set
- [x] 7.2 Helper text explains that the URL is the source of truth and the
      preset buttons are conveniences

## 8. Verification (IF)

- [ ] 8.1 `npm run lint` clean
- [ ] 8.2 `npm run typecheck` clean
- [ ] 8.3 `npm test` green (with the rebrand-related test string updates)
- [ ] 8.4 `docker build -t agentverse-runtime:latest -f infra/runtime/Dockerfile .`
      builds and the container responds with HTTP 200 on `/health`
- [ ] 8.5 `./start.sh` (local mode) brings up the runtime + SPA end-to-end
- [ ] 8.6 `./start.sh cloud-deploy` runs cleanly end-to-end against a real GCP
      project (requires user creds; documented as a runbook deliverable)

## Out of scope

- Per-tenant CAO isolation (next change after this lands)
- Auth-proxy implementation choice (Cloud Run IAM vs custom FastAPI proxy)
- Cost monitoring of Cloud Run + Firebase Hosting (folds into FinOps Tier 2)
- Replacing internal `CaoClient` / `caoQueryKeys` / file paths with neutral
  names (cosmetic-only refactor, deferred until brand decision is final)
