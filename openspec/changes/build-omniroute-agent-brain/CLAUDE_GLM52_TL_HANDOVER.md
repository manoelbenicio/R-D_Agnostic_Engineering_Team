# Handover Zero-Context — Claude/GLM-5.2 Technical Lead

> **SUPERSEDED 2026-07-18:** Claude was closed by the owner after G2. Current authority is
> `KIRO_OPUS48_CODEX56A_TL_HANDOVER.md`. This file is historical G0/G1 evidence only.

## 1. Destinatário e autoridade

**Destinatário:** Claude usando GLM-5.2, atuando como Technical Lead/Manager do programa Agent Brain.

**Decisão do dono recebida em 2026-07-17:** Claude/GLM-5.2 liderará toda a orquestração, planejamento agentic e preparação coordenada do ambiente.

O TL é responsável por manter OpenSpec e GSD alinhados, decompor o trabalho, atribuir tarefas aos quatro Codex, controlar dependências e arquivos, validar evidências, resolver gaps e sintetizar o resultado para o dono. O TL é delegation-only para implementação: não produz código de produto, não altera produção e não assume silenciosamente tarefas dos workers.

## 2. Estado atual — leia antes de agir

- Planejamento OpenSpec completo: proposal, design, cinco specs e `tasks.md`.
- Plano atual: 85 tarefas, 0 executadas.
- Implementação Waves 0–3/tier 20 autorizada em 2026-07-17: Seção 7.1 de `OMNIROUTE_ARCHITECT_RESPONSE.md` marcada `AUTORIZADO`.
- Nome provisório aprovado: `Agent Brain`.
- Primeiro tier: 20 tarefas; 50/100 exigem aprovações futuras.
- Prodex não pode ser removido nesta autorização inicial.
- Cutover default para OmniRoute não está autorizado.
- OmniRoute é o target hot-path exclusivo; Brain será cold control plane e credentialless.
- O GSD atual é legado RPP/Prodex v2.1 e contradiz o target atual. Ele deve ser preservado como histórico e rebaselined, não usado como plano concorrente.
- A regra histórica dizia que apenas Kiro escrevia `.planning/`. A decisão atual nomeia Claude/GLM-5.2 como novo planning/orchestration owner; essa mudança precisa ser registrada formalmente no novo GSD antes de executar código.
- Nenhum segredo deve ser lido, impresso, copiado para docs ou enviado a agentes. Use referências de secret, nunca valores.

## 3. Ordem obrigatória de leitura

Raiz canônica do projeto:

```text
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team
```

Leia nesta ordem:

1. Planejamento e governança geral:

```text
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/build-omniroute-agent-brain/MASTER_PLANNING_AND_GOVERNANCE.md
```

2. Arquitetura AS-IS e TO-BE:

```text
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/build-omniroute-agent-brain/architecture.md
```

3. Proposal e design:

```text
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/build-omniroute-agent-brain/proposal.md
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/build-omniroute-agent-brain/design.md
```

4. Requisitos formais:

```text
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/build-omniroute-agent-brain/specs/agent-brain-runtime/spec.md
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/build-omniroute-agent-brain/specs/omniroute-agent-routing/spec.md
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/build-omniroute-agent-brain/specs/credentialless-agent-execution/spec.md
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/build-omniroute-agent-brain/specs/parallel-agent-capacity/spec.md
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/build-omniroute-agent-brain/specs/brain-cutover-operations/spec.md
```

5. OmniRoute supplier contract and response:

```text
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/build-omniroute-agent-brain/OMNIROUTE_REQUIREMENTS_HANDOFF.md
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/build-omniroute-agent-brain/OMNIROUTE_ARCHITECT_RESPONSE.md
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/build-omniroute-agent-brain/omniroute-architecture-acceptance-checklist.md
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/build-omniroute-agent-brain/prodex-omniroute-feature-parity.md
```

6. Execução de quatro agentes:

```text
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/build-omniroute-agent-brain/tasks.md
```

7. Handover atual:

```text
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/build-omniroute-agent-brain/CLAUDE_GLM52_TL_HANDOVER.md
```

## 4. GSD legado — leitura obrigatória, não executar como target

Raiz GSD:

```text
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/.planning
```

Leia para preservar governança/evidências e identificar decisões superseded:

```text
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/.planning/PROJECT.md
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/.planning/REQUIREMENTS.md
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/.planning/ROADMAP.md
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/.planning/STATE.md
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/.planning/DECISIONS.md
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/.planning/GOLDEN_RULES.md
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/.planning/EVIDENCE_CONTRACT.md
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/.planning/TASKS_STATUS.md
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/.planning/RCA-2026-07-04-001-orchestrator-errors.md
```

Não sobrescreva estes arquivos imediatamente. Crie primeiro a proposta de rebaseline Agent Brain v3 e preserve v2.1 como histórico. Recomenda-se uma área isolada inicialmente:

```text
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/.planning/agent-brain-v3/
```

