# Master Agentic Deployment Plan — Credential Isolation + Rotation

**Orchestrator/SME:** Opus 4.8 (planning, contract ownership, hotspot integration,
independent validation). **Coders:** CODEX-1, CODEX-2, GLM-5.2 (+ optional CODEX-3).
**Rule:** máximo paralelismo em arquivos disjuntos; hotspots (`execenv.go`,
`daemon.go`) e integração = dono único (Opus). Verificação real no container a
cada DONE. Postgres-only. Source original intocado (trabalho em `multica-auth-work/`).

## Matching agente → força (por que cada um)
- **CODEX** (1/2/3): implementação Go rigorosa, defensiva, aderente a contrato
  (provou nos vendors + métricas). Ideal para: pacotes novos de lógica (detector,
  rotação, adapter), verificação Go.
- **GLM-5.2**: infra/observabilidade e visão sistêmica (achou o `--entrypoint
  promtool`, o `/metrics` existente). Ideal para: observability, deploy, schema
  SQL/migrations, dashboards, docs de ops.
- **Opus 4.8**: contrato, hotspots, integração, RCA, aceite. Serial por natureza.

## Estado atual (Wave 0/1 — DONE + validado por mim)
| Stream | Agente | Status |
|--------|--------|--------|
| W-INT-contract (contrato + Codex piloto) | Opus 4.8 | ✅ DONE, verde |
| W-VENDORS (kiro_home, antigravity_home + testes) | CODEX-1 | ✅ DONE, validado |
| W-METRICS (credential_metrics coletores) | CODEX-2 | ✅ DONE, validado |
| W-OBS (observability stack) | GLM-5.2 | ✅ DONE, validado |
| Fiação vendors no core (execenv+daemon) | Opus 4.8 | ✅ feito; verificação final delegada |

## Wave 1.5 — Fechar Fase 1 (curto, em andamento)
| Stream | Agente | Arquivos | Dep |
|--------|--------|----------|-----|
| W-VERIFY (build ./… + testes COM git no container; não-regressão fallback) | CODEX (especialista verificação) | só `_test.go` novos se preciso | vendors, contrato |
| W-DEBRAND (higienizar branding nos arquivos tocados, sem mudar comportamento) | GLM-5.2 | apenas arquivos já tocados, pós-DONE | W-VERIFY |

## Wave 2 — Fase 2 Rotação (paralelo real, pacotes NOVOS)
| Stream | Agente | Arquivos NOVOS (lock) | Dep |
|--------|--------|-----------------------|-----|
| W-DETECT (detecção de esgotamento: regex tela por vendor + HTTP 429 + distinguir 503) | CODEX-1 | `internal/rotation/detector.go` (+ test) | contrato |
| W-ROTATE (máquina de estados: lock→snapshot→logout→select→login→resume; prioridade por expertise) | CODEX-2 | `internal/rotation/service.go`, `models.go`, `pool.go` (+ tests) | contrato |
| W-PGSTORE (Postgres: accounts, credentials(ref), assignments, rotation_events + migrations) | GLM-5.2 | `internal/rotation/store_pg.go`, `migrations/*` (+ tests) | — |
| W-EMIT (instrumentar pontos de emissão das métricas nos hotspots) | Opus 4.8 | `execenv.go`, `daemon.go`, `rotation/*` call sites | W-METRICS, W-ROTATE |

> W-DETECT / W-ROTATE / W-PGSTORE são arquivos disjuntos em `internal/rotation/` →
> paralelizam. A integração no core (W-EMIT) e a fiação da rotação no daemon são
> do Opus (serial, hotspots).

## Wave 3 — Integração + Observabilidade real + Aceite final (Opus + GLM)
| Stream | Agente | Ação |
|--------|--------|------|
| W-INT-final | Opus 4.8 | juntar rotação no daemon, build+test full, RCA |
| W-OBS-real | GLM-5.2 | ligar dashboards às métricas reais emitidas; validar alertas com dados |
| W-ACCEPT | Opus 4.8 | checklist de aceite: não-regressão AS-IS, isolamento por vendor, rotação e2e |

## Disciplina (inegociável)
1. Check-in em `.deploy-control/` antes de editar; `files_locked` declarado.
2. Nunca editar arquivo travado por outro; hotspots só Opus.
3. Build+test verde no container ANTES de DONE (verificado pelo Opus, não confiado).
4. Postgres-only; nada de SQLite próprio; sem branding novo "Multica".
5. Se bloquear: status BLOCKED + nota; Opus redistribui.
6. Cada stream = pacote/arquivo NOVO sempre que possível → zero colisão.