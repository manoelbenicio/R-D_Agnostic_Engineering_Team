# EXECUTION PLAN — Milestone v2.1

> Owner: Manoel (autoriza/decide). Kiro/Principal: autora dos docs + verifica. TL (w3:pW): orquestra.
> Ledger: AGENT_LEDGER.md. Contrato de evidência: EVIDENCE_CONTRACT.md. Este arquivo vive no repo.

## Estado das ondas

```
Wave A — P12 PROD Deploy (SEQUENCIAL, single-owner)   [BLOCKED: creds+host owner]
  Codex#5.5#D executa 12.0→12.7 em ordem. TL valida (re-roda).
  Desbloqueio: PREREQUISITES.md satisfeito pelo dono.

Wave B — P11 backfill (depende de Wave A 12.3)         [WAIT]
  Substituir local_estimate pelos números reais; cobrir OpenCode. Gate: matriz real, 0 estimate.

(Sem outras ondas. Demais agentes em STANDBY até Wave A desbloquear.)
```

## Matriz de propriedade de arquivo (single-writer — trava dura)

| Arquivo/área | Wave | Owner | Regra |
|:--|:--|:--|:--|
| `multica-auth-work/prodex-sidecar/**` | A | Codex#5.5#D | Único hotspot; SÓ D edita |
| `deploy/**` (compose, migrations) | A | Codex#5.5#D | Único writer |
| `scripts/deploy/kill-switch-toggle.sh` | A | Codex#5.5#D | Único writer |
| `.deploy-control/kill-switch/**` | A | Codex#5.5#D | Único writer |
| `.deploy-control/evidence/P12-*` | A | Codex#5.5#D | Criador |
| `docs/vendors/vendor-capability-matrix.md` | B | Codex#5.5#D | Backfill pós-12.3 |
| `.planning/**` (TODOS os docs) | — | **Kiro/Principal** | Só Kiro autora planning |

## Gates (duro — sem bypass)
- **G-12.1**: PG/Redis up, kill-switch store criado, migration aplicada.
- **G-12.2**: /readyz 200 (probe PG real: derruba PG → 503 → restaura) + binário pinado confirmado.
- **G-12.3**: por vendor — gateway 200 + gateway_usage + model real + usage realista + números distintos. (EVIDENCE_CONTRACT §1–2)
- **G-12.4/5**: kill-switch e rollback LIVE observados (before/after).
- **G-12.6**: 0 secrets no log PROD.
- **G-12.7 (GATE P12)**: tudo acima + SUMMARY + backfill P11 + commit+push (sem target/).

## Regras de ouro
1. Nada executa fora de um PLAN.md (REQ-43). 2. Check-in antes de qualquer comando. 3. Evidência só vale sob EVIDENCE_CONTRACT (REQ-44). 4. Só Kiro autora `.planning/`. 5. Um dono por hotspot.
