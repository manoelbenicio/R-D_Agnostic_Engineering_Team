# GTL-R39 — Peer review independente do plano V2 de Browser QA descartável (ORQ-39)

- Card: **ORQ-39** · UUID `c03941bc-3bde-4de1-ab19-1ba93de0ad51` · dependência declarada **ORQ-26**
- Revisor: **Kiro-Opus5** — diferente de Agy-P0-A7 (revisor do V1) e do autor Opus48#B
- Alvo: `.deploy-control/p0/evidence/gtl-browser-qa-disposable-plan.md`, 40.166 bytes
- Data UTC: 2026-07-27T13:05Z
- Modo READ-ONLY. Não editei workflow nem código, não instalei, não fiz `docker pull`, não rodei
  teste, não commitei, não fiz push, PR, merge, nem disparei Actions, não toquei banco live nem
  credencial. Nenhuma mutação de card. Única escrita: este arquivo.

---

## VEREDITO

- **DESIGN PASS — com 4 correções obrigatórias** (§C1 a §C4). Nenhuma delas invalida a arquitetura;
  duas são defeitos que fariam o job falhar na primeira execução.
- **EXECUTION BLOCK** — mantido. Nada foi executado, e três gates só fecham com execução autorizada
  (§7).

O plano é honesto, mede o que afirma e declara o que não mediu. Todos os identificadores imutáveis
que ele publica **conferem** com a minha verificação independente. As correções abaixo são de
conteúdo técnico, não de credibilidade.

---

## 1. Congelamento e reparo de citação (item 1 do dispatch)

Hash congelado **confere**:

```console
$ sha256sum .deploy-control/p0/evidence/gtl-browser-qa-disposable-plan.md
9a4f51413840b8f68cb4ca0da87dc9fdbaf6c3729bdded5dd1b37a4e1d4d95b9
```

O reparo de citação (V2.14) é **somente metadata e histórico**: cabeçalho do V2 com
`ORQ-39` + UUID + assignee + dependência `ORQ-26`, e a narrativa do identificador provisório
`ORQ-31`. O card verificado bate com o board: `ORQ-39` = `c03941bc-3bde-4de1-ab19-1ba93de0ad51`,
título "Ephemeral Browser QA & Playwright Supply-Chain Pipeline", criado 2026-07-27T12:53:27Z.
O autor declara e eu não encontrei contradição: ele não criou, deletou, renumerou nem mutou card, e
não é o registrar nomeado.

### Classificação dos `orq31-` remanescentes: **renome de implementação, não corrupção de citação**

Ocorrências medidas (10 linhas), todas em **identificadores de implementação**, nenhuma em campo de
rastreabilidade:

| linha | ocorrência | natureza |
|---|---|---|
| 332, 569, 663 | nome de arquivo `.github/workflows/orq31-browser-qa.yml` (V2.1, FILES_LOCKED, critério 1) | nome de artefato de implementação |
| 554 | `paths:` do trigger referenciando o mesmo arquivo | consequência do nome |
| 594 | `concurrency: group: orq31-browser-qa-...` | nome de grupo |
| 613 | `name: orq31-playwright-trace` | nome de artefato |
| 324, 655-657 | prosa explicando a retenção deliberada | histórico |

Isto **não é corrupção de citação**: a citação de rastreabilidade (card, UUID, dependência) foi
corrigida e está correta. São nomes de recurso que herdaram um número inválido e que **precisam ser
renomeados para `orq39-`** como mudança técnica declarada, exatamente como o autor registrou em V2.14
sem executá-la. Concordo com a decisão de não renomear dentro de um reparo de citação. A renomeação
entra em §C1.

---

## 2. Raiz sem `.github/workflows` e workflows aninhados inertes (item 2) — **CONFIRMADO**

Medido por mim, de forma independente:

```console
$ ls .github/workflows                 → NO_ROOT_WORKFLOW (não existe)
$ ls multica-auth-work/.github/workflows
ci.yml  desktop-smoke.yml  mobile-verify.yml  release.yml
```

Comportamento oficial do GitHub, citação verbatim da documentação:

> "GitHub searches the `.github/workflows` directory **in the root of your repository** for workflow
> files that are present in the associated commit SHA or Git ref of the event."
> — <https://docs.github.com/en/actions/concepts/workflows-and-actions/workflows#workflow-triggers>

E, para o disparo manual:

