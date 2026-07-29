# Supersession — OmniRoute owns provider routing and credentials

Date: 2026-07-22
Decision: owner-approved architectural supersession
Replacement authority: `openspec/changes/build-omniroute-agent-brain`

This change is canceled and archived incomplete. It MUST NOT be implemented, tested, dispatched, or treated as active backlog.

OmniRoute is the sole owner of provider/model mapping, provider account discovery, OAuth/API credentials, provider sessions, login/revoke, quota, account health, rotation, retry/failover and provider-specific configuration. Main Brain receives only the approved router-neutral route/readiness contract, uses one restricted OmniRoute transport reference, and fails closed when OmniRoute is unavailable. Main Brain never manages provider accounts or provider-native credentials.

## Disposition of the 17 formerly pending tasks

| Task | Disposition | Replacement, if any |
|---|---|---|
| 0.1 provider map | RETIRE | OmniRoute internal responsibility |
| 0.2 `resolveSessionEnv` mapping | RETIRE | No provider-session env mapping in Main Brain |
| 0.3 provider session-store contract | RETIRE | OmniRoute internal responsibility |
| 1.1 per-provider/per-account config dirs | RETIRE | OmniRoute credential vault/account pools |
| 1.2 provider-native environment injection | RETIRE | Main Brain removes provider credentials/auth homes from child environments |
| 1.3 global provider credential fallback | RETIRE | Forbidden; Main Brain fails closed with no direct-provider fallback |
| 2.1 provider account/session listing | RETIRE | OmniRoute administration surface, outside this product change |
| 2.2 provider account login | RETIRE | OmniRoute administration surface, outside this product change |
| 2.3 provider account revoke | RETIRE | OmniRoute administration surface, outside this product change |
| 3.1 provider `session_id` propagation | RETIRE | Main Brain keeps only router-neutral task/execution/session correlation under `end-to-end-observability` |
| 3.2 assigned-account environment | RETIRE | Replaced by credentialless child boundary; no provider account environment is injected |
| 3.3 provider-specific runtime credential coverage | RETIRE | CLI adapters target OmniRoute without provider-account lookup |
| 4.3 local automatic account reassignment | RETIRE | OmniRoute rotation/account selection |
| 4.4 local account-switch alerting | RETIRE | Main Brain may consume bounded router readiness/result metadata, never account identity |
| 5.1 build/tests for this capability | RETIRE | Main Brain validation tasks 5.2/5.3 cover active product code |
| 5.3 local rotation test | RETIRE | OmniRoute-owned validation, not Main Brain product scope |
| 5.4 provider credential log sanitization | RETIRE HERE | Stronger active requirements already exist in `credentialless-agent-execution` and `end-to-end-observability`: no credentials, auth values, account identity or content in diagnostics |

## Active replacement guarantees

The active Main Brain change already requires:

- OmniRoute as the only router owner;
- no provider account selection or direct/alternate fallback;
- no provider-native credentials, auth homes or direct endpoints in child environments;
- router-neutral task/session/request correlation without account identity;
- metadata-only, secret-free logs, metrics and traces;
- fail-closed admission when OmniRoute is not ready.

The obsolete delta spec MUST NOT be synchronized into main specs.
