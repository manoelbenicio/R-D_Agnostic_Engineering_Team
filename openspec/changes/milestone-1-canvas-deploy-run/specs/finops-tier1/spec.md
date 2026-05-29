## ADDED Requirements

### Requirement: Tier 1 Cost Estimation Formula

The system SHALL compute estimated cost for any time window as `Σ (PROVIDER_COST_PER_HOUR[provider] × active_hours)` per master spec §8.7. `active_hours` for a terminal SHALL be measured from the terminal's `last_active` timestamp (or its creation time if never active) up to the present. The provider-cost table SHALL match master spec §8.7 exactly:

- `kiro_cli`: $15.00/hr
- `claude_code`: $15.00/hr
- `codex`: $5.00/hr
- `gemini_cli`: $0.50/hr
- `kimi_cli`: $2.00/hr
- `copilot_cli`: $3.00/hr
- `opencode_cli`: $1.00/hr
- `q_cli`: $5.00/hr

The cost table SHALL be exposed as a single source of truth (`PROVIDER_COST_PER_HOUR` constant) consumed by every cost-displaying surface (Dashboard, FinOps page, Templates picker, voice "cost" command).

#### Scenario: Mixed-provider cost sums correctly

- **WHEN** the user has run a Claude developer for 2 hours and a Gemini reviewer for 30 minutes
- **THEN** the estimate is `(15.00 × 2) + (0.50 × 0.5) = 30.25` USD with the ⚠️ label

### Requirement: Mandatory Estimate Label

Every UI surface that displays a Tier 1 cost number SHALL include the ⚠️ glyph and the text "Rough estimate based on active time. Actual costs may differ significantly. See provider dashboard for real billing." per master spec §8.7. The text SHALL appear as a tooltip on the ⚠️ glyph at minimum and as inline copy in the FinOps page. UIs SHALL NOT display a cost number without this association.

#### Scenario: Cost on Dashboard is labeled

- **WHEN** the Dashboard renders the Cost / MTD KPI
- **THEN** the ⚠️ glyph is adjacent and its tooltip contains the master-spec disclaimer text verbatim

#### Scenario: Cost on Templates picker is labeled

- **WHEN** the Templates picker renders the 10 entries
- **THEN** every cost-per-hour estimate is rendered with the ⚠️ glyph

### Requirement: FinOps Page

The system SHALL expose a FinOps page at `/finops` showing: month-to-date cost (currency-formatted), budget utilization gauge, cost-by-provider breakdown table, cost-by-canvas breakdown (top 10 most expensive canvases this month), and a configuration affordance for the monthly budget. Costs SHALL refresh on the same cadence as the underlying CAO data (session list polling, 5 s).

#### Scenario: Budget utilization renders proportionally

- **WHEN** the user has set a monthly budget of $100 and the MTD estimate is $47
- **THEN** the budget gauge renders 47% filled with the ⚠️ label

#### Scenario: Top-10 canvases sort by descending cost

- **WHEN** the user has used 12 canvases in the current month
- **THEN** the Cost-by-canvas table shows exactly 10 rows ordered by descending cost

### Requirement: Tier 2 and Tier 3 Out of Scope

Token-level cost estimation by parsing terminal output (Tier 2) and exact billing via provider APIs (Tier 3) are explicitly post-launch (master spec §13). The v1 capability SHALL NOT include these and SHALL surface the limitation in the FinOps page footer: "Tier 1 estimate. Token-level billing accuracy arrives in a future release."

#### Scenario: FinOps page footer notes the limitation

- **WHEN** the user opens `/finops`
- **THEN** the page footer contains the post-launch notice