> "This event will only trigger a workflow run if the workflow file exists on the default branch."
> — <https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#workflow_dispatch>

Logo: os 4 workflows aninhados **estão inertes** e a afirmação V2.0 do plano está correta, assim como
a escada de autorizações E0-E5 de V2.9 e a impossibilidade de `workflow_dispatch` antes do merge.
Registro convergência: eu havia chegado à mesma conclusão de forma independente no manifesto do
ORQ-26 (`gtl-orq26-ci-commit-manifest.md` §1), antes de ler este plano.

---

## 3. Identificadores imutáveis (item 3) — **TODOS CONFEREM**

### 3.1 SHAs de Action, verificados via API oficial `git/ref/tags`

| action | versão | SHA do plano | meu resultado |
|---|---|---|---|
| `actions/checkout` | v6.1.0 | `d23441a48e516b6c34aea4fa41551a30e30af803` | idêntico ✓ |
| `actions/setup-node` | v6.5.0 | `249970729cb0ef3589644e2896645e5dc5ba9c38` | idêntico ✓ |
| `actions/setup-go` | v5.6.0 | `40f1582b2485089dde7abd97c1529aa768e1baff` | idêntico ✓ |
| `actions/upload-artifact` | v4.6.2 | `ea165f8d65b6e75b540449e92b4886f43607fa02` | idêntico ✓ |
| `pnpm/action-setup` | v4.4.0 | `a15d269cd4658e1107c09f1fabf4cbd7bd1f308a` | idêntico ✓ |

### 3.2 Digests de imagem, verificados por `HEAD` no registry (sem pull)

| imagem | digest do plano | meu resultado |
|---|---|---|
| `mcr.microsoft.com/playwright:v1.58.2-noble` | `sha256:6446946a1d9fd62d9ae501312a2d76a43ee688542b21622056a372959b65d63d` | idêntico ✓ (`HTTP/2 200`, header `docker-content-digest`) |
| `pgvector/pgvector:pg17` | `sha256:d2ef61f42ef767baa5a1475393303cc235bcd92febd9d7014eddb48b41f3bad0` | idêntico ✓ (índice OCI multi-arch; eu já o havia obtido independentemente para o ORQ-26) |

### 3.3 Guarda de lock e versão — **correta**

`package.json:43` usa `^1.58.2` (caret) e `pnpm-lock.yaml:122-123` resolve `1.58.2`: medido, confere.
O par `--frozen-lockfile` + guard `require('@playwright/test/package.json').version == 1.58.2`
é a defesa certa, e a recomendação de pin exato como mudança de outro dono está bem colocada.
`package.json` **não tem** `engines`: confere.

### 3.4 Hostname do Postgres em job-container — **correto**

Com `container:` no job, o service é resolvido pelo nome (`postgres:5432`), não `localhost`. A
remoção do bloco `ports:` é apropriada. A imagem escolhida é a mesma do produto
(`docker-compose.selfhost.yml:23`), o que evita divergência de Postgres, e traz `contrib`, de que a
migration `001_init` precisa para `pgcrypto`.

### 3.5 Caminhos raiz/`working-directory` — **corretos, com a ressalva certa**

`defaults.run.working-directory: multica-auth-work` é necessário (`playwright.config.ts:5`
`testDir: "./e2e"`, e os manifestos vivem em `multica-auth-work/`), e o plano registra
corretamente que `defaults.run` **não** se aplica a `uses:` nem a `services:`.

### 3.6 Guarda de `.env` fail-fast — **correta e bem justificada**

`e2e/env.ts:5-13` carrega `.env.worktree` ou `.env` do cwd e sobrescreveria o ambiente do CI. A
guarda **falha e nunca apaga**. Concordo integralmente: apagar mascararia checkout errado.

### 3.7 Invariante "nenhum dado de produção" — **satisfeita, e por construção**

Verifiquei a razão de fundo: `e2e/fixtures.ts:29-51` conecta em `DATABASE_URL`, executa
`DELETE FROM verification_code WHERE email = $1` e faz `SELECT` na mesma tabela. Apontar isso para o
Postgres do ORQ1 executaria `DELETE` em tabela real com um segredo vivo. A escolha de stack efêmera
completa elimina o segredo do problema em vez de protegê-lo. Nenhum `secrets.*`, nenhum
`{{resolve:secretsmanager:...}}`, `permissions: contents: read`. Aprovado.

---

## 4. Comandos e env mínimo (item 4) — **duas correções e uma prova que o plano não tinha**

