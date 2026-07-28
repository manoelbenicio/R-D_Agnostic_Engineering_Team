# Plano de QA de browser em ambiente DESCARTAVEL - ORQ-26 / board / chat

Autor: Opus48#B (ORQ2, pane w6:p2) · UTC 2026-07-27T12:40Z
Modo: READ-ONLY / DESENHO. **Nao instalei nada, nao rodei Playwright, nao baixei browser, nao dei
push, nao executei workflow.** Nada foi instalado no ORQ1 nem no ORQ2. Os unicos arquivos criados sao
este e o meu check-out.
Escritor/integrador: Codex56-TL (GENERAL-TECH-LEAD). Autoridade final: owner humano.

---

## 1. PRINCIPIO: O AMBIENTE E DESCARTAVEL, A STACK TAMBEM

O ponto que decide o desenho inteiro: **o ambiente descartavel precisa trazer a stack junto**, nao
apontar para ORQ1/ORQ2. Dois motivos medidos, nao estilisticos:

1. `playwright.config.ts:19-21` (literal):
   ```ts
   // Don't auto-start servers — they must be running already
   // This avoids complexity and port conflicts during testing
   ```
   Nao ha `webServer`. O runner **nao** sobe nada; alguem tem de servir `baseURL`.
2. O login de teste **fala com o banco**. `e2e/fixtures.ts:29-51` (literal, recortado):
   ```ts
   async login(email: string, name: string) {
     const client = new pg.Client(DATABASE_URL);
     await client.connect();
     ...
       await client.query("DELETE FROM verification_code WHERE email = $1", [email]);
       const sendRes = await fetch(`${API_BASE}/auth/send-code`, { ... });
       const result = await client.query(
         "SELECT code FROM verification_code WHERE email = $1 AND used = FALSE AND expires_at > now() ORDER BY created_at DESC LIMIT 1",
         [email],
       );
   ```
   Ou seja: o suite **le o codigo de verificacao direto da tabela `verification_code`**, com
   `DATABASE_URL`, e ainda **DELETA** linhas dessa tabela por e-mail.

Consequencia dura, e e a recomendacao central deste plano: apontar o QA descartavel para o Postgres
do ORQ1 seria (a) usar um segredo vivo — o mesmo `DATABASE_URL` que o TL registrou como exposto em
historico e pendente de rotacao — e (b) executar `DELETE` em tabela de producao. **Ambos inaceitaveis.**

Portanto: **stack efemera completa dentro do job**, com Postgres proprio. Isso elimina o segredo do
problema em vez de proteger o segredo — nao existe credencial de producao no job.

## 2. TOPOLOGIA DO JOB DESCARTAVEL

```
runner efemero (GitHub Actions ubuntu-24.04, ou container throwaway em host de CI)
├── service: postgres:17.x@sha256:<digest>      -> banco novo, vazio, morre com o job
├── step:    migrate  (cmd/migrate do repo)     -> aplica migrations no banco efemero
├── service/step: backend (docker-compose.selfhost.build.yml)  -> :8080
├── step:    next build + start (apps/web)      -> :3000  == PLAYWRIGHT_BASE_URL
└── container: mcr.microsoft.com/playwright:v1.58.2-noble@sha256:<digest>
              -> Chromium ja embutido, NENHUM download de browser
```
Nada disso toca ORQ1 ou ORQ2. Sem SSH para os hosts, sem tunel, sem `~/.agent-cred-homes`, sem
OmniRoute. O QA de browser exercita UI + backend + DB; **nao** exercita inferencia.

Alternativa aceitavel se nao houver CI: um unico container descartavel em host de CI dedicado, com
`--rm` e `--network` propria. **Nao** no ORQ1 nem no ORQ2, porque `/tmp` do ORQ2 esta 100% cheio
(`tmpfs 7.7G 7.7G 0 100%`, medido) e o ORQ1 esta com disco em 75% — nenhum dos dois tem folga para
imagem de Playwright, que passa de 1 GB.

## 3. VERSOES E LOCK - RISCO REAL DE MISMATCH

Medido no repo:
```
package.json:43        "@playwright/test": "^1.58.2"
pnpm-lock.yaml:122-123 specifier: ^1.58.2   version: 1.58.2
package.json           packageManager: pnpm@10.28.2
```
**Defeito de supply chain a corrigir antes de rodar:** o especificador e `^1.58.2` (caret). O lock
resolve `1.58.2` hoje, mas qualquer `pnpm install` sem `--frozen-lockfile` pode subir para `1.59.x`,
e a imagem `playwright:v1.58.2` traz o **Chromium daquela versao**. Playwright exige que a versao do
pacote case com a do driver/browser da imagem; divergencia gera erro de "Executable doesn't exist" ou
comportamento nao reproduzivel.

Regras obrigatorias:
1. `pnpm install --frozen-lockfile` **sempre**. Falha o job se o lock nao satisfizer o `package.json`.
2. A tag da imagem tem de ser derivada do lock, nao escrita a mao. Passo de guarda:
   ```bash
   PW=$(node -e 'console.log(require("./node_modules/@playwright/test/package.json").version)')
   test "$PW" = "1.58.2" || { echo "playwright $PW != imagem v1.58.2"; exit 1; }
   ```
3. Recomendo trocar `^1.58.2` por `1.58.2` exato no `package.json`. Isso e mudanca de arquivo e
   **nao esta neste plano** — e proposta para o escritor unico.

## 4. CACHE DE BROWSER

- **Preferencia: nenhum cache.** A imagem `mcr.microsoft.com/playwright:v1.58.2-noble` ja contem os
  browsers em `/ms-playwright`. Nao rodar `playwright install`; nao baixar nada. Isso remove uma
  superficie de rede e de supply chain inteira.
- Se, por algum motivo, o job usar runner nu em vez da imagem: `PLAYWRIGHT_BROWSERS_PATH=$HOME/.cache/ms-playwright`
  com chave de cache **incluindo a versao exata do pacote** (`ms-playwright-1.58.2-<os>`), e
  `playwright install --with-deps chromium` **apenas chromium**. Chave sem versao e a causa classica
  de browser velho com pacote novo.
