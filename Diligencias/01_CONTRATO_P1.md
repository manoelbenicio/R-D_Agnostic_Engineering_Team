# P1 — Diligência: Contrato Go↔L2 (rpp.l2.v1)

## Objetivo
Definir o contrato local versionado entre Multica Go (control plane) e o runtime prodex/Rust (L2),
com o invariante de **roteador único por sessão**.

## REQ-IDs
REQ-04 (contrato + schema + single-router). Spec: `specs/l2-runtime-contract/spec.md`.

## Pré-requisitos
- P0 verde (binário resolvível).

## Passos
- 1.1 Definir operações: `HealthCheck`, `ApplyPolicy`, `RegisterAccounts`, `StartSession`,
  `StopSession`, `RouteDecisionEvent`, `RuntimeEventStream`, `KillSwitch`.
- 1.2 Schema de eventos em **JSON Schema Draft 2020-12**, versionado como `rpp.l2.v1`.
  - Atenção ao bug já corrigido: `tenant_id`/`session_id` **condicionais** (obrigatórios só em eventos
    session-scoped; eventos globais como `sidecar_started` não exigem) via `allOf if/then`.
- 1.3 Especificar o invariante **roteador único por sessão** de forma testável: Go empurra desired-state;
  Rust roteia o request em voo; Go **não** roteia mid-flight.

## Verificação / evidência
- Schema **compila** (validador Draft 2020-12).
- Contrato revisado; sem segredo.
- Fixture de evento válida/ inválida (positiva/negativa).

## Critério de GATE (DONE)
✅ Schema compila · ✅ contrato `rpp.l2.v1` versionado · ✅ invariante single-router especificado e testável.

## Fronteira (alvo do fork, não agora)
Sidecar local HTTP/gRPC-like JSON sobre loopback, bearer efêmero de alta entropia, schema versionado,
health/readiness. **Não** FFI, **não** subprocesso-por-request. (Spec de segurança do bearer/porta é item do marco de fork.)