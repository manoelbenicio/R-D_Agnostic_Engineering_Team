# PROMPT — CODEX (high-thinking) — Stream W-INTEGRATE

> Cole TUDO abaixo (dentro do bloco) na IDE do Codex. É autossuficiente.

---

## 1. PAPEL E MODO
Você é o **CODEX**, engenheiro Go sênior, operando em **high-thinking**. Nesta
rodada você é o **DONO ÚNICO E EXCLUSIVO** dos arquivos hotspot
`server/internal/daemon/daemon.go` e `server/internal/daemon/execenv/execenv.go`.
Enquanto você trabalha, mais ninguém edita esses arquivos. O SME (Opus 4.8) fará a
validação independente DEPOIS que você marcar DONE — portanto a qualidade e a
verificação são de sua responsabilidade.

## 2. REGRA DE OURO (LEIA ANTES DE TUDO)
- Trabalhe **SOMENTE** em `/mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work/`.
- **NUNCA** edite o source original fora dessa pasta.
- **NÃO FAÇA COMMIT.** Não rode `git commit`, `git add`, `git push` nem crie branch.
  Apenas edite arquivos no working copy. O versionamento é feito por outra pessoa.
- Se em qualquer momento faltar informação do contrato, **PARE** e marque `BLOCKED`
  no seu check-in com a dúvida exata. NÃO invente API nem altere arquivos proibidos.

## 3. CONTEXTO DO PROJETO (por que isto existe)
O produto (Multica) roda agentes CLI (codex, kiro, antigravity). Estamos adicionando,
de forma **cirúrgica**, isolamento de credencial por conta + rotação automática quando
a cota de ~5h esgota. **O produto deve permanecer 100% AS-IS quando não há rotação
configurada.** Fase 1 (isolamento) já está pronta e verde. Você agora faz a **Fase 2 —
integração da rotação no daemon**.

## 4. O QUE JÁ EXISTE, PRONTO E VERDE (NÃO REESCREVER, NÃO EDITAR)
Pacote `server/internal/rotation/`:
- `contract.go` — interfaces e tipos. Contém: `ExhaustionDetector`, `Store`,
  `AccountAuthenticator`, `RotationService`, `Account`, `AccountStatus`,
  `RotationReason` (`ReasonQuotaReactive` etc.), `ExhaustionSignal`,
  `DetectionResult`, `ErrNoAccountAvailable`.
- `detector.go` — `NewExhaustionDetector() *Detector`; método
  `Detect(vendor, screenText string, httpStatus int) DetectionResult`.
- `pool.go` — `NewPool(store Store) *Pool`.
- `service.go` — `NewService(store Store, detector ExhaustionDetector, auth AccountAuthenticator, opts ...ServiceOption) *Service`;
  método `OnExhaustion(ctx, agentID, vendor, tenantID string, reason RotationReason, now time.Time) (Account, error)`
  e `SelectNext(...)`. Opções: `WithAuthenticationTimeout`, `WithMaxLoginAttempts`.
- `store_pg.go` — `NewPGStore(pool *pgxpool.Pool) *PGStore` implementa `Store`.
- **`AccountAuthenticator` NÃO tem implementação concreta ainda — você vai criá-la.**

Pacote `server/internal/metrics/`:
- `credential_metrics.go` — `NewCredentialMetrics(registerers ...prometheus.Registerer) *CredentialMetrics`
  com `Collectors()` e métodos: `ObserveRestore(vendor,result)`,
  `ObserveEnvInjection(vendor,result)`, `ObservePrepare(vendor,seconds)`,
  `SetAccountStatus(vendor,accountID,status,value)`, `SetAccountTokensUsed(...)`,
  `SetAccountWindowSecondsRemaining(...)`, `SetAccountsAvailable(vendor,count)`,
  `SetAllAccountsExhausted(vendor,bool)`, `ObserveRotation(vendor,reason,result,seconds)`,
  `ObserveExhaustionDetected(vendor,signal)`.
