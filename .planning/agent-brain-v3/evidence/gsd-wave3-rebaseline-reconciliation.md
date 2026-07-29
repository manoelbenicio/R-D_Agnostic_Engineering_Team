# EVIDENCE — P0 GSD Wave 3 Full Live Rebaseline and OpenSpec Traceability (ORQ-59)

- **Task ID**: `b36e6bd1-401e-4339-b97f-f57cbec6c4a6` (ORQ-59)
- **Title**: P0 GSD Wave 3 — Full live rebaseline and OpenSpec traceability
- **Author**: Agent `agy-p0-a7` (ID: `780104f1-1ff4-4292-8207-44b9ac4f5fca`)
- **Timestamp**: `2026-07-29T15:01:08Z`
- **Scope**: Documentation-only GSD rebaseline for `.planning/agent-brain-v3/` package

---

## 1. Executive Summary

This evidence artifact records the complete, full live rebaseline of the `.planning/agent-brain-v3/` GSD package against current production facts, the DB inventory grouped by `project_id, status` (UTC `2026-07-29T15:01:08Z`), and the OpenSpec topic content SHA reconciled under ORQ-62 (`7618599f29d43e964a485ab12a9932a9fd037e1f` accepted-in-review on branch `agent/agy-p0-a7/eb2c0b57`).

Zero product code or issue status changes were executed. All historical claims (including "not-in-production", old 51/96 counts, retired PD-08/STOP instructions, and July 18-21 lane states) have been explicitly superseded without erasing historical context.

---

## 2. Production Facts & Infrastructure Posture

1. **Base Integration Pointer**: `main` branch tip at `b657129` is recorded as a stale integration pointer reference.
2. **Production Overlay**: Production container deployment runs overlay commit `8227241`.
3. **Active Incidents & Outages**:
   - **ORQ-26 Outage**: Active Chat Lifecycle materialization and squad routing failure.
   - **GitHub Billing Lock**: Active external billing restriction preventing automated GitHub workflow execution.
   - **ORQ-61/59 Helper-Label Incident**: Auxiliary issue label incident recorded as metadata-only and corrected.
4. **Gates Separation**:
   - **Readiness Gates**: Local synthetic harness validation (e.g. 20-task local load harness in ORQ-50).
   - **Production Acceptance Gates**: Live production canary validation requiring multi-account deployment, DB persistence, and production telemetry verification.

---

## 3. OpenSpec Lineage & Reconciliation SHA

- **Lineage Defect**: Remote SHA `fc98e218` omitted mainline `rotation-parity-polyglot` artifacts.
- **OpenSpec Topic Content SHA**: Consumed verified remote ORQ-62 tip `7618599f29d43e964a485ab12a9932a9fd037e1f` (branch `agent/agy-p0-a7/eb2c0b57`), which is classified as **accepted-in-review / topic content** (not canonical-main or integrated), integrating ORQ-61 link repairs and passing strict validation (`openspec validate --all --strict`).

---

## 4. Live Unchecked OpenSpec Task Mappings

| Spec Domain | Task Range | Target Kanban Card | Description / Status |
|---|---|---|---|
| `native-runtimes-onboarding` | 1.6 | **ORQ-52** (DONE) | Native Runtimes Onboarding: Design System Parity & Web QA |
| `native-runtimes-onboarding` | 2.4 – 3.4 | **ORQ-53** (BACKLOG) | Native Runtimes Integration: NIM/Cline Backend Deploy & End-to-End Smoke |
| `credential-account-home-restoration` | 3.2, 3.4 | **ORQ-13** (IN_REVIEW) + **ORQ-14** (BLOCKED) | Canonical usage integration (ORQ-13) + real counters (ORQ-14) |
| `credential-account-home-restoration` | 3.3 | **ORQ-14** (BLOCKED) | Usage Telemetry — Real AGY/Kiro counters or explicit unavailability |
| `credential-account-home-restoration` | 4.5 | **ORQ-23** (IN_PROGRESS) | Independent safety review before live gate 4.5 |
| `chat-orchestration-standard` | 2.2 | **ORQ-54** (IN_PROGRESS) | Direct @agent routing and focused acceptance |
| `chat-orchestration-standard` | 2.3 | **ORQ-59** (IN_PROGRESS) | P0 GSD Wave 3 — Full live rebaseline and OpenSpec traceability |
| `build-omniroute-agent-brain` | 6.3, 6.4 | **ORQ-50** (IN_REVIEW) | Capacity — Prepare 20/50/100 harness and zero-queue load window |

---

## 5. Live DB Inventory Breakdown (Grouped by `project_id, status` at UTC 2026-07-29T15:01:08Z)

- **Main Project (`4b0ef49b-df06-4e83-9a29-8a23b34821d4`)**: Total **39 cards**
  - `done`: 21 | `in_review`: 7 | `in_progress`: 6 | `blocked`: 3 | `backlog`: 2
- **Unassigned (`project_id IS NULL`)**: Total **13 cards**
  - `done`: 3 | `in_review`: 3 | `in_progress`: 1 | `blocked`: 4 | `todo`: 1 | `cancelled`: 1
- **Validation Project (`abe3c461-c921-4a51-b91a-b08529429145`)**: Total **1 card**
  - `cancelled`: 1
- **Workspace Grand Total**: **53 cards** (ORQ-11 through ORQ-63)

Every executable residual task in GSD and OpenSpec has exactly one mapped Kanban card.

---

## 6. Verification & Quality Checks

1. **Git Diff Check**: `git diff --check` executed with zero whitespace errors.
2. **Relative Links Audit**: All relative links across `.planning/agent-brain-v3/` validated.
3. **Secrets Scan**: Zero secrets or credentials included in any documentation edit.
