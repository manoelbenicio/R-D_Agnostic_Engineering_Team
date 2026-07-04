# Plano de Paralelização — Deploy de isolamento de credencial

**Frota:** CODEX-1, CODEX-2, GEMINI-31-PRO, GLM-52 (+ orquestrador/integração).
**Meta:** máximo paralelismo com **zero colisão de arquivo** — cada stream toca
arquivos disjuntos; hotspots têm dono único.

## Princípio de particionamento
- **Um vendor por stream**, em arquivos NOVOS por vendor (`kiro_home.go`,
  `antigravity_home.go`, etc.) → sem edição concorrente do mesmo arquivo.
- **Hotspots compartilhados** (`execenv.go`, `daemon.go`) = **apenas** stream
  W-INT. Todos programam contra o contrato dela.
- **Observabilidade e docs** são streams independentes (não tocam auth core).

## Onda 0 — Contrato (bloqueante, dono único) → W-INT
| Stream | Dono | Arquivos (lock) | Entrega |
|--------|------|-----------------|---------|
| W-INT-contract | CODEX-1 | `execenv/execenv.go` (PrepareParams+call sites), `daemon/daemon.go` (agentEnv inject) | publica: campo `CredentialAccountHome` em PrepareParams + como cada provider recebe seu dir/env. **Baseline Codex já pronto** serve de molde. |

> Nenhum outro stream de auth começa a editar core até W-INT publicar o contrato.
> Streams de vendor podem começar seus arquivos NOVOS em paralelo desde já.

## Onda 1 — Vendors (paralelo, arquivos disjuntos)
| Stream | Agente | Arquivos NOVOS (lock) | Env var |
|--------|--------|-----------------------|---------|
| W-CODEX | CODEX-1 | `execenv/codex_home.go` (JÁ FEITO) + wiring | `CODEX_HOME` |
| W-KIRO | CODEX-1 | `execenv/kiro_home.go`, `kiro_home_test.go` | `XDG_DATA_HOME` / `KIRO_API_KEY` |
| W-AGY | CODEX-1 | `execenv/antigravity_home.go`, `_test.go` | `HOME` |
| W-METRICS | CODEX-2 | `internal/metrics/credential_metrics.go` (+ test) — arquivo NOVO | — (define coletores; Opus pluga emissão nos hotspots) |
| W-GLM | GLM-52 | `deploy/observability/*` | — |

## Onda 2 — Transversais (paralelo, fora do auth core)
| Stream | Agente | Arquivos (lock) |
|--------|--------|-----------------|
| W-OBS | GLM-52 | `deploy/observability/*` (compose, prometheus.yml, dashboards) |
| W-DEBRAND | GEMINI-31-PRO | só arquivos JÁ tocados pelos vendors (após DONE deles) |
| W-DOCS | CODEX-2 | `docs/project/*` (após contrato estabilizar) |

## Onda 3 — Integração final (dono único) → W-INT
| Stream | Dono | Ação |
|--------|------|------|
| W-INT-final | CODEX-1 | juntar wiring de todos os vendors no `daemon.go`/`execenv.go`, build+test full no container, marcar DONE |

## Dependências (DAG)
```
W-INT-contract ──▶ W-CODEX ─┐
               ├──▶ W-KIRO ─┤
               ├──▶ W-AGY  ─┼──▶ W-DEBRAND ──▶ W-INT-final
               └──▶ W-GLM  ─┘
W-OBS  (independente, paralelo total)
W-DOCS (após contrato)
```

## Regras de disciplina (todos os agentes)
1. Check-in em `.deploy-control/` ANTES de editar (ver README do protocolo).
2. Nunca editar arquivo em `files_locked` de outro agente ativo.
3. Build+test verde no container ANTES do check-out DONE.
4. Hotspots (`execenv.go`, `daemon.go`) só o W-INT.
5. Postgres-only; nada de SQLite próprio.
6. Se bloquear: `status: BLOCKED` + nota; orquestrador redistribui.
