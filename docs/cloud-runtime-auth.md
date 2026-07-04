# Cloud Runtime — server-side auth enforcement (DECISION POINT)

> Status: **OPEN — needs a human decision.** This doc frames the choice for
> task 6.5 (`cloud-runtime-deployment`). It does not pick a winner, and the
> skeleton below requires **no GCP credentials** to read or extend.

## Problem

The SPA attaches a Firebase JWT (`Authorization: Bearer <jwt>`, see
`src/shell/app-fetch.ts`) when `VITE_AUTH_REQUIRED=true`. Nothing on the
**server side** validates that token yet. The Cloud Run service is deployed
*without* `--allow-unauthenticated`, so today the only thing standing between
the internet and the runtime is whatever default IAM the project has. We must
decide how requests are authenticated **before** they reach the orchestration
runtime.

## Option A — Cloud Run IAM (Google-managed, ID-token based)

Require IAM `roles/run.invoker` on the service; callers send a Google-signed
**ID token** in `Authorization: Bearer`. Google's front end validates it
before the request ever hits the container.

- ➕ Zero auth code in our container; Google validates the token.
- ➕ Smallest attack surface; no token-verification bug we can write.
- ➕ Per-tenant service already gives a per-tenant IAM boundary (task 3.4).
- ➖ The token must be a **Google-issued ID token for the service audience**,
  not a raw Firebase Auth user JWT. The browser SPA cannot mint one directly —
  it needs a token broker (e.g. a tiny Cloud Function / IAP) to exchange the
  Firebase session for an invoker token.
- ➖ Browser → Cloud Run direct calls don't fit cleanly; usually pairs with
  **IAP** or a backend. WebSocket upgrade through IAP needs validation.
- ➖ Coupled to GCP; harder to run the same enforcement locally.

## Option B — Custom auth proxy (e.g. FastAPI/Envoy sidecar)

A thin proxy in front of the runtime verifies the **Firebase ID token**
(`firebase-admin` / public JWKs), checks tenant claims, then forwards to the
runtime on localhost.

- ➕ Verifies the *Firebase user JWT the SPA already sends* — no token broker.
- ➕ Portable: identical enforcement locally and in cloud; testable with MSW.
- ➕ Room for app-level policy: per-tenant claims, rate limits, audit logs,
  topology-guard hooks (overlaps with the `validation-proxy` change).
- ➖ We own the security-critical code path (JWT verification, JWKS caching,
  clock-skew, revocation). Bugs here are auth bypasses.
- ➖ Extra hop + process to run, scale, and monitor per tenant.
- ➖ WebSocket proxying must be implemented correctly (upgrade, backpressure).

## Recommendation framing (not a decision)

- If the runtime stays **GCP-only** and a token broker/IAP is acceptable →
  **Option A** minimises code we must secure.
- If enforcement must be **portable, locally testable, and policy-rich**
  (and especially if it merges with `validation-proxy`) → **Option B**.

A common compromise: **Option A for the network boundary** (block
unauthenticated traffic at Cloud Run) **plus a slim Option B layer** for
tenant-claim / topology policy. That keeps Google doing token crypto while we
do app policy.

## What is implemented now (credential-free skeleton)

No credentials are required for any of the below:

1. `infra/runtime/service.yaml` deliberately omits `--allow-unauthenticated`
   and documents the per-tenant IAM/secret wiring (commented `secretKeyRef`,
   `BLOCKED` markers where a real project is needed).
2. The SPA already sends the Firebase JWT (`src/shell/app-fetch.ts`,
   task 6.3) — owned by the SUP agent, out of this change's file scope.
3. `infra/runtime/auth-proxy/` contains a **non-functional reference
   skeleton** (`README.md` + a commented FastAPI sketch) for Option B. It has
   no dependencies wired and is never built into the image.

## Blocked / needs a real GCP project

- Creating the per-tenant Secret Manager secrets referenced in `service.yaml`.
- Granting `roles/run.invoker` (Option A) or standing up IAP / a token broker.
- End-to-end auth verification (task 8.6) — requires user GCP creds.

## Bundling a worker CLI (cross-ref task 2.6)

Auth choice is independent of which worker CLI ships in the image. The image
bakes **no** CLI by default; `infra/runtime/Dockerfile` exposes a `WORKER_CLI`
build ARG and `cloudbuild.yaml` a `_WORKER_CLI` substitution. Options the user
can pass (illustrative, not endorsements):

| WORKER_CLI value                          | Notes                                  |
| ----------------------------------------- | -------------------------------------- |
| `npm i -g @anthropic-ai/claude-code`      | Node-based; needs node in the image.   |
| `uv tool install kiro-cli`                | Python/uv; uv already present.         |
| *(empty, default)*                        | No CLI — canvases cannot execute yet.  |

Decision deferred to the user.
