# GOLDEN RULES & CHECK-IN/OUT — RPP (CANÔNICO, vigente v2.1)

> Documento vigente e VINCULANTE. Todo agente (inclusive o TL) DEVE seguir. Fonte da verdade das regras.
> Complementa EVIDENCE_CONTRACT.md (o que conta como evidência REAL) e ORCHESTRATION_v2.1.md (comando/roster).

## GOLDEN RULES (inegociáveis — TODAS as 10)
1. **SIGN-IN/OUT em disco** — check-in ANTES de tocar em qualquer arquivo; check-out ao terminar. Caminho absoluto.
2. **Propriedade de arquivo DISJUNTA** — dois agentes não podem ter `files_locked` sobreposto ao mesmo tempo. Hotspots (ex.: `prodex-sidecar`, `internal/daemon`) = dono único serial.
3. **Verde-em-container COM EVIDÊNCIA antes de DONE** — não confie no tail; o validador (TL/Opus) re-roda. IPv6 OFF nos builds (`--sysctl net.ipv6.conf.all.disable_ipv6=1`).
4. **Nada inventado** — só fonte primária. Nunca invente flag/comando do prodex/Herdr, nem número, nem upstream. (Ver EVIDENCE_CONTRACT: localhost/fake-upstream/smoke/placeholder/usage trivial/números idênticos/sign-off forjado = INVALID.)
5. **Sem segredo** em log/trace/evidência/check-in. **SQLite proibido** p/ estado compartilhado (Postgres).
6. **Invariantes runtime:** roteador único/sessão, hard affinity, rotate-before-commit, troca de perfil fail-closed.
7. **Segurança prodex:** Caveman/hook DESABILITADO por padrão (RCE); `PRODEX_ALLOW_UNSAFE_CHILD_ENV=off`.
8. **QA NUNCA bypassado.** Deploy só com kill-switch + rollback TESTADOS + logs scrubbed.
9. **Só o TL commita** (após validação). Se travar/ambíguo: PARE e escale (TL/dono). Não decida sozinho.
10. **Comunicação com o TL só via Herdr** (`pane run`, não `agent send`), e só com autorização do dono.

## REGRA ADICIONAL v2.1 (reforço pós-incidente de evidência fake 2026-07-06)
11. **Nenhuma task fora de um PLAN.md** (task-ID rastreável). Passo não planejado → PARA, Kiro planeja, só então delega.
12. **Só Kiro/Principal autora `.planning/`.** Agentes não criam artefatos de planejamento.

## FORMATO DO CHECK-IN (obrigatório ANTES de trabalhar)
Arquivo: `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/.deploy-control/<AGENT>__<STREAM>__<START_UTC>.md`
```
agent: Codex#5.5#D
stream: P12-AGENTIC
phase: P12
priority: P0
status: IN_PROGRESS            # IN_PROGRESS ao iniciar
progress: 0                    # 0..100
started_at: 2026-07-06T01:51:00Z
finished_at:                   # vazio até terminar
files_locked:                  # escopo EXCLUSIVO e disjunto
  - multica-auth-work/prodex-sidecar/**
  - deploy/**
  - .deploy-control/evidence/P12-*
depends_on: PREREQUISITES (host) ; plan_ref: .planning/phases/12-prod-deploy/PLAN.md
build_result:                  # colar quando houver (comando + resultado)
notes: iniciando P12 agentic conforme AGENTIC-REAL-SESSION.md
```

## FORMATO DO CHECK-OUT (obrigatório AO TERMINAR)
Atualize o MESMO arquivo:
```
status: DONE                   # DONE | BLOCKED
progress: 100
finished_at: 2026-07-06T0X:XXZ
build_result: |
  <comando + saída scrubbed real>
notes: <task> concluída; evidência em .deploy-control/evidence/ (sob EVIDENCE_CONTRACT).
```
- **BLOCKED:** `status: BLOCKED` + motivo exato + o que precisa. Escale ao TL. (Melhor BLOCKED honesto que fake.)
- **Evidência real** (comando+saída scrubbed) em `.deploy-control/evidence/`. Sem isso, NÃO é DONE.

## STATUS REPORTING STANDARD (campos mínimos por report)
`agent · stream · phase · task · priority · status · progress · eta · started_at · finished_at · depends_on · blockers · build_result · notes`
ACK assinado em ≤15min quando o TL solicitar; cadência de reporte a cada 60s durante P12.

## OWNERSHIP (conflito)
Antes de iniciar, verifique check-in ativo que trave seus `files_locked`. Se sim → coordene com o TL. Nunca toque arquivo de outra stream.
