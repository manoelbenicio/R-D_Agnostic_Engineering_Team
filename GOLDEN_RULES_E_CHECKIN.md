# GOLDEN RULES & CHECK-IN/OUT — Rotation-Parity Polyglot (CANÔNICO, atualizado 2026-07-04)

> **Este é o documento vigente.** O `CHECKIN_OUT.md` da raiz é de outro projeto (HerdMaster, jun/22) — SUPERSEDED.
> Todo agente (inclusive o TL) DEVE seguir isto. Regras + formato EXATO do check-in/out obrigatório em disco.

## GOLDEN RULES (inegociáveis)
1. **SIGN-IN/OUT em disco** — check-in ANTES de tocar em qualquer arquivo; check-out ao terminar. Caminho absoluto.
2. **Propriedade de arquivo DISJUNTA** — dois agentes não podem ter escopo (`files_locked`) sobreposto ao mesmo tempo. Hotspots (ex.: `internal/daemon`) = dono único serial.
3. **Verde-em-container COM EVIDÊNCIA antes de DONE** — não confie no tail; o validador (TL/Opus) re-roda. IPv6 OFF nos builds (`--sysctl net.ipv6.conf.all.disable_ipv6=1`).
4. **Nada inventado** — só fonte primária. Nunca invente flag/comando do prodex/Herdr.
5. **Sem segredo** em log/trace/evidência/check-in. **SQLite proibido** p/ estado compartilhado (Postgres).
6. **Invariantes runtime:** roteador único/sessão, hard affinity, rotate-before-commit, troca de perfil fail-closed.
7. **Segurança prodex:** Caveman/hook DESABILITADO por padrão (RCE); `PRODEX_ALLOW_UNSAFE_CHILD_ENV=off`.
8. **QA NUNCA bypassado.** Deploy só com kill-switch + rollback TESTADOS + logs scrubbed.
9. **Só o TL commita** (após validação). Se travar/ambíguo: PARE e escale (TL/dono). Não decida sozinho.
10. **Comunicação com o TL só via Herdr** (`pane run`, não `agent send`), e só com autorização do dono.

## FORMATO DO CHECK-IN (obrigatório ANTES de trabalhar)
**Arquivo:** `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/.deploy-control/<AGENT>__<STREAM>__<START_UTC>.md`
(ex.: `Codex-5.5-C__P0-FOUNDATION__20260705T003000Z.md`). Front-matter obrigatório:

```
agent: Codex#5.5#C
stream: P0-FOUNDATION
phase: P0
priority: P0
status: IN_PROGRESS            # IN_PROGRESS ao iniciar
progress: 0                    # 0..100
started_at: 2026-07-05T00:30:00Z
finished_at:                   # vazio até terminar
files_locked:                  # escopo EXCLUSIVo que você vai tocar (disjunto)
  - ~/runtime/prodex-src/**
  - multica-auth-work/server/.env (config prodex)
depends_on: none               # ou outra stream/task
build_result:                  # colar quando houver (comando + resultado)
notes: iniciando P0 conforme Diligencias/00_FUNDACAO_P0.md
```

## FORMATO DO CHECK-OUT (obrigatório AO TERMINAR)
Atualize o MESMO arquivo (ou anexe entrada) com:
```
status: DONE                   # DONE | BLOCKED
progress: 100
finished_at: 2026-07-05T01:10:00Z
build_result: |
  green — docker run --rm --sysctl net.ipv6.conf.all.disable_ipv6=1 ... cargo build --release => target/release/prodex
  prodex --version => v0.246.0 ; sha256=<hash>
notes: P0 concluída; binário pinado + Multica resolve o executável. Evidência em .deploy-control/evidence/.
```
- **BLOCKED:** `status: BLOCKED` + `notes:` com o motivo exato + o que precisa. Escale ao TL.
- **Evidência real** (comando+saída scrubbed) em `.deploy-control/evidence/`. Sem isso, NÃO é DONE.

## STATUS REPORTING STANDARD (campos mínimos por report)
`agent · stream · phase · task · priority · status · progress · eta · started_at · finished_at · depends_on · blockers · build_result · notes`
ACK assinado em ≤15min quando o TL solicitar; cadência de 30min.

## REGRAS DE OWNERSHIP (conflito)
- Antes de iniciar, verifique se algum check-in ativo (`STARTING/IN_PROGRESS`) já trava seus `files_locked`. Se sim → coordene com o TL antes.
- Nunca toque arquivo de outra stream.

> Fonte da verdade das regras: este doc + `Diligencias/00_LEIA_PRIMEIRO_MISSAO.md`. Formato herdado dos blocos `<mandatory_signin_signout>` dos prompts.