- **Somente Chromium.** `playwright.config.ts:13-18` declara um unico projeto `chromium`. Instalar
  Firefox/WebKit seria custo e download sem teste que os use.
- Cache de pnpm (`~/.pnpm-store`) e opcional e seguro, chaveado por hash do `pnpm-lock.yaml`.

## 5. RESTRICOES DA RAIZ DO WORKFLOW (config existente, nao inventada)

`playwright.config.ts` fixa quatro coisas que o job precisa respeitar:

| linha | valor | consequencia para o job |
|---|---|---|
| `:5` | `testDir: "./e2e"` | `cwd` do comando tem de ser `multica-auth-work/`, senao 0 testes encontrados |
| `:6` | `timeout: 60000` | 60 s por teste; um teste de upload lento falha por timeout, nao por bug |
| `:7` | `workers: 1` | serial. O tempo total e a soma dos testes; nao ha ganho em runner grande |
| `:8` | `retries: 0` | flake vira falha vermelha. Bom para honestidade, ruim para sinal — ver 10.3 |
| `:10` | `baseURL` de `PLAYWRIGHT_BASE_URL` / `FRONTEND_ORIGIN` / `localhost:3000` | o job deve setar `PLAYWRIGHT_BASE_URL` explicitamente |

E `e2e/env.ts:5-12` carrega `.env.worktree` ou `.env` do `cwd`. **No job descartavel esses arquivos
nao devem existir** — se existirem, sobrescrevem silenciosamente o ambiente do CI. Passo de guarda:
falhar se `multica-auth-work/.env` ou `.env.worktree` estiver presente no checkout.

`workers: 1` combina com o isolamento por worker que os helpers ja fazem
(`e2e/helpers.ts:4-8`): `DEFAULT_E2E_EMAIL = e2e-${E2E_WORKER}-${E2E_RUN_ID}@multica.ai` e
`DEFAULT_E2E_WORKSPACE = e2e-workspace-${E2E_WORKER}-${E2E_RUN_ID}`. Cada execucao ja cria e-mail e
workspace unicos — bom, e reduz colisao mesmo se um dia subir para 2 workers.

## 6. AUTENTICACAO SEM SEGREDO EM CONTEXTO

O mecanismo real, ja no repo, e favoravel: **nao existe senha**. O fluxo e e-mail + codigo de
verificacao lido do banco.

Desenho:
1. `DATABASE_URL` do job aponta para o **Postgres efemero do proprio job**
   (`postgres://postgres:postgres@localhost:5432/multica_test`). Isso nao e segredo: e credencial
   descartavel de um banco que nasce e morre no job, sem dado real.
2. Nenhum segredo de producao entra no job. Zero uso de `secretsmanager`, zero
   `{{resolve:secretsmanager:...}}`, zero `DATABASE_URL` de ORQ1. Nada a rotacionar depois.
3. `MULTICA_DEV_VERIFICATION_CODE` (`fixtures.ts:56`) pode fixar o codigo no ambiente efemero,
   dispensando o `SELECT` na tabela. Recomendo usar, porque encurta o login e reduz acoplamento ao
   schema de `verification_code`.
4. **Cookies e `storageState`**: `auth.spec.ts`/`helpers.ts` injetam o token via
   `page.addInitScript` (`helpers.ts:51`). Se o plano futuro adotar `storageState` para acelerar,
   o arquivo de estado **e credencial** — tem de ser gerado dentro do job, ficar fora dos artefatos e
   nunca ser cacheado. Regra: `storageState` em `$RUNNER_TEMP`, nunca em `test-results/`.
5. Nada de conta de teste "real" compartilhada. Cada execucao cria seu proprio usuario
   `e2e-<worker>-<runid>@multica.ai` em banco vazio.

**Consequencia importante e honesta:** este QA valida UI + backend + DB. Ele **nao** valida OmniRoute,
credencial de provedor, slot ou rotacao — nada disso existe no job. Um teste de chat que exigisse
resposta real de modelo esta fora de escopo; ver 7.3.

## 7. TESTES A EXECUTAR

### 7.1 Ja existentes, reaproveitar (7 specs)
`auth.spec.ts`, `chat-attachments.spec.ts`, `comments.spec.ts`, `issues.spec.ts`,
`navigation.spec.ts`, `onboarding-v2-smoke.spec.ts`, `settings.spec.ts`.

`chat-attachments.spec.ts` e diretamente relevante para o ORQ-26: ele exercita
`POST /api/upload-file` e ja assere `uploaded.url` e o vinculo `chat_session_id` (linhas 132-141).
**Mas ele fica na camada HTTP** (o proprio cabecalho do arquivo diz "Stays at the HTTP layer"),
portanto **nao** reproduz o `ApiContractError` do ORQ-26, que ocorre no parse do cliente
(`packages/core/api/schema.ts:56`). Para o ORQ-26 e preciso um teste de **browser**, nao de HTTP.

### 7.2 Novos, na ordem de valor para o ORQ-26
1. **upload no chat pela UI** — o unico teste que reproduz o ORQ-26. Deve exercitar as duas rotas de
   fallback que eu identifiquei no root cause: (a) resposta sem `download_url` e (b) resposta sem
   contexto de workspace. Como o servidor so cai nessas rotas em condicao de falha, o caminho pratico
   e `page.route()` interceptando `**/api/upload-file` e devolvendo o payload degradado
   `{"id":"...","url":"...","filename":"..."}` — e assere que a UI mostra erro honesto em vez de
   quebrar. Isso testa o comportamento do cliente sem precisar quebrar o servidor.
2. **botoes do painel de chat inoperantes** — e uma issue `in_progress` **sem nenhuma task** (medi
   isso no T3). Teste: cada botao do painel dispara a acao esperada (envio, anexar, cancelar).