### C1..C4 abaixo são as correções obrigatórias

### §C1 — Renomear os identificadores `orq31-` para `orq39-` (obrigatório)

Arquivo, `concurrency.group`, `paths:` do trigger, `name:` do artefato e a entrada correspondente em
FILES_LOCKED. Sem isso, o primeiro workflow ativo do repositório nasce carregando um número de card
inválido, contrariando o próprio ORQ-38.

### §C2 — O comando de migration está errado: falta o subcomando `up` (obrigatório)

V2.7 propõe `DATABASE_URL=... go run ./cmd/migrate`. Medido em
`server/cmd/migrate/main.go:109-115`:

```go
if len(os.Args) < 2 { ... }          // :109
direction := os.Args[1]              // :114
if direction != "up" && direction != "down" { ... }
```

O runner **exige** `up` ou `down` como argumento. Como está, o passo de migration falha antes de
aplicar qualquer coisa. Correto: `go run ./cmd/migrate up` (cwd `multica-auth-work/server`). É o
mesmo comando que o `ci.yml:131-133` já usa e que o harness GTL-51 documentou.

### §C3 — Env mínimo do backend: `APP_ENV` é obrigatório, senão o processo se recusa a subir (obrigatório)

O plano deixou "conjunto mínimo de env" como pendente de execução. **Parte disso é provável
estaticamente e eu provei:**

| variável | necessidade | âncora |
|---|---|---|
| `APP_ENV` | **obrigatória na prática.** `main.go:125-128` chama `auth.ValidateJWTConfiguration(APP_ENV, JWT_SECRET)` e faz `os.Exit(1)` em erro. `jwt.go:46-58` só dispensa o segredo quando `APP_ENV` ∈ {`dev`,`development`,`test`}; fora disso exige ≥32 bytes e não-placeholder | `cmd/server/main.go:125`, `internal/auth/jwt.go:46-58` |
| `JWT_SECRET` | só se `APP_ENV` não for dev/test; nesse caso ≥32 bytes | idem |
| `DATABASE_URL` | necessária de fato: sem ela o default é `postgres://multica:multica@localhost:5432/...` (`main.go:145-147`) e o `Ping` falha com `os.Exit(1)` em job-container | `cmd/server/main.go:145-165` |
| `PORT` | opcional, default `8080` | `cmd/server/main.go:140-143` |
| `RESEND_API_KEY` / `SMTP_HOST` | **NÃO obrigatórias.** Ausentes, o start só emite `WARN` (`main.go:129-131`), e `EmailService` cai no *dev tier*: registra e **retorna `nil`** (`internal/service/email.go:345-352`), logo `POST /auth/send-code` responde OK | `main.go:129`, `service/email.go:345-352` |
| `MULTICA_DEV_VERIFICATION_CODE` | opcional; se usada, precisa ter 6 dígitos e `APP_ENV != production` | `internal/handler/auth.go:176-190` |

Correção: fixar `APP_ENV: test` no `env:` do job (que também habilita o dev code) **ou** um
`JWT_SECRET` de ≥32 bytes descartável. O plano não menciona nenhum dos dois — como está, o backend
não sobe, porque `docker-compose.selfhost.yml` usa `APP_ENV: ${APP_ENV:-production}` e a intenção
declarada do produto é produção por default.

Ressalva honesta de escopo: isso cobre as barreiras de startup até o `Ping` do banco
(`main.go:125-165`). O que vem **depois** dessa linha eu não verifiquei linha a linha, então
"suficiente" continua sendo afirmação de execução, não minha.

### §C4 — `MULTICA_DEV_VERIFICATION_CODE` **não** dispensa o `SELECT` na tabela (obrigatório corrigir a afirmação)

V1 §6.3 afirma que a variável "dispensa o `SELECT` na tabela". Medido em `e2e/fixtures.ts:47-56`: o
fixture faz o `SELECT` **e lança** `No verification code found for <email>` quando não há linha,
**antes** de escolher entre o código do banco e o `configuredDevCode`. Ou seja, a variável só troca o
**valor** do código; o fixture continua exigindo (a) acesso ao Postgres efêmero e (b) uma linha
criada por `/auth/send-code`. Além disso o servidor também exige a linha
(`GetLatestVerificationCode` em `handler/auth.go:431-436`) mesmo no caminho de dev code. A
consequência prática é boa para o plano — a stack efêmera já dá as duas coisas —, mas a afirmação
precisa ser corrigida para não induzir alguém a remover `DATABASE_URL` do job de teste.

