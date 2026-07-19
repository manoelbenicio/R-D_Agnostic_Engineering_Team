# EVIDENCE_CONTRACT — Agent Brain v3 (o que conta como REAL)

> Absorve EVIDENCE_CONTRACT legado (rejeição de fabricação). Binding on TL + all 4 Codex.

## O que conta
- **Proveniência exata**: comando, host, versão+commit/digest OmniRoute, timestamp UTC, quem rodou. Sem "as" outro agente ou o dono.
- **Development topology**: host/WSL real for tier-20 validation (loopback 127.0.0.1:20128),
  not production and not a fake-upstream; OmniRoute version/digest remains explicit.
- **Per-model per-protocol**: 200 em `/v1/models` NÃO prova protocol fidelity/tools/streaming/rotation. Cada rota aprovada exige conformance.
- **Failure-injection real**: expiry, revoked, quota, 429 (account/global), 5xx, timeout, broken stream (pre/post output), cancel, restart/account add-remove under load — capturado, não descrito.
- **Capacity reproduzível**: mix, stream/tool ratio, prompt/output sizes, p50/p95/p99, filas, CPU/mem/sockets, fairness, fallback — por tier 20/50/100.
- **Kill switch real**: aplicado sob serviço vivo, rotação para/resume — before/after, não descrição.
- **Logs scrubbed**: grep por secret/token → 0 matches mostrado; sem chave real em arquivo.
- **Smart Context**: SC01–SC10 (shadow→controlled dev cohort→exact fallback→self-check)
  OR signed product+security waiver.

## Regras (inegociáveis)
1. Proveniência completa em cada evidence file.
2. Check-in antes de execução (files_locked, plan_ref, status). Evidence não pode predar o check-in/execução.
3. Toda evidence mapeia a task-ID + AB-REQ + acceptance ID (gate de aceite).
4. Nada fabricado: localhost/fake-upstream/smoke/placeholder/ números idênticos cross-vendor/ sign-off forjado = INVALID.
5. Sem segredo/cookie/prompt/repo-content/tool-payload em log/evidence/screenshots.
6. Distinguir reviewed · implemented · verified · accepted. "DONE" só com artifact rastreável.
7. TL valida independentemente; manda re-rodar; só TL commita plano (não código). TL nunca roda prodex/bash de produto.
8. Órfão (req/comp/iface/task/ev sem cadeia) bloqueia a fase.

## Rejection
On violation: marcar `> [!CAUTION] INVALID — <reason>`, reverter status p/ BLOCKED, escalar ao TL/dono. Não apagar histórico da fabricação.

## IDs de evidence
Formato: `EV-<fase>-<nn>`. Registrados em EVIDENCE_INDEX.md (imutável). Nenhum AB-REQ fecha sem evidence ID vinculado.
