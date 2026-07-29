# T20 — config.go tier-20 parsing edge-case review (READ-ONLY)

- agent `Opus48#B` · lane `T20-CONFIG-REVIEW` · pane `w6:p2` · task `T20-CONFIG-REVIEW`
- check-in: `.deploy-control/p0/checkins/Opus48-B__T20-CONFIG-REVIEW__20260723T211249Z.json`
- posture: **READ-ONLY** — no source writes. `loadAgentBrainIntegrationConfig`, `internal/daemon/config.go` + `internal/daemon/brain/config.go` at HEAD.

## 1. `AGENT_BRAIN_TASK_CAPACITY_TIER` validation (20/50/100)

Path: capacity `ResolveConfig` (:766) → `tierValue, err := strconv.Atoi(strings.TrimSpace(capacity.Value))`
(:774) → stored `CapacityTier: brain.CapacityTier(tierValue)` (:811) → `result.Validate()` (:816).
- `Validate()` (:184) calls `c.Neutral.Validate()` **first, unconditionally** → `brain.Config.Validate`
  (:189) → `return c.CapacityTier.Validate()` (:196) → enum check ∈ {20,50,100}. **Range IS enforced in
  ALL modes** (dev on or off). Then dev-only: `CapacityTier != CapacityTier20 → error` (:194).
- Outcomes:
  - non-integer (`"abc"`, `"20x"`, `"20.0"`, `"0x14"`) → Atoi error → `"…must be 20, 50, or 100"` (:776). ✓
  - out-of-range integer (`7`,`21`,`0`,`-20`,`999`) → Atoi OK → stored → `CapacityTier.Validate()`
    rejects `"capacity tier must be one of 20, 50, or 100"`. ✓ (enforced even with dev OFF)
  - valid `50`/`100` + **dev ON** → rejected by `:194` (dev authorized only for tier-20). ✓
  - valid `50`/`100` + dev OFF → passes Validate (valid enum) but **inert** (only tier-20 honored at
    admission `:678`).
- **Edge risks (low):**
  - **R1 (low/cosmetic):** `strconv.Atoi` accepts `"+20"` and `"020"` (base-10 → 20) and trims spaces; a
    stray `+`/leading-zero silently normalizes to 20. Harmless but lenient.
  - **R2 (low):** the Atoi-step error message `"must be 20, 50, or 100"` fires for ANY non-integer even
    though the true enum check is later; two different messages ("must be 20, 50, or 100" vs "must be one
    of 20, 50, or 100") describe the same constraint — minor operator confusion.

## 2. `AGENT_BRAIN_CAPACITY_GATE_ENABLED` bool parse (:784-786)

```go
capacityGateEnabled := false
if v, ok := os.LookupEnv("AGENT_BRAIN_CAPACITY_GATE_ENABLED"); ok {
    capacityGateEnabled, _ = strconv.ParseBool(strings.TrimSpace(v))   // <- error SWALLOWED
}
```
- unset → default **false** ✓; `""` → ParseBool error → swallowed → **false** ✓ (fail-closed);
  `0/f/F/false/FALSE` → false ✓; `1/t/T/true/TRUE` → true ✓.
- **R3 (MEDIUM — operability, silent misconfig):** `strconv.ParseBool` rejects common bool spellings
  (`yes`, `on`, `enabled`, `enable`, `2`, `y`, and typos like `ture`, `treu`). The error is discarded
  (`_`), so any such value → **silently `false`** with no log/error. An operator intending to enable the
  canary tier-20 via `=yes`/`=on`/`=enabled` gets it silently OFF and no signal. This is **fail-closed
  (safe: never accidentally enables)** but is **inconsistent** with `AGENT_BRAIN_DEVELOPMENT_ENABLED`
  and `AGENT_BRAIN_GATEWAY_REQUIRED`, which use `parseConfigBool(...)` and return a hard error on garbage.
- **R4 (low):** the gate is read via raw `os.LookupEnv`, bypassing the translator/`ConfigCandidate`
  precedence machinery used by every other setting — no CLI override, no legacy alias, no source
  tracking. Inconsistent sourcing (not wrong, but asymmetric with the tier/gateway config).
- **Recommendation:** parse with the shared strict `parseConfigBool` (hard error on unrecognized), or at
  minimum `slog.Warn` on ParseBool failure so a mistyped enable value is visible. Keep the fail-closed
  default.

## 3. Precedence vs `MULTICA_DAEMON_MAX_CONCURRENT_TASKS`

Capacity `ResolveConfig` candidate order (first `Set` wins):
1. `AGENT_BRAIN_TASK_CAPACITY_TIER` (neutral CLI)
2. `MULTICA_DAEMON_MAX_CONCURRENT_TASKS` (legacy CLI) — **only if `developmentEnabled && overrides.MaxConcurrentTasks != 0`** (:761-764)
3. `AGENT_BRAIN_TASK_CAPACITY_TIER` (neutral env)
4. `MULTICA_DAEMON_MAX_CONCURRENT_TASKS` (legacy env) — **conditional on `developmentEnabled`** (:770)
5. default `"20"`

Precedence itself is correct (neutral CLI > legacy CLI > neutral env > legacy env > default). Risks come
from the **semantic collision**: the legacy var historically carried an *arbitrary concurrency count*,
but here it feeds the **tier enum** (must be 20/50/100).
- **R5 (HIGH — migration footgun / hard fail):** a legacy dev config with
  `MULTICA_DAEMON_MAX_CONCURRENT_TASKS=8` (or 4/16/etc.) and no neutral tier set → `capacity.Value="8"` →
  `tierValue=8` → `result.Validate()` fails (`CapacityTier.Validate` → not in {20,50,100}) → **daemon
  config load fails to start**. A previously-valid legacy concurrency value now breaks boot in dev mode.
- **R6 (medium):** even a legacy value that IS a valid enum (`50`/`100`) is rejected in dev mode by the
  `== CapacityTier20` check (:194) — so the legacy var can only ever be `20` without erroring in dev.
- **R7 (low — silent no-op):** in **non-dev** (production), candidates 2 & 4 are gated off, so
  `MULTICA_DAEMON_MAX_CONCURRENT_TASKS` is **silently ignored** for the tier (falls through to neutral env
  or default 20). Operators relying on the legacy var in production see no effect and no warning.
- **Recommendation:** when the legacy `MULTICA_DAEMON_MAX_CONCURRENT_TASKS` candidate is `Set` but its
  value is not a valid tier (e.g. `8`), emit a dedicated error/warning naming the legacy var and the
  enum requirement (rather than the generic tier message), and document that the legacy var now maps to
  the tier enum. Consider a distinct message for the dev-only silent-ignore-in-production case.

## Severity summary
| ID | Area | Severity | Fail-safe? |
|---|---|---|---|
| R3 | GATE bool ParseBool error swallowed (garbage→silent false) | **MEDIUM** (operability) | yes (fail-closed) |
| R5 | legacy MAX_CONCURRENT_TASKS non-enum → dev config load hard-fails | **HIGH** (migration) | yes (fails closed, but blocks boot) |
| R6 | legacy valid 50/100 rejected in dev (must be 20) | medium | yes |
| R7 | legacy var silently ignored in production | low | yes |
| R1/R2 | Atoi leniency + dual error messages | low/cosmetic | yes |
| R4 | gate bypasses precedence/translator | low | n/a |

Tier **range validation is sound** (enforced all modes via `Neutral.Validate → CapacityTier.Validate`).
Primary real risks: **R3** (silent gate misconfig) and **R5** (legacy concurrency value breaks dev boot).

## Non-claims
Static read-only review; no source/test written; no run. Advisory only. The admission-honoring site
(`config.go:678` `CapacityGateEnabled && CapacityTier==CapacityTier20`) is noted but out of this parsing
review's scope.
