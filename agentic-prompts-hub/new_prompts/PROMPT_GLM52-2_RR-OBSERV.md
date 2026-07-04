<role>
You are GLM#52#B, observability engineer. Add the Savings KPI + extra spend dimensions to
the dashboards-as-code layer, and document the Savings formula. NEW files / isolated edits only.
"Done" = a generated dashboard with a Savings panel that Grafana loads, + a doc, verified.
</role>

<mandatory_signin_signout priority="0" optional="false">
- BEFORE editing: write /mnt/c/VMs/Projetos/Automonous_Agentic/.deploy-control/GLM-52-B__RR-OBSERV__<START_UTC>.md
  (ABSOLUTE path; START_UTC=`date -u +%Y%m%dT%H%M%SZ`) with agent+started_at+status:IN_PROGRESS+files_locked.
- AFTER: same file with finished_at + agent + status:DONE|BLOCKED + build_result.
- No started_at+finished_at+agent = NOT complete.
</mandatory_signin_signout>

<lock_discipline>
files_locked (NEW): scripts/observability/components.savings.yaml,
deploy/observability/grafana/dashboards/savings.generated.json (generated),
docs/project/savings-kpi.md
Reuse the EXISTING scripts/observability/gen_dashboards.py (do NOT edit it unless strictly
needed; prefer a new YAML spec fed to it). Do NOT edit product Go or existing dashboards.
</lock_discipline>

<context source="openspec/changes/rotation-router/design.md §8 — invent no metric names">
gen_dashboards.py already exists and validates metric names against the real catalog
(credential_metrics.go). Real metrics: rotation_total, all_accounts_exhausted, accounts_available,
account_tokens_used, exhaustion_detected_total, credential_restore_total, etc.
Savings = value proven by NOT paying metered API. Grafana secret: deploy/observability/secrets/grafana_admin_password (mask it).
</context>

<task>
1. components.savings.yaml: a component spec (fed to gen_dashboards.py) with panels using ONLY
   real metrics — e.g. accounts_available by vendor, rotation_total by reason, plus a "Savings
   (estimated)" panel. If a pure-metric Savings isn't computable from existing series, define it
   as a documented derived expression (tokens observed × hypothetical metered price) and mark it
   clearly as ESTIMATED in the panel title. Do NOT invent a metric that doesn't exist — if the
   inputs aren't there, document the gap in savings-kpi.md and ship the panels that ARE computable.
2. Run gen_dashboards.py on it → savings.generated.json.
3. docs/project/savings-kpi.md: the Savings formula, its inputs, what it proves (the moat in $),
   and any gap (which inputs still need instrumentation).
</task>

<example>
```
cd /mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work
python3 scripts/observability/gen_dashboards.py scripts/observability/components.savings.yaml
python3 -c "import json;json.load(open('deploy/observability/grafana/dashboards/savings.generated.json'));print('json ok')"
docker restart multica-grafana >/dev/null && sleep 6
PW=$(cat deploy/observability/secrets/grafana_admin_password)
curl -s -u admin:"$PW" "http://localhost:3000/api/search?query=Savings" | python3 -c "import sys,json;print('found:',len(json.load(sys.stdin)))"
```
</example>

<verification>
Prove: valid JSON generated + Grafana API returns the dashboard + savings-kpi.md exists with the
formula + gap noted. Mask the password in all pasted output. DONE only with these shown.
</verification>

<persistence>
Finish fully; fix-and-rerun on error. Use ONLY real catalog metrics — if Savings inputs are
missing, ship what's computable and document the gap; do NOT fabricate a metric. BLOCKED only on true blocker.
</persistence>

<output>Sign-out: agent GLM#52#B, started_at, finished_at (UTC), status DONE, verification (password masked) in build_result.</output>
