# PROJECT — Agent Brain v3 (OpenSpec/GSD rebaseline)

Planning/adjudication key: Kiro `claude-opus-4.8` — Technical Lead/Manager.
planning_owner: Kiro/Opus-4.8 · operational_co_lead: Codex#56#A (owner transition 2026-07-18)
active_milestone: Agent Brain v3 — G3 serial integration READY; Waves 0–3/tier 20 authorized
supersedes_planning: .planning/ (RPP/Prodex v2.1) — preservado como histórico, NÃO executado como plano ativo
active_openspec_change: build-omniroute-agent-brain (MASTER ATIVO)

## O que é este projeto

Agent Brain é o **cold control plane** brand-neutral: orquestra tasks, workspaces,
repositories, lifecycle de processos, launch/cancelamento de CLIs, streaming de
resultados, admission control (tiers 20/50/100) e a política `CLIKind` + `RouteModel`.
Não possui credenciais de provider. Guarda apenas **uma** chave OmniRoute estável e
limitada.

OmniRoute é o **hot data plane exclusivo**: credenciais/subscriptions de provider,
account pools, strict round-robin, continuation affinity, token refresh/expiry,
quota/reset/redeem, 401/403/429, circuit breakers, bounded retry/fallback,
protocol translation, streaming/tools, Smart Context (SC01–SC10) e telemetria/auditoria.

## Fronteira cold / hot

| Responsabilidade | Agent Brain (cold) | OmniRoute (hot) |
|---|---|---|
| Tasks, workspaces, repos, processos, launch/cancel/stream | Owns | — |
| Admission 20/50/100 | Task-level admission | Inference/account limits |
| Credenciais/subscriptions de provider | **Nunca** | Exclusive owner |
| Account selection + strict round-robin | Não duplica | Exclusive owner |
| Continuation affinity | Fornece IDs opacos | Enforce |
| Refresh/expiry/quota/reset/redeem | Observa status seguro | Exclusive owner |
| 429/5xx/retry/circuit/fallback | Define deadline/policy | Executa bounded pre-commit |
| Protocol translation / streaming / tools | Configura adapter | Preserva ou rejeita |
| Smart Context/token saving | Kill switch only | Computa e executa |
| Secrets observability | Redige chave+conteúdo | Redige chave/secrets+conteúdo |

## Non-goals (deste plano, e do target)

- Não remover Prodex nesta fase (não autorizado até gate G6).
- Não ativar tiers 50/100 (somente após gates G7).
- Não ativar gateway-required como default (somente após gate G6).
- Não permitir dual router (Prodex + OmniRoute ativos simultaneamente como owners).
- Não pôr credenciais provider dentro do Agent Brain.
- Não ler, imprimir, copiar ou registrar a chave OmniRoute.
- Não executar mudanças de produção. O sistema é não-produtivo; production canary/soak foi
  removido pelo dono, mas testes de integração, segurança, falhas, rollback e capacidade em
  desenvolvimento continuam obrigatórios.
- Não reescrever cegamente lifecycle/workspace/repo/cancel/streaming funcionais.

## Owners

- **Planning/adjudication owner:** Kiro/Opus-4.8 (não escreve código de produto).
- **Operational co-lead:** Codex#56#A (Herdr transport, independent verification, state/docs,
  execution control; não disputa decisões arquiteturais com Kiro).
- **Integrator-líder (após autorização):** Codex 1 — único owner de `daemon.go`,
  `config.go`, `health.go`, `execenv/execenv.go`, `execenv/codex_home.go`,
  `pkg/agent/models.go`, `cmd/multica/cmd_daemon.go`.
- **Streams paralelos (após autorização):** Codex 2 (gateway), Codex 3 (runtime/CLI),
  Codex 4 (ops/paridade/evidência).
- **Sign-off:** arquiteto OmniRoute, dono do produto, segurança (para waivers) — separados,
  nenhum agente forja aceite.

## Mapeamento de histórico

- RPP/Prodex v2.1 (`.planning/`) → SUPERSEDED como plano ativo; preservado como histórico.
- decisions legadas D-007 (isolamento credenciais) e D-008 (TL delegation-only) e
  EVIDENCE_CONTRACT (proveniência, rejeição de fabricação) → absorvidos como governança
  permanente do Agent Brain v3 (ver DECISIONS.md).

## Índice de artefatos (.planning/agent-brain-v3/)

PROJECT.md · REQUIREMENTS.md · ROADMAP.md · STATE.md · DECISIONS.md · TRACEABILITY.md ·
COMPONENT_REGISTER.md · INTERFACE_REGISTER.md · REMOVAL_REGISTER.md · RISKS.md ·
EVIDENCE_CONTRACT.md · FILE_OWNERSHIP.md · AGENT_LEDGER.md · EVIDENCE_INDEX.md ·
HERDR_TRANSPORT.md · DISPATCH_QUEUE.md · G4_ACCELERATED_PACKET.md · phases/G0..G8/PLAN.md