- `registry.go` — `NewRegistry(RegistryOptions{...})` cria o `prometheus.Registry`,
  registra Go/Process/business/db collectors via `reg.MustRegister(...Collectors()...)`.
  É AQUI que o `CredentialMetrics` deve ser registrado (siga o padrão de `NewBusinessMetrics`).

Estado atual dos hotspots (já contêm a Fase 1 — PRESERVE):
- `execenv.go`: `PrepareParams.CredentialAccountHome` e `ReuseParams.CredentialAccountHome`
  já existem; `Prepare`/`Reuse` já chamam `prepareCodexHomeWithOpts` / `prepareKiroHome` /
  `prepareAntigravityHome` quando `CredentialAccountHome != ""`. `PrepareParams{...}` é
  populado em `daemon.go` por volta da linha **3244**.
- `daemon.go`: injeta `CODEX_HOME` / `XDG_DATA_HOME` / `HOME` em `agentEnv` por volta das
  linhas **3378–3393**.

## 5. TAREFA (faça nesta ordem)

### Passo 1 — Adapter de autenticação (arquivo NOVO)
Crie `server/internal/rotation/auth_authenticator.go` com uma implementação concreta
de `AccountAuthenticator` (device-login/OAuth, store/restore AS-IS):
- `Login(ctx, acc Account) (sessionID string, err error)`: restaura a credencial da
  conta (AS-IS) para o dir isolado da conta (`acc.HomeDir`/`acc.ConfigDir`) e retorna
  um sessionID (ex.: derivado de `acc.AccountID`).
- `Logout(ctx, acc Account) error`: remove/limpa a credencial do dir isolado da conta.
- `WaitAuthenticated(ctx, sessionID string, timeout time.Duration) (bool, error)`:
  verifica presença/validade do arquivo de credencial (sem ler segredo).
- Deve ser injetável e testável (aceite dependências por parâmetro/opção).
- **NUNCA logar conteúdo de credencial** — apenas caminho/tipo/mtime, se necessário.
- Crie também `auth_authenticator_test.go` provando login→wait→logout com um dir temporário.

### Passo 2 — Registrar as métricas de credencial
Em `server/internal/metrics/registry.go`, registre o `CredentialMetrics` no registry
(mesmo padrão do `businessMetrics`), e exponha-o para o daemon consumir (ex.: retorne-o
em `RegistryOptions`/struct de retorno, seguindo o estilo já existente do arquivo).
NÃO altere `credential_metrics.go`.

### Passo 3 — Fiar a RotationService no daemon (hotspot)
Em `daemon.go`:
- Instancie (na inicialização do daemon, onde o pool pgx já existe): `store := rotation.NewPGStore(pool)`,
  `detector := rotation.NewExhaustionDetector()`, `auth := rotation.New<SeuAdapter>(...)`,
  `svc := rotation.NewService(store, detector, auth)`.
- No ponto onde uma tarefa termina com falha de quota / parada suspeita, use `detector.Detect(...)`
  sobre a saída/So HTTP status; se `DetectionResult.Exhausted`, chame
  `svc.OnExhaustion(ctx, agentID, vendor, tenantID, rotation.ReasonQuotaReactive, time.Now())`,
  obtenha a `Account` nova e **re-despache a tarefa** setando
  `prepParams.CredentialAccountHome = <dir da conta nova>` (linha ~3244).
- **Backward-compat OBRIGATÓRIA:** se não houver contas configuradas / `SelectNext`
  retornar `ErrNoAccountAvailable`, o comportamento deve ser **idêntico ao atual**
  (nenhuma rotação, nenhuma env nova). O teste `execenv/vendor_credential_fallback_test.go`
  DEVE continuar verde.
- Emita métricas nos pontos reais: `ObserveExhaustionDetected` na detecção,
  `ObserveRotation` na troca, `ObserveRestore`/`ObserveEnvInjection` no restore/injeção.

### Não faça
- NÃO altere: `contract.go`, `detector.go`, `service.go`, `pool.go`, `store_pg.go`,
  `credential_metrics.go`, `codex_home.go`, `kiro_home.go`, `antigravity_home.go`.