### Demais comandos: **conferem**

| etapa | comando do plano | verificação |
|---|---|---|
| frontend build | `pnpm --filter @multica/web build` | `apps/web/package.json:2` name `@multica/web`, `:8` `"build": "next build"` ✓ |
| frontend start | `pnpm --filter @multica/web start` | `:9` `"start": "next start"` ✓ (porta 3000 default) |
| e2e | `pnpm exec playwright test --trace=retain-on-failure --screenshot=only-on-failure --video=off --max-failures=5` | coerente com `playwright.config.ts` ✓ |
| backend | `PORT=8080 DATABASE_URL=... go run ./cmd/server` | correto **desde que** somado ao §C3 |

Observação adicional, não bloqueante: `NEXT_PUBLIC_API_URL` é lido em
`apps/web/config/runtime-urls.ts:7` e `components/web-providers.tsx:66`. Variáveis `NEXT_PUBLIC_*`
são inlinadas **no build** do Next, então a variável precisa estar no ambiente do passo de `build`,
não só do `start`. O plano coloca ambas no `env:` do job, o que satisfaz isso por acidente feliz;
vale tornar explícito. `e2e/fixtures.ts:13` também deriva `API_BASE` de `NEXT_PUBLIC_API_URL`, então
o mesmo valor serve aos dois consumidores.

---

## 5. Auditoria estática dos testes, artefato, redação, cleanup e teto de tempo (item 5)

### 5.1 Os 7 specs existentes — inventário confirmado

`auth.spec.ts` (46), `chat-attachments.spec.ts` (170), `comments.spec.ts` (64), `issues.spec.ts`
(196), `navigation.spec.ts` (43), `onboarding-v2-smoke.spec.ts` (109), `settings.spec.ts` (40) —
668 linhas no total. Nenhum deles foi executado por ninguém neste ciclo; o plano é explícito sobre
isso e coloca "7 specs verdes antes de escrever teste novo" como passo 6, o que aprovo.

A afirmação mais importante do plano sobre eles **confere**: `chat-attachments.spec.ts` fica na
camada HTTP — o cabeçalho do arquivo diz literalmente *"Stays at the HTTP layer (auth → upload-file →
send-chat-message → DB check)"* — e as asserções em `:137-141` são sobre o JSON da resposta
(`uploaded.chat_session_id`, `chat_message_id`, `url`). Logo ele **não** pode reproduzir o
`ApiContractError` do ORQ-26, que nasce no parse do cliente. A necessidade de um teste de browser
está corretamente fundamentada.

### 5.2 Os 6 testes propostos — avaliação estática

| # | teste | avaliação |
|---|---|---|
| 1 | upload no chat pela UI, com `page.route()` devolvendo `{id,url,filename}` | **É o único que reproduz o ORQ-26 e o desenho está certo.** Confirmo pela minha própria auditoria do ORQ-26: o payload degradado sem `download_url` é exatamente o que dispara `ApiContractError`. Interceptar em vez de quebrar o servidor é a escolha correta |
| 2 | botões do painel de chat | válido; é a issue ORQ-26 original. Risco de flake se depender de runtime de agente — deve assertar despacho da ação, não resposta |
| 3 | reasoning / thinking level | válido e barato |
| 4 | squad + dropdown de modelos | a ressalva do plano é a correta: sem OmniRoute, assertar "lista não vazia", nunca um número |
| 5 | delete de issue e de runtime | válido em banco efêmero; atenção ao overlap com o worktree do ORQ-18, que o plano declara |
| 6 | board/Kanban com persistência após reload | válido e de bom valor |

Lacuna que eu registro: **nenhum dos 6 cobre o contrato corrigido do ORQ-26 no lado servidor**
(id vazio + `download_url` na resposta contextless, 400 pré-upload, 500 com cleanup). Isso é
coberto por S1-S7 no worktree ORQ-26, ainda congelado. Recomendo que o plano cite essa dependência
explicitamente para não duplicar cobertura nem deixar buraco.

### 5.3 Artefato, redação e varredura — **aprovado, com um ajuste**

A política é sólida: `--trace=retain-on-failure`, `--screenshot=only-on-failure`, `--video=off`,
`retention-days: 3`, upload só em `failure()`, e **falhar** o upload quando o grep de segredo casar,
em vez de redigir. Concordo com a preferência por falhar.

