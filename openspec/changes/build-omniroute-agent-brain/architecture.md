# Agent Brain and OmniRoute Architecture

## Scope

These diagrams describe the discovered runtime today and the target boundary. They intentionally distinguish the Docker backend from the host/WSL daemon that actually launches coding-agent CLIs.

## AS-IS: discovered current runtime

```mermaid
flowchart LR
    U[Operator / Web UI] --> API[Multica backend containers\nAPI + task state]
    API <--> DB[(Postgres)]
    API <-->|daemon API / task stream| D[Multica host/WSL daemon\ncurrent executable]

    subgraph HOST[Host / WSL execution runtime]
        D --> TL[Task lifecycle, workspace,\ncancellation, streaming]
        D --> LR[Legacy Go account selection\nand rotation paths]
        D --> CE[Credential account-home resolution\nand execenv preparation]
        D -. partial / legacy hooks .-> PX[Prodex / Rust L2 code path\nnot accepted as final architecture]
        CE --> H1[Per-task Claude/Codex/Kimi/\nAgy/Kiro/NIM homes]
        H1 --> C[Installed coding-agent CLIs]
    end

    C -->|provider-native credentials\nmay be inherited or copied| P[Anthropic / OpenAI / Google /\nKimi / NVIDIA providers]

    OR[OmniRoute Docker container\n127.0.0.1:20128\naccounts and model routes exist]
    OR --> P
    D -. health/network reachability only;\nagent traffic not yet enforced .-> OR

    classDef risk fill:#4b1f24,stroke:#ff6b6b,color:#fff;
    class LR,CE,H1,PX risk;
```

### Current-state facts and risks

- The active daemon runs on the WSL/host and launches host-installed CLIs. The `multica-backend-1` container is not the CLI runtime.
- For this daemon, OmniRoute is reached through `http://127.0.0.1:20128`; Docker DNS `http://omniroute:20128` applies only to containers on `multica_default`.
- OmniRoute is alive and has account pools/model routes, but the active daemon environment does not yet enforce OmniRoute base URLs or the single stable key.
- Existing execution code can copy native credentials into task homes and can inherit provider secrets from the daemon environment.
- Custom agent environment settings can currently override routing/authentication variables unless gateway-required mode applies a denylist and injects trusted values last.
- Prodex/L2 and legacy rotation hooks remain in the source, but they are superseded by the target design and must not remain an alternative credential/router owner.

## TO-BE: brand-neutral Agent Brain plus OmniRoute

```mermaid
flowchart LR
    U[Operator / Product UI] --> CP[Brand-neutral control API\nprojects, tasks, policy]
    CP <--> DB[(Task / workspace / audit data)]
    CP <-->|neutral daemon contract| B[Agent Brain daemon\nCOLD CONTROL PLANE]

    subgraph BRAIN[Agent Brain responsibilities]
        B --> O[Task orchestration and admission\n20 / 50 / 100 capacity tiers]
        B --> W[Workspace, process lifecycle,\ncancellation, result streaming]
        B --> M[CLIKind + RouteModel policy\nmodel/capability validation]
        B --> E[Credentialless task environment\ndeny provider secrets]
        B --> H[OmniRoute health/readiness gate\nrequest/session correlation]
    end

    E --> CC[Claude Code adapter\nAnthropic Messages + SSE]
    E --> CX[Codex adapter\nOpenAI Responses + SSE]
    E --> OA[OpenAI-compatible adapter\nKimi / GLM / NVIDIA]
    E --> AG[Antigravity adapter\ndirect endpoint if supported;\nClaude/Codex fallback otherwise]

    CC -->|/v1/messages| OR[OmniRoute\nHOT DATA PLANE]
    CX -->|/v1/responses| OR
    OA -->|/v1/chat/completions\nor approved Responses path| OR
    AG -->|documented Antigravity or\ncompatible protocol| OR

    subgraph HOT[OmniRoute exclusive responsibilities]
        OR --> A[Stable tenant-key auth\nprovider credential vault]
        OR --> R[Atomic model routes and account pools\nstrict RR for independent requests]
        OR --> F[Continuation affinity\npre-commit retry/fallback]
        OR --> Q[Token refresh, expiry, quota,\nsubscription/reset lifecycle]
        OR --> CB[429 classification, cooldown,\ncircuit breakers, provider fallback]
        OR --> SC[Smart Context/token saving\nshadow, canary, exact fallback]
        OR --> OT[Protocol translation, streaming,\ntool integrity, usage, telemetry]
    end

    OR --> PA[Provider account pools\nAnthropic / OpenAI / Google /\nKimi / NVIDIA / others]

    ADM[Restricted OmniRoute administration\naccounts, routes, keys, kill switches] --> OR
    OBS[Metrics / logs / traces / alerts\nno secrets or raw content] <-->|correlation IDs| B
    OBS <-->|route/account evidence| OR

    classDef cold fill:#17324d,stroke:#63b3ed,color:#fff;
    classDef hot fill:#4a2c17,stroke:#f6ad55,color:#fff;
    class B,O,W,M,E,H cold;
    class OR,A,R,F,Q,CB,SC,OT hot;
```

