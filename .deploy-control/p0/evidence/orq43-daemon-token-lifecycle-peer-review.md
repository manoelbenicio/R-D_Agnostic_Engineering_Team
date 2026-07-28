# ORQ-43 — Peer Review Adversarial de Governança: Desenho de Ciclo de Vida e Rotação do Token `mdt_` do Daemon

- **Autor do Peer Review:** Antigravity (wB:p1 / w8:p2)
- **Documento Auditado:** `.deploy-control/p0/evidence/orq43-daemon-token-mdt-lifecycle-design.md` por Codex56#A (`w7:p3`)
- **Data UTC:** 2026-07-27T15:37:00Z
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1)
- **Governança:** Issue `ORQ-43` (`d1149dd3-9da8-4678-a4b7-d98f3eddca14`)
- **Skill Carregada:** `.agents/skills/aws-secrets-manager/SKILL.md` (zero texto claro, zero `GetSecretValue`, segredos via `asm-exec` no processo filho).
- **Modo:** PEER REVIEW ADVERSARIAL READ-ONLY — Zero edição de código, zero geração de token, zero mutações de DB/AWS/container/board.

---

## 1. Veredito Final
- **VEREDITO: PASS (DESENHO DE CICLO DE VIDA DO TOKEN `mdt_` APROVADO)**
- O documento `.deploy-control/p0/evidence/orq43-daemon-token-mdt-lifecycle-design.md` reflete de forma factual o estado real da base de código, distingue corretamente validadores de emissores, exige a divisão obrigatória entre **Fase 1 (Habilitação/Emissão `ORQ-43A`)** e **Fase 2 (Rotação `ORQ-43B`)**, impõe o gate de fila zero em 4 estados e garante a conformidade com a skill `aws-secrets-manager`.

---

## 2. Auditoria Adversarial das Descobertas Críticas e Travas de Governança

### 2.1 Confirmação Factual da Descoberta Crítica (Emissão Desconectada)
- **Busca Independente no Código**:
  - `internal/auth/jwt.go:70-76` `GenerateDaemonToken()` produz `"mdt_"+40 hex`, mas possui **ZERO chamadores** fora do próprio arquivo de testes.
  - `pkg/db/queries/daemon_token.sql:1` `CreateDaemonToken` possui **ZERO chamadores** fora do código gerado pelo `sqlc`.
  - Não existe cliente CLI (`cmd/multica`) ou handler de produção populando a tabela `daemon_token` ou persistindo o token `mdt_` no cliente do daemon.
- **Divisão Obrigatória do Projeto**:
  - **Fase 1 (`ORQ-43A` - Habilitação)**: Criar rota de emissão/renovação de `mdt_`, armazenamento no cliente do daemon e expurgo automatizado de expirados.
  - **Fase 2 (`ORQ-43B` - Rotação por Sobreposição)**: Execução da rotação com janelas de sobreposição e invalidação de cache.
  - *Nota*: Executar a rotação antes da Fase 1 causaria disrupção total (revogaria a credencial sem capacidade de emitir nova).

### 2.2 Gate de Fila Ativa em 4 Estados
- A rotação exige estritamente a verificação de zero tarefas em 4 estados cruciais em duas leituras consecutivas:
  ```sql
  SELECT count(*) FROM agent_task_queue
  WHERE status IN ('queued','dispatched','running','waiting_local_directory');
  ```
- A inclusão de `waiting_local_directory` impede interrupções prematuras durante o provisionamento de diretórios locais pelo daemon.

### 2.3 Semântica de Reconexão e Janela de Sobreposição (Dual-Token Overlap)
- Como a tabela `daemon_token` possui índice único em `token_hash` e índice **não-único** em `(workspace_id, daemon_id)`, o sistema suporta nativamente múltiplos tokens válidos simultâneos para o mesmo daemon.
- Durante a janela de sobreposição (S1->S5), o token antigo permanece válido enquanto o daemon adota o token novo, garantindo **downtime zero e zero desconexão não planejada**.

### 2.4 Invalidação Obrigatória de Cache Redis
- O validador utiliza `DaemonTokenCache` com TTL clampado (`TTLForExpiry`).
- Para evitar que um token revogado continue autenticando por cache de memória, a etapa de revogação (S5) exige chamadas explícitas a `DaemonTokenCache.Invalidate(hash)` para cada item em `RevokedTokenHashes`.

### 2.5 Limites de Escopo e Exclusão Estrita
- Exclusão expressa de tokens de tarefa (`mat_` / `MULTICA_TOKEN`, escopo da `ORQ-36`) e PATs (`mul_` / `mcn_`).

---

## 3. Conformidade com a Skill `aws-secrets-manager`

- **Zero Texto Claro**: O runbook proíbe categoricamente qualquer impressão de segredos em stdout, logs, argumentos de linha de comando ou documentos de evidência.
- **Resolução via `asm-exec`**: Toda gravação de credenciais nos hosts de daemon utiliza a resolução por `asm-exec` executada dentro de um diretório privado `0700`.

---

## 4. Veredito Final
- **STATUS: PASS (AUDITORIA ADVERSARIAL DA ORQ-43 CONCLUÍDA COM SUCESSO)**
- **Documento Gravado**: `.deploy-control/p0/evidence/orq43-daemon-token-lifecycle-peer-review.md`
- *Operação 100% Read-Only. Nenhum segredo lido, nenhuma mutação de código, banco, AWS ou container.*