Ajuste recomendado (não bloqueante): o `grep -rIlE` da V2.12 roda antes do `upload-artifact`, mas
ambos têm condições diferentes (`if: always()` vs `if: failure()`). Em job que passa, o scan roda e o
upload não; correto. Em job que falha, os dois rodam na ordem declarada; também correto. Só falta
`continue-on-error: false` explícito — é o default, então é apenas documentação. Além disso o padrão
`Bearer [A-Za-z0-9._-]{20,}` **vai** casar com traces legítimos de login, então esse gate tende a
falhar sempre que houver trace: é seguro, mas na prática significa "nunca publicar trace". Melhor
declarar isso como decisão consciente do que descobrir na primeira falha.

### 5.4 Teto de tempo e cleanup — **coerente, com o risco certo identificado**

`timeout-minutes: 25`, `--max-failures=5`, `workers: 1`, `timeout: 60000`, `concurrency` com
`cancel-in-progress: true`. O plano identifica corretamente que **`next build` é o custo dominante e
não foi medido**, e oferece plano B (job separado que publica `.next/` por artefato). Cleanup: runner
efêmero, service morre com o job, e a recusa explícita a `docker system prune` em host compartilhado
está alinhada com AA-001 §0.1.

---

## 6. FILES_LOCKED verificado

Do plano (V2.10), com a correção §C1 aplicada aos nomes:

```
.github/workflows/orq39-browser-qa.yml                         (novo, raiz)   ← renomeado
multica-auth-work/e2e/chat-upload-ui.spec.ts                   (novo)
multica-auth-work/e2e/chat-panel-buttons.spec.ts               (novo)
multica-auth-work/e2e/chat-reasoning-level.spec.ts             (novo)
multica-auth-work/e2e/squad-model-dropdown.spec.ts             (novo)
multica-auth-work/e2e/delete-flows.spec.ts                     (novo)
multica-auth-work/e2e/board-kanban.spec.ts                     (novo)
.deploy-control/p0/evidence/gtl-browser-qa-disposable-plan.md   (o próprio plano)
```

Fora do lock, exigindo outro card e outro dono: `package.json` (pin exato de Playwright,
`@axe-core/playwright`), `pnpm-lock.yaml`, `playwright.config.ts`, os 7 specs existentes, e qualquer
movimentação dos workflows aninhados. Non-overlap: nenhum arquivo colide com o worktree ORQ-26
(`server/internal/handler/file.go`, `file_test.go`) nem com o ORQ-17
(`server/internal/middleware/auth_test.go`). **Confirmo zero sobreposição** com as duas frentes que
eu mesmo revisei.

---

## 7. DESIGN PASS versus EXECUTION PASS — separação explícita

### Aprovado por desenho (não requer execução)

Topologia efêmera; hostname `postgres:5432` sem `ports:`; `working-directory`; guarda de `.env`
fail-fast; SHAs e digests pinados e verificados; `--frozen-lockfile` + guard de versão; ausência de
browser download; invariante de nenhum dado ou credencial de produção; `permissions: contents: read`;
escada de autorizações E0-E5; FILES_LOCKED e non-overlap; política de artefato, redação e cleanup;
teto de tempo; escopo dos 7 + 6 testes; condições de parada.

### Continuam sendo **gates de execução**, e ninguém pode declará-los verdes sem rodar

| # | gate | por que não fecha estaticamente |
|---|---|---|
| E-a | backend sobe de fato com o env mínimo | §C3 prova o que é *necessário* até `main.go:165`; suficiência exige processo vivo |
| E-b | duração real do `next build` e cabimento no `timeout-minutes: 25` | não medido por ninguém |
| E-c | os 7 specs existentes passam hoje | nunca executados neste ciclo |
| E-d | os 6 testes novos reproduzem o que prometem, em especial o do ORQ-26 | os testes ainda não existem |
| E-e | Node/pnpm/Go disponíveis dentro da imagem Playwright | o passo `Prove toolchain` é o desenho certo, mas a prova só existe no job |

Portanto o veredito de execução é **BLOCK**, e isso não é demérito do plano: é a consequência
correta de nada ter sido executado.

---

## 8. Autorização de governança do owner, separada, para ativar o primeiro workflow da raiz

