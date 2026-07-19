# MODERNIZATION_PACKETS — Agent Brain v3

> Planning-only artifact prepared by Kiro/Opus-4.8 (planning/adjudication lead).
> These packets are **prepared, not dispatched**. Dispatch is Codex#56#A's control and only
> after each packet's gate clears. No credential/auth/secret access, no Multica-daemon Codex
> dispatch, no production, cutover, Prodex removal, or tier 50/100.
> Context: STATE was already corrected by Codex#56#A to `G4 IN_PROGRESS` (G3 accepted at 44/85);
> this file does not re-raise that item.

## Global constraints (apply to both packets)

- PD-01: preserve the dirty baseline; no reset/stash/revert/discard.
- PD-08: never read, copy, print, rewrite, rotate, quarantine, or mutate any credential/auth/secret; synthetic or reference-only values only.
- D-V3-14: non-production; development validation only. No result authorizes production/cutover/removal/tiers.
- Do not edit Codex1 hotspots (central daemon/config/health/cmd/go.mod/execenv/models.go/brain/**/prodex*).
- Do not dispatch Codex through the current Multica daemon. Use isolated Herdr panes.
- Treat native adapters 5.6–5.8 as fail-closed contracts only; no native-acceptance claim.

## Packet A — Event-driven read-only schema-v1 status/evidence monitoring

- **Owner:** Codex4 (`w3:pA`).
- **Dependency:** G2D schema-v1 (`EV-G2D-03`, `internal/daemon/observability/schema.go`) exists; G3 accepted. Runs alongside/after G4 with no G3/G4 scope overlap.
- **File scope (exclusive):** `multica-auth-work/server/internal/daemon/observability/**` (new read-only event consumer + status/transition view). Must NOT touch central entrypoints, `brain/**`, `gateway/**`, `runtimeenv/**`, `deploy/**`, or the active request path.
- **Goal:** replace call-log/pane polling and full-transcript ingestion with a read-only tap on emitted schema-v1 events (admission, gateway-readiness, selection, affinity, refresh, quota, 401/403, 429/circuit, retry/fallback, cancellation, usage, overload); content-off, no account identity.
- **Acceptance evidence:**
  - **EV-MON-01:** consumer ingests schema-v1 events read-only; zero content/secret/account-identity fields; unknown fields rejected.
  - **EV-MON-02:** status/transition view derives phase/worker/model state from events (no transcript ingestion, no active-path change); deterministic on a synthetic event fixture.
  - **EV-MON-03:** redaction test proves no secret/prompt/tool-payload/account identity leaks into monitor output.
- **Stop conditions:** any need to wire into the active daemon beyond read-consumption (Codex1 hotspot) → STOP; any credential/secret access → STOP (PD-08); no Multica-daemon dispatch; read-only only.

## Packet B — Vendor/model visibility repair (gated on EV-G4-03)

- **Owner:** Codex4 (`w3:pA`) for the attribution/telemetry data layer. Any frontend/dashboard file is a **separate, explicitly gated** scope and is read-only with respect to the active request path.
- **Dependency (hard gate):** **G4 isolation evidence `EV-G4-03` accepted** (active-path safety) before any UI change.
- **File scope (exclusive):** `multica-auth-work/server/internal/daemon/observability/**` reporting/attribution view (requested-model → actual served model/route → pseudonymous account/connection). Frontend/dashboard files, if present in-repo, are a distinct scoped sub-task, read-only w.r.t. the active path; must NOT touch central entrypoints or the active request path.
- **Goal:** correct vendor/model attribution so the earlier class of mislabel (e.g. a Claude request surfacing as a wrong vendor/model) cannot recur; show true served model/route/vendor with pseudonymized account.
- **Acceptance evidence:**
  - **EV-VIS-01:** accurate requested→served model/route/vendor mapping on a synthetic dataset; pseudonymous account only.
  - **EV-VIS-02:** UI/report reflects the correct vendor/model; regression fixture proves the mislabel case is fixed.
  - **EV-VIS-03:** redaction — no account identity, secret, or content in the view.
- **Stop conditions:** must not start before `EV-G4-03` accepted; read-only telemetry; no active-path change; no credential/secret access (PD-08); no Multica-daemon dispatch.

## Parking rule (unchanged)

- Kanban automation stays **parked** until credentialless dispatch is proven end-to-end (post-G3/G4).
- MUL-2..MUL-25 remain parked; MUL-11/12/15 must be reconciled/superseded against OmniRoute exclusive credential/account/rotation ownership before any bulk action.

## Sequencing

1. Packet A is dependency-ready now (read-only; G2D schema-v1 present) — dispatch at Codex#56#A discretion.
2. Packet B is blocked until `EV-G4-03` (G4 isolation) is accepted.
3. Neither packet enables tier 20 (Codex1 task 9.2, separately gated), production, cutover, Prodex removal, or tiers 50/100.
