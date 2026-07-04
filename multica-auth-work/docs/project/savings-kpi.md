# Savings KPI — the Moat, in Dollars

> Source of truth for the **SAVINGS** tab in Analytics (design.md §8).
> Authored by GLM#52#B (Wave-1 stream RR-OBSERV). Overwrites any pre-existing stub.
> Grafana dashboard: `deploy/observability/grafana/dashboards/savings.generated.json`
> (generated from `scripts/observability/components.savings.yaml`).

The Savings KPI is the metric that proves the product's value **in a number**:
*"$ economizado rotacionando assinaturas vs metered"* — dollars saved by serving
traffic on rotated flat-rate subscriptions instead of paying a metered per-token API.

## What it proves (the moat)

The Rotation Router keeps traffic on flat-rate **subscription** accounts and rotates
across them as quota windows exhaust. Per design.md §11 (decision 1), Requesty is a
**reference, never a paid vendor**, so metered spend in production is **$0**. Every
token we serve on a subscription is a token we did **not** pay metered rates for.

The Savings KPI monetizes that gap — it turns "we rotate subscriptions" into a dollar
figure a buyer can act on:

```
Savings ($) = (tokens served on subscriptions) × (metered price we DID NOT pay)
```

Because metered spend is architecturally zero, the tokens observed on subscription
accounts (`account_tokens_used`) are exactly the tokens that would otherwise have been
billed at metered rates. The KPI is therefore a direct readout of the moat.

## Formula (as implemented in the dashboard)

The Savings panel is **ESTIMATED** and computed purely from a real catalog series
multiplied by a documented constant — no new metric is invented:

```
Savings_est($)            = sum(account_tokens_used) * P_metered
Savings_est_by_vendor($)  = sum by (vendor) (account_tokens_used) * P_metered
```

- `account_tokens_used{vendor,account_id}` — **real gauge** from
  `server/internal/metrics/credential_metrics.go` (tokens consumed per account).
- `P_metered` — **hypothetical blended metered price**, currently a constant embedded
  in the PromQL query: `3e-6` USD/token (~$3 per 1,000,000 tokens). This is a
  placeholder blended rate, NOT a per-vendor real price. It lives in the query and the
  panel titles are clearly labelled `ESTIMATED`.

Both Savings panels are flagged `ESTIMATED` in their titles because the price side of
the formula is not yet a real series.

### Inputs

| Input | Source | Status |
|-------|--------|--------|
| Tokens served on subscriptions | `account_tokens_used` (real gauge, `credential_metrics.go`) | ✅ instrumented |
| Metered reference price per token | constant `P_metered = 3e-6` in the PromQL query | ⚠️ hard-coded estimate |
| Per-vendor metered price | — | ❌ not instrumented (gap) |
| Model / context-tier price weighting | — | ❌ not instrumented (gap) |
| Prompt vs completion token split | — | ❌ not instrumented (gap) |

## Supporting spend dimensions (real metrics, no estimation)

The same dashboard ships exact panels straight from the catalog — these need no
estimate and are the context around the Savings number:

- `sum by (vendor) (account_tokens_used)` — observed token spend proxy by vendor.
- `account_tokens_used` (table) — per-account token usage, the **feedstock** the
  Savings sum aggregates (GROUP BY account).
- `sum by (vendor) (accounts_available)` — headroom: flat-rate accounts still usable.
- `sum by (reason) (rotation_total)` — rotation pressure by reason (request volume /
  failover dimension).
- `max by (vendor) (all_accounts_exhausted)` — full-exhaustion alarm by vendor (when
  the moat is at risk of running out).
- `sum by (signal) (rate(exhaustion_detected_total[5m]))` — exhaustion signal rate
  (detection dimension: proactive vs reactive).

Every one of these references a real series from `credential_metrics.go`; none is
fabricated.

## Gap — what still needs instrumentation to make Savings EXACT

The Savings panel is **ESTIMATED** because the metered-price side of the formula is
not yet a real series. To promote it from ESTIMATED to exact, instrument:

1. **A per-vendor metered reference price** (USD/token) as a metric or
   config-backed series, e.g. `metered_reference_price_usd_per_token{vendor}` — today
   it is the hard-coded constant `3e-6` in the query. **(primary gap)**
2. **Model / context-tier weighting** — `account_tokens_used` is not split by model or
   context tier, so a single blended price is applied. Real metered pricing varies by
   model; a `..._by_model` breakdown would sharpen the number.
3. **Prompt vs completion token split** — metered APIs price input and output tokens
   differently; the current gauge is a single total.

Until (1)–(3) exist, the dashboard ships the **computable** panels: the spend
dimensions are exact, and Savings is a clearly-labelled ESTIMATE. No fabricated metric
is used — every panel references a real series from `credential_metrics.go`; only the
price multiplier is an openly-documented constant.

## Regenerate the dashboard

```bash
cd /mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work
python3 scripts/observability/gen_dashboards.py scripts/observability/components.savings.yaml
# -> deploy/observability/grafana/dashboards/savings.generated.json
```

Grafana auto-provisions `deploy/observability/grafana/dashboards/*.json`; restart (or
wait for provisioning) then search by title "Savings":

```bash
docker restart multica-grafana >/dev/null && sleep 6
PW=$(cat deploy/observability/secrets/grafana_admin_password)   # never print/commit
curl -s -u admin:"$PW" "http://localhost:3000/api/search?query=Savings"
```

Never print or commit the Grafana admin password.
