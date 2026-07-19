# Credential isolation 1.1–1.3 architecture remediation decision

## Status and preflight claim

- Decision ID: `DEC-CREDISO-1.1-1.3-20260718`.
- Status: **RECOMMENDATION READY; PRODUCT-OWNER DECISIONS REQUIRED BEFORE IMPLEMENTATION**.
- Recommended option: **A — explicit tenant-scoped account, no implicit global fallback**.
- Input audit: `credential-isolation-config-env-audit.md`, SHA-256 `7fb12ec8b1a4e85209cef4f85f4d88c82dc5520b5797bb29df1339ddd7abcef4` (verified before this decision).
- Repository commit: `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`.
- Preflight: docs-only architecture work. Source, product, OpenSpec, STATE, ledger, and index remained read-only. No test, binary, database, network, provider, credential path, auth home, secret, token, or real environment value was accessed.
- This document specifies future edits; it does not authorize or perform them and does not mark any task complete.

## Problem to decide

The current artifacts and implementation encode incompatible requirements:

1. OpenSpec task/spec 1.3 says an unassigned agent preserves the global credential home.
2. Production fails closed when the rotation store, assignment, account, or account home is absent (`server/internal/daemon/daemon.go:4031-4056,4077-4079`).
3. The design says the literal proposal env map is wrong for installed Kiro and likely wrong for Antigravity, while Claude is not in the actual roster (`design.md:1-16,57-72,105-135`).
4. `Account` stores both `HomeDir` and `ConfigDir`, but production uses only `HomeDir` and does not compare `Account.TenantID` with `Task.WorkspaceID` (`internal/rotation/contract.go:57-65`; `daemon.go:4045-4056`).
5. Codex auth is account-sourced, but configuration, sessions, and plugin cache still reach the shared Codex home (`execenv/codex_home.go:104-165`).
6. Alternate provider discovery roots can be supplied through local custom env after isolated values are injected (`daemon.go:3653-3667,4860-4866`); `CLOUDSDK_CONFIG` also escapes the credentialless `runtimeenv` classifier (`runtimeenv/policy.go:24-74,92-130`).

The remediation must not turn the OpenSpec inconsistency into a production credential fallback.

## Threat model

### Protected assets

- Provider OAuth/token state, provider configuration, session metadata, and account-scoped refresh state.
- Tenant/workspace isolation and the binding `(tenant, provider, agent) → account`.
- Trusted child-process routing roots and the guarantee that one task cannot discover another account's or the daemon user's provider state.

### Adversaries and failure modes

| Threat | Capability | Required defense |
| --- | --- | --- |
| Cross-tenant assignment bug or corrupted row | Causes an agent to reference a valid account ID owned by another workspace | Resolve assignment and account under the task tenant atomically; reject mismatch before filesystem access. |
| Malicious/stale custom env | Supplies an alternate home, config root, credential file, direct key, or endpoint after isolation injection | Central case-normalized deny policy; remove inherited values; reject custom values by key name; trusted provider roots applied last. |
| Provider CLI precedence/version drift | Starts honoring an alternate root such as `KIRO_HOME` or `CLOUDSDK_CONFIG` | Deny the complete discovery-root family even when the current binary reportedly ignores it; versioned provider contract and acceptance gate. |
| Shared global-state contamination | A task refreshes auth/config/session/plugin state visible to another account | No implicit global source; account-owned mutable state; task overlays; explicit persistence/write-back policy. |
| Path traversal/symlink substitution | A stored path escapes the managed credential root or is replaced before use | Derived canonical paths, component-wise no-symlink validation, containment under a trusted root, and ExactEnv pre-launch checks. Same-UID post-check TOCTOU remains an OS-level residual. |
| Operator compatibility pressure | Enables a global fallback to keep a legacy workstation working | Make compatibility explicit, development-only or represented as an assigned account; never infer it from missing state. |
| Concurrent tasks sharing one account | Mutate the same refresh/session state concurrently | Account-scoped lock/CAS for mutable write-back; task-local overlays; no cross-account shared writable paths. |

