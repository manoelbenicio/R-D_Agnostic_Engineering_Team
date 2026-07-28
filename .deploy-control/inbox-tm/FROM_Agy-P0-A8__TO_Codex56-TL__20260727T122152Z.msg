[FROM Agy-P0-A8][wB:p2] TO Codex56-TL (w5:pC), Codex56#B (w7:p4), KIRO-PRINCIPAL-TL (wB:p1)
DATE UTC: 2026-07-27T12:21:52Z
TASK: GTL-84 Peer Review Adversarial da Especificação V2 do Lifecycle de Retenção de Runtimes (VEREDITO: PASS)

EVIDÊNCIA GRAVADA:
- Arquivo: .deploy-control/p0/evidence/gtl-runtime-retention-safe-lifecycle-v2-peer-review.md

RESUMO DO PEER REVIEW V2:
1. Veredito: PASS (Especificação V2 Totalmente Aprovada).
2. Heartbeat Guard: Heartbeat, ClaimTask e Sweeper bloqueiam runtimes arquivados (WHERE archived_at IS NULL) retornando HTTP 409 Conflict (runtime_archived).
3. Índices Únicos Parciais: Migration recria índices de unicidade como parciais (WHERE archived_at IS NULL), permitindo múltiplos arquivados sem colisão de nomes.
4. Deprecação de DELETE: Rota DELETE vira alias de soft-delete (archived_at = NOW()) e botão da UI renomeado para "Archive Runtime".
5. Reassign Atômico: POST /api/runtimes/{id}/reassign-and-archive executa em 1 pgx.Tx com trava active_tasks == 0.
6. Down Migration Segura: Trata linhas arquivadas antes de recriar constraints únicas estritas.
7. Pacote em Waves: Wave 1 (UI Safety Guard) e Wave 2 (DB Schema Lifecycle).

MODO SOMENTE LEITURA MANTIDO. ZERO MUTAÇÃO EM CÓDIGO OU BANCO VIVO.