Este é o ponto de maior consequência do card e merece decisão isolada: **ligar o primeiro workflow
ativo do repositório**. Hoje nenhum workflow roda (§2). Autorizações, em cadeia, todas do owner:

| # | ação | irreversibilidade / blast radius |
|---|---|---|
| G1 | criar `.github/workflows/orq39-browser-qa.yml` na raiz, em branch de feature | local, reversível |
| G2 | `git push` da branch — **transmissão de código para remoto público** | reversível por delete da branch |
| G3 | abrir PR — a partir daqui o job roda pelo trigger `pull_request` | runs efêmeros; primeiro consumo de minutos de Actions |
| G4 | **merge em `main` — liga o GitHub Actions no repositório de forma permanente** e faz o `workflow_dispatch` existir | decisão de governança, não técnica; muda o modelo de execução do repo |
| G5 | primeira execução manual pós-merge | efêmera |
| G6 | *(separado, outro card)* pin exato de Playwright e `@axe-core/playwright` em `package.json`/`pnpm-lock.yaml` | outro dono |

Recomendo tratar **G4** como item próprio na tabela de decisões do owner, com uma consideração que o
plano não levanta: com o primeiro workflow na raiz, o `paths:` proposto inclui
`multica-auth-work/server/**` e `apps/web/**`, o que fará o job rodar em PRs de outras frentes —
inclusive o ORQ-26 e o ORQ-17. Isso é desejável a médio prazo, mas no primeiro merge significa que
duas frentes congeladas passam a depender de um pipeline nunca executado. Sugiro, para o primeiro
merge, restringir `paths:` ao próprio workflow e a `multica-auth-work/e2e/**`, e ampliar depois.

---

## 9. Correções obrigatórias, consolidadas

| # | correção | severidade |
|---|---|---|
| **C1** | Renomear `orq31-` → `orq39-` no nome do arquivo, `concurrency.group`, `paths:`, nome do artefato e FILES_LOCKED | obrigatória, cosmética-mas-de-rastreabilidade |
| **C2** | `go run ./cmd/migrate` → **`go run ./cmd/migrate up`** (`cmd/migrate/main.go:109-115`) | obrigatória, o job falharia |
| **C3** | Adicionar `APP_ENV: test` (ou `JWT_SECRET` ≥32 bytes) ao `env:` do job (`main.go:125-128`, `jwt.go:46-58`) | obrigatória, o backend não subiria |
| **C4** | Corrigir a afirmação de que `MULTICA_DEV_VERIFICATION_CODE` dispensa o `SELECT` (`e2e/fixtures.ts:47-56`) | obrigatória, é afirmação factual errada |
| R1 | Declarar `NEXT_PUBLIC_API_URL` explicitamente no passo de `build` (inline de `NEXT_PUBLIC_*`) | recomendada |
| R2 | Declarar que o padrão `Bearer …` do secret-scan implica, na prática, nunca publicar trace | recomendada |
| R3 | Citar a dependência do ORQ-26 (S1-S7 server-side) para não duplicar nem deixar buraco de cobertura | recomendada |
| R4 | Restringir `paths:` no primeiro merge ao workflow e a `e2e/**` | recomendada |

---

## Check-out — ORQ-39 · `c03941bc-3bde-4de1-ab19-1ba93de0ad51`

- Veredito: **DESIGN PASS com C1-C4 obrigatórias; EXECUTION BLOCK** (gates E-a a E-e).
- Revisor distinto de Agy-P0-A7 e do autor, conforme exigido.
- Verificado de forma independente: hash congelado, card e UUID no board, ausência de
  `.github/workflows` na raiz, inércia dos 4 workflows aninhados com citação oficial do GitHub,
  5 SHAs de action, 2 digests de imagem, pins de pnpm/Node/Go/Playwright, ausência de `engines`,
  comandos de build/start/e2e, assinatura do runner de migration, env mínimo de startup do backend,
  comportamento do EmailService sem backend, fluxo do fixture de login, inventário dos 7 specs e a
  natureza HTTP-only do `chat-attachments.spec.ts`.
- Não executei: nenhum teste, build, install, pull, workflow, DB live ou credencial. Nenhuma mutação
  de card. Nenhum arquivo do plano alterado — o congelamento segue em
  `9a4f51413840b8f68cb4ca0da87dc9fdbaf6c3729bdded5dd1b37a4e1d4d95b9`.
- Dependências que continuam congeladas e intocadas por mim: worktree ORQ-26 e worktree ORQ-17.