## Options

| Option | Behavior | Security | Compatibility/operations | Decision |
| --- | --- | --- | --- | --- |
| **A. Explicit managed account; no implicit fallback** | Every credential-bearing launch requires a valid tenant/provider/account assignment. A legacy/global credential may be retained only by explicitly importing or registering it as a managed account and assigning it. Missing assignment always fails closed in every deployment class. | Strongest. Absence and corruption cannot select daemon-user state. Audit and rotation share one account model. | Requires bootstrap/import and explicit assignment. Current production already fails closed, so unassigned production behavior does not regress. Legacy workstation setup becomes one explicit step. | **RECOMMENDED**. |
| **B. Explicit development/test-only implicit fallback** | Default remains required. A new `legacy-global-development` mode may use one approved global source only when explicitly selected and when hard gates pass. Production startup rejects the mode. | Weaker: retains ambient state and adds a second launch path. Safe only with strict constraints and dedicated tests. | Lowest-friction local migration. Adds config surface and ongoing support burden. | Accept only if principal confirms a real development bootstrap need that Option A cannot meet. |
| **C. Literal OpenSpec fallback in production** | Missing store/assignment silently uses the daemon user's provider home. | Unacceptable: tenant/account attribution disappears, custom/global state becomes ambient, rotation/account auditing can be bypassed, and false isolation is likely. | Preserves legacy behavior but invalidates the capability's security objective. | **REJECT**; must not be implemented. |

### Recommended end state: Option A

There is no semantic category called “unassigned but credential-bearing.” A launch is either:

```text
credentialless gateway route
        OR
tenant-scoped assigned provider account
        OR
fail closed before provider state is read
```

A legacy global home, if it must be preserved, becomes an explicit account enrollment/import operation. Its files are copied into the managed credential root; production never points a child at the daemon user's live global home. The account receives a normal tenant, provider, account ID, `HomeDir`, `ConfigDir`, status, and assignment. This reconciles compatibility with production fail-closed behavior without a hidden fallback branch.

### If Option B is retained

All of the following are mandatory, not advisory:

- New policy enum defaults to `required`; no boolean whose zero value enables fallback.
- `legacy-global-development` is accepted only with an explicit development/test deployment class. Production configuration validation fails startup before task polling.
- It is mutually exclusive with gateway-required mode, rotation, multi-account enrollment, session APIs, and more than one concurrent credential-bearing task.
- Provider allowlist initially contains only Codex because it has an explicit compatibility helper. Kiro, Antigravity, Claude, GLM/OpenCode, Cline, and NIM remain fail closed until separately approved.
- Only `ErrNoAssignment` may invoke it. Nil store, store errors, missing account, tenant mismatch, provider mismatch, invalid paths, or partial environment always fail closed.
- It emits a content-free warning/metric containing mode/provider/reason only—never credential paths or values.
- It is not inferred from `AgentBrain.DevelopmentEnabled`; that is a separate gateway feature. A dedicated credential-isolation deployment policy must be validated at startup.
- CI must prove production rejects the mode and that the compatibility path cannot run concurrently or under a non-development class.

## Tenant/workspace assignment contract

### Recommended resolver

Introduce a narrow resolver contract rather than making the daemon compose two unscoped store calls:

```go
type CredentialAssignmentResolver interface {
    ResolveAssignedAccount(ctx context.Context, agentID, tenantID string) (rotation.Account, error)
}
```

The PostgreSQL implementation performs one assignment/account join filtered by both `agent_id` and `accounts.tenant_id`. The daemon then validates:

1. task `AgentID` and `WorkspaceID` are non-empty;
2. account ID and tenant are non-empty;
3. `account.TenantID == task.WorkspaceID`;
4. normalized `account.Vendor == normalized task provider`;
5. provider is in the versioned credential-isolation contract;
6. `HomeDir` and `ConfigDir` satisfy the provider path contract;
7. account status permits launch.