- NÃO introduza SQLite próprio (Postgres-only). NÃO armazene token em claro.
- NÃO adicione novas strings de branding "Multica". NÃO faça commit.

## 6. CHECK-IN / CHECK-OUT (localização e formato EXATOS)
- **Local correto do arquivo de controle:** `Automonous_Agentic/.deploy-control/`
  (o BOARD PRINCIPAL na RAIZ do projeto). **NÃO** salve dentro de
  `multica-auth-work/.deploy-control/` — isso quebra o rastreamento.
- **Nome do arquivo:** `CODEX__W-INTEGRATE__<START_UTC>.md`, onde `<START_UTC>` é o
  timestamp UTC do MOMENTO em que você começa, no formato `AAAAMMDDTHHMMSSZ`
  (ex.: `20260701T210500Z`). Gere com: `date -u +%Y%m%dT%H%M%SZ`.
- **No início (antes de editar qualquer código)** crie o arquivo com:
  ```
  agent: CODEX
  stream: W-INTEGRATE
  started_at: <UTC ISO 8601, ex.: 2026-07-01T21:05:00Z>
  finished_at:
  status: IN_PROGRESS
  files_locked:
    - server/internal/daemon/daemon.go
    - server/internal/daemon/execenv/execenv.go
    - server/internal/metrics/registry.go
    - server/internal/rotation/auth_authenticator.go
    - server/internal/rotation/auth_authenticator_test.go
  depends_on: [W-ROT-contract, W-ROTATE, W-DETECT, W-PGSTORE, W-METRICS]
  build_result:
  notes:
  ```
- **`started_at`** = timestamp de quando você COMEÇA. **`finished_at`** = timestamp de
  quando você TERMINA (só preencha no check-out). Ambos em UTC ISO 8601.
- **No check-out (ao terminar):** atualize o MESMO arquivo: `finished_at`,
  `status: DONE` (ou `BLOCKED`), e `build_result` com o **tail real** da saída de
  verificação (cole as últimas ~15 linhas). Se `BLOCKED`, descreva o motivo exato em `notes`.

## 7. VERIFICAÇÃO OBRIGATÓRIA (antes de marcar DONE)
Rode EXATAMENTE isto (precisa de git E Postgres no container, senão dá falso-vermelho):
```
cd /mnt/c/VMs/Projetos/Automonous_Agentic/multica-auth-work
docker network create rotnet 2>/dev/null
docker run -d --rm --name rotpg --network rotnet -e POSTGRES_PASSWORD=pw \
  -e POSTGRES_USER=aop -e POSTGRES_DB=aop postgres:17-alpine
sleep 6
docker run --rm --network rotnet -v "$PWD":/src -w /src/server \
  -e DATABASE_URL="postgres://aop:pw@rotpg:5432/aop?sslmode=disable" golang:1.26-alpine \
  sh -c "apk add --no-cache git >/dev/null 2>&1; git config --global user.email t@t; \
         git config --global user.name t; \
         go build ./... && go vet ./internal/... && \
         go test ./internal/daemon/... ./internal/rotation/... ./internal/metrics/..."
docker stop rotpg; docker network rm rotnet
```
**Critério de DONE:**
- `go build ./...` = OK (server inteiro compila).
- `go vet ./internal/...` = OK.
- Testes de `daemon`, `rotation`, `metrics` = verdes.
- **Único fail tolerado:** `TestValidateLocalPath/rejects_a_symlink_pointing_at_the_user_home`
  (ambiental — container roda como root; NÃO é do seu código). Qualquer OUTRO fail
  bloqueia o DONE.

## 8. RESUMO DO QUE EU ESPERO (para não haver dúvida depois)
- ANTES: check-in criado no board principal, com `started_at` UTC correto.
- DURANTE: só arquivos permitidos; sem commit; sem tocar arquivos proibidos; backward-compat preservada.
- DEPOIS: verificação verde no container (git+postgres), check-out com `finished_at`,
  `status: DONE` e `build_result` colado. Nada de branding novo; nada de segredo em log.
- Se algo do contrato faltar ou um arquivo proibido precisar mudar: `BLOCKED` + nota,
  sem improvisar.