O GSD v3 deve conter `PROJECT`, `REQUIREMENTS`, `ROADMAP`, `STATE`, `DECISIONS`, `TRACEABILITY`, `COMPONENT_REGISTER`, `INTERFACE_REGISTER`, `REMOVAL_REGISTER`, `RISKS`, `EVIDENCE_CONTRACT`, `FILE_OWNERSHIP`, `AGENT_LEDGER`, `EVIDENCE_INDEX` e planos G0–G8.

## 5. Código e paths de integração

Código do daemon/backend atual:

```text
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/multica-auth-work/server
```

Principais áreas:

```text
multica-auth-work/server/internal/daemon/daemon.go
multica-auth-work/server/internal/daemon/config.go
multica-auth-work/server/internal/daemon/health.go
multica-auth-work/server/internal/daemon/client.go
multica-auth-work/server/internal/daemon/types.go
multica-auth-work/server/internal/daemon/execenv/
multica-auth-work/server/internal/daemon/prodex.go
multica-auth-work/server/internal/daemon/l2_runtime.go
multica-auth-work/server/internal/rotation/
multica-auth-work/server/pkg/agent/
multica-auth-work/server/pkg/agent/nim.go
multica-auth-work/server/pkg/agent/codex.go
multica-auth-work/server/pkg/agent/claude.go
multica-auth-work/server/pkg/agent/kimi.go
multica-auth-work/server/pkg/agent/antigravity.go
multica-auth-work/server/cmd/multica/
```

Hotspots com owner único:

```text
server/internal/daemon/daemon.go
server/internal/daemon/config.go
server/internal/daemon/health.go
server/internal/daemon/execenv/execenv.go
server/internal/daemon/execenv/codex_home.go
server/pkg/agent/models.go
server/cmd/multica/cmd_daemon.go
server/go.mod
```

Não permita dois agentes editarem qualquer hotspot simultaneamente. O Codex 1 integrador é o único owner de `daemon.go`, `config.go`, `health.go` e entrypoints centrais.

Documentos históricos de contrato/paridade Prodex:

```text
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/docs/contract/rpp-l2-v1-contract.md
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/docs/prodex/prodex-invariants.md
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/docs/prodex/prodex-runtime-invariants.md
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/Diligencias/00c_PRODEX_CRATE_COVERAGE.md
```

## 6. OmniRoute runtime e paths

Estado conhecido:

```text
Container: omniroute
Imagem declarada: diegosouzapw/omniroute:latest
Working directory interno: /app
Estado interno: /app/data
Volume host Docker: /var/lib/docker/volumes/omniroute-data/_data
Porta host: 20128
Host/WSL daemon base URL: http://127.0.0.1:20128
Container em multica_default: http://omniroute:20128
```

Não existe checkout host do source OmniRoute localizado dentro de `/mnt/c/VMs/Projects` ou no diretório de usuário pesquisado. A imagem contém o runtime em `/app`. Antes de qualquer alteração OmniRoute, obtenha o repositório/source mapping e fixe o digest da imagem; não edite o filesystem efêmero do container como source de produção.

Referência do secret, sem ler/imprimir o valor:

```text
Windows: C:\Users\dataops-lab\.omniroute\claude-code-api-key.txt
WSL: /mnt/c/Users/dataops-lab/.omniroute/claude-code-api-key.txt
```

Essa origem Windows não deve ser injetada diretamente como arquivo world-readable. O plano exige secret Linux/service com permissão restrita e somente uma chave OmniRoute nos child processes.

## 7. Arquitetura target resumida

```text
Product/API
   │
   ▼
Agent Brain — cold plane
   ├─ tasks/workspaces/repositories
   ├─ admission 20/50/100
   ├─ CLI launch/cancel/result streaming
   ├─ CLIKind + RouteModel
   └─ zero provider credentials
             │
             ▼
OmniRoute — exclusive hot plane
   ├─ provider credentials/subscriptions
   ├─ model routing/account pools
   ├─ strict RR + continuation affinity
   ├─ refresh/expiry/quota/reset
   ├─ 429/circuit/retry/fallback
   ├─ Anthropic/Responses/Chat translation
   ├─ Smart Context/token saving
   └─ telemetry/audit/security
             │
             ▼
Provider accounts
```

Prodex e legacy Go rotation não fazem parte do target final. Contudo, não podem ser removidos antes da matriz de paridade e dos gates de cutover.

## 8. Decisões e limites que não podem ser reinterpretados

1. OmniRoute substitui Prodex e todo credential management no target.
2. Agent Brain guarda uma única chave OmniRoute estável e nenhum provider key.
3. O daemon real atualmente executa CLIs no host/WSL; usa loopback, não DNS Docker.
4. `CLIKind` e `RouteModel` são conceitos diferentes.
5. Round-robin por request lógico não é limite global de concorrência.
6. Stateful continuations precisam affinity explícita.
7. Retry/fallback automático é somente pre-commit.
8. Smart Context SC01–SC10 e reset/redeem não podem desaparecer silenciosamente.
9. Tier inicial é 20. Tiers 50/100 não estão autorizados.
10. Prodex removal e gateway-required default não estão autorizados na fase inicial.
11. Código funcional existente de lifecycle/workspace/repo/cancel/streaming deve ser extraído e preservado, não reescrito cegamente.
12. Nenhum segredo ou prompt/repository content em logs/evidências.
13. Nenhum agente declara DONE sem evidência rastreável.
14. Nenhum change histórico executa em paralelo como plano master.

