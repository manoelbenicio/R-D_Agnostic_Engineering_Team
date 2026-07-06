# CHECK-IN / CHECK-OUT — PROTOCOLO MANDATÓRIO (vigente v2.1)

> VINCULANTE p/ TODO agente, INCLUSIVE o TL. Sem exceção, sem atalho. Complementa GOLDEN_RULES.md (R1)
> e EVIDENCE_CONTRACT.md. Violar qualquer item abaixo → trabalho REJEITADO/revertido + escala ao Kiro.

```
═══════════════════════════════════════════════════════════════════════
<mandatory_signin_signout>  — OBRIGATÓRIO. NÃO NEGOCIÁVEL.
═══════════════════════════════════════════════════════════════════════
```

## A. ANTES de tocar QUALQUER arquivo ou rodar QUALQUER comando (CHECK-IN)
1. **Ler** `.planning/GOLDEN_RULES.md`, `.planning/AGENT_LEDGER.md`, e o `PLAN.md` da sua task.
2. **Verificar task-ID**: a task existe no PLAN.md? Se não → PARE e chame o TL/Kiro (R11). Não improvise.
3. **Verificar ownership**: algum check-in ativo (IN_PROGRESS) já trava seus `files_locked`? Se sim → PARE, coordene com o TL (R2). Escopo deve ser DISJUNTO.
4. **Criar o arquivo de check-in** em `/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/.deploy-control/<AGENT>__<STREAM>__<START_UTC>.md` com o front-matter COMPLETO:
   ```
   agent:            # ex: Codex#5.5#D
   stream:           # ex: P12-AGENTIC
   phase:            # ex: P12
   priority:         # P0..P3
   status: IN_PROGRESS
   progress: 0
   started_at:       # UTC ISO8601
   finished_at:      # vazio
   files_locked:     # escopo EXCLUSIVO e disjunto (globs)
   depends_on:       # task/stream ou "none"
   plan_ref:         # .planning/phases/.../PLAN.md
   build_result:     # vazio até haver
   notes:            # o que vai fazer, doc de referência
   ```
5. **Atualizar o File Lock Table** (AGENT_LEDGER.md): você = owner, 🔴 Locked nos `files_locked`.

> NENHUM comando/edição pode ocorrer antes de A1–A5 completos. Evidência NUNCA pode predatar o check-in.

## B. DURANTE (obrigatório)
6. **Reporte** ao TL a cada 60s (P12) no STATUS REPORTING STANDARD:
   `agent · stream · phase · task · priority · status · progress · eta · started_at · finished_at · depends_on · blockers · build_result · notes`
7. **ACK** assinado em ≤15min quando o TL solicitar.
8. **Só toque seus `files_locked`.** Nunca arquivo de outra stream.

## C. AO TERMINAR (CHECK-OUT — obrigatório; DONE não existe sem isto)
9. **Evidência real primeiro**: comando + saída scrubbed em `.deploy-control/evidence/`, sob EVIDENCE_CONTRACT (real, não fake/estimate/inventado). Sem evidência real → NÃO é DONE.
10. **Atualizar o MESMO check-in**:
    ```
    status: DONE            # ou BLOCKED
    progress: 100
    finished_at:            # UTC
    build_result: |         # comando + resultado real scrubbed
    notes:                  # concluído; ponteiro p/ evidência
    ```
11. **Liberar o lock** (AGENT_LEDGER.md): owner limpo, 🟢 Available.

## D. Se FALHAR ou BLOQUEAR (obrigatório — honestidade > fingir)
12. `status: BLOCKED` + `notes:` com motivo EXATO + o que precisa. Linha FAILED no ledger com o erro. Escale ao TL. **BLOCKED honesto é aceito; evidência fake é REJEITADA e revertida.**

## E. ENFORCEMENT (quem garante)
- **TL**: recusa qualquer DONE sem check-in+check-out+evidência real; re-roda a evidência (R3, R9). Só o TL commita.
- **Kiro/Principal**: audita em disco/git; marca fake como INVALID; reverte task para BLOCKED.
- **Consequência de pular**: trabalho invalidado, task revertida, incidente registrado em VERIFICATION.md.

```
</mandatory_signin_signout>
```