3. **reasoning / thinking level** — o seletor renderiza os niveis suportados e persiste a escolha.
   Assere que o valor escolhido chega ao payload de criacao, sem afirmar nada sobre custo.
4. **squad** — criar squad, adicionar membro, e o dropdown de modelos **popular**. Aqui vale a
   ressalva do GTL-03: em ambiente efemero sem OmniRoute, o dropdown so tera catalogo estatico
   (claude/codex). Assere que a lista **nao esta vazia**, nao um numero especifico.
5. **delete** — deletar issue e deletar runtime (ORQ-18 esta em worktree separado); assere confirmacao
   e desaparecimento da lista, sem tocar dado real porque o banco e efemero.
6. **board / Kanban** — mover cartao entre colunas e persistir apos reload. Cobre a familia de
   inconsistencias issue↔status que eu medi no T3 (3 issues `blocked` com task `completed`).

### 7.3 Explicitamente fora
Qualquer teste que exija resposta de modelo real. Sem OmniRoute no job, um "chat responde" seria
falso. Testar chat de ponta a ponta com inferencia e outra frente, com gate proprio.

## 8. SCREENSHOTS E TRACES COM REDACTION

Config recomendada para o job (via CLI/env, sem editar o config do repo):
```
--trace=retain-on-failure  --screenshot=only-on-failure  --video=off
```
Riscos concretos e mitigacao:
1. **Trace contem corpo de requisicao e header.** O `Authorization: Bearer <token>` do login e o
   `storageState` apareceriam. Mitigacao: `retain-on-failure` (nao `on`), e o token e de usuario
   efemero em banco efemero — expira com o job e nao da acesso a nada real. Ainda assim, o artefato
   nao deve ser publico.
2. **Artefato de PR e publico em repo publico.** Regra: `retention-days: 3` e artefato privado; se o
   repo for publico, **nao** anexar trace, apenas o relatorio JSON de PASS/FAIL.
3. **Screenshot pode capturar e-mail e nome do usuario de teste.** Sao sinteticos
   (`e2e-<worker>-<runid>@multica.ai`), logo aceitavel.
4. Nunca ligar `--video=on`: custo alto e nenhuma informacao que o trace nao de.
5. Passo de varredura antes do upload do artefato: grep por padroes de segredo
   (`AKIA[0-9A-Z]{16}`, `sk-[A-Za-z0-9]{20,}`, `Bearer [A-Za-z0-9._-]{20,}`,
   `postgres://[^ ]*:[^ ]*@`) em `test-results/`; se casar, **falhar o upload**, nao redigir
   silenciosamente. Falhar e mais seguro do que confiar num regex de redacao.

## 9. FERRAMENTAS ADICIONAIS - COM JUSTIFICATIVA E LIMITE

| ferramenta | veredito | justificativa |
|---|---|---|
| `@axe-core/playwright` | **RECOMENDO** | O repo tem requisito explicito de acessibilidade. axe roda no mesmo browser da suite, sem servico externo, custo de segundos por pagina. Regra: assere **zero violacoes `critical`/`serious`** e trate `moderate`/`minor` como relatorio, senao o gate vira ruido |
| visual diff (`toHaveScreenshot`) | **RECOMENDO COM ESCOPO ESTREITO** | Nativo do Playwright, sem dependencia nova. Mas so para 2-3 componentes estaveis (painel de chat vazio, coluna do board). Snapshot de pagina inteira em CI e fonte cronica de flake por fonte e antialias; e com `retries: 0` cada flake vira vermelho. Exigir `--update-snapshots` deliberado e baseline commitada por plataforma |
| Lighthouse | **NAO RECOMENDO AGORA** | Mede performance de rede/render, e a app roda em `next start` de runner compartilhado com Postgres e backend no mesmo host: o numero mediria o runner, nao o produto. Se o objetivo for orcamento de performance, o lugar e um job dedicado com hardware estavel. Como auditoria pontual de a11y/SEO, `axe` ja cobre a11y melhor |
| `@lhci/cli` | **NAO** | Mesma objecao, mais um servidor de historico para manter |
| `playwright-msw` / mock de rede | **NAO PRECISA** | `page.route()` nativo resolve o caso do ORQ-26 (secao 7.2.1) sem dependencia nova |

Principio aplicado: cada ferramenta nova e superficie de supply chain. Duas adicoes (`axe`, e o visual
diff que ja vem no Playwright) sao justificaveis; Lighthouse nao, pelo ambiente.

## 10. SUPPLY CHAIN, CUSTO E CLEANUP

### 10.1 Pins e digests
- imagem Playwright: `mcr.microsoft.com/playwright:v1.58.2-noble@sha256:<digest>` — **tag + digest**,
  nunca so tag. Tag e mutavel; digest nao.
- `postgres:17.x@sha256:<digest>` idem.
- Actions de terceiro fixadas por **SHA de commit**, nunca por `@v4`.
- `pnpm install --frozen-lockfile`; `pnpm@10.28.2` via `packageManager` do `package.json`
  (Corepack), nao instalado a mao.
- `@axe-core/playwright` entra no lock com versao **exata**, e o job roda com o lock congelado.
- Nao coletei nenhum digest neste documento: coletar exige `docker pull`/consulta de registry, que
  nao fiz. **Os `<digest>` acima sao placeholders a preencher por quem executar** — declarado, nao
  inventado.

### 10.2 Custo
- Imagem Playwright > 1 GB: dominado por pull. Em runner hospedado, ~1-2 min de pull a frio.
- `workers: 1` com `timeout: 60000`: 7 specs existentes + 6 novos, na pior hipotese de 60 s cada,
  dao teto teorico bem abaixo de 20 min. Recomendo `timeout-minutes: 25` no job e
  `--max-failures=5` para nao queimar minuto depois que a suite ja esta claramente vermelha.
- Custo real e minuto de runner + pull, nenhum custo de provedor de modelo, porque nao ha inferencia.