## 9. Disposição dos changes OpenSpec

```text
build-omniroute-agent-brain     MASTER ATIVO
native-runtimes-onboarding      INCORPORAR adapters/model discovery; remover native credentials
chat-orchestration-standard     PRESERVAR TL/Manager/delegação/escape hatch
agent-credential-isolation      ABSORVER objetivo; retirar per-provider credential ownership do Brain
persist-prodex-runtime-integration  BASELINE DE SEGURANÇA ATIVO — preservar/auditar/concluir sob lock Codex1
rotation-parity-polyglot        HISTÓRICO/PARIDADE
rotation-router                 SUPERSEDED/VAZIO
```

Paths:

```text
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/native-runtimes-onboarding/
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/chat-orchestration-standard/
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/agent-credential-isolation/
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/persist-prodex-runtime-integration/
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/rotation-parity-polyglot/
/mnt/c/VMs/Projects/RD_Agnostic_Engineering_Team/openspec/changes/rotation-router/
```

## 10. Quatro streams após autorização

| Agente | Stream | Ownership |
|---|---|---|
| Codex 1 | Lead integrator/Brain core | Contratos, facade, central daemon/config/health, merge |
| Codex 2 | OmniRoute gateway | Novo gateway client, models, capabilities, protocol/route policy |
| Codex 3 | Runtime/CLI security | Sanitizer, credentialless homes, Claude/Codex/Kimi/NIM/Agy adapters |
| Codex 4 | Operations/parity | Secrets, deploy, observability, failure/capacity evidence, runbooks |

Sequência:

```text
G0 GSD/rebaseline
→ G1 freeze
→ G2 quatro streams paralelas
→ G3 integração serial Codex 1
→ G4 protocolos/falhas/segurança/tier 20
→ G5 paridade
→ G6 cutover/removal
→ G7 tiers 50/100
→ G8 debrand
```

## 11. Primeira missão do TL — G0 concluído, iniciar freeze G1

Antes de delegar código de produto:

1. Ler todos os documentos da ordem obrigatória.
2. Confirmar por escrito que nenhuma Wave foi iniciada.
3. Registrar no novo GSD que Claude/GLM-5.2 substitui Kiro como planning/orchestration owner para este milestone, preservando o histórico.
4. Criar/revisar o GSD v3 em área isolada, com AB-REQs e G0–G8.
5. Criar traceability, component, interface, removal, ownership e evidence registers.
6. Cruzar os 85 tasks contra todos os specs, P01–P34, SC01–SC10 e changes herdados.
7. Relatar qualquer órfão, contradição ou requisito sem owner/evidência.
8. Preservar a worktree `persist-prodex-runtime-integration` conforme PD-01 e registrar lock exclusivo Codex1.
9. Executar o freeze G1; depois delegar as streams G2 com ownership disjunto e evidência obrigatória.

## 12. ETA que o TL deve gerenciar

```text
G0 rebaseline:                       2–4h
G1 freeze:                           0,5–1h
G2 quatro streams:                   3–6h
G3 integração:                       2–4h
G4 aceite tier 20:                   4–8h
Primeiro canary 20 aceito:           1–2 dias úteis
Total melhor caso:                   5–7 dias úteis
Total provável:                      8–14 dias úteis
Total com gaps relevantes:          15–25 dias úteis
```

O TL não deve prometer redução linear de 4×: implementação paraleliza; integração, failure-injection, soak/capacidade e sign-offs têm caminhos seriais.

## 13. Formato de reporte do TL ao dono

Cada reporte deve conter:

```text
phase
task IDs
agents ativos
owners/files locked
progresso real
evidence IDs
decisões tomadas
blockers
riscos novos
ETA atualizado
próxima ação que exige autorização
```

Nunca declare “completo” apenas com descrição de código. Cite artifacts/evidence reais e separe claramente reviewed, implemented, verified e accepted.

## 14. Definição final de sucesso

- Agent Brain neutro e credentialless.
- OmniRoute único hot router/credential owner.
- Claude, Codex, Kimi, GLM/NVIDIA e Antigravity aceitos por modelo/protocolo.
- Rotation, affinity, expiry, quota, 429, retry/fallback e cancellation comprovados.
- Smart Context/Prodex parity fechada ou waivers explícitos.
- Tier 20 inicialmente; 50/100 somente após gates.
- Prodex/legacy rotation removidos somente após cutover autorizado.
- Debrand completo sem quebrar consumers.
- OpenSpec, GSD, components, interfaces, tasks, owners e evidências sem órfãos.
