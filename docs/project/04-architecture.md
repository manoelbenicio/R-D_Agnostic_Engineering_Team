# 04 — Arquitetura

**Data:** 2026-07-01

---

## 1. Componentes (e responsabilidade)

| Componente | Papel | Toca nesta mudança? |
|------------|-------|---------------------|
| Daemon (`daemon.go`) | prepara ambiente por tarefa; injeta env vars | **Sim** (injeção de env por vendor) |
| execenv (`codex_home.go` etc.) | prepara home/credencial por tarefa | **Sim** (fonte da credencial por conta) |
| Agent backends (`pkg/agent/*`) | invocam o CLI de cada vendor | Leitura (contrato de env) |
| Orquestração / canvas / dispatch | "cérebro" do produto | **Não** (intocado) |
| Store de credencial | armazena/serve credencial por conta (AS-IS) | **Sim** (Postgres) |
| Rotação (Fase 2) | detecção + troca + retomada | **Sim** (Fase 2) |
| Observabilidade | métricas + dashboards | **Sim** (doc 05) |

## 2. Diagrama lógico (Fase 1)

```mermaid
flowchart TD
    T[Tarefa atribuída a um agente] --> R[Resolver conta do agente\n(mapa de atribuição)]
    R --> P[Preparar credencial isolada da conta\n(restore AS-IS, dir 0700)]
    P --> I[Injetar env var nativa do vendor\n(daemon.go agentEnv)]
    I --> C[CLI do agente lê a credencial da SUA conta]
    C --> X[Execução sem sobreposição de contas]

    subgraph Persistência
      DB[(Postgres\nseats/sessions/accounts)]
    end
    R -. lê atribuição .-> DB
    P -. lê credencial AS-IS .-> DB
```

## 3. Diagrama lógico (Fase 2 — rotação)

```mermaid
flowchart TD
    M[Monitor do agente] --> D{Esgotou cota?\nregex tela / 429 / ledger}
    D -- não --> M
    D -- sim --> L[Lock do agente + snapshot da tarefa]
    L --> S[Selecionar próxima conta\n(prioridade por expertise)]
    S --> Q{Há conta disponível?}
    Q -- não --> W[Park: agendar wake em min(cooldown_until) + alerta]
    Q -- sim --> RC[Restaurar credencial AS-IS da nova conta]
    RC --> I2[Injetar env] --> RES[Retomar tarefa do checkpoint]
    RES --> EV[Registrar evento de rotação\n(métricas)]
    EV --> M
```

## 4. Contrato de env por vendor (autoritativo)

| Vendor | Env var(s) | Conteúdo |
|--------|-----------|----------|
| Codex | `CODEX_HOME` | dir da conta contendo `auth.json` restaurado |
| Kiro | `XDG_DATA_HOME` (sessão) **ou** `KIRO_API_KEY` (headless) | dir por conta / chave `ksk_...` |
| Antigravity | `HOME` | aponta para `~/.gemini/antigravity-cli` da conta |

## 5. Modelo de dados (Postgres — mínimo)

> Reaproveita o conceito de seats/sessions já pesquisado; nomes agnósticos.

- `accounts(account_id, vendor, tenant_id, priority, status, window_start,
  tokens_used, cooldown_until, ...)`
- `credentials(account_id, vendor, blob_ref, format, refresh_token_present,
  created_at, expires_at)` — blob AS-IS via KMS/secret ref, **nunca** em log.
- `assignments(agent_id, account_id, assigned_at)` — mapa agente → conta.
- `rotation_events(id, agent_id, from_account, to_account, reason, ts)` (Fase 2).

Todas as tabelas em Postgres (RNF-04). Acesso via pool de conexões.

## 6. Pontos de decisão de design

- **Symlink → fonte por conta:** hoje `auth.json` é symlink para o global; passa a
  apontar/copiar a credencial da conta atribuída (cópia AS-IS evita contaminação de
  refresh entre contas).
- **Injeção única:** todo env por vendor entra no mesmo ponto (`daemon.go agentEnv`),
  mantendo a mudança localizada.
- **Fallback:** ausência de atribuição → caminho global atual (compatibilidade).

## 7. Riscos residuais (ver design.md do change para a lista completa)

- Divergência de env var por vendor (Kiro ignora `KIRO_HOME`, usa `XDG_DATA_HOME`) —
  mitigado pela tabela §4, validada empiricamente.
- `CODEX_HOME` inexistente causa erro fatal → `mkdir -p` antes do spawn.
- Modelo de custo: API-key (metered) vs sessão OAuth (5h flat) — decisão do dono.