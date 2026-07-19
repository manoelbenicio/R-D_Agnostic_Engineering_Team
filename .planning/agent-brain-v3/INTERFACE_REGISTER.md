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