### 10.3 Flake vs `retries: 0`
`retries: 0` esta no config do repo e eu **nao** proponho alterar o arquivo. Mas registro o
tradeoff: em CI, `retries: 1` com `retain-on-failure` distingue flake de bug real sem esconder nada
(o relatorio marca `flaky`). Se o General-TL preferir manter `0`, esperar vermelho ocasional por
timing e nao tratar como regressao sem olhar o trace.

### 10.4 Cleanup
- Runner efemero: destruido pelo provedor. Nada a limpar.
- Container throwaway: `docker run --rm`, rede dedicada, `docker compose down -v` no `always()`.
- **Nao** rodar `docker system prune` em host compartilhado: prune e classe STOP-AND-WAIT e o
  ORQ1 guarda as imagens de rollback do OmniRoute.
- Dados de teste: o banco morre com o job. Os helpers ja criam usuario e workspace unicos por
  execucao, entao nem ha residuo a limpar.
- Artefatos: `retention-days: 3`.

## 11. SEQUENCIA PROPOSTA (nada executado)

1. Owner autoriza; General-TL define onde o job roda.
2. Corrigir `^1.58.2` -> `1.58.2` e adicionar `@axe-core/playwright` com versao exata (**mudanca de
   arquivo, do escritor unico**).
3. Coletar os digests reais das duas imagens e fixa-los no workflow.
4. Escrever o workflow descartavel: Postgres efemero -> migrate -> backend -> web -> Playwright.
5. Guardas: `--frozen-lockfile`; versao do pacote == tag da imagem; ausencia de `.env`/`.env.worktree`.
6. Rodar os 7 specs existentes. Verde antes de escrever teste novo.
7. Adicionar os 6 testes da secao 7.2, comecando pelo upload do chat (ORQ-26).
8. Ligar `axe` com gate em `critical`/`serious`.
9. Visual diff em 2-3 componentes, baseline commitada deliberadamente.
10. Varredura de segredo nos artefatos antes do upload.

## 12. LIMITES E NAO-AFIRMACOES
- **Nao instalei, nao rodei, nao baixei browser, nao dei push, nao executei workflow.** Nada instalado
  no ORQ1/ORQ2.
- Nao coletei digests de imagem: exigiria `docker pull` ou consulta a registry. Placeholders declarados.
- Nao verifiquei o **formato da tag** `mcr.microsoft.com/playwright:v1.58.2-noble` contra o registry.
  O padrao `v<versao>-<codename>` e o usado pela Microsoft, mas o codename correto para 1.58.2 e a
  existencia da tag precisam ser confirmados no passo 3, junto do digest.
- Nao executei nenhum dos 7 specs existentes: nao sei se passam hoje. O passo 6 existe por isso.
- Nao li nenhum valor de segredo. `DATABASE_URL` e `MULTICA_DEV_VERIFICATION_CODE` aparecem aqui como
  **nomes** de variavel, lidos do codigo; nunca acessei os valores.
- Nao inspecionei `.github/workflows/ci.yml`: nao sei se ja existe job de e2e nem qual runner e usado.
  O plano assume runner novo e dedicado; integrar com o CI existente pode mudar detalhes.
- Nao verifiquei se o backend sobe corretamente a partir de
  `docker-compose.selfhost.build.yml` num runner limpo; isso e pre-condicao do passo 4 e precisa de
  validacao real.
- Nao afirmo que os 6 testes novos reproduzem o ORQ-26: o de upload e desenhado para reproduzir, com
  `page.route()` devolvendo o payload degradado que eu identifiquei no root cause, mas so a execucao prova.
- Nao alterei nenhum arquivo do repo, incluindo `playwright.config.ts` e `package.json`. Todas as
  mudancas de arquivo listadas sao propostas para o escritor unico.
- Nao toquei nas 9 issues preservadas (ORQ-12, 13, 15, 16, 17, 18, 21, 22, 23) e nao disparei rerun.

---
---

# V2 - CORRECAO PARA FECHAR GTL-R31A (ORQ-39)

Autor: Opus48#B (ORQ2, pane w6:p2) · UTC 2026-07-27T12:50Z
Cartao: **ORQ-39** · UUID `c03941bc-3bde-4de1-ab19-1ba93de0ad51` · assignee Opus48-B
Dependencia: **ORQ-26** (root cause do ApiContractError em uploadFile, que o teste de browser deste cartao existe para reproduzir)
Citacao anterior: `ORQ-31` era **provisoria e invalida** e foi corrigida (ver secao V2.14)
Veredito anterior: **BLOCK** (GTL-R31A). Este V2 responde item por item.
Modo: READ-ONLY. **Nao criei workflow, nao editei codigo, nao instalei, nao baixei, nao dei pull,
nao rodei teste, nao commitei, nao dei push, nao abri PR, nao fiz merge, nao fiz deploy, nao acessei
ambiente live.** Todos os identificadores abaixo vieram de consulta **read-only** a registry oficial
e a `git ls-remote` oficial - nenhuma imagem foi baixada.

## V2.0 ACHADO ESTRUTURAL QUE MUDA O PLANO (o mais importante deste V2)

O repositorio e `github.com/manoelbenicio/R-D_Agnostic_Engineering_Team`, default branch **`main`**.
Medido:
```
$ git ls-files | grep -E '^\.github/workflows/'
(vazio)
$ git ls-files | grep -cE '^multica-auth-work/\.github/workflows/'
4
$ find .github -maxdepth 2
.github/CODEOWNERS
.github/prompts/...
.github/skills/...
```
**Nao existe `.github/workflows/` na raiz do repositorio.** Os 4 workflows (`ci.yml`,
`desktop-smoke.yml`, `mobile-verify.yml`, `release.yml`) estao em
`multica-auth-work/.github/workflows/`, e o GitHub Actions **so le `.github/workflows/` na raiz**.
Portanto:
1. Esses 4 workflows estao **inertes** neste repositorio. O `ci.yml` que fixa `node-version: 22`,
   `go-version: "1.26.1"` e `pnpm/action-setup@v4` nao roda hoje.
