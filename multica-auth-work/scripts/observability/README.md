# Dashboards as Code

`gen_dashboards.py` reads a YAML component spec and writes Grafana dashboard JSON into the provisioning directory Grafana already auto-loads:

```bash
python3 scripts/observability/gen_dashboards.py scripts/observability/components.example.yaml
```

The default output directory is `deploy/observability/grafana/dashboards`. Generated files are named `<component-name>.generated.json`, so hand-written dashboards such as `rotation.json` are not overwritten.

## Requirements

The generator uses PyYAML for YAML parsing:

```bash
pip install pyyaml
```

Python 3.10+ is recommended.

## Spec Format

Each component becomes one Grafana dashboard. Each panel must declare a real metric from the catalog and a Grafana panel type:

```yaml
components:
  - name: rotation-overview
    title: "Account Rotation"
    panels:
      - title: "Rotations total"
        metric: rotation_total
        type: stat
        query: "sum by (reason) (rotation_total)"
      - title: "All accounts exhausted"
        metric: all_accounts_exhausted
        type: gauge
        query: "max by (vendor) (all_accounts_exhausted)"
      - title: "Accounts available"
        metric: accounts_available
        type: timeseries
        query: "accounts_available"
```

Supported panel types are `timeseries`, `stat`, `gauge`, and `table`.

Allowed metrics are:

- `rotation_total{vendor,reason,result}`
- `rotation_duration_seconds{vendor}`
- `all_accounts_exhausted{vendor}`
- `accounts_available{vendor}`
- `account_status{vendor,account_id,status}`
- `account_tokens_used{vendor,account_id}`
- `account_window_seconds_remaining{vendor,account_id}`
- `exhaustion_detected_total{vendor,signal}`
- `credential_restore_total{vendor,result}`
- `cred_env_injection_total{vendor,result}`
- `credential_prepare_seconds{vendor}`

Unknown metrics fail loudly with a non-zero exit and no dashboard output for that invalid run.

## Grafana Loading

The observability stack provisions dashboards from `deploy/observability/grafana/dashboards/*.json`. After generating JSON, restart or wait for Grafana provisioning to refresh, then search for the dashboard by title:

```bash
docker restart multica-grafana >/dev/null
PW=$(cat deploy/observability/secrets/grafana_admin_password)
curl -s -u admin:"$PW" "http://localhost:3000/api/search?query=Account%20Rotation"
```

Do not print or commit the Grafana admin password.
