# Kill-Switch Operational Procedure

Status: DRY-RUN READY - LIVE EXECUTION F0-GATED

This procedure applies and verifies `rpp.l2.v1` kill switches. It is designed to stop unsafe runtime behavior before the next request or immediately, depending on feature scope. LIVE execution is prohibited until F0/F7 owner approval is recorded.

References:

- `docs/contracts/l2-runtime-contract.md`
- `docs/deploy/rollback-runbook.md`
- `docs/deploy/prod-rollout-runbook.md`
- `scripts/smoke/kill-switch-smoke.sh`

## 1. Supported Feature Keys

Use the narrowest feature and scope that stops the unsafe behavior:

```text
smart_context    next_request preferred for protocol/tool-call/JSON/continuation risk
gateway          immediate preferred for gateway policy or admission risk
auto_redeem      immediate preferred for redeem or account-state risk
provider_bridge  immediate preferred for provider/profile bridge risk
runtime_proxy    immediate preferred for broad runtime proxy risk
```

If a scoped kill switch cannot be confirmed, broaden scope and begin rollback.

## 2. Dry-Run Procedure

Run from the repo root:

```bash
bash scripts/smoke/kill-switch-smoke.sh --dry-run --feature smart_context
SMOKE_KILL_EFFECTIVE_AT=immediate bash scripts/smoke/kill-switch-smoke.sh --dry-run --feature gateway
SMOKE_KILL_EFFECTIVE_AT=immediate bash scripts/smoke/kill-switch-smoke.sh --dry-run --feature auto_redeem
SMOKE_KILL_EFFECTIVE_AT=immediate bash scripts/smoke/kill-switch-smoke.sh --dry-run --feature provider_bridge
SMOKE_KILL_EFFECTIVE_AT=immediate bash scripts/smoke/kill-switch-smoke.sh --dry-run --feature runtime_proxy
```

Dry-run pass criteria:

- each command exits 0;
- each command prints the planned `/v1/killswitch/apply` loopback request;
- no command requires a bearer token;
- no runtime state is changed;
- feature names match `rpp.l2.v1`.

Record dry-run evidence under `.deploy-control/evidence/` with feature, intended scope, exit code, and scrubbed summary.

## 3. LIVE Procedure

Only run after the owner gate is open:

```bash
SMOKE_ALLOW_EXECUTE=1 \
SMOKE_TARGET_ENV=prod \
DEPLOY_OWNER_APPROVED=true \
L2_BASE_URL=http://127.0.0.1:43117 \
L2_BEARER_TOKEN=<from-approved-secret-boundary> \
SMOKE_TENANT_ID=<tenant-id-or-approved-test-tenant> \
SMOKE_PROVIDER=<provider> \
SMOKE_PROFILE_ID=<profile-id> \
SMOKE_KILL_EFFECTIVE_AT=next_request \
bash scripts/smoke/kill-switch-smoke.sh --execute --feature smart_context
```

For immediate broad stop, change `SMOKE_KILL_EFFECTIVE_AT=immediate` and select the required feature. Do not run a broad kill switch as a convenience action; record the trigger and owner/operator decision first.

LIVE pass criteria:

- response confirms `contract_version=rpp.l2.v1`;
- response confirms `applied=true`;
- `effective_at` matches the requested or stricter accepted value;
- durable kill-switch store reflects the disabled state;
- runtime event acknowledgement is present when event stream is available;
- subsequent smoke or admission check proves the feature is disabled for the requested scope;
- evidence contains no raw tokens, prompts, provider payloads, database URLs, or Redis URLs.

## 4. Failure Response

If kill-switch apply or confirmation fails:

1. Treat the runtime as unsafe for the affected scope.
2. Freeze new affected sessions.
3. Broaden kill-switch scope only as needed to stop the failure.
4. Start rollback if the switch cannot be confirmed quickly.
5. Notify owner and Opus 4.8 with scrubbed trigger, feature, scope, and confirmation status.

## 5. Evidence Fields

Minimum evidence:

```text
kill_switch_id
timestamp_utc
operator
feature
scope
requested_effective_at
confirmed_effective_at
durable_store_result
event_ack_result
post_apply_behavior_result
rollback_required
secrets_present
remaining_risks
```