2. A conclusao do plano V1 de que "nao existe job e2e" era certa, mas por motivo mais forte do que
   eu supunha: **nenhum job roda**.
3. O `orq31-browser-qa.yml` na raiz seria o **primeiro** workflow ativo do repositorio. Isso muda a
   avaliacao de risco: nao ha CI existente para regredir, e tambem nao ha CI provado para reutilizar.

Nao afirmo por que os workflows estao aninhados - pode ser vestigio de import de subarvore. Nao
alterei nada e nao proponho move-los neste cartao.

## V2.1 Caminho do workflow e working-directory

Arquivo: **`.github/workflows/orq31-browser-qa.yml`** (raiz do repositorio, obrigatorio para o
Actions enxergar).

```yaml
defaults:
  run:
    working-directory: multica-auth-work
```
Necessario porque `playwright.config.ts:5` fixa `testDir: "./e2e"` relativo ao cwd, e o
`package.json`/`pnpm-lock.yaml` vivem em `multica-auth-work/`. Sem isso, `pnpm` nao encontra manifesto
e o Playwright encontra 0 testes.

Ressalva de precisao: `defaults.run` **nao** se aplica a `uses:` nem a `services:`. Passos de action
que recebem caminho (`actions/setup-node` com `cache-dependency-path`, `actions/upload-artifact` com
`path`) precisam do caminho explicito `multica-auth-work/...`.

## V2.2 URL do banco dentro do job-container

O job roda **dentro de container**, logo o hostname e o **nome do service**, nao `localhost`:
```yaml
services:
  postgres:
    image: pgvector/pgvector:pg17@sha256:d2ef61f42ef767baa5a1475393303cc235bcd92febd9d7014eddb48b41f3bad0
    env:
      POSTGRES_USER: multica
      POSTGRES_PASSWORD: multica-ephemeral
      POSTGRES_DB: multica
    options: >-
      --health-cmd "pg_isready -U multica -d multica"
      --health-interval 5s --health-timeout 5s --health-retries 20
    # SEM bloco `ports:` — nada precisa ser exposto no host
env:
  DATABASE_URL: postgres://multica:multica-ephemeral@postgres:5432/multica?sslmode=disable
```
Duas correcoes frente ao V1:
- host **`postgres:5432`**, nao `localhost:5432`. Em job-container o service e resolvido pelo nome na
  rede do job; `localhost` apontaria para o proprio container de trabalho.
- **sem `ports:`**. Mapear porta para o host nao serve a ninguem aqui e so cria colisao. O
  `docker-compose.selfhost.yml:6` mapeia `127.0.0.1:${BACKEND_PORT:-8080}:8080` porque ali ha
  consumidor externo; no job nao ha.

A senha `multica-ephemeral` **nao e segredo**: e credencial de um banco que nasce vazio e morre com o
job. Nao vai para Secrets, nao e reutilizada, nao existe fora do job.

## V2.3 Guarda de `.env` - falhar, nunca apagar

`e2e/env.ts:5-12` (literal) carrega `.env.worktree` ou `.env` do cwd e **sobrescreveria** o ambiente
do CI em silencio:
```ts
const envCandidates = [".env.worktree", ".env"];
for (const filename of envCandidates) {
  const path = resolve(process.cwd(), filename);
  if (existsSync(path)) { config({ path }); break; }
}
```
Guarda obrigatoria, **antes** de qualquer passo que rode teste:
```yaml
- name: Fail if a dotenv file would override CI env
  run: |
    set -euo pipefail
    for f in .env .env.worktree; do
      if [ -e "$f" ]; then
        echo "::error::$f is present and would override CI env (e2e/env.ts:5-12). Refusing to run."
        echo "This step never deletes the file — remove it deliberately or fix the checkout."
        exit 1
      fi
    done
    echo "no dotenv override present"
```
Regra explicita: **nunca `rm`**. Apagar mascararia um checkout errado e destruiria um arquivo que
pode ser de outra pessoa. Falhar e a acao correta.

## V2.4 Identificadores IMUTAVEIS - todos verificados agora, read-only

### Imagens (digest obtido por `HEAD` no registry oficial, sem pull)
| imagem | tag | digest verificado | fonte |
|---|---|---|---|
| Playwright | `v1.58.2-noble` | `sha256:6446946a1d9fd62d9ae501312a2d76a43ee688542b21622056a372959b65d63d` | `mcr.microsoft.com/v2/playwright/manifests/v1.58.2-noble` -> `HTTP/2 200`, header `docker-content-digest` |
| pgvector | `pg17` | `sha256:d2ef61f42ef767baa5a1475393303cc235bcd92febd9d7014eddb48b41f3bad0` | `registry-1.docker.io/v2/pgvector/pgvector/manifests/pg17` -> `HTTP/2 200`, header `docker-content-digest` |

A existencia da tag `v1.58.2-noble` foi confirmada em
`mcr.microsoft.com/v2/playwright/tags/list`, que lista `v1.58.2`, `v1.58.2-noble`,
`v1.58.2-noble-amd64`, `v1.58.2-noble-arm64`. Isso fecha a nao-afirmacao do V1 sobre o formato da tag.
`pgvector/pgvector:pg17` e exatamente a imagem que `docker-compose.selfhost.yml:23` usa, logo o job
nao introduz Postgres diferente do produto.

Uso obrigatorio: **tag + digest**, nunca so tag.
```yaml
container:
  image: mcr.microsoft.com/playwright:v1.58.2-noble@sha256:6446946a1d9fd62d9ae501312a2d76a43ee688542b21622056a372959b65d63d
```

### Actions (SHA completo via `git ls-remote --tags` oficial)
Pinados na **mesma linha de major que o repo ja adota**, para nao divergir do `ci.yml`:

