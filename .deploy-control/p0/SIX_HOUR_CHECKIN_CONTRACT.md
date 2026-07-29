# Contrato de check-in/check-out — janela Main Brain P0 de seis horas

Este documento especializa `.deploy-control/p0/PROTOCOL.md`; não cria um board concorrente. `p0_control.py`, `control.json`, `events.jsonl` e o estado Herdr continuam autoritativos.

## 1. Convenção de arquivos imutáveis

Cada assignment produz dois receipts separados:

```text
.deploy-control/p0/checkins/CHECKIN__<agent>__<lane>__<task>__<YYYYMMDDTHHMMSSZ>.json
.deploy-control/p0/checkins/CHECKOUT__<agent>__<lane>__<task>__<YYYYMMDDTHHMMSSZ>.json
```

Use slugs ASCII (`Opus48-A`, `Agy-P0-A7`), lane `L1`–`L8` e task sem espaços. Timestamps são UTC real. Não sobrescreva receipt existente.

Além do receipt, execute o comando correspondente de `scripts/orchestration/p0_control.py` para manter `control.json`/`events.jsonl` consistentes.

## 2. CHECKIN obrigatório antes de qualquer edit

Primeiro:

```bash
herdr pane current --current
python3 scripts/orchestration/p0_control.py check-in \
  --agent '<label>' --pane-id "$HERDR_PANE_ID" \
  --lane '<L1-L8>' --task '<task-id>' \
  --activity '<bounded activity>' --files <exact paths>
```

Depois grave o receipt:

```json
{
  "event": "CHECKIN",
  "timestamp_utc": "YYYY-MM-DDTHH:MM:SSZ",
  "agent": "exact Herdr label",
  "pane_id": "opaque pane id",
  "lane": "L1",
  "task_ids": ["5.1"],
  "plan_ref": ".planning/agent-brain-v3/P0_SIX_HOUR_EXECUTION_PLAN.md",
  "prompt_ref": ".planning/agent-brain-v3/P0_SIX_HOUR_AGENT_PROMPTS.md#L1",
  "files_locked": ["exact/path"],
  "new_files_declared": [],
  "dependencies": [],
  "preflight": {
    "cwd": "path",
    "git_head": "sha",
    "git_status_count": 0,
    "toolchains": {"go": "version-or-unavailable", "node": "version-or-unavailable"},
    "disk_free": "value"
  },
  "constraints_ack": {
    "no_deploy": true,
    "no_inference": true,
    "no_secret_read": true,
    "no_commit_push": true,
    "no_reset_stash_revert_clean": true,
    "ownership_exclusive": true
  },
  "status": "IN_PROGRESS"
}
```

Um receipt com arquivo genérico (`server/**`) quando o prompt exige paths exatos é inválido.

## 3. Heartbeat

No máximo a cada 10 minutos e sempre que status/progresso/blocker mudar:

```bash
python3 scripts/orchestration/p0_control.py heartbeat \
  --agent '<label>' --task '<task-id>' --progress <0-99> \
  --activity '<material result or current command>'
```

Conteúdo mínimo do heartbeat:

- progresso baseado em entregável, não tempo;
- comando/teste atual ou último material result;
- ETA atualizado;
- blocker concreto, owner e next action se houver;
- arquivos adicionados ao lock (somente após manager validar zero-overlap).

Heartbeat sem mudança material pode ser curto, mas não pode inventar progresso.

## 4. BLOCKED

Bloqueie imediatamente quando faltar contrato, ferramenta, autorização ou arquivo fora do ownership:

```bash
python3 scripts/orchestration/p0_control.py block \
  --agent '<label>' --task '<task-id>' \
  --blocker '<fact; owner; next action>'
```

Não use workaround destrutivo, credencial, fake upstream, mock de produção, instalação ou edição fora do scope para contornar o blocker.

## 5. CHECKOUT

Antes do receipt, execute focused validation. Depois:

```bash
python3 scripts/orchestration/p0_control.py check-out \
  --agent '<label>' --task '<task-id>' \
  --evidence '<artifact path>' \
  --validation '<exact command: PASS|FAIL|BLOCKED>' \
  --summary '<bounded result and non-claims>'
```

Receipt obrigatório:

```json
{
  "event": "CHECKOUT",
  "timestamp_utc": "YYYY-MM-DDTHH:MM:SSZ",
  "agent": "exact Herdr label",
  "pane_id": "opaque pane id",
  "lane": "L1",
  "task_ids": ["5.1"],
  "status": "DONE|BLOCKED|FAILED",
  "files_changed": ["exact/path"],
  "files_created": [],
  "files_deleted": [],
  "tests": [
    {"command": "exact", "exit_code": 0, "assertions_or_tests": 1, "result": "PASS"}
  ],
  "format_and_diff": {
    "format": "PASS|FAIL|NOT_AVAILABLE",
    "git_diff_check": "PASS|FAIL"
  },
  "evidence": ["path"],
  "requirements": ["AB-REQ-xx"],
  "openspec_tasks": ["5.1"],
  "summary": "what is actually proven",
  "non_claims": ["what was not executed or accepted"],
  "blocker": null,
  "handoffs": [
    {"to_lane": "L1", "reason": "shared anchor", "file": "exact/path"}
  ]
}
```

Zero tests/assertions não conta como PASS. Static review deve ser classificado `REVIEWED`, não `VERIFIED`.

## 6. Separação de responsabilidades

| Papel | Pode editar produto | Pode verificar | Pode aceitar/fechar OpenSpec |
|---|---:|---:|---:|
| L1–L7 producer | somente ownership | focused own scope | não |
| L8 evaluator | não | sim | não |
| Opus48-Kiro manager | não | inspeciona artifacts | recomenda |
| Principal Orchestrator | somente docs autoritativos | amostra independentemente | sim, com evidência |

Produtor não autoaceita. Um verificador não corrige o finding que avalia. Principal não fecha task por narrativa.

## 7. Monitoramento de panes

Antes de cada wave e a cada 10 minutos:

```bash
herdr agent list
python3 scripts/orchestration/p0_control.py monitor --once
```

O manager compara cada assignment ativa com:

- pane ainda existe;
- estado Herdr `working`, ou `blocked` com blocker concreto;
- heartbeat ≤10 min (RED após 15 min);
- lock sem interseção;
- receipt CHECKIN existente;
- tool preflight registrado.

Agente `idle|done` só recebe nova assignment se houver trabalho real, disjunto e com critério de aceite. Caso contrário, standby é correto e não conta como falha de saturação.

## 8. Gates de segurança

- `implementation_authorized=true` autoriza somente engenharia offline dentro do ownership exato de L1–L7 e das tasks desta janela; L8 continua read-only e manager/Principal continuam sem código de produto. Essa flag não autoriza deploy, restart, inferência, leitura de segredo, commit/push nem edição fora do lock.
- `live_runs.*.authorized=false`: zero inference.
- Nenhum receipt contém secret, auth header, cookie, prompt, tool payload, repository content ou account identity.
- Não registrar body bruto de logs/spans.
- Nenhum deploy ou mutação ORQ1 nesta janela por estes agentes.

## 9. Gate de encerramento T+360

A janela encerra somente quando cada lane possui CHECKOUT `DONE|BLOCKED|FAILED`, o manager publica consolidado pane→lane→status→evidence e o Principal registra:

- tasks realmente fecháveis;
- blockers externos;
- findings ainda abertos;
- build/test provenance;
- non-claims de deploy/live/capacidade;
- próxima ação serial crítica.