## Target responsibility boundary

| Responsibility | Agent Brain | OmniRoute |
|---|---|---|
| Product tasks, workspaces, repositories, processes | Owns | Does not own |
| Task admission and 20/50/100 concurrency tiers | Owns task-level admission | Owns inference/account-level limits and queues |
| CLI installation, launch, cancellation, stdout/event parsing | Owns | Does not own |
| Provider/model selection intent | Sends approved route/model ID | Validates and resolves route/model |
| Provider credentials and subscriptions | Must never possess | Exclusive owner |
| Account selection and strict round-robin | Must not duplicate | Exclusive owner |
| Continuation/account affinity | Supplies opaque session/continuation IDs | Enforces required hot-path affinity |
| Token refresh, expiry, quota, reset/redeem | Observes safe status only | Exclusive owner |
| 429/5xx retry, circuit breaker, account/provider fallback | Sets policy/deadline; no account retry | Executes bounded pre-commit policy |
| Protocol translation and SSE/tool fidelity | Configures correct CLI adapter | Preserves or explicitly rejects capability |
| Smart Context/token saving | Chooses policy/kill switch only | Computes, validates, and performs hot-path optimization |
| Provider usage/cost evidence | Aggregates by task/project | Produces redacted route/model/account usage evidence |
| Secrets in logs/traces | Redacts stable key and content | Redacts stable/provider secrets and content |
| Product kill switch | Can stop tasks or disable route policy | Stops route/account/model before next request |

## Required request flow

```mermaid
sequenceDiagram
    participant CP as Control API
    participant B as Agent Brain
    participant CLI as Coding-agent CLI
    participant O as OmniRoute
    participant P as Provider account

    CP->>B: Assign task + CLIKind + RouteModel + policy
    B->>O: Readiness/model capability check
    O-->>B: Ready + approved capability metadata
    B->>CLI: Launch isolated task env with one OmniRoute key
    CLI->>O: Model request + task/session/request IDs
    O->>O: Select eligible account / preserve continuation affinity
    O->>P: Dispatch using provider credential
    alt expired token, quota, 429, or safe pre-commit failure
        P-->>O: Classified failure
        O->>O: Refresh, cooldown/circuit, bounded fallback
        O->>P: Retry on eligible account/provider per policy
    end
    P-->>O: Stream / tools / usage
    O-->>CLI: Protocol-faithful stream + actual-route telemetry
    CLI-->>B: Agent events/result
    B-->>CP: Task state/result and redacted evidence
```

## Non-negotiable cutover rules

1. The Agent Brain must fail closed when OmniRoute is not ready; it must not fall back to direct provider credentials.
2. Trusted gateway configuration is injected after user/custom environment processing, and provider-native keys/base URLs are denied.
3. Strict round-robin applies to new independent requests. Stateful continuations are explicitly affinitized; SSE chunks and internal retries never advance rotation as new logical requests.
4. Rotation policy never creates a global one-request-at-a-time limit. Account/model/global concurrency and admission are separately configurable.
5. No mid-stream replay after partial model output or a potentially non-idempotent tool action.
6. Prodex is removable only after the feature-parity matrix and OmniRoute acceptance checklist are complete with evidence.
7. Legacy Multica names remain only in a time-bounded compatibility facade until API, CLI, stored configuration, and UI consumers migrate.