Every failure returns a generic launch error and a content-free reason code. No path, account email, token, or secret appears in logs/errors. Defense-in-depth comparisons remain in the daemon even though the query is tenant-filtered.

The assignment write path must enforce the same tenant/provider rules before storing the relation. A database constraint or tenant-bearing assignment key is desirable, but application validation is still required because the current schema relates only agent ID to account ID.

## Managed path and `ConfigDir` semantics

### Common rules

- Trusted credential root is operator-owned configuration, never task/custom env.
- Canonical account root: `<credential-root>/tenants/<tenant-id>/<provider>/<account-id>/`.
- `HomeDir` is the account ownership boundary. `ConfigDir` must be equal to or a canonical descendant of `HomeDir` and must follow the provider contract below.
- Paths are absolute, clean, non-root, component-wise non-symlink, exist with directories restricted to `0700`, and are derived/validated from IDs rather than accepted as arbitrary external paths.
- Existing DB path strings are migration inputs, not automatically trusted.
- Task-local copies live under the task execution root. Only explicitly named account-scoped mutable files may be reconciled back, under an account lock and atomic replace/CAS.

### Provider contract

| Provider | `HomeDir` | `ConfigDir` meaning | Trusted child projection | Boundaries |
| --- | --- | --- | --- | --- |
| Codex | Account root | `<HomeDir>/codex-home`, the complete account-owned Codex source | `CODEX_HOME=<task overlay>` | `auth.json` account-scoped; config files copied only from `ConfigDir`; sessions account-scoped, never daemon-global; plugin cache shared only if separately proven immutable/content-addressed/non-secret and mounted read-only. |
| Kiro | Account root | `<HomeDir>/xdg-data`, containing `kiro-cli/data.sqlite3` | `XDG_DATA_HOME=<task copy>` | Block `KIRO_HOME`, `KIRO_CONFIG_DIR`, `HOME`, other XDG roots, and direct Kiro/AWS credential selectors. Current native lever remains version-gated. |
| Antigravity/Agy | `<account-root>/home` | `<HomeDir>/.gemini/antigravity-cli` | `HOME=<task home>` | Copy only the account token/config subtree. Block Gemini/Google/Cloud SDK alternate roots. Native acceptance remains pending binary proof. |
| OpenCode/GLM | Account root | `<HomeDir>/.config`, with provider config under `opencode/`; data under `<HomeDir>/.local/share/opencode` | task `XDG_CONFIG_HOME` plus task `XDG_DATA_HOME` | Both roots are mandatory and account-scoped; no global OpenCode auth/provider config. |
| Cline | Account root | `<HomeDir>/.cline` (or one normalized canonical data root selected at enrollment) | `CLINE_DATA_DIR`, `CLINE_SANDBOX=1`, `CLINE_SANDBOX_DATA_DIR` | Stop runtime-time heuristic search across multiple layouts after migration; enrollment normalizes once. |
| NIM | Account root | Not a filesystem config directory; `ConfigDir` must be empty or a documented metadata root | controlled credential reference/value projection only | Raw API-key files are a transitional exception and should not define the general `ConfigDir` model. Principal must confirm whether NIM stays in this capability. |
| Claude | **Unsupported pending discovery** | Undefined until installed-version store/root proof exists | none; fail closed | Do not guess `CLAUDE_CONFIG_DIR`. The proposal/task roster must be reconciled with the design before implementation. |

### Codex state decision

Recommended boundary:

- `auth.json`: account-owned mutable credential. Each task receives a regular-file snapshot. A successful refresh may return only this file to the same account using an account-scoped lock, source-generation/CAS check, `0600` temporary file, fsync, and atomic rename. Never write to another account or the daemon-global home.
- `config.toml`, `config.json`, `instructions.md`: source only from account `ConfigDir`; task overlay may add daemon-managed sandbox/skills/routes but never writes those transformations back.
- `sessions/`: persistent per-account directory. A task may link to that account directory, but never to the daemon user's global sessions. Concurrent writes require provider-supported semantics or per-account serialization.
- `plugins/cache`: no global writable link. Share only an audited immutable/content-addressed cache mounted read-only; otherwise copy/account-scope it. Plugin registration/config remains account-scoped.