| action | versao | SHA imutavel |
|---|---|---|
| `actions/checkout` | v6.1.0 | `d23441a48e516b6c34aea4fa41551a30e30af803` |
| `actions/setup-node` | v6.5.0 | `249970729cb0ef3589644e2896645e5dc5ba9c38` |
| `actions/setup-go` | v5.6.0 | `40f1582b2485089dde7abd97c1529aa768e1baff` |
| `actions/upload-artifact` | v4.6.2 | `ea165f8d65b6e75b540449e92b4886f43607fa02` |
| `pnpm/action-setup` | v4.4.0 | `a15d269cd4658e1107c09f1fabf4cbd7bd1f308a` |

Forma obrigatoria, SHA com comentario de versao:
```yaml
- uses: actions/checkout@d23441a48e516b6c34aea4fa41551a30e30af803 # v6.1.0
```
Nenhum `@v4`/`@v6` mutavel. Nenhuma action de terceiro alem de `pnpm/action-setup`.

## V2.5 Versoes exatas de ferramenta e prova de disponibilidade

Medido nos manifestos do repo:
| ferramenta | valor | fonte |
|---|---|---|
| pnpm | **10.28.2** | `package.json` -> `"packageManager": "pnpm@10.28.2"` |
| Node | **22** | `multica-auth-work/.github/workflows/ci.yml:43` -> `node-version: 22` (workflow inerte, mas e a intencao declarada) |
| Go | **1.26.1** | `server/go.mod` -> `go 1.26.1`; e `ci.yml:108` -> `go-version: "1.26.1"` |
| Playwright | **1.58.2** | `pnpm-lock.yaml:122-123` -> `specifier: ^1.58.2`, `version: 1.58.2` |

`package.json` **nao tem** campo `engines` (medido), logo a versao de Node nao e imposta pelo
manifesto - mais uma razao para fixa-la no workflow.

**Nao ha prova de disponibilidade dentro da imagem Playwright**, e eu nao posso obte-la sem `docker
pull`, que este cartao proibe. Portanto o V2 **nao assume** nada e adiciona setup pinado explicito:
```yaml
- uses: actions/setup-node@249970729cb0ef3589644e2896645e5dc5ba9c38 # v6.5.0
  with:
    node-version: 22
    cache: pnpm
    cache-dependency-path: multica-auth-work/pnpm-lock.yaml
- uses: pnpm/action-setup@a15d269cd4658e1107c09f1fabf4cbd7bd1f308a # v4.4.0
  with:
    version: 10.28.2
- uses: actions/setup-go@40f1582b2485089dde7abd97c1529aa768e1baff # v5.6.0
  with:
    go-version: "1.26.1"
    cache-dependency-path: multica-auth-work/server/go.sum
- name: Prove toolchain
  run: node -v && pnpm -v && go version
```
O passo `Prove toolchain` e a prova exigida, executada no job, e nao uma suposicao minha.
Se `setup-go` nao funcionar dentro da imagem Playwright, a alternativa e um **job separado** que
compila o backend e passa o binario por artefato - registrado como plano B em V2.11.

## V2.6 Lock congelado e guarda de versao contra a imagem

```yaml
- name: Install with frozen lockfile
  run: pnpm install --frozen-lockfile

- name: Guard — installed Playwright must match the container image
  run: |
    set -euo pipefail
    PW=$(node -p "require('@playwright/test/package.json').version")
    echo "installed=$PW expected=1.58.2"
    test "$PW" = "1.58.2" || { echo "::error::Playwright $PW != image v1.58.2"; exit 1; }
- name: Guard — no browser download
  env:
    PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD: "1"
  run: echo "browsers come from the image; download disabled"
```
Motivo do guard: `package.json:43` usa **caret** `^1.58.2`. O lock resolve `1.58.2`, mas um
`pnpm install` sem `--frozen-lockfile` pode subir para `1.59.x` e brigar com o Chromium embutido na
imagem `v1.58.2`. O guard transforma um erro obscuro de runtime em falha imediata e legivel.
Recomendo tambem trocar `^1.58.2` por `1.58.2` exato - **mudanca de arquivo, do escritor unico, nao
feita aqui.**

## V2.7 Comandos exatos, extraidos dos manifestos

| etapa | comando | fonte |
|---|---|---|
| migration | `DATABASE_URL=... go run ./cmd/migrate` (cwd `multica-auth-work/server`) | `server/cmd/migrate/main.go:120` -> `dbURL := os.Getenv("DATABASE_URL")` |
| backend | `PORT=8080 DATABASE_URL=... go run ./cmd/server` | `docker-compose.selfhost.yml:10-11` define `DATABASE_URL` e `PORT: "8080"` para o servico backend |
| frontend build | `pnpm --filter @multica/web build` (`next build`) | `apps/web/package.json` -> `"build": "next build"` |
| frontend start | `pnpm --filter @multica/web start` (`next start`, porta 3000) | `apps/web/package.json` -> `"start": "next start"` |
| e2e | `pnpm exec playwright test --trace=retain-on-failure --screenshot=only-on-failure --video=off --max-failures=5` | `playwright.config.ts` |

`PLAYWRIGHT_BASE_URL=http://localhost:3000` e `NEXT_PUBLIC_API_URL=http://localhost:8080` no env do
job, porque backend e frontend rodam **no mesmo container** do job (nao sao services).

**Nao verifiquei** se `go run ./cmd/server` sobe sem variavel adicional (SMTP, Redis, JWT). O
`docker-compose.selfhost.yml` lista `SMTP_PORT` entre outras; o conjunto minimo obrigatorio precisa
ser determinado por execucao, e e a primeira coisa que o peer reviewer deve atacar. Ver V2.12.

## V2.8 Credenciais: nenhuma de producao, auth efemera de runner

1. `DATABASE_URL` aponta **somente** para o service `postgres` do job.
2. **Zero** `secrets.*` de producao no workflow. Nada de `DATABASE_URL` do ORQ1 - e o mesmo segredo
   que o TL registrou como exposto e pendente de rotacao, e o login de teste faz
   `DELETE FROM verification_code WHERE email = $1` (`e2e/fixtures.ts:35`), ou seja apontar para
   producao executaria DELETE em tabela real.
