# Owner ruling — production-first proportional validation

- **Effective:** 2026-07-28T20:47Z
- **Authority:** Owner + General Tech Manager
- **Scope:** current pre-release Multica engineering phase

## Ruling

The current environment is explicitly allowed to validate reversible changes in
production. The default validation strategy is therefore:

1. run the smallest relevant static/unit check before integration;
2. prepare an exact, fast rollback;
3. deploy the bounded change;
4. run a short production smoke against the real user path;
5. keep the change on success or roll back immediately, correct and repeat.

Agents must not spend hours reproducing production in an artificial environment
when a reversible production smoke gives stronger evidence faster. Repeating a
large pre-production matrix and then repeating the same validation in production
is prohibited unless the risk class below requires it.

## Risk-proportional gates

| Change class | Required gate |
|---|---|
| Reversible UI, presentation, isolated handler behavior, documentation or operational wiring | Focused test/build plus production smoke and rollback |
| Schema/migration, account attribution, concurrency invariant | Focused DB up/down/up and affected integration tests, then production smoke |
| Authentication, credential or secret rotation | Content-free preflight, old/new acceptance proof, rollback credentials/window, then controlled production test |
| Destructive delete/cascade | Exact target resolution, backup, transaction/rollback proof, then bounded production execution |
| Irreversible external/provider mutation | Explicit target-specific owner decision remains required |

Full package, browser, race or baseline matrices are required only when they cover
a credible regression surface not already proved by an independent prior run.
Previously reproduced evidence must not be rerun without a specific changed-risk
reason. A review of a narrow delta validates the delta, not the entire historical
stack again.

## Execution governance

- All executable agent activation remains Kanban-only with exactly one product task.
- Herdr remains supervision-only.
- The General Tech Manager chooses test depth and approves rollback readiness.
- The owner has granted standing authorization for technically approved bounded
  execution; generic approval must not be requested repeatedly.
- Failures are corrected and retested; they are not hidden by status changes.