The principal must decide whether auth refresh write-back is required now or whether tasks are deliberately read-only snapshots that force re-enrollment after refresh. Silent “task refresh succeeds but next task reloads stale source” is not acceptable.

## Environment/discovery-root policy

One case-normalized policy registry must drive inherited removal, custom-env rejection, trusted apply-last injection, `isBlockedEnvKey`, and pre-launch assertions. Duplicated hand-maintained lists are not acceptable.

### Always blocked as untrusted discovery roots/selectors

```text
HOME
USERPROFILE
XDG_DATA_HOME
XDG_CONFIG_HOME
CODEX_HOME
CODEX_CONFIG_DIR
CLAUDE_CONFIG_DIR
GEMINI_CONFIG_DIR
CLOUDSDK_CONFIG
GOOGLE_APPLICATION_CREDENTIALS
KIRO_HOME
KIRO_CONFIG_DIR
CLINE_DATA_DIR
CLINE_SANDBOX_DATA_DIR
OPENCLAW_CONFIG_PATH
OPENCLAW_STATE_DIR
OPENCLAW_HOME
OPENCLAW_INCLUDE_ROOTS
AWS_CONFIG_FILE
AWS_SHARED_CREDENTIALS_FILE
AWS_PROFILE
```

Provider-native trusted values are re-added only by the selected provider contract. A provider may trust `HOME` or XDG roots for its own task-local projection; custom/inherited values never win.

### Credential/routing bypass families

Reject custom and remove inherited provider credential aliases and direct-route selectors, including:

- `OPENAI_*`, `CODEX_*`, `ANTHROPIC_*`, `CLAUDE_*`, `GOOGLE_*`, `GEMINI_*`, `GCP_*`, `KIRO_*`, `AWS_*`, `NVIDIA_*`, `NIM_*`, and provider-specific Cline/OpenCode/OpenClaw credential/endpoint fields;
- generic `*_API_KEY`, `*_ACCESS_TOKEN`, `*_REFRESH_TOKEN`, `*_AUTH_TOKEN`, `*_BEARER_TOKEN`, `*_OAUTH*`, `*_COOKIE*`, `*_CREDENTIAL*`, `*_BASE_URL`, `*_API_BASE`, `*_API_URL`, and `*_ENDPOINT*` names for credential-isolated launches.

Compatibility exception lists must be per provider, non-secret, minimal, and validated before trusted injection. Rejection diagnostics contain key names and reason codes only.

## Exact OpenSpec edits required after owner approval

These are proposed edits, not changes made by this pack.

### `proposal.md`

1. Replace the stale legacy `PROVIDERS` map as implementation truth with the versioned Go provider contract: Codex `CODEX_HOME`; Kiro `XDG_DATA_HOME`; Antigravity `HOME` pending acceptance; OpenCode/GLM XDG pair; Cline data/sandbox roots; Claude unsupported pending discovery.
2. State that missing assignment fails closed in production and that compatibility is either an explicitly enrolled account (Option A) or a development-only policy (Option B).
3. Replace references to absent legacy Python/TypeScript paths as affected implementation with current Go account/store, daemon resolver, execenv, runtimeenv, and future credential-session API work.

### `design.md`

Add a normative decision section containing:

- chosen fallback option and startup/launch state machine;
- atomic tenant-scoped resolver and defense-in-depth tenant/provider checks;
- canonical account-root/HomeDir/ConfigDir rules and the provider table above;
- Codex auth/config/session/plugin boundaries and refresh write-back decision;
- central alternate-root/credential-bypass registry and trusted-apply-last order;
- migration rules for existing account rows and global Codex state;
- explicit unsupported/provider-validation gates for Claude and native Antigravity.