3. Login sem senha: o suite le o codigo da tabela, ou usa `MULTICA_DEV_VERIFICATION_CODE`
   (`e2e/fixtures.ts:56`) fixado no job.
4. Usuario e workspace unicos por execucao, ja implementados em `e2e/helpers.ts:4-8`
   (`e2e-${E2E_WORKER}-${E2E_RUN_ID}@multica.ai`).
5. `permissions: contents: read` no topo do workflow. Sem `id-token`, sem `packages`, sem `issues`.
6. Se algum dia usar `storageState`: em `$RUNNER_TEMP`, nunca em `test-results/`, nunca cacheado.

## V2.9 Gatilho e integracao - `workflow_dispatch` NAO existe antes do merge

Fato do GitHub Actions: `workflow_dispatch` so aparece na UI/API depois que o arquivo existe **no
default branch** (`main`). Um workflow so em branch de feature nao pode ser disparado manualmente.
Nao ha atalho.

Estrategia escalonada, e **cada seta abaixo e uma autorizacao separada do owner**. Nao presumo
nenhuma:

| # | passo | autorizacao necessaria |
|---|---|---|
| E0 | escrever o arquivo em branch de feature | **autorizacao 1** (criar arquivo) |
| E1 | `git push` da branch | **autorizacao 2** (push) |
| E2 | abrir PR | **autorizacao 3** (PR) |
| E3 | rodar via `pull_request` trigger, ainda sem estar em `main` | coberto por E2; e o unico jeito de validar antes do merge |
| E4 | merge em `main` | **autorizacao 4** (merge) |
| E5 | `workflow_dispatch` passa a existir; primeira execucao manual | **autorizacao 5** |

Recomendacao para reduzir risco no primeiro merge:
```yaml
on:
  workflow_dispatch:
  pull_request:
    paths:
      - '.github/workflows/orq31-browser-qa.yml'
      - 'multica-auth-work/e2e/**'
      - 'multica-auth-work/apps/web/**'
      - 'multica-auth-work/server/**'
```
Deliberadamente **sem `push: branches: [main]`** na primeira versao: o job nao deve rodar em todo
commit de `main` antes de provar estabilidade. Adicionar `push` depois e mudanca separada.

Como este seria o **primeiro** workflow ativo do repositorio (V2.0), o merge tem efeito colateral de
governanca: liga o Actions no repo. Isso e decisao do owner, nao detalhe tecnico.

## V2.10 FILES_LOCKED

Arquivos que este cartao pode criar/alterar quando autorizado, e **nada alem**:
```
.github/workflows/orq31-browser-qa.yml                         (novo, raiz)
multica-auth-work/e2e/chat-upload-ui.spec.ts                   (novo)
multica-auth-work/e2e/chat-panel-buttons.spec.ts               (novo)
multica-auth-work/e2e/chat-reasoning-level.spec.ts             (novo)
multica-auth-work/e2e/squad-model-dropdown.spec.ts             (novo)
multica-auth-work/e2e/delete-flows.spec.ts                     (novo)
multica-auth-work/e2e/board-kanban.spec.ts                     (novo)
.deploy-control/p0/evidence/gtl-browser-qa-disposable-plan.md  (este arquivo)
```
**Fora do lock, exige outro cartao e outro dono:**
`multica-auth-work/package.json` (pin exato de Playwright e `@axe-core/playwright`),
`multica-auth-work/pnpm-lock.yaml`, `playwright.config.ts`, os 7 specs existentes, e qualquer
movimentacao dos workflows aninhados.
Non-overlap declarado: nao toco em `server/**` nem em `packages/**`; o worktree
`gtl-i03-orq13-phase1` (ORQ-13) e outro dono e nao e afetado.

## V2.11 Teto de custo e tempo

- `timeout-minutes: 25` no job; `--max-failures=5`.
- `workers: 1` e `timeout: 60000` vem do config: 7 specs + 6 novos, pior caso teorico ~13 min de
  teste, mais ~2 min de pull de imagem, mais build do Next.
- **`next build` e o custo dominante e eu nao o medi.** Em app Next grande pode passar de 5 min. Se o
  job estourar 25 min, a causa provavel e build, nao teste. Plano B: job separado que faz `build` e
  passa `.next/` por artefato, e o job de browser so faz `start`.
- Zero custo de provedor de modelo: nao ha inferencia no job.
- `concurrency: group: orq31-browser-qa-${{ github.ref }}`, `cancel-in-progress: true`, para nao
  empilhar execucoes.

## V2.12 Cleanup e varredura de segredo em artefato

```yaml
- name: Secret-scan artifacts before upload
  if: always()
  run: |
    set -euo pipefail
    D=multica-auth-work/test-results
    [ -d "$D" ] || { echo "no artifacts"; exit 0; }
    if grep -rIlE 'AKIA[0-9A-Z]{16}|sk-[A-Za-z0-9]{20,}|Bearer [A-Za-z0-9._-]{20,}|postgres://[^ ]*:[^ ]*@' "$D"; then
      echo "::error::secret-like pattern found in artifacts; refusing upload"
      exit 1
    fi
- uses: actions/upload-artifact@ea165f8d65b6e75b540449e92b4886f43607fa02 # v4.6.2
  if: failure()
  with:
    name: orq31-playwright-trace
    path: multica-auth-work/test-results/
    retention-days: 3
```
Falhar o upload em vez de redigir: confiar em regex de redacao e pior do que recusar.
Cleanup: runner efemero e destruido pelo provedor; o service Postgres morre com o job; nenhum
`docker system prune` em host algum - prune e STOP-AND-WAIT e o ORQ1 guarda as imagens de rollback do
OmniRoute.

## V2.13 Condicoes de PARADA e autorizacao

Paro e escalo, sem executar, se qualquer uma ocorrer:
1. Falta autorizacao escrita do owner para qualquer um dos 5 passos de V2.9.
2. `pnpm install --frozen-lockfile` falhar: significa `package.json` e lock divergentes - e cartao de
   outro dono.
