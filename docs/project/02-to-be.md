# 02 — Estado Alvo (TO-BE)

**Data:** 2026-07-01

---

## 1. Visão do estado alvo

Cada conta de vendor vive **isolada** (diretório/credencial própria). Cada agente é
apontado para a conta atribuída via a **env var nativa** do seu CLI, no ponto de
injeção já existente do daemon. Logar/renovar uma conta **não afeta** as outras.
Na Fase 2, ao esgotar a cota (~5h), o sistema **troca de conta automaticamente** e
retoma a tarefa.

## 2. Mecanismo de isolamento por vendor (confirmado por pesquisa)

| Vendor | Alavanca de isolamento | Restauração AS-IS |
|--------|------------------------|-------------------|
| **Codex** | `CODEX_HOME` → diretório da conta (com `auth.json` da conta) | escreve `auth.json` bruto |
| **Kiro** | `XDG_DATA_HOME` → dir próprio (sqlite por conta) **ou** `KIRO_API_KEY` | sessão sqlite ou env key |
| **Antigravity** | `HOME` → `~/.gemini/antigravity-cli` da conta | escreve token bruto |

Regra transversal: **armazenar/restaurar exatamente o formato do vendor**, incluindo
o `refresh_token`; não guardar credencial já usada/expirada.

## 3. Fluxo TO-BE (Fase 1 — isolamento)

```
Tarefa é atribuída a um agente
      │
      ▼
Resolver conta do agente (config de atribuição)
      │
      ▼
Preparar credencial isolada da conta (dir próprio, restore AS-IS)
      │
      ▼
Injetar a env var nativa do vendor no agentEnv (daemon.go)
      │
      ▼
CLI do agente lê a credencial da SUA conta → sem sobreposição
```

## 4. Fluxo TO-BE (Fase 2 — rotação automática)

```
Detecção de esgotamento (regex na tela do vendor + HTTP 429 / ledger de cota)
      │
      ▼
Lock do agente → snapshot da tarefa (prompt + checkpoint)
      │
      ▼
Selecionar próxima conta disponível (prioridade por expertise)
      │
      ▼
Restaurar credencial da nova conta (AS-IS) → injetar env
      │
      ▼
Retomar tarefa (re-dispatch a partir do checkpoint)
      │
      ▼
Registrar evento de rotação (observabilidade)
```
> Compartilhamento concorrente é permitido: uma credencial válida pode servir
> N agentes durante seu ciclo (~5h). Não há lease de conta única.

## 5. Superfície de mudança (cirúrgica)

| Arquivo | Mudança |
|---------|---------|
| `execenv/codex_home.go` | trocar o symlink global do `auth.json` pela fonte da conta atribuída |
| `daemon.go` (~l.3380) | estender a injeção de env para as demais env vars por vendor |
| config de atribuição | mapa agente/conta (superfície mínima; sem mexer no cérebro) |

Fora de escopo: orquestração, canvas, dispatch, UI — **intocados**.

## 6. Diferença AS-IS → TO-BE (resumo)

| Dimensão | AS-IS | TO-BE |
|----------|-------|-------|
| Credencial por vendor | 1 global compartilhada | N isoladas por conta |
| Sobreposição ao logar | sim (sobrescreve) | não |
| Troca ao esgotar 5h | manual | automática (Fase 2) |
| Capacidade ociosa | desperdiçada | aproveitada via rotação |
| Persistência nova | — | Postgres-only |
| Observabilidade de cota | inexistente | Grafana/Prometheus (doc 05) |

## 7. Critérios de aceite (alto nível)

- AC1: duas contas do mesmo vendor coexistem sem sobreposição.
- AC2: cada agente usa a credencial da sua conta atribuída.
- AC3: fallback preservado — sem atribuição, comportamento global atual.
- AC4: nenhum segredo em log.
- AC5 (Fase 2): ao esgotar, o agente troca de conta e retoma sem intervenção.
- AC6: build verde no container; sem regressão nas suítes tocadas.