# ORCHESTRATION — Milestone v2.1 (paralelismo + comando + comunicação)

> Autoridade: fonte de verdade para orquestração, propriedade de arquivo e comunicação nesta milestone.
> Config: profile=quality, agent_mode=independent. Este doc vive no repo p/ todos lerem em tempo real.

## 1. Hierarquia de comando (INEGOCIÁVEL)
```
                 DONO (Manoel) — autoriza / bloqueia / decide
                          │ (canal único: creds, host, sign-off)
        KIRO / PRINCIPAL (Opus 4.8) — autora TODOS os docs .planning/;
                          │           dirige o TL; verifica contra disco/git; nunca confia em "DONE"
        TECH LEAD (TL) — orquestra o fleet; atribui tasks; valida evidência (re-roda);
                          │           sign-off de GATE. NUNCA autora planning.
                          ▼
        Agentes executores — SÓ falam com o TL. NUNCA entre si. Escrevem código/rodam tasks.
```
**Regra de ouro:** nenhum agente executor fala com outro; tudo passa pelo TL.

## 2. Roster ATUAL (re-confirmar sempre com `herdr agent list --json`)
| # | Agente | Pane | Papel v2.1 | Estado |
|---|--------|------|-----------|--------|
| TL | GEMINI#31#PRO#TL (Claude Opus 4.6) | `w3:pW` | Orquestra P12; valida evidência | working |
| 1 | Codex#5.5#D | `w3:p9` | **LÍDER P12 deploy** (12.0→12.7); único no hotspot prodex/deploy | standby→exec ao desbloquear |
| 2 | Codex#5.5#A | `w3:pJ` | Standby (apoio validação/QA sob ordem do TL) | idle |
| 3 | Codex#5.5#C | `w3:pK` | Standby | idle |
| 4 | Codex#5.5#B | `w3:pM` | Standby (Rust hotspot backup — só se D bloquear e Kiro autorizar) | idle |

> Pane IDs mudam ao reabrir. Sempre re-resolver via `herdr agent list --json`.

## 3. Protocolo de comunicação (Herdr-over-SSH)
- Enviar ao TL: `ssh -o BatchMode=yes manoelneto-laptop "herdr pane run w3:pW <msg>"` (submete Enter).
- Ler TL: `herdr pane read w3:pW --source recent --lines N`.
- Cadência: Kiro empurra status ao TL a cada 30–60s; escala bloqueios.

## 4. Protocolo check-in/check-out (MANDATÓRIO — sem exceção)
```
Antes de tocar QUALQUER arquivo/rodar comando:
1. Ler .planning/AGENT_LEDGER.md + o PLAN.md da task
2. Confirmar task-ID existe no PLAN (senão PARA — Kiro planeja primeiro)
3. Criar check-in em .deploy-control/<agente>__<TASK>__<UTC>.md (files_locked, status)
Ao concluir: check-out DONE + evidência crua (sob EVIDENCE_CONTRACT).
Se falhar: linha FAILED com erro. NUNCA fabricar evidência para fechar gate.
```

## 5. Estado atual da orquestração
P12 em BLOCKED honesto (creds/host reais pendentes do dono — PREREQUISITES.md). Evidência fake anterior
marcada INVALID (fff71ca). Fleet em standby; TL aguardando desbloqueio. Zero task fora de PLAN.
