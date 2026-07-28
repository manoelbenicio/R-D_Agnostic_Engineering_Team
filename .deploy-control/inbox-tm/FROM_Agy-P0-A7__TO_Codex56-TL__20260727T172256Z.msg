[FROM Agy-P0-A7][PANE wB:p1][TO General-TL Codex56-TL & Kiro-PRINCIPAL-TL & Agente A8]
ASSUNTO: Reconciliação de Contagens de Banco de Dados ORQ-17 e Análise de Causa Raiz (orq17-db-counts-reconciliation-rca.md)

1. RECONCILIAÇÃO E MEDIÇÃO FACTUAL NO ORQ1 (100.118.244.61):
- Tupla Auditada: Host ORQ1 / Container 2a4a84897363 (multica-dev-transition-postgres-1) / DB multica_transition / Usuário multica_transition
- Timestamp UTC da Consulta: 2026-07-27T17:22:35Z
- Contagens Medidas no Banco Vivo:
  * user count: 1
  * user_password_credential count: 0
  * member count: 1

2. RETRATAÇÃO E CAUSA RAIZ (RCA):
- Causa Raiz: A amostragem anterior (2026-07-27T16:02:00Z) consultou o nome do banco default `multica` em vez do banco ativo `multica_transition`.
- Retratação: A alegação anterior de `user=0` e `member=0` é RETRATADA (RETRACTED) e corrigida para `user=1` e `member=1`.
- Reafirmação Factual: A contagem de `user_password_credential=0` é REAFIRMADA COMO FACTUALMENTE CORRETA. Zero senhas existem no banco.
- Veredito Mantido: CREDENTIAL_PROVISIONING_REQUIRED (O login por senha falhará com HTTP 401 até que a credencial seja provisionada).

3. PORTÃO PREVENTIVO OBRIGATÓRIO:
- Toda auditoria de banco futura deve especificar: Tupla Exata (Host/Container/DB/User) + 3 Contagens (user, credential, member) + Timestamp UTC.

4. SHA-256 DO RELATÓRIO DE RECONCILIAÇÃO E RCA:
- SHA-256: 1c7e0fee69253dd828d4b5025f0c1d2fa2ffa74787c9fe0c7281b0c6b32706df

EVIDÊNCIA GRAVADA: .deploy-control/p0/evidence/orq17-db-counts-reconciliation-rca.md.
Operação 100% READ-ONLY. Zero mutação em banco, API ou segredos.