3. O guard de versao Playwright falhar: nao mexer na imagem para "fazer passar"; corrigir o pin.
4. A guarda de `.env` disparar: nao apagar arquivo; escalar.
5. O backend nao subir por variavel faltante (V2.7): parar e pedir a lista minima ao dono do backend.
6. Qualquer necessidade de tocar host ORQ1/ORQ2, credencial real ou dado real.
7. **Freeze de Kanban ativo** para criacao de cartao (V2.14).

## V2.14 IDENTIFICADOR - RESOLVIDO (reparo de citacao, ORQ-39)

Cartao verificado e fornecido pelo GTL: **ORQ-39**, UUID `c03941bc-3bde-4de1-ab19-1ba93de0ad51`,
assignee **Opus48-B**. Dependencia declarada: **ORQ-26**.

Historico do identificador, preservado porque explica a correcao:
- este trabalho foi originalmente despachado citando `ORQ-31`;
- a frota reportou **dois UUIDs distintos** sob `ORQ-31` (Browser QA e Security Wave A);
- o GTL determinou depois que o unico `ORQ-31` real e **Security Wave A** e que o Browser QA **nao
  tinha cartao persistido**, tornando a citacao `ORQ-31` **provisoria e invalida**;
- `ORQ-38` registra o incidente de nivel de reporte;
- `ORQ-39` e o cartao verificado que passa a cobrir este plano.

Durante todo o periodo eu **nao criei** cartao, **nao deletei**, **nao renumerei**, **nao mesclei** e
**nao mutei** nenhum cartao, e **nao consultei o board**. Nao sou o registrar nomeado, logo nunca
executei `POST /api/issues`. O numero acima foi **fornecido**, nao previsto por mim.

Este reparo e **somente de citacao**: nenhum conteudo tecnico das secoes V2.0 a V2.13, V2.15 e V2.16
foi alterado. Hashes antes/depois e o diff exato estao no check-out de reparo.

Item tecnico que deliberadamente **nao** foi renomeado, porque renomear seria mudanca tecnica e nao
de citacao: o nome de arquivo proposto `.github/workflows/orq31-browser-qa.yml`, o grupo de
`concurrency` `orq31-browser-qa-...` e o nome de artefato `orq31-playwright-trace` mantem o prefixo
`orq31-`. Recomendo renomear para `orq39-` quando o plano sair do congelamento, mas isso e mudanca de
conteudo e precisa de autorizacao propria - **nao a fiz aqui**.

## V2.15 CRITERIOS DE PASS-READINESS

O plano fica pronto para PASS quando **todos** forem verdadeiros:
1. `.github/workflows/orq31-browser-qa.yml` na **raiz**, com `defaults.run.working-directory: multica-auth-work`. **(V2.1 — atendido no desenho)**
2. `DATABASE_URL` usando `postgres:5432`, sem `ports:`. **(V2.2 — atendido)**
3. Guarda de `.env`/`.env.worktree` que **falha** e nunca apaga. **(V2.3 — atendido)**
4. Toda action por SHA completo e as duas imagens por tag+digest **verificados**. **(V2.4 — atendido, digests obtidos read-only)**
5. Versoes exatas de pnpm/Node/Go/Playwright, com passo `Prove toolchain` no job. **(V2.5 — atendido)**
6. `--frozen-lockfile` + guard pacote-vs-imagem + download de browser desabilitado. **(V2.6 — atendido)**
7. Comandos de migration/backend/build/start extraidos de manifesto. **(V2.7 — atendido, com pendencia de env minimo do backend)**
8. Nenhuma credencial ou dado de producao; auth efemera de runner. **(V2.8 — atendido)**
9. Estrategia de gatilho escalonada, com cada push/PR/merge rotulado como autorizacao separada. **(V2.9 — atendido)**
10. FILES_LOCKED e non-overlap explicitos. **(V2.10 — atendido)**
11. Teto de custo/tempo, cleanup e varredura de segredo em artefato. **(V2.11/V2.12 — atendido)**
12. 7 specs existentes + 6 testes de browser propostos. **(V1 secao 7 + V2.10 — atendido)**
13. Condicoes de parada explicitas. **(V2.13 — atendido)**
14. **PENDENTE, nao atendido por mim:** conjunto minimo de env do backend provado por execucao (V2.7),
    e custo real do `next build` (V2.11). Sao os dois unicos itens que exigem execucao e por isso
    ficam para o peer reviewer/executor - nao os declaro atendidos.

Portanto: **12 de 14 criterios atendidos no desenho; 2 dependem de execucao autorizada.** Nao me
auto-aprovo e nao declaro PASS.

## V2.16 NAO-AFIRMACOES DO V2
- Nao criei workflow, nao editei codigo, nao instalei, nao baixei, nao dei `docker pull`, nao rodei
  teste, nao commitei, nao dei push, nao abri PR, nao fiz merge, nao fiz deploy, nao acessei live.
- Os digests foram obtidos por requisicao **`HEAD`** a `mcr.microsoft.com` e `registry-1.docker.io`,
  e os SHAs por `git ls-remote --tags` a `github.com`. Sao consultas de metadado, **nao** downloads de
  camada. Nenhuma imagem foi materializada no disco.
- Nao verifiquei o conteudo interno da imagem Playwright: nao sei se ela tras Node 22, pnpm ou Go.
  Por isso V2.5 adiciona setup pinado e um passo de prova, em vez de assumir.
- Nao executei nenhum dos 7 specs existentes; nao sei se passam hoje.
- Nao provei que o backend sobe com apenas `DATABASE_URL` e `PORT`.
- Nao medi o tempo do `next build`.
- Nao consultei o Kanban, nao criei, deletei, renumerei, mesclei nem mutei cartao algum.
- Nao movi nem propus mover os workflows aninhados; apenas registrei que estao inertes.
- Nao me auto-aprovo: solicito peer reviewer **diferente** do revisor do V1.