Remove or label historical any statement that says both “Claude is out of roster” and “Claude is required” without a decision. Historical empirical notes may remain, but normative behavior must be unambiguous.

### `spec.md`

Replace `Scenario: Fallback quando não há atribuição` with:

```text
#### Scenario: Produção sem atribuição falha fechada
- WHEN um runtime que exige isolamento não possui atribuição válida para o mesmo tenant e provedor
- THEN o daemon recusa o launch antes de ler qualquer home/config global
- AND nenhum estado de credencial do usuário do daemon é projetado ao processo filho
```

If Option A is chosen, add:

```text
#### Scenario: Compatibilidade global exige conta explícita
- WHEN o operador precisa migrar uma credencial global legada
- THEN ele a importa/registra como conta tenant-scoped e atribui essa conta explicitamente
- AND a produção nunca seleciona o home global implicitamente
```

If Option B is chosen instead, add a development-only scenario with every gate listed in this decision and an explicit production-startup rejection scenario.

Add requirements/scenarios for:

- tenant and provider equality on assignment read/write;
- absolute/canonical/contained/no-symlink HomeDir and ConfigDir;
- provider-specific config semantics and complete required env projection;
- removal/rejection of all alternate roots before trusted apply-last injection;
- Codex config/session/plugin state never referencing daemon-global writable state;
- generic, content-free errors/logs.

### `tasks.md`

Replace tasks 1.1–1.3 with:

```text
- [ ] 1.1 Implementar raiz canônica tenant/provedor/conta, validar HomeDir/ConfigDir e resolver atribuição atomicamente no mesmo tenant.
- [ ] 1.2 Implementar contrato versionado de env nativa por provedor, remoção/rejeição de raízes alternativas e injeção trusted-last.
- [ ] 1.3 Falhar fechado sem atribuição em produção; implementar somente a compatibilidade explicitamente aprovada (conta importada ou modo development-only restrito).
```

Add separate unchecked verification tasks for pure fake-store assignment negatives, provider path/layout tests, alternate-root matrix, Codex state boundaries, production fallback rejection, and migration/backward compatibility. Do not reuse the DB-gated aggregate as evidence for these pure contracts.

## Pure offline acceptance matrix

All tests use direct structs/fakes, `t.TempDir()`, synthetic sentinels, and value-free log capture. They must run without `TestMain`, build tags, PostgreSQL, HTTP/loopback, provider binaries, real env/home reads, or credentials.

