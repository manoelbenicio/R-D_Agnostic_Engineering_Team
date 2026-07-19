# INTERFACE_REGISTER — Agent Brain v3

> Toda integração: endpoint, auth, message format, timeout, retry, cancellation, observability, owner, compat.

G1 freeze: `agent-brain.v1` em `multica-auth-work/server/internal/daemon/brain`.
Consumidores Codex 2–4 implementam contra esse pacote; mudanças exigem revisão do Codex1.
G2 packages are complete in isolation; none of the interfaces below is considered active-path
until Codex1 finishes G3 wiring and development isolation evidence.

| Interface | Contrato | Auth | Timeout/retry | Cancel | Obs | Owner | Compat |
|---|---|---|---|---|---|---|---|
| Brain↔Control API | task assign + CLIKind + RouteModel + policy | internal | n/a (control) | task cancel | redacted events | Codex1 | facade p/ legado |
| Brain↔OmniRoute readiness | GET /v1/models, liveness/readiness | Bearer OmniRoute (scoped) | bounded | n/a | route-health redacted | Codex2 | — |
| Claude Code ↔ OmniRoute | POST /v1/messages (Anthropic + SSE) | ANTHROPIC_AUTH_TOKEN=OmniRoute key | per route | HTTP cancel | correlation headers | Codex3 | root URL sem /v1 duplicado |
| Codex ↔ OmniRoute | POST /v1/responses (Responses + SSE) | custom provider key=OmniRoute | per route; HTTP/SSE | HTTP cancel | X-Session/Request-Id | Codex3 | wire_api=responses; sem auth.json |
| Kimi/GLM/NVIDIA ↔ OmniRoute | POST /v1/chat/completions (ou Responses) | OmniRoute key | per route | HTTP cancel | usage redacted | Codex3 | proven endpoint/model |
| Antigravity ↔ OmniRoute | /v1/antigravity (ou fallback agy/ via Claude/Codex) | OmniRoute key | per route | HTTP cancel | — | Codex3 | native OR documented fallback |
| Secret injection | Linux restricted secret file (ref only) | file perm restricted | n/a | n/a | never log value | Codex4 | origem Windows não é injetada direta |
| Telemetry/events | structured redacted metrics/log/events | n/a | n/a | n/a | correlation IDs; no secrets/content | Codex4 | schema-versioned |
| Env vars (neutral) | CLIKind/RouteModel/RouterOwner/gateway-required | n/a | n/a | n/a | trusted applied last | Codex1 + Codex3 | legacy aliases bounded |

## Contratos Go congelados

- `TaskRequest`, `TaskResult`, `Correlation`
- `TaskExecutor`, `RuntimeRegistry`, `GatewayReadinessChecker`, `ModelCapabilityRegistry`, `ResultSink`
- `CLIKind`, `RouteModel`, `RouterOwner`, `ProtocolFamily`
- `Config`, `GatewayConfig`, `ReadinessPolicy`, `SecretFileRef`, `CapacityTier`
- `TranslateLegacyTask`, `TaskToken`, `LegacyUseRecorder`
## Endpoints topologia (BCO spec)
- Host/WSL daemon (atual): `http://127.0.0.1:20128` (loopback). Codex custom provider: `http://127.0.0.1:20128/v1`.
- Container em `multica_default`: `http://omniroute:20128` (Docker DNS). Exposição LAN `192.168.1.27:20128` (portproxy + firewall `192.168.1.0/24`).
- Endpoint = runtime topology, não hard-coded universal; containeriza exige decidir DNS/gateway host.

## Variáveis de ambiente (deny/allow)
- DENY no child (gateway-required): ANTHROPIC_API_KEY, OPENAI_API_KEY, GOOGLE_*_API_KEY, KIMI_*, NVIDIA_API_KEY (diretos), provider base URL overrides, OAuth cookies.
- ALLOW/TRUSTED (applied last): ANTHROPIC_BASE_URL=OmniRoute root, ANTHROPIC_AUTH_TOKEN=OmniRoute key, ANTHROPIC_BASE_URL/Claude markers controlados, OmniRoute provider p/ Codex.
## Headers de correlação
- X-Session-Id, X-Request-Id, task/session/request IDs (Brain cria); OmniRoute retorna request ID + actual model/route + pseudonymous account + selection reason + retries/fallback + quota/circuit safe.

## Correlação E2E de 8 hops (Wave A, D-V3-17 / AB-REQ-39/40)
Schema versionado metadata-only; `secrets_present=false`; sem prompts/tool payloads/repo content/reasoning/secrets/cookies/account emails/connection strings.

| Hop | Span | IDs (join keys) | Campos metadata-only | Owner |
|---|---|---|---|---|
| 1 ingress API | `ingress` | `request_id`→`task_id` | método/rota, principal pseudônimo, status, latência | W6 |
| 2 DB queue | `queue` | `queue_msg_id`↔`task_id` | enqueue/dequeue ts, depth, wait ms | W7 |
| 3 daemon admission | `admission` | `task_id`,`session_id`,`launch_id` | decisão admissão, readiness, CLIKind/RouteModel labels, fail-closed class | W1 |
| 4 CLI process | `cli` | `launch_id`,`proc_id` | launch/exit, exit code, cancel, argv redigido estruturalmente | W3 |
| 5 OmniRoute/provider | `route` | `request_id`↔`omni_request_id` | route/model real, account/connection pseudônimo, selection reason, retries/fallback, quota/circuit, usage safe | W2 |
| 6 terminal persistence | `persist` | `task_id`,`result_id` | persist latency, byte/token counts, status | W7 |
| 7 WS/UI delivery | `delivery` | `session_id`,`delivery_id` | delivery latency, backpressure/drops, reconnects | W6 |
| 8 trace assembly | `trace` | todos acima | gap/orphan detection, hop-completeness, um trace por task | W5 |

Carriers: headers/metadata internos (alinhados a task 4.6 + AB-REQ-21). Lib: `observability/e2e` (W5), chamada pelos donos de cada hop. Gate: G4-OBS bloqueante (OBS-1..OBS-11).

## Interface: máquina de estados de recovery da plataforma (Wave A, D-V3-16 / AB-REQ-41)
- Estados: `NORMAL` (router_owner=omniroute; Prodex OFF), `DEGRADED` (OmniRoute not ready → fail-closed queue/reject; router=none; NUNCA auto-promove Prodex), `RECOVERY` (router_owner=rust_l2 Prodex cold; OmniRoute quiesced; mutuamente exclusivo).
- Transições (só em session boundary, nunca mid-flight): `NORMAL→DEGRADED` em OmniRoute-not-ready; `DEGRADED→RECOVERY` só via `ENABLE_RECOVERY` explícito+autorizado, exige OmniRoute quiesced; `RECOVERY→NORMAL` via `RESTORE`, exige Prodex drained.
- Invariante: exatamente um router owner por sessão; default OFF; sem mutação de credencial nas transições (PD-08); cada transição emite evento OBS metadata. Wire no único select `health.go:177-184`. Owner: W1.
