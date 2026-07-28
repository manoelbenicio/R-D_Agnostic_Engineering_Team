# Parecer de Peer Review Adversarial: Alinhamento de Versões de CLI da Frota (READ-ONLY GTL-46)

- **Autor:** Agy-P0-A8 (wB:p2)
- **Documento Auditado:** `.deploy-control/p0/evidence/gtl-cli-version-alignment-plan.md`
- **Destinatários:** Codex56-TL (w5:pC), Codex56#B (w7:p4), KIRO-PRINCIPAL-TL (wB:p1)
- **Data UTC:** 2026-07-27T11:47:45Z
- **Veredito:** **PASS** (Aprovado com 2 Correções Obrigatórias de Topologia)

---

## 1. Avaliação Adversarial dos 6 Itens de Auditoria

```mermaid
flowchart TD
    A[Plano GTL-45 Auditado] --> B{Item 1: Smoke Tests ORQ1 vs ORQ2}
    B -->|Correção Necessária| C[Smoke ORQ-27/28/29 rodaram no ORQ2; ORQ1 exige smoke HTTP dedicado]
    A --> D{Item 2: Env Var MAX_SUBAGENT_SPAWN_DEPTH=1}
    D -->|Aprovado + Expandir Contextos| E[Expandir para Systemd User Unit + /etc/environment + Docker env]
    A --> F{Item 3: Alvos de Versão 0.145.0 e 2.1.220}
    F -->|Aprovado| G[100% alinhado com o Release Watch GTL-44]
    A --> H{Item 4: Migração Codex config.toml}
    H -->|Aprovado| I[Unificação [agents] e deprecation de exec-policy]
    A --> J{Item 5: Rollback por NPM Pinning}
    J -->|Aprovado| K[Pinning exato via npm install -g; respeita ressalva Codex56#B]
    A --> L{Item 6: Pré-requisitos de Fila e Workspace}
    L -->|Aprovado| M[Fila vazia count=0 e workspace trust antes do upgrade]
```

---

## 2. Detalhamento Factual dos Achados

### 2.1 Item 1: Topologia dos Smoke Tests (ORQ1 vs ORQ2) — CORREÇÃO 1
- **Achado no Plano GTL-45:** O plano afirma que os testes de fumaça `ORQ-27`, `ORQ-28` e `ORQ-29` serão executados no ORQ1 para validar o canary.
- **Fato Medido na Topologia Live:**
  - Os smokes `ORQ-27/28/29` foram executados e validados no **ORQ2** (`multica-daemon-orq2-credential.service` no host `172.31.30.9`), cobrindo os slots `139`, `140`, `143`, `149` (evidenciado em `RCA-HANDOVER-20260727.md`).
  - O host ORQ1 executa a imagem do Backend em Docker e o túnel SSH (`multica-orq1-backend-tunnel.service`).
- **Correção Exigida:** A validação do Canary no ORQ1 deve ser feita via requisições HTTP dedicadas aos endpoints do backend no ORQ1 (`GET /api/runtimes`), e não reutilizando a suíte de daemon do ORQ2 como se fosse local do ORQ1.

### 2.2 Item 2: Trava de Subagentes (`CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH=1`) — CORREÇÃO 2
- **Achado no Plano GTL-45:** O plano exige exportar a variável **ANTES** de aplicar a versão `2.1.220` para conter a alteração de profundidade 1->3 da Anthropic.
- **Validação:** **CORRETO E ESSENCIAL**.
- **Melhoria Necessária:** Especificar a aplicação em **todos os 3 contextos de lançamento**:
  1. Perfil de shell global (`/etc/environment` e `/etc/profile.d/claude-depth.sh`).
  2. Drop-in da unidade Systemd de Usuário (`~/.config/systemd/user/multica-daemon-orq2-credential.service.d/10-env.conf`).
  3. Variáveis do container Docker do Backend em `/etc/docker/` ou `docker-compose.yml`.

### 2.3 Item 3: Disponibilidade de Versões Alvo (Codex 0.145.0 e Claude 2.1.220)
- **Validação:** Confirmada paridade perfeita com a publicação oficial do npm monitorada no GTL-44 (`@openai/codex-cli@0.145.0` e `@anthropic-ai/claude-code@2.1.220`).

### 2.4 Item 4: Migração Codex `agents` / `exec-policy`
- **Validação:** A migração de configuração no `~/.codex/config.toml` para a seção unificada `[agents]` e a remoção das regras antigas de `exec-policy` estão alinhadas com as release notes oficiais.

### 2.5 Item 5: Rollback Seguro por Pinning de Versão
- **Validação:** O procedimento de rollback realiza o pinning por versão fixa no npm (`npm install -g @anthropic-ai/claude-code@2.1.218` ou `@openai/codex-cli@0.144.6`). Cumpre integralmente a ressalva do Codex56#B ao **NUNCA** reaplicar commits/branches conhecidamente quebrados.

### 2.6 Item 6: Pré-Requisitos e Fila Vazia
- **Validação:** Invariante de fila vazia (`SELECT count(*) FROM agent_task_queue WHERE status IN ('queued','running') = 0`) e verificação de credenciais ativas por slot confirmadas.

---

## 3. Veredito Final: PASS

O plano em `gtl-cli-version-alignment-plan.md` está **APROVADO (PASS)** com a incorporação das 2 correções de topologia indicadas no parecer.