| Acceptance ID | Proposed named test | Required assertion | Evidence target |
| --- | --- | --- | --- |
| `CI113-A01` | `TestCredentialAssignmentAcceptsSameTenantProvider` | Fake resolver returns account only for exact agent+tenant; daemon accepts exact provider and complete paths. | `EV-CREDISO-1.1-ASSIGNMENT` |
| `CI113-A02` | `TestCredentialAssignmentRejectsNoAssignmentAndNilResolver` | Both paths fail before execenv; no fallback/home resolution. | same |
| `CI113-A03` | `TestCredentialAssignmentRejectsTenantMismatch` | Cross-tenant account is rejected with generic error; no path logged. | same |
| `CI113-A04` | `TestCredentialAssignmentRejectsProviderMismatch` | Account vendor mismatch fails closed. | same |
| `CI113-A05` | `TestCredentialAssignmentRejectsResolverError` | Arbitrary store error never becomes legacy fallback. | same |
| `CI113-P01` | `TestCredentialAccountPathsRejectOutsideRootTraversalAndSymlink` | Absolute/canonical/contained/no-symlink rules for both HomeDir and ConfigDir. | `EV-CREDISO-1.1-PATHS` |
| `CI113-P02` | `TestCredentialProviderPathContractMatrix` | Exact provider layout and mandatory ConfigDir semantics; unsupported Claude rejects. | same |
| `CI113-E01` | `TestCredentialDiscoveryRootPolicyRejectsAllAlternateRoots` | Table includes every exact key listed above, case-insensitively. | `EV-CREDISO-1.2-ENV` |
| `CI113-E02` | `TestCredentialCustomEnvironmentRejectsProviderBypassFamiliesWithoutValues` | Key names/reasons only; sentinel values absent from errors/logs. | same |
| `CI113-E03` | `TestCredentialEnvironmentRemovesInheritedAndAppliesTrustedRootsLast` | Hostile inherited/custom roots are absent; exactly the task provider's complete roots remain. | same |
| `CI113-E04` | `TestCredentialEnvironmentRejectsCrossProviderProjection` | Prepared state for provider A cannot satisfy provider B; no alternate provider key survives. | same |
| `CI113-C01` | `TestCodexAccountOverlayNeverUsesGlobalConfigSessionsOrPlugins` | Synthetic global sentinels never enter task overlay; account sentinels do. | `EV-CREDISO-1.1-CODEX-BOUNDARY` |
| `CI113-C02` | `TestCodexRefreshWritesBackOnlySameAccountWithCAS` | If write-back is approved: A updates A only; stale generation and B path reject. | same |
| `CI113-F01` | `TestCredentialIsolationProductionRejectsMissingAssignment` | Production/default policy always fails closed. | `EV-CREDISO-1.3-FALLBACK` |
| `CI113-F02` | `TestLegacyGlobalCompatibilityRequiresExplicitManagedAccount` | Option A: an explicitly assigned imported account works; absence never imports/uses global. | same |
| `CI113-F03` | `TestLegacyGlobalDevelopmentModeRejectedOutsideDevelopment` | Option B only: startup/config validation rejects production, rotation, concurrency >1, unsupported provider, and gateway-required combinations. | same |
| `CI113-R01` | `TestCredentialReuseRevalidatesAssignmentTenantAndPaths` | Reuse cannot carry a stale account/root after reassignment or tenant/provider change. | `EV-CREDISO-1.1-REUSE` |

The fake must be constructed directly; do not use the current `newTestDaemon`, which starts an `httptest.Server` (`daemon_test.go:1265-1274`). A narrow resolver fake avoids implementing unrelated rotation methods and makes the test genuinely network/DB-free.

Acceptance requires verbose non-zero proof, ×20, race, vet, normal build-tag inclusion, no `TestMain` skip, source hashes, and a negative proof that the DB aggregate was not used as a substitute.

## Owned file plan and dependency order

No two owners should edit a hotspot concurrently.

| Order | Owner scope | Planned files | Dependency/output |
| --- | --- | --- | --- |
| 0 | Product owner + OpenSpec docs owner | `proposal.md`, `design.md`, `spec.md`, `tasks.md` | Select Option A/B and provider/Codex decisions before source work. |
| 1 | Credential-policy owner | New narrow pure contract package under `server/internal/daemon/credentialisolation/` with tests | Versioned provider/path/env registry; no filesystem secrets or process launch. |
| 2 | Rotation/store owner | `internal/rotation/contract.go`, `store_pg.go`, tests, next migration if required | Atomic tenant-scoped resolver and assignment-write validation. Depends on provider/path contract. |
| 3 | Execenv owner | `execenv/execenv.go`, `codex_home.go`, provider home files and focused tests | Consume validated account paths; implement task overlays and approved Codex boundaries. Depends on 1–2. |
| 4 | Runtimeenv owner | `runtimeenv/policy.go`, `env.go`, `assert.go`, focused tests | Reuse central discovery-key policy and trusted-last assertions. Depends on 1. |
| 5 | Daemon/config owner | `daemon.go`, `config.go`, `cmd/multica/cmd_daemon.go` only if Option B, direct pure tests | Wire resolver, policy, startup validation, launch/reuse gate. Depends on 1–4. |
| 6 | Independent QA owner | Focused pure test files/evidence only | Run `CI113-*`; verify hashes/build tags/TestMain and no DB/network. No product edits unless a separately authorized minimal test correction is required. |

## Migration and backward-compatibility risks

