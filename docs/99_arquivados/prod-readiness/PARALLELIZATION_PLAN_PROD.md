# PARALLELIZATION PLAN — Prod Readiness (máx paralelismo, zero colisão)
> Orquestrador: Opus 4.8. Regra-mãe: ownership DISJUNTO. Hotspots têm dono único e são
> SERIAIS; todo o resto é ARQUIVO NOVO por stream. Nenhum agente edita arquivo travado
> por outro. Opus valida cada DONE no container (não confia no tail).
> Base: MASTER_PROD_READINESS.md (B/H/O/F). Layout real conferido 2026-07-02.

## MAPA DE HOTSPOTS (dono único, SERIAL — nunca 2 agentes juntos)
- `daemon/daemon.go`            → OWNER serial (fila W-*-INT). 1 stream por vez.
- `daemon/execenv/execenv.go`   → OWNER serial (fase 3 isolamento).
- `rotation/contract.go`        → CONGELADO. Ninguém edita (só Opus, se preciso).
- `metrics/credential_metrics.go`→ OWNER serial (só quando métrica nova exigir).
Regra: mudança em hotspot = seu próprio stream, lock exclusivo, sem paralelo no MESMO arquivo.

## STREAMS PARALELOS (arquivos NOVOS/disjuntos — rodam JUNTOS sem colisão)
Cada linha = 1 agente. Nenhuma toca o arquivo da outra.

### GRUPO 1 — Robustez do domínio rotação (arquivos NOVOS em rotation/)
| Stream | Arquivos (lock, NOVOS) | Item | Depende |
|--------|------------------------|------|---------|
| PR-TOKEN-LIFECYCLE | rotation/token_refresh.go (+_test) | B1 | contract (read-only) |
| PR-DETECT-HARDEN   | rotation/detector_reactive_ext.go (+_test) | B3,F2 | usa detector.go read-only; NÃO edita detector.go — novo tipo/func |
| PR-COOLDOWN        | rotation/cooldown_return_test.go (+ helper novo se preciso) | H3 | service/pool read-only |
| PR-CONCURRENCY     | rotation/pool_concurrency_test.go | H4 | pool read-only |
| PR-ROBUST-SUBSWAP  | rotation/swap_snapshot.go (+_test) | H5 | contract read-only |
> Todos em rotation/, mas CADA UM cria arquivo próprio novo. Zero colisão entre si.
> Se um precisar MUDAR service.go/pool.go/detector.go (hotspot do domínio) → vira stream
> serial separado com lock daquele arquivo (não paraleliza com quem lê o mesmo).

### GRUPO 2 — Auth real + provisionamento (scripts/docs NOVOS)
| Stream | Arquivos (lock, NOVOS) | Item | Depende |
|--------|------------------------|------|---------|
| PR-ENROLL-SCRIPT   | scripts/staging/enroll_account.sh + enroll_*.sql | B4 | schema 123 read-only |
| PR-AUTH-HARNESS    | rotation/real_auth_switch_test.go (//go:build staging) | B5 | usa CredentialAuthenticator read-only |
| PR-ENROLL-RUNBOOK  | docs/project/account-enrollment-runbook.md | O2 | — |

### GRUPO 3 — Observabilidade (config/docs NOVOS, sem Go de produto)
| Stream | Arquivos (lock) | Item | Depende |
|--------|-----------------|------|---------|
| PR-OBS-ALERTS      | deploy/observability/alerts.yml (edição isolada) | H6 | stack up (feito) |
| PR-OBS-RUNBOOK     | docs/project/observability-runbook.md | O3 | stack up |
> Observability já está de pé (WAVE A). Estes 2 não tocam Go nenhum → 100% paralelos.

### GRUPO 4 — Docs de operação (NOVOS, zero código)
| Stream | Arquivos (lock) | Item |
|--------|-----------------|------|
| PR-DEPLOY-RUNBOOK  | docs/project/prod-deploy-runbook.md | O1 |
| PR-SECRETS-DECISION| docs/project/secrets-at-rest.md | O4 |

## STREAMS SERIAIS (hotspots — 1 de cada vez, na fila; NÃO paralelizam entre si)
| Stream | Hotspot (lock EXCLUSIVO) | Item | Ordem |
|--------|--------------------------|------|-------|
| S-DAEMON-DBURL-GUARD | daemon/daemon.go | B2 (WARN se rotação sem DATABASE_URL) | 1º |
| S-DAEMON-METRICS     | daemon/daemon.go | H2 (expor metrics server do daemon) | 2º (após S1) |
| S-INT-TOKEN          | daemon/daemon.go | fiar B1 no loop (após PR-TOKEN-LIFECYCLE) | 3º |
> daemon.go = 1 dono por vez. S-DAEMON-* entram em SÉRIE. O Opus valida cada um antes do próximo.

## MATRIZ DE COLISÃO (garantia de não-quebra)
- GRUPO 1/2/3/4 = arquivos novos disjuntos → rodam TODOS em paralelo, inclusive entre grupos.
- Serial S-DAEMON-* = fila única no daemon.go; roda EM PARALELO com Grupos 1-4 (arquivos
  diferentes), mas NUNCA 2 streams no daemon.go ao mesmo tempo.
- Nenhum stream de Grupo edita hotspot; se precisar, PARA e vira pedido de stream serial.

## CAPACIDADE (quantos agentes ao mesmo tempo)
Onda máxima possível AGORA (sem esperar dependência dura):
- 1 agente em S-DAEMON-DBURL-GUARD (daemon.go)         [B2]
- 1 agente PR-DETECT-HARDEN (arquivo novo)             [B3]
- 1 agente PR-ENROLL-SCRIPT (scripts novos)            [B4]
- 1 agente PR-OBS-ALERTS + PR-OBS-RUNBOOK (config/doc) [H6,O3]
- 1 agente PR-DEPLOY-RUNBOOK / PR-SECRETS (docs)       [O1,O4]
→ até 5+ agentes SIMULTÂNEOS sem colisão. Token-lifecycle (B1) e auth-harness (B5)
  entram assim que houver 2ª conta real (dependência de credencial, não de código).

## DISCIPLINA
0. OBRIGATÓRIO (gate duro, NÃO opcional): ASSINAR ANTES e DEPOIS.
   - ANTES de qualquer trabalho: criar check-in em .deploy-control/
     `<AGENTE>__<STREAM>__<START_UTC>.md` com agent, stream, started_at (UTC), status:
     IN_PROGRESS, files_locked. Nada é editado antes disso existir.
   - LOGO AO TERMINAR: atualizar o MESMO arquivo com finished_at (timestamp UTC) +
     nome do agente confirmado + status: DONE|BLOCKED + build_result colado.
   - Sem started_at + finished_at + nome do agente = NÃO está concluído (Opus rejeita).
1. Check-in/out em .deploy-control/ com files_locked ANTES de editar.
2. Ler check-ins IN_PROGRESS; se seu arquivo já está travado → esperar/outro stream.
3. Verde no container antes de DONE; Opus re-roda e valida.
4. Nada inventado (fonte primária p/ vendor). Sem segredo em log. Tokens mascarados.
5. Hotspot = lock exclusivo + serial. Resto = arquivo novo + paralelo.