1. Existing account rows may contain arbitrary, relative, symlinked, outside-root, or semantically ambiguous `HomeDir`/`ConfigDir`. Mark invalid/degraded and require explicit migration; never normalize silently into trust.
2. Current Codex users may rely on global `config.toml` provider definitions, instructions, sessions, or plugin cache. Moving to account scope can change model/provider routing and resume behavior. Provide an explicit import preview and content-class inventory without reading/logging secret values.
3. Existing session IDs may point to the global sessions directory. Choose migrate-to-account, invalidate, or retain an account-scoped compatibility copy; never keep a writable global symlink.
4. Task-local Codex token refresh currently does not define authoritative write-back. A policy decision is required to prevent stale next-task credentials or unsafe concurrent overwrites.
5. Kiro and Antigravity variable semantics are installed-version observations. A CLI upgrade can change precedence; version-gate and fail closed when the contract is unknown.
6. The proposal names credential-session REST APIs that do not exist in the Go server. There is no shipped API compatibility contract to preserve, but future API design must use the same tenant/provider/path rules.
7. Option B adds permanent configuration/support burden and creates a tempting production bypass. Its flag must not be accepted by production startup or inherited from unrelated Agent Brain development settings.
8. ExactEnv's same-UID validate-to-exec TOCTOU remains an OS-isolation residual and is not solved by path validation alone.

## Principal/Kiro decisions required

Implementation must stop until the principal answers these questions:

1. **Fallback choice:** Approve **Option A (recommended)**—no implicit fallback, with legacy state imported as an explicit managed account—or require Option B's development-only fallback? Option C is not security-acceptable.
2. **Provider roster:** Is the normative task-1 roster the current production isolation matrix (Codex, Kiro, Antigravity, GLM/OpenCode, Cline, with NIM exception), or must Claude/Gemini CLI be implemented now? Recommended: declare Claude unsupported and treat Antigravity separately from Gemini until binary contracts are proven.
3. **Codex refresh:** Must refreshed `auth.json` be reconciled back to the account store with lock/CAS, or are task credentials read-only snapshots requiring re-enrollment? Recommended: account-scoped CAS write-back.
4. **Codex sessions/plugins:** Approve account-scoped sessions and only audited read-only shared plugin artifacts, or require a different compatibility boundary? Recommended: no writable daemon-global link.
5. **Existing path migration:** May legacy paths be imported by an explicit operator action, or must existing rows be grandfathered temporarily? Recommended: explicit import; invalid rows fail closed.
6. **NIM semantics:** Keep NIM in credential-isolation tasks despite its value/reference model, or split it into a separate native-secret contract? Recommended: split the exception so `ConfigDir` remains a filesystem-root concept.

## Source pins and non-claims

Critical current source SHA-256 values (rechecked for this decision):

```text
a1d96a3c8a4edfc9f9c583326c6f74779643473904f13f0a8aab63ea1cd6fe07  internal/daemon/daemon.go
8099867594f18f6d5e5c47fa88581f06ac771e456f203fc0161d09c6584fc0e6  internal/daemon/execenv/execenv.go
aa3f8bbd82ffb3fe70ed72da037eaae3028af4b558a7b1d0c207ccaeb4cb3fe0  internal/daemon/execenv/codex_home.go
80d3d990c470ad9e7a21d661d51553bd37690edc13e16c7e95246ebe211df834  internal/daemon/runtimeenv/policy.go
eef92d45127137f5f339a430490c7b188f5704c9ebf55b545a54fc0c3c85fd4e  internal/rotation/contract.go
e47757784ba22d9c4d749c51028247523a6d8cfd3db002343a503d5fba67edf8  internal/rotation/store_pg.go
```

No claim is made that any proposed API, migration, resolver, policy package, path validation, fallback mode, Codex write-back, or acceptance test exists. No tests were run because this pack is docs-only. No provider-native behavior was revalidated. No credential/session API compatibility is claimed; the existing session-API audit says those Go routes are absent. ExactEnv evidence remains accepted only for its documented point-in-time controls and retains the same-UID TOCTOU residual.
