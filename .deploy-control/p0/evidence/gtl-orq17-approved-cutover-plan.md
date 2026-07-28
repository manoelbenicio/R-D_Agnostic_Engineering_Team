# ORQ-17 - Plano de cutover aprovado (V3) - READ-ONLY

- autor: Opus48#A - ORQ2 - pane w6:p1 - 2026-07-27T12:35Z
- politica aprovada pelo owner: FQDN MagicDNS do ORQ1, HTTPS 443, acesso restrito ao
  dispositivo/identidade Tailscale do owner, TTL absoluto 24h + idle 4h
- modo: READ-ONLY. Nada instalado, configurado, reiniciado ou exposto. Nenhum arquivo de codigo,
  unit, container ou env alterado.
- integrador: Codex56-TL (GENERAL-TECH-LEAD) w5:pC. **Peer review obrigatorio antes de qualquer
  execucao.** Autoridade final: owner humano.

---

## 1. FQDN real, descoberto (nao inventado)

`tailscale status --json` no ORQ1, campos literais:

| campo | valor medido |
|---|---|
| `Self.DNSName` | `orq1.tail96e2c0.ts.net.` |
| `Self.HostName` | `orq1` |
| `MagicDNSSuffix` | `tail96e2c0.ts.net` |
| `CurrentTailnet.Name` | `cloud.labs.brazil@gmail.com` |
| `CurrentTailnet.MagicDNSEnabled` | `true` |
| `Self.TailscaleIPs` | `100.118.244.61`, `fd7a:115c:a1e0::5034:f43e` |

**FQDN canonico**: `orq1.tail96e2c0.ts.net`
**Origin canonico**: `https://orq1.tail96e2c0.ts.net`

---

## 2. GATE ZERO - HTTPS do tailnet esta DESLIGADO

Medido: `CertDomains: None` no `tailscale status --json`, e `tailscale serve status` responde
`No serve config`.

`CertDomains` vazio significa que o recurso **HTTPS Certificates do tailnet nao esta habilitado**.
Sem ele, `tailscale cert` e `tailscale serve --https` **falham**, e nao ha como emitir certificado
para `orq1.tail96e2c0.ts.net`. Isso **nao se resolve no host**: e um toggle no admin console do
tailnet, sob a identidade `cloud.labs.brazil@gmail.com`.

**Consequencia**: nenhuma fase de execucao pode comecar antes de o owner habilitar HTTPS
Certificates e o `CertDomains` passar a listar o FQDN. Este e o primeiro item do gate, e e acao do
owner, nao do agente.

---

## 3. Topologia medida e alvo

### 3.1 Hoje
```
owner --(tunel SSH, frágil)--> ORQ1 127.0.0.1:13100 (frontend)
                                    127.0.0.1:18080 (backend)
```
`ss -tln` no ORQ1: **apenas** `127.0.0.1:18080` e `127.0.0.1:13100`. Nada em 0.0.0.0, nada em 443.
Ou seja backend e frontend ja estao corretamente em loopback - isso **nao deve mudar**.

O tunel atual e `multica-orq1-backend-tunnel.service`, **user unit do ORQ2** ("Multica ORQ2 to ORQ1
backend tunnel"), nao do ORQ1 (GTL-50). Ele serve o daemon do ORQ2, nao o browser do owner.

### 3.2 Alvo
```
owner (dispositivo Tailscale autorizado)
   |  HTTPS 443, cert do tailnet
   v
ORQ1 :443  (borda: tailscale serve OU Caddy)
   |-- /            -> 127.0.0.1:13100  (frontend Next.js)
   |-- /api, /ws    -> 127.0.0.1:18080  (backend)
```
Invariante: os dois processos continuam **loopback-only**. A borda e o unico ouvinte publico, e
"publico" aqui significa **apenas dentro do tailnet**.

### 3.3 Ferramenta de borda - decisao
Medido no ORQ1: `caddy`, `nginx` e `traefik` **NAO INSTALADOS**. `tailscaled` ja esta presente e
autenticado.

**Recomendo `tailscale serve`**, com dois motivos objetivos:
1. **zero instalacao** - nao adiciona pacote, e a diretriz do gate e nao instalar;
2. **cert automatico** - `tailscale serve --https=443` obtem e renova o certificado do tailnet sem
   `certbot`, sem cron e sem arquivo de chave para vazar.
Caddy so se ganharmos algo que o serve nao da (rewrite complexo, buffering, header custom). Nao
identifiquei tal necessidade. Se o integrador preferir Caddy, o plano exige um item novo de gate:
**instalar pacote**, que e STOP-AND-WAIT proprio.

Forma esperada (a validar em dry-run, **nao executar agora**):
```bash
tailscale serve --bg --https=443 --set-path / http://127.0.0.1:13100
tailscale serve --bg --https=443 --set-path /api http://127.0.0.1:18080
tailscale serve --bg --https=443 --set-path /ws http://127.0.0.1:18080
```
RESSALVA declarada: **nao validei a sintaxe exata de `--set-path` nesta versao do tailscale** - nao
executei nem `tailscale serve --help`. O integrador deve confirmar a sintaxe antes, e o roteamento
por path pode exigir uma unica origem com o Next.js fazendo proxy de `/api`. Isso e detalhe de
implementacao, nao de politica.

### 3.4 Funnel: PROIBIDO
`tailscale funnel` expoe na internet publica. A politica aprovada e acesso **somente** por
dispositivo/identidade Tailscale do owner. Portanto: `serve` sim, `funnel` **nunca** neste plano.

---

## 4. Restricao de acesso ao dispositivo/identidade do owner

`tailscale serve` publica para **todo o tailnet**, nao para um dispositivo. A restricao por
identidade exige duas camadas, ambas fora do host:

1. **ACL do tailnet** (admin console, acao do owner): regra permitindo `orq1:443` apenas para a
   identidade/dispositivo do owner, negando o resto do tailnet - incluindo os proprios agentes.
2. **Tailscale identity headers**: quando servido por `serve`, o backend passa a receber
   `Tailscale-User-Login` / `Tailscale-User-Name`. **Nao verifiquei** se o backend le esses headers
   hoje - meu grep de auth nao encontrou nada equivalente. Portanto **nao proponho** depender deles
   nesta rodada; a autorizacao continua sendo o login proprio do Multica, e a ACL e a camada de rede.

Ponto de honestidade: sem a ACL, expor via `serve` significa **qualquer no do tailnet** alcancar a
UI autenticada. A ACL nao e opcional; e parte da politica aprovada.

---

## 5. TTL absoluto 24h + idle 4h - o que o codigo suporta hoje

### 5.1 TTL absoluto 24h: SUPORTADO, sem codigo novo
`internal/auth/cookie.go`:
- `:23 defaultAuthTokenTTL = 30 * 24 * time.Hour // 30 days` - o default atual e **30 dias**.
- `:36-61 parseAuthTokenTTL` aceita `time.ParseDuration` ("24h") **ou** inteiro em segundos.
- `:74 AuthTokenTTL()` le `AUTH_TOKEN_TTL` uma vez (`sync.Once`) e cai no default se ausente/invalido.
- `:152-165 SetAuthCookies` usa esse TTL em `MaxAge` e `Expires`.

Acao: `AUTH_TOKEN_TTL=24h` no ambiente do backend. **Cuidado**: `sync.Once` significa que a mudanca
so vale para processo novo - exige recriar o container, e recriar container e mutacao sob gate.

### 5.2 Idle 4h: **NAO SUPORTADO**. Precisa mecanismo novo.
Evidencia negativa: `grep -rn "refresh|Refresh|rolling|sliding|renew"` em `internal/auth/` e
`internal/handler/auth.go` (sem testes) retorna **VAZIO**. Nao existe renovacao, nem sliding window,
nem tabela de sessao (`grep user_session|session_token|refresh_token` em `migrations/` acha apenas
`005_daemon_pairing` e `029_drop_daemon_pairing`, ambos de pairing de daemon, nao de sessao de
usuario).

O token e **stateless**: o servidor nao guarda "ultima atividade", logo **e impossivel expirar por
inatividade sem introduzir estado ou reemissao**. Qualquer afirmacao de que idle 4h "ja funciona"
seria falsa.

#### 5.2.1 Mecanismo minimo real, sem inventar `user_session`
Opcao recomendada: **cookie deslizante com dois relogios no proprio token**, zero tabela nova.
- o claim de expiracao passa a ser `min(iat + 24h, last_seen + 4h)`;
- `last_seen` nao precisa de banco: e o proprio momento de emissao do cookie. A cada requisicao
  autenticada, se o cookie tem mais de N minutos (ex. 5, para nao reescrever a cada request), o
  middleware **reemite** o cookie com nova validade de 4h, **mas nunca alem de `iat_original + 24h`**;
- o teto absoluto exige que o token carregue o `iat` original de forma imutavel, para a reemissao
  nao renovar indefinidamente. Sem isso, "sliding" viraria sessao eterna - e o erro classico dessa
  implementacao.

Custos e limites que declaro: (a) o `Set-Cookie` de reemissao precisa passar pela borda sem ser
bufferizado - com `tailscale serve` isso e transparente, mas e item de teste; (b) logout continua
dependendo do que ja existe hoje, porque sem estado no servidor nao ha revogacao imediata - o teto
de 24h e o unico limite duro; (c) o CSRF e derivado do token (`cookie.go:141-145 hmac(authToken)`),
logo **toda reemissao precisa reemitir tambem o cookie CSRF**, senao o par quebra e requests
mutantes passam a falhar. Este ultimo ponto e a armadilha principal.

Opcao alternativa, se o owner exigir revogacao real: tabela de sessao com `last_seen_at`. Isso e
schema novo, migration, e **muda o modelo de auth** - fica **FORA** deste plano.

#### 5.2.2 GATE SEPARADO
Conforme instruido, o idle 4h **nao entra na mesma janela do cutover**. Divisao:
- **Gate A - cutover de rede** (secoes 3, 4, 6): FQDN, HTTPS, ACL, borda, origin, CORS/CSRF/WS.
  Entrega o acesso estavel sem tunel. Usa `AUTH_TOKEN_TTL=24h`, que ja e suportado.
- **Gate B - idle 4h**: exige codigo novo (5.2.1), peer review, testes e build. Nao bloqueia o
  Gate A. Enquanto B nao existir, a politica efetiva e "absoluto 24h, sem idle", e isso deve estar
  **escrito** para o owner, nao implicito.

---

## 6. Ordem obrigatoria: `bypass=false` + `FRONTEND_ORIGIN` ANTES da borda

### 6.1 Interlock que o codigo ja tem - e que trabalha a nosso favor
`internal/middleware/auth.go:38-55`, literal:
```go
38 func localAuthBypassEmail() string {
39 	if !strings.EqualFold(strings.TrimSpace(os.Getenv(localAuthBypassEnv)), "true") {
40 		return ""
41 	}
43 	frontendOrigin := strings.TrimSpace(os.Getenv("FRONTEND_ORIGIN"))
48 	host := parsed.Hostname()
49 	ip := net.ParseIP(host)
50 	if !strings.EqualFold(host, "localhost") && (ip == nil || !ip.IsLoopback()) {
51 		return ""
52 	}
```
com o comentario `:36-37`: *"The second check prevents a copied production configuration from
silently turning a public deployment into an unauthenticated instance."*

Isto e importante: **assim que `FRONTEND_ORIGIN` deixar de ser loopback, o bypass se desativa
sozinho**, mesmo que `MULTICA_LOCAL_AUTH_BYPASS=true` continue no ambiente. Ou seja o codigo ja
falha fechado. Mesmo assim, o plano exige **remover explicitamente** o bypass, por dois motivos: nao
depender de defesa em profundidade como mecanismo primario, e nao deixar a flag `true` para o
proximo operador copiar.

Estado atual medido no processo do daemon (GTL-02): `MULTICA_LOCAL_AUTH_BYPASS=true` e
`MULTICA_LOCAL_AUTH_EMAIL=owner@local.test`, com `MULTICA_APP_URL=http://localhost:13100`.

### 6.2 Sequencia, e ela nao pode ser reordenada
1. `MULTICA_LOCAL_AUTH_BYPASS=false` (ou remover a var) e remover `MULTICA_LOCAL_AUTH_EMAIL`.
2. `FRONTEND_ORIGIN=https://orq1.tail96e2c0.ts.net` (e `MULTICA_APP_URL` idem, usado por
   `handler/config.go:83` e pelos comandos de CLI `cmd_auth.go:86`/`cmd_login.go:18`).
3. `AUTH_TOKEN_TTL=24h`.
4. Recriar backend e frontend para que as vars valham (o TTL e `sync.Once`).
5. Garantir que existe **usuario real com credencial** antes de desligar o bypass - senao o cutover
   entrega uma UI onde o owner nao consegue entrar. Nao verifiquei se ha senha configurada para
   `owner@local.test`; `125_user_password_credential` existe como migration, logo o mecanismo existe.
   **Item de gate**: confirmar login funcional por loopback ANTES de expor.
6. Somente entao habilitar a borda (`tailscale serve`).

Inverter 1-2 e 6 significaria expor na rede uma instancia **sem autenticacao** durante a janela.

---

## 7. CORS, CSRF, WebSocket e uploads sob o novo origin

| eixo | codigo | efeito da mudanca de origin |
|---|---|---|
| CORS | `cmd/server/router.go:42-62 allowedOrigins()`: usa `CORS_ALLOWED_ORIGINS`, senao `FRONTEND_ORIGIN`, senao `defaultOrigins` | basta `FRONTEND_ORIGIN` correto. Se `CORS_ALLOWED_ORIGINS` existir no ambiente, ele **ganha** e precisa conter o FQDN |
| Cookie `Secure` | `internal/auth/cookie.go:115-119`: derivado do **scheme de `FRONTEND_ORIGIN`** | com `https://` o cookie vira `Secure`. Se `FRONTEND_ORIGIN` ficar `http://`, o cookie sai sem `Secure` sobre HTTPS - downgrade silencioso. Motivo pelo qual o scheme importa |
| SameSite | `cookie.go:164 SameSite: http.SameSiteStrictMode` | com frontend e API sob o **mesmo** host (`orq1.tail96e2c0.ts.net`), Strict funciona. Se a borda servir frontend e API em hosts diferentes, Strict **quebra** o login. Requisito de desenho: host unico |
| CSRF | `cookie.go:141-145`: `hmac(authToken)` em cookie legivel | inalterado, desde que toda reemissao de auth reemita o CSRF (ver 5.2.1) |
| WebSocket | `internal/realtime/hub.go:158-180 checkOrigin` | aceita same-origin comparando `Origin` com `r.Host`; atras de proxy usa `X-Forwarded-Host` **somente se** `isTrustedProxy(r.RemoteAddr)`. Com `tailscale serve` o backend vera `RemoteAddr` de loopback e `Host` reescrito - **item de teste obrigatorio**: se `Host` chegar como `127.0.0.1:18080` e o `Origin` do browser como o FQDN, o WS **e recusado** a menos que `MULTICA_TRUSTED_PROXIES` inclua o loopback da borda |
| `MULTICA_TRUSTED_PROXIES` | `router.go:65-70` | provavelmente precisa incluir `127.0.0.1/32` e/ou `::1/128`. Nao verifiquei o valor atual - item de gate |
| WS do daemon | `internal/daemonws/hub.go:182 CheckOrigin: func(r *http.Request) bool { return true }` | nao afetado pela mudanca de origin. Registro que e permissivo por desenho (clientes nativos), e que a borda **nao deve** expor `/daemon-ws` ao tailnet sem necessidade |
| uploads | nao inspecionei limites de body na borda | item de gate: `tailscale serve` ou Caddy impondo limite menor que o do backend causaria 413 em upload grande. Testar com arquivo real acima de 10 MB |

---

## 8. Health, rollback e o tunel

### 8.1 Health, antes e depois (read-only, seguro)
```bash
# no ORQ1, loopback - deve continuar 200 em todas as fases
curl -fsS http://127.0.0.1:18080/health
curl -fsSI http://127.0.0.1:13100/ | head -1
# do dispositivo do owner, apos a borda
curl -fsSI https://orq1.tail96e2c0.ts.net/ | head -1
```
Evidencia a anexar: as tres saidas antes e depois, mais `tailscale serve status` e
`ss -tln | grep 443`.

### 8.2 Rollback
1. **Borda**: `tailscale serve reset` (ou remover o path especifico). Reversivel em um comando, sem
   reiniciar backend, frontend ou daemon.
2. **Env**: restaurar `FRONTEND_ORIGIN=http://localhost:13100`, `MULTICA_LOCAL_AUTH_BYPASS=true` e
   remover `AUTH_TOKEN_TTL`, recriando os containers. Custo: uma recriacao.
3. **Tunel do ORQ2**: `multica-orq1-backend-tunnel.service` **nao deve ser desligado** no cutover.
   Ele serve o daemon do ORQ2, nao o browser. Desligar quebraria o executor. Somente apos o cutover
   estar estavel e o owner decidir, avaliar se ainda faz sentido - e isso e outra issue.
4. **Invariante mantida** (ressalva Codex56#B): nenhum rollback reaplica branch antiga nem
   reintroduz copia bruta de diretorio de credencial.

### 8.3 O que NAO e rollback
Se o `Set-Cookie` ou o WS falharem apos a borda, a correcao e ajustar `MULTICA_TRUSTED_PROXIES` e o
mapeamento de path - **nao** reabilitar o bypass. Reabilitar bypass com origin publico e o cenario
que `middleware/auth.go:36-37` existe para impedir.

---

## 9. Ferramentas verificadas (nada instalado)

| ferramenta | ORQ1 | uso no plano |
|---|---|---|
| `tailscaled` / `tailscale` | presente e autenticado (tailnet `cloud.labs.brazil@gmail.com`) | borda + cert |
| `caddy` | **NAO INSTALADO** | so se o integrador justificar; exige gate de instalacao |
| `nginx` | **NAO INSTALADO** | nao usado |
| `traefik` | **NAO INSTALADO** | nao usado |
| `curl`, `ss` | presentes | health |

---

## 10. Gate checklist (ordem literal)

- [ ] **G0** Owner habilita HTTPS Certificates no tailnet; `CertDomains` passa a listar
      `orq1.tail96e2c0.ts.net`. **Sem isso nada comeca.**
- [ ] **G1** Owner define a ACL do tailnet restringindo `orq1:443` a sua identidade/dispositivo.
- [ ] **G2** Confirmar login real por loopback (usuario com credencial), com o bypass ainda ativo.
- [ ] **G3** Fila de tasks vazia: `count(*) FROM agent_task_queue WHERE status IN ('queued',
      'dispatched','running','waiting_local_directory')` = 0, saida anexada.
- [ ] **G4** Aplicar env: bypass=false, `FRONTEND_ORIGIN`/`MULTICA_APP_URL` = FQDN https,
      `AUTH_TOKEN_TTL=24h`, `MULTICA_TRUSTED_PROXIES` com loopback. Recriar containers.
- [ ] **G5** Health loopback 200 em backend e frontend; login funcionando por loopback com bypass OFF.
- [ ] **G6** Habilitar borda (`tailscale serve`), confirmar `443` em escuta e HTTPS 200 do dispositivo
      do owner.
- [ ] **G7** Testes de borda: login completo, WS conectando, upload > 10 MB, CSRF em request mutante.
- [ ] **G8** ~~Registrar evidencia na ORQ-17~~ **SUPERSEDIDO por V4.7/G8**: evidencia vai para
      `.deploy-control/p0/evidence/`, **nunca** como comentario na issue (GTL ASSIGNMENT/COMMENT
      FREEZE: comentario em issue atribuida pode enfileirar task paga). Confirmar que o tunel do
      ORQ2 e o daemon seguem ativos.
- [ ] **G9** (Gate B, separado) idle 4h: desenhar, revisar, testar e so entao implementar.

Cada item de G4 a G8 e mutacao e exige autorizacao escrita do owner. Este documento nao autoriza
nada.

## 11. Perguntas em aberto, declaradas sem afirmar

- sintaxe exata de `tailscale serve --set-path` nesta versao: nao executei `--help`.
- se o backend le `Tailscale-User-Login`: nao encontrei, mas nao varri exaustivamente; por isso nao
  dependo disso.
- valor atual de `MULTICA_TRUSTED_PROXIES` e de `CORS_ALLOWED_ORIGINS` no ambiente do backend: nao li.
- se existe senha configurada para o usuario do owner: `125_user_password_credential` existe como
  migration, mas nao consultei o banco.
- limites de body/upload na borda: nao medidos.

## 12. Nada mutado

Nenhuma instalacao, configuracao, restart, exposicao de porta, ACL, env, unit, container ou codigo
alterado. Somente leitura: `tailscale status --json`, `tailscale serve status`, `command -v`,
`ss -tln`, e leitura de fonte no ORQ2. As 9 issues preservadas seguem intactas; zero rerun.

---
---

# SECAO V4 - CORRECAO QUE SUPERSEDE A V3 (ORQ-17)

- autor: Opus48#A - ORQ2 - pane w6:p1 - 2026-07-27T12:48Z
- card: **ORQ-17** "Publicar frontend 13100 com acesso LAN estavel" (existente; nenhum card criado -
  Kanban em FREEZE pelo incidente de ORQ-31 duplicado)
- fecha: BLOCK do GTL-R17
- **SUPERSEDE as secoes 3.3, 4, 5.2 e 10 da V3.** O que a V4 nao contradiz permanece valido.
- modo: READ-ONLY. Zero mutacao de codigo, config, cert, serve, ACL, env, container, restart,
  instalacao ou credencial. **Nao executar.** Re-review independente solicitada.

## V4.0 Tailscale 1.98.9 - sintaxe verificada no host, read-only

`tailscale version` no ORQ1: **`1.98.9`** (commit `4fb758c39ae5b208b974af14ba6bc896a250394c`,
long `1.98.9-t4fb758c39-g200941d74`).

`tailscale serve --help` nesta versao, flags literais relevantes:
```
USAGE
  tailscale serve <target>
  tailscale serve status [--json]
  tailscale serve reset
FLAGS
  --bg, --bg=false      Run the command as a background process (default false, ...)
  --http value          Expose an HTTP server at the specified port
  --https value         Expose an HTTPS server at the specified port (default mode)
  --set-path value      Appends the specified path to the base URL for accessing the underlying service
  --yes, --yes=false    Update without interactive prompts (default false)
```
Confirmado: `--https`, `--set-path`, `--bg`, `--yes`, `serve status [--json]` e `serve reset`
existem nesta versao. A V3 declarava a sintaxe como nao validada; agora esta validada.
`--set-path` e **prefixo**, e a raiz e montada **sem** `--set-path`.

## V4.1 CORRECAO DE FATO - `/auth/login`, `/auth/google` e `/auth/logout` NAO sao rotas do Next.js

O dispatch pede preservar essas tres como rotas do frontend. Medido: **sao rotas do BACKEND.**
`cmd/server/router.go`, literal:
```go
487 	r.With(authRL).Post("/auth/login", h.Login)
488 	r.With(authRL).Post("/auth/google", h.GoogleLogin)
489 	r.Post("/auth/logout", h.Logout)
```
E o app router do Next.js (`apps/web/app/`) contem:
```
app/page.tsx                      -> /
app/(auth)/login/page.tsx         -> /login          (grupo "(auth)" NAO aparece na URL)
app/(auth)/onboarding/page.tsx    -> /onboarding
app/(auth)/invitations/page.tsx   -> /invitations
app/(auth)/invite/[id]/page.tsx   -> /invite/{id}
app/(auth)/workspaces/new/page.tsx-> /workspaces/new
app/auth/callback/page.tsx        -> /auth/callback   <-- UNICA rota /auth/* do frontend
app/lark/...                      -> /lark
app/[workspaceSlug]/...           -> /{workspaceSlug}/...
```
**Consequencia direta e a armadilha central deste cutover**: existe **colisao de prefixo** em
`/auth`. Se a borda mandar `/auth` inteiro para o backend, `/auth/callback` (pagina do frontend)
**quebra** e o login por Google termina em 404. Se mandar `/auth` para o frontend, os tres POSTs de
autenticacao param de existir. A unica forma correta e montar **os tres paths exatos** no backend e
deixar `/auth/callback` cair na raiz do frontend, o que funciona porque `--set-path` usa
precedencia de prefixo mais longo.

## V4.2 Mapa de rotas exato (derivado do codigo, nada inventado)

Backend em `127.0.0.1:18080`, prefixos de primeiro nivel medidos em `router.go`:
`/health` (443), `/readyz` (444), `/healthz` (445), `/health/realtime` (456), `/ws` (468),
`/uploads/*` (474), `/auth/login` (487), `/auth/google` (488), `/auth/logout` (489),
`/api/...` (492 em diante, incluindo `r.Route("/api/daemon", ...)` em 510 e `/api/daemon/ws` em 516).

Frontend em `127.0.0.1:13100`: tudo o mais, incluindo `/`, `/login`, `/auth/callback`, `/lark`,
`/{workspaceSlug}/...`, `/_next/*`, `/favicon.ico`, `/robots.txt`.

### Mapa proposto (ordem de aplicacao irrelevante; a precedencia e por prefixo mais longo)
```bash
# raiz -> frontend (captura /login, /auth/callback, /_next, /{workspaceSlug}, /lark)
tailscale serve --bg --yes --https=443 http://127.0.0.1:13100

# API
tailscale serve --bg --yes --https=443 --set-path /api      http://127.0.0.1:18080
# WebSocket do browser (realtime)
tailscale serve --bg --yes --https=443 --set-path /ws       http://127.0.0.1:18080
# arquivos servidos pelo backend
tailscale serve --bg --yes --https=443 --set-path /uploads  http://127.0.0.1:18080
# os TRES endpoints de auth do backend, paths exatos, para nao capturar /auth/callback
tailscale serve --bg --yes --https=443 --set-path /auth/login  http://127.0.0.1:18080
tailscale serve --bg --yes --https=443 --set-path /auth/google http://127.0.0.1:18080
tailscale serve --bg --yes --https=443 --set-path /auth/logout http://127.0.0.1:18080
```
**Nao expor** `/health`, `/readyz`, `/healthz`, `/health/realtime`: continuam loopback-only, o que e
suficiente para o preflight (V4.5) e evita superficie desnecessaria.

### Risco residual declarado: `/api/daemon`
Montar `/api` expoe tambem `/api/daemon/*`, incluindo o WS de daemon em `router.go:516`. Agrava que
`internal/daemonws/hub.go:182` tem `CheckOrigin: func(r *http.Request) bool { return true }` por
desenho, para clientes nativos. `tailscale serve` **nao tem regra de negacao**, logo nao existe
"excluir /api/daemon" na borda. Tres opcoes, e recomendo a primeira:
1. **Confiar na ACL de V4.3** como controle: se a 443 do orq1 so aceita o dispositivo do owner, o
   alcance de `/api/daemon` fica limitado ao proprio owner. O daemon do ORQ2 continua entrando por
   loopback via tunel, sem passar pela borda.
2. Montar prefixos mais estreitos em vez de `/api` - inviavel na pratica: `router.go` tem 54
   ocorrencias de `/api`, e enumerar todas cria divergencia silenciosa a cada rota nova.
3. Colocar um proxy com regra de deny na frente - exige instalar pacote, o que este plano proibe.
Sem a ACL, esta exposicao e inaceitavel. Com a ACL, e aceitavel e declarada.

## V4.3 ACL / Grants de menor privilegio - proposta explicita

### V4.3.1 ACHADO QUE MUDA O DESENHO: identidade nao separa owner de agentes
`tailscale status --json` no ORQ1, medido:
```
users: 7650130887043022  cloud.labs.brazil@gmail.com | Manuel Filho
self:  orq1.tail96e2c0.ts.net.  UserID=7650130887043022  tags=None  OS=linux
peers (14): lenovo-lab, ipad-air-5th-gen-wifi, orq2, a07-de-manoel, msi-laptop-1, ec2-jump-box,
            poco-x8-pro-max, manoelneto-laptop-1, wsl-dataops-labs, manoelneto-laptop, hp-laptop,
            x8promax, wsl-lenovo-lab, msi-laptop  -- TODOS UserID=7650130887043022, TODOS tags=None
```
**Todos os 15 nos pertencem a MESMA identidade e nenhum tem tag.** Portanto uma ACL baseada em
`src: ["cloud.labs.brazil@gmail.com"]` concederia acesso **tambem** aos nos de agente (`orq1`,
`orq2`, `wsl-dataops-labs`, `ec2-jump-box`). O requisito "restringir a identidade do owner
excluindo nos de agente" **nao e expressavel por identidade** neste tailnet. Tem de ser por
**dispositivo**.

### V4.3.2 Proposta A - por IP de dispositivo (aplicavel hoje, sem re-keying)
Usa `hosts` + `src` por host. O owner precisa **declarar** quais dispositivos sao dele; eu **nao
invento**. Comando read-only para ele levantar os IPs:
```bash
tailscale status --json | python3 -c "import json,sys;d=json.load(sys.stdin);\
print(d['Self']['DNSName'], d['Self']['TailscaleIPs']);\
[print(p['DNSName'], p['TailscaleIPs']) for p in d['Peer'].values()]"
```
Esqueleto de ACL (placeholders `<...>` a preencher pelo owner):
```jsonc
{
  "hosts": {
    "orq1":        "100.118.244.61",
    "orq2":        "100.110.178.47",
    "owner-dev-1": "<IP-do-dispositivo-do-owner>"
    // adicionar apenas os dispositivos que o owner declarar
  },
  "acls": [
    // ORQ-17: UI do Multica no orq1, HTTPS 443, somente dispositivos do owner
    { "action": "accept", "src": ["owner-dev-1"], "dst": ["orq1:443"] },

    // manter o que a frota precisa hoje, sem alargar
    { "action": "accept", "src": ["orq2"], "dst": ["orq1:22", "orq1:18080"] }
    // NOTA: nao ha regra dando orq1:443 a orq2, wsl-dataops-labs ou ec2-jump-box.
    // Em ACL Tailscale o default e DENY, logo a ausencia de regra ja e a exclusao.
  ]
}
```
Ponto que precisa estar explicito para o owner: **nao existe "deny" em ACL Tailscale**; a exclusao
dos nos de agente e obtida pela **ausencia** de regra para `orq1:443`. Portanto qualquer regra
ampla pre-existente (por exemplo um `accept` de `*` para `*`, comum em tailnets novos) **anula** esta
restricao. Antes de aplicar, o owner precisa **ler a ACL atual** no admin console; eu nao tenho
acesso a ela e nao a inferi.

### V4.3.3 Proposta B - por tag nos nos de agente (mais robusta, mas e mutacao maior)
Marcar `orq1`, `orq2`, `wsl-dataops-labs` e `ec2-jump-box` com `tag:agent` permite regra por classe
em vez de por IP, e sobrevive a troca de IP. Custo: aplicar tag a um no **transfere a posse do no
para a tag e re-autentica o dispositivo**, o que pode derrubar conectividade e, no caso do orq1,
derrubar o tunel do ORQ2 e o daemon. **Nao recomendo dentro do Gate A.** Fica como melhoria
posterior, com janela propria.

### V4.3.4 Grants (sintaxe nova)
Se o tailnet ja usa `grants` em vez de `acls`, o equivalente e:
```jsonc
{ "grants": [ { "src": ["owner-dev-1"], "dst": ["orq1"], "ip": ["tcp:443"] } ] }
```
**Nao verifiquei** qual das duas sintaxes a policy atual usa - nao tenho acesso ao admin console.
O owner deve escolher a que corresponde ao arquivo dele; misturar `acls` e `grants` no mesmo policy
file e fonte comum de erro.

### V4.3.5 Verificacao e rollback da ACL
Verificacao, do dispositivo do owner e de um no de agente:
```bash
# DO DISPOSITIVO DO OWNER: deve SUCEDER
curl -fsSI https://orq1.tail96e2c0.ts.net/ | head -1
# DO ORQ2 (no de agente): deve FALHAR por ACL, nao por 404
curl -sS --max-time 5 -o /dev/null -w '%{http_code}\n' https://orq1.tail96e2c0.ts.net/ ; echo "exit=$?"
# no proprio orq1, ver se o filtro recebeu a regra:
tailscale debug prefs 2>/dev/null | head -20
tailscale status --json | python3 -c "import json,sys;print(json.load(sys.stdin).get('CertDomains'))"
```
Sinal de sucesso: do ORQ2 a conexao **nao completa** (timeout/refused no nivel de rede), e nao um
HTTP 403 - ACL bloqueia antes do HTTP.
Rollback da ACL: **o admin console versiona o policy file**; o rollback e restaurar a revisao
anterior no console. Nao ha rollback por CLI no host. Isso e acao do owner, e precisa estar dito
antes de aplicar - nao depois.

## V4.4 Idle 4h REMOVIDO do Gate A

Removido integralmente do escopo de execucao, por motivo tecnico ja evidenciado: o CSRF e derivado
do proprio token de auth - `internal/auth/cookie.go:141-145`, `mac := hmac.New(sha256.New,
[]byte(authToken))` - logo **qualquer renovacao/reemissao do token invalida o par CSRF existente**,
e uma janela deslizante implica reemissao continua. Somado a isso, nao existe estado de sessao:
grep de `refresh|rolling|sliding|renew` em `internal/auth/` e `handler/auth.go` = vazio; nao ha
tabela de sessao (migrations tem apenas `005_daemon_pairing` e `029_drop_daemon_pairing`).

Status: **problema futuro, separado, ENFILEIRADO**. Nao abro card agora - o Kanban esta em FREEZE
pelo incidente de ORQ-31 duplicado. Evidencia enfileirada aqui, para virar card quando houver
unfreeze e busca de duplicata por outro agente:

> **[FILA - sem numero]** Expiracao por inatividade da sessao web. Hoje impossivel: token stateless
> sem `last_seen`, e CSRF = `hmac(authToken)` (cookie.go:141-145), logo renovar o auth quebra o CSRF.
> Qualquer solucao precisa reemitir os DOIS cookies atomicamente e ter teto duro no `iat` original,
> ou introduzir estado de sessao (migration nova). Impacto: sem idle, a politica efetiva e apenas o
> teto absoluto. Dependencia: cutover ORQ-17 concluido. Risco/rollback: mexe no caminho de auth de
> todos os usuarios.

**Politica efetiva do Gate A, a ser dita ao owner sem ambiguidade**: teto absoluto de 24h, **sem**
timeout de inatividade.

## V4.5 Gate A - politica final, preflight, health e rollback

### V4.5.1 Politica do Gate A (e so isso)
1. login real (bypass **false**);
2. origin HTTPS: `FRONTEND_ORIGIN=https://orq1.tail96e2c0.ts.net` e `MULTICA_APP_URL` idem;
3. `Secure`/`SameSite` corretos por consequencia: `cookie.go:115-119` deriva `Secure` do **scheme**
   de `FRONTEND_ORIGIN`, e `cookie.go:164` fixa `SameSite=Strict`;
4. `AUTH_TOKEN_TTL=24h` (suportado: `cookie.go:36-61` aceita `"24h"`; default atual e 30 dias em
   `cookie.go:23`; `sync.Once` em `:74` obriga processo novo);
5. **roteamento em host unico** - `SameSite=Strict` exige frontend e API no mesmo host, e o mapa de
   V4.2 satisfaz isso;
6. **tunel preservado**: `multica-orq1-backend-tunnel.service` e user unit do **ORQ2** e serve o
   daemon, nao o browser. **Nao desligar.**

### V4.5.2 Preflight (tudo read-only, antes de qualquer mutacao)
```bash
# G0: cert do tailnet habilitado?
tailscale status --json | python3 -c "import json,sys;print('CertDomains=',json.load(sys.stdin).get('CertDomains'))"
# esperado: lista contendo orq1.tail96e2c0.ts.net  (hoje: None -> PARAR)

# borda ainda limpa e portas em loopback
tailscale serve status
ss -tln | grep -E '13100|18080|:443'   # esperado hoje: apenas 127.0.0.1:13100 e 127.0.0.1:18080

# saude interna
curl -fsS http://127.0.0.1:18080/health
curl -fsSI http://127.0.0.1:13100/ | head -1

# fila vazia (4 estados ativos, conforme 109_agent_task_waiting_local_directory.up.sql:15)
SELECT count(*) FROM agent_task_queue
WHERE status IN ('queued','dispatched','running','waiting_local_directory');

# env atual do backend, para saber o que muda
# (ler valores de FRONTEND_ORIGIN, CORS_ALLOWED_ORIGINS, MULTICA_TRUSTED_PROXIES,
#  MULTICA_LOCAL_AUTH_BYPASS, AUTH_TOKEN_TTL)
```

### V4.5.3 Verificacao pos-cutover (todas obrigatorias)
| # | verificacao | comando/esperado | se falhar |
|---|---|---|---|
| 1 | cert emitido | `tailscale status --json` -> `CertDomains` contem o FQDN | PARAR |
| 2 | rotas montadas | `tailscale serve status` lista raiz + `/api` + `/ws` + `/uploads` + os 3 `/auth/*` | PARAR |
| 3 | frontend | `curl -fsSI https://orq1.tail96e2c0.ts.net/` -> 200 | PARAR |
| 4 | rota do callback preservada | `curl -sSI https://.../auth/callback` -> resposta do **Next.js**, nao 404/405 do backend | PARAR - e a colisao de V4.1 |
| 5 | login real | POST em `/auth/login` pelo browser, com `Set-Cookie` contendo `Secure` e `SameSite=Strict` | PARAR |
| 6 | CSRF | um request mutante (ex. PATCH de issue) apos login -> 2xx | PARAR |
| 7 | WS | `/ws` conecta e recebe evento; se recusar, ajustar `MULTICA_TRUSTED_PROXIES` (ver `realtime/hub.go:158-180`, que so confia em `X-Forwarded-Host` se `isTrustedProxy(RemoteAddr)`) | PARAR |
| 8 | upload | arquivo > 10 MB, e leitura de volta por `/uploads/...` | PARAR |
| 9 | ACL | do ORQ2 a conexao a `:443` **falha na rede**; do dispositivo do owner sucede | PARAR |
| 10 | daemon intacto | `systemctl --user is-active` das duas units no ORQ2, e `daemon status` mostrando os 3 runtimes | PARAR e rollback |

**Regra de parada**: falha em qualquer um de 1-10 interrompe o cutover e dispara o rollback de
V4.5.4. Nao ha "seguir e ajustar depois".

### V4.5.4 Rollback, em ordem de custo crescente
1. **Borda** (segundos, sem reiniciar nada): `tailscale serve reset`. Volta a nao haver ouvinte em
   443. Nenhum efeito em backend, frontend, daemon ou tunel.
2. **Env** (uma recriacao de container): restaurar `FRONTEND_ORIGIN=http://localhost:13100`,
   remover `AUTH_TOKEN_TTL`, e - somente se o acesso do owner tiver sido perdido -
   `MULTICA_LOCAL_AUTH_BYPASS=true`. Note que o bypass volta a funcionar **automaticamente** quando o
   origin volta a ser loopback, por `middleware/auth.go:38-55`.
3. **ACL** (admin console): restaurar a revisao anterior do policy file. Nao ha rollback por CLI.
4. **NUNCA**: desligar o tunel do ORQ2; reabilitar bypass com origin publico; reaplicar branch antiga
   ou copia bruta de diretorio de credencial (ressalva Codex56#B - reintroduziria o AGY
   task-incapaz e antecede o hardening `0600`/`O_NOFOLLOW`).

## V4.6 GATE ZERO - acao de negocio do owner, literal

Estado medido hoje: `CertDomains: None` e `tailscale serve status` = `No serve config`.

Acao exata, e **somente o owner** pode fazer, no admin console do tailnet
`cloud.labs.brazil@gmail.com`:
1. abrir **admin console > DNS**;
2. em **HTTPS Certificates**, clicar **Enable HTTPS**;
3. confirmar que o MagicDNS continua habilitado (ja esta: `MagicDNSEnabled: true`);
4. voltar ao ORQ1 e confirmar que `tailscale status --json` passou a listar
   `orq1.tail96e2c0.ts.net` em `CertDomains`.

Sem o passo 2, `tailscale serve --https=443` nao consegue certificado e **o cutover nao comeca**.
Nenhum agente pode executar isso, e nao ha contorno tecnico no host.

## V4.7 Checklist V4 (substitui a secao 10 da V3)

- [ ] **G0** owner habilita HTTPS Certificates (V4.6); `CertDomains` lista o FQDN.
- [ ] **G1** owner declara os dispositivos dele e **le a ACL atual**; aplica a proposta de V4.3.2
      (ou a de `grants`, V4.3.4), garantindo que nao ha regra ampla pre-existente anulando.
- [ ] **G2** preflight de V4.5.2 integral, com saidas anexadas; fila = 0.
- [ ] **G3** confirmar login real por **loopback** com bypass ainda ativo (garante que existe
      credencial utilizavel antes de desligar o bypass).
- [ ] **G4** aplicar env de V4.5.1 e recriar backend/frontend.
- [ ] **G5** health loopback 200 e login funcionando por loopback com bypass **OFF**.
- [ ] **G6** montar a borda com os 7 comandos de V4.2.
- [ ] **G7** verificacoes 1-10 de V4.5.3, todas PASS.
- [ ] **G8** registrar evidencia do card **ORQ-17** (UUID `7d873133-16d5-42c6-8595-629d6fb16251`)
      em `.deploy-control/p0/evidence/`, **NAO como comentario na issue**: sob o GTL
      ASSIGNMENT/COMMENT FREEZE, comentario em issue atribuida pode enfileirar task paga, e
      `POST` de comentario exige autorizacao de execucao do GTL. Confirmar daemon e tunel do
      ORQ2 ativos.
- [ ] **G9** idle 4h **fora deste gate**: item enfileirado em V4.4, aguardando unfreeze do Kanban.

G4 a G8 sao mutacao e exigem autorizacao escrita do owner. G0 e G1 sao acoes do owner no console.
Este documento nao autoriza nada.

## V4.8 Perguntas em aberto (declaradas, nao afirmadas)

- **qual dispositivo e o do owner**: 14 peers, todos da mesma identidade e sem tag. Nao escolho por
  ele. Candidatos online hoje: `msi-laptop-1` (windows), `hp-laptop` (windows), `ec2-jump-box`
  (linux), `wsl-dataops-labs` (linux), `orq2` (linux). Os tres ultimos parecem infraestrutura, nao
  dispositivo pessoal - mas isso e leitura de nome, nao fato.
- **conteudo da ACL atual**: sem acesso ao admin console. Se houver regra ampla, a proposta nao
  restringe nada.
- **`acls` ou `grants`** na policy atual: nao verificado.
- **valores atuais** de `CORS_ALLOWED_ORIGINS` e `MULTICA_TRUSTED_PROXIES` no backend: nao lidos.
- **senha do usuario do owner**: `125_user_password_credential` existe como migration; nao consultei
  o banco. E o risco do G3.
- **limite de body na borda** do `tailscale serve`: nao documentado no `--help`; medir no G7.8.

## V4.9 Nada mutado

Nenhuma execucao. Somente leitura: `tailscale version`, `tailscale serve --help`,
`tailscale serve status`, `tailscale status --json`, `ss -tln`, e leitura de fonte
(`cmd/server/router.go`, `apps/web/app/`, `internal/auth/`, `internal/middleware/`,
`internal/realtime/`, `internal/daemonws/`). Nenhum cert, serve, ACL, env, container, restart,
instalacao ou credencial tocado. Nenhum card criado (Kanban em FREEZE). As 9 issues preservadas
seguem intactas; zero rerun.

**Re-review independente solicitada antes de qualquer execucao.**

---

# SECAO V4.10 - RULING DE DISPOSITIVOS DO OWNER: **STOP POR AMBIGUIDADE** (ORQ-17)

- autor: Opus48#A - ORQ2 - pane w6:p1 - 2026-07-27T12:54Z
- card: **ORQ-17** (existente; Kanban em FREEZE, nenhum card criado ou mutado)
- ruling recebido: autorizar HTTPS 443 apenas para `msi-laptop`, `hp`, `manoelneto-laptop`
- modo: READ-ONLY. Nenhuma ACL, cert, serve, env, codigo, container ou rede tocado. Nenhum valor de
  segredo lido.

## V4.10.1 VEREDITO: **STOP**. Os tres nomes do ruling nao resolvem para um no unico.

A instrucao manda parar se houver nome ambiguo ou duplicado. Ha **os dois casos**:

| nome do ruling | resolve para | veredito |
|---|---|---|
| `msi-laptop` | **2 nos** com o MESMO `HostName` | **AMBIGUO** |
| `hp` | **0 nos** com esse `HostName` | **INEXISTENTE** |
| `manoelneto-laptop` | **2 nos** com o MESMO `HostName` | **AMBIGUO** |

Nao escolho por conta propria qual dos pares e o dispositivo autorizado, nem assumo que `hp`
significa `hp-laptop`. Autorizar o no errado concede 443 a um dispositivo que o owner nao aprovou;
autorizar o certo por adivinhacao e sorte, nao controle.

## V4.10.2 Mapeamento exato medido (`tailscale status --json` no ORQ1, read-only)

`HostName` NAO e unico neste tailnet. O identificador unico e o `DNSName` (e o `ID` do no).

| HostName | DNSName (unico) | Node ID | IPv4 | IPv6 | online | OS |
|---|---|---|---|---|---|---|
| `msi-laptop` | `msi-laptop-1.tail96e2c0.ts.net.` | `nFo3jVWzSA11CNTRL` | `100.112.85.91` | `fd7a:115c:a1e0::5834:555b` | **true** | windows |
| `msi-laptop` | `msi-laptop.tail96e2c0.ts.net.` | `n4VGspiHL721CNTRL` | `100.122.21.119` | `fd7a:115c:a1e0::e234:1577` | false | linux |
| `hp-laptop` | `hp-laptop.tail96e2c0.ts.net.` | `nWs3fyq48A21CNTRL` | `100.85.79.80` | `fd7a:115c:a1e0::fb34:4f52` | **true** | windows |
| `manoelneto-laptop` | `manoelneto-laptop-1.tail96e2c0.ts.net.` | `nM39yzyBqz11CNTRL` | `100.86.110.121` | `fd7a:115c:a1e0::1e34:6e7a` | false | windows |
| `manoelneto-laptop` | `manoelneto-laptop.tail96e2c0.ts.net.` | `nox87nmLpd11CNTRL` | `100.98.214.121` | `fd7a:115c:a1e0::af34:d67b` | false | linux |

Nos **nao autorizados** pelo ruling, listados para que a ACL os exclua por omissao e para o owner
conferir que nenhum foi esquecido:

| HostName | DNSName | Node ID | IPv4 | online | OS |
|---|---|---|---|---|---|
| `orq1` | `orq1.tail96e2c0.ts.net.` | `nus9yXJufe11CNTRL` | `100.118.244.61` | true | linux |
| `orq2` | `orq2.tail96e2c0.ts.net.` | `nUgByTFSuw11CNTRL` | `100.110.178.47` | true | linux |
| `wsl-dataops-labs` | `wsl-dataops-labs.tail96e2c0.ts.net.` | `nGe5CJnDeC21CNTRL` | `100.117.245.15` | true | linux |
| `ec2-jump-box` | `ec2-jump-box.tail96e2c0.ts.net.` | `ndHZovrehX11CNTRL` | `100.94.211.42` | true | linux |
| `lenovo-lab` | `lenovo-lab.tail96e2c0.ts.net.` | `nK8hGrpyb811CNTRL` | `100.120.203.49` | false | linux |
| `wsl-lenovo-lab` | `wsl-lenovo-lab.tail96e2c0.ts.net.` | `n7tNLoRv1k11CNTRL` | `100.104.184.20` | false | linux |
| `localhost` | `ipad-air-5th-gen-wifi.tail96e2c0.ts.net.` | `n2LYSsxzE511CNTRL` | `100.71.184.65` | false | iOS |
| `A07 de Manoel` | `a07-de-manoel.tail96e2c0.ts.net.` | `nMQDP2jYY411CNTRL` | `100.79.199.50` | false | android |
| `POCO X8 Pro Max` | `poco-x8-pro-max.tail96e2c0.ts.net.` | `nujmjeLaqu11CNTRL` | `100.116.153.53` | false | android |
| `X8ProMax` | `x8promax.tail96e2c0.ts.net.` | `nyyVb6mb1721CNTRL` | `100.80.41.51` | false | linux |

Total: **15 nos**, todos do user `7650130887043022` (`cloud.labs.brazil@gmail.com`), **nenhum com
tag** - confirmando o achado de V4.3.1 de que identidade nao separa owner de agente.

## V4.10.3 As tres perguntas que so o owner responde (bloqueiam o G1)

1. `msi-laptop` = `msi-laptop-1` (windows, **online**) **ou** `msi-laptop` (linux, offline)? Ou os dois?
2. `hp` = `hp-laptop` (windows, online)? Nao existe no com HostName `hp`; confirmar a correspondencia.
3. `manoelneto-laptop` = `manoelneto-laptop-1` (windows) **ou** `manoelneto-laptop` (linux)? Ou os dois?

A resposta deve vir por **DNSName ou Node ID**, nunca por HostName - HostName e ambiguo aqui.

## V4.10.4 Fragmento de ACL / Grants - parametrizado, NAO aplicavel ate V4.10.3

Restricao estrutural que preciso declarar: `src` de ACL Tailscale aceita usuarios, grupos, tags,
`autogroup:*`, IPs e nomes definidos em `hosts` (que sao apelidos para IP). **Nao aceita Node ID.**
Ou seja o unico seletor simultaneamente **inequivoco e expressavel** e o **IP** - o Node ID e estavel
mas inutilizavel em `src`, e o HostName e utilizavel mas ambiguo. Consequencia direta: a ACL fica
amarrada a IP, e por isso o risco de renumeracao de V4.10.5 e **estrutural**, nao um detalhe.

Variante `acls` (sintaxe classica):
```jsonc
{
  "hosts": {
    "orq1": "100.118.244.61",
    "orq2": "100.110.178.47",
    // preencher APENAS apos V4.10.3, com o IP do no confirmado por DNSName/NodeID:
    "owner-msi":       "<IP do no msi-laptop confirmado>",       // 100.112.85.91 OU 100.122.21.119
    "owner-hp":        "<IP do no hp confirmado>",               // provavelmente 100.85.79.80
    "owner-manoelneto":"<IP do no manoelneto-laptop confirmado>" // 100.86.110.121 OU 100.98.214.121
  },
  "acls": [
    // ORQ-17: UI do Multica no ORQ1 via HTTPS 443, somente os 3 dispositivos autorizados
    { "action": "accept",
      "src": ["owner-msi", "owner-hp", "owner-manoelneto"],
      "dst": ["orq1:443"] },

    // PRESERVAR o que a frota ja usa: ORQ2 -> ORQ1 SSH e backend por tunel
    { "action": "accept", "src": ["orq2"], "dst": ["orq1:22", "orq1:18080"] }
  ]
}
```
Variante `grants` (sintaxe nova), equivalente:
```jsonc
{
  "grants": [
    { "src": ["owner-msi", "owner-hp", "owner-manoelneto"],
      "dst": ["orq1"], "ip": ["tcp:443"] },
    { "src": ["orq2"], "dst": ["orq1"], "ip": ["tcp:22", "tcp:18080"] }
  ]
}
```
**Nao sei qual das duas a policy atual usa** - nao tenho acesso ao admin console e nao li a policy.
O owner deve usar a variante que corresponde ao arquivo dele. Misturar `acls` e `grants` no mesmo
policy file e erro comum e silencioso.

Se a decisao de V4.10.3 incluir os **dois** nos de um par, basta adicionar a segunda entrada em
`hosts` e cita-la no mesmo `src` - o fragmento nao muda de forma.

## V4.10.5 "Provar que nenhuma wildcard concede 443" - o que eu **posso** e o que **nao posso** provar

Nao posso provar. Motivo factual: a policy do tailnet vive no admin console, **nao no host**, e eu
nao tenho acesso a ela. `tailscale debug prefs` e `tailscale status` mostram o estado do no e o
filtro efetivo compilado, nao o texto da policy. Portanto qualquer afirmacao minha de que "nao existe
wildcard" seria invencao. Declaro isso em vez de simular a prova.

O que posso entregar e o **procedimento de prova**, para o owner executar no console, e a razao pela
qual ele e obrigatorio:
1. Em ACL Tailscale **nao existe regra de negacao**. O default e deny, e a exclusao dos agentes vem
   da **ausencia** de regra concedendo `orq1:443`.
2. Logo **qualquer** regra pre-existente cuja intersecao inclua `orq1:443` **anula** a restricao.
   Os padroes que anulam, e que sao exatamente o default de tailnets novos:
   - `{"action":"accept","src":["*"],"dst":["*:*"]}`
   - `{"action":"accept","src":["autogroup:member"],"dst":["*:*"]}`
   - `{"action":"accept","src":["cloud.labs.brazil@gmail.com"],"dst":["*:*"]}` - concede tambem aos
     agentes, porque **todos os 15 nos sao dessa identidade** (V4.10.2)
   - qualquer `dst` com `orq1:*`
3. Procedimento: abrir **Access controls** no console, e antes de aplicar o fragmento, buscar
   literalmente por `"*"`, `*:*`, `autogroup:member`, `autogroup:self` e `orq1:*`. Se qualquer um
   existir num `accept` que alcance `orq1:443`, o fragmento de V4.10.4 **nao restringe nada** e o
   cutover **para** ate a regra ampla ser removida ou estreitada.
4. Prova empirica que substitui a leitura da policy, e essa **eu especifico como gate**: apos
   aplicar, a tentativa do ORQ2 contra `orq1:443` tem de falhar **na camada de rede** (V4.10.6,
   teste 4). Se responder HTTP, existe wildcard.

## V4.10.6 Validacao obrigatoria - 3 permitidos + 1 negado

Executar **apos** G0 (cert), G1 (ACL) e G6 (borda), e todos os quatro tem de dar o resultado esperado:

| # | de onde | comando | esperado |
|---|---|---|---|
| 1 | dispositivo `msi-laptop` confirmado | `curl -fsSI https://orq1.tail96e2c0.ts.net/` | `HTTP/2 200` |
| 2 | dispositivo `hp` confirmado | idem | `HTTP/2 200` |
| 3 | dispositivo `manoelneto-laptop` confirmado | idem | `HTTP/2 200` |
| 4 | **ORQ2** (`100.110.178.47`) | `curl -sS --max-time 5 -o /dev/null -w '%{http_code} exit=%{exitcode}\n' https://orq1.tail96e2c0.ts.net/` | **falha de rede**: timeout ou connection refused, `%{http_code}` = `000`. **Qualquer** codigo HTTP, inclusive 403, indica wildcard ativa -> **PARAR** |

Complemento no ORQ2, para distinguir ACL de DNS/rota:
```bash
tailscale ping orq1            # deve continuar OK: ACL de 443 nao afeta o ping do tailnet
nc -z -w3 100.118.244.61 22    # deve continuar ABERTO: regra ORQ2->ORQ1:22 preservada
nc -z -w3 100.118.244.61 18080 # deve continuar ABERTO: tunel do daemon preservado
nc -z -w3 100.118.244.61 443   # deve FECHAR: e o unico efeito desejado
```
Os dois `nc` do meio sao a prova de que a nova ACL **nao quebrou** o tunel `ORQ2 -> ORQ1` nem o SSH.
Se algum deles fechar, a ACL foi restritiva demais e o daemon perde o backend - rollback imediato.

## V4.10.7 Risco de renumeracao de dispositivo

Como a ACL amarra em IP (V4.10.4), o acesso quebra **silenciosamente** se o IP do no mudar. Quando
isso acontece:
- no **reinstalado** ou re-autenticado com nova chave: recebe **novo** node ID e **novo** IP;
- no **removido e readicionado** ao tailnet: idem;
- WSL/VM recriada: idem;
- **nao** acontece por reboot, troca de rede ou mudanca de IP publico - o IP `100.x` do Tailscale e
  estavel enquanto o no existir.

Sintoma: o dispositivo do owner passa a receber **falha de rede**, nao 403 - identico a um bloqueio
de ACL correto, o que torna o diagnostico confuso. Deteccao:
```bash
tailscale status --json | python3 -c "import json,sys;d=json.load(sys.stdin);\
[print(p['DNSName'],p['ID'],p['TailscaleIPs']) for p in d['Peer'].values()]"
```
comparar com a tabela de V4.10.2. Mitigacao estrutural seria `tag:owner-device`, que sobrevive a
renumeracao - mas aplicar tag **re-autentica o no** e, no caso dos hosts de infraestrutura, poderia
derrubar tunel e daemon; por isso segue **fora do Gate A** (V4.3.3). Mitigacao aceita agora:
revalidar a tabela de IPs sempre que um dispositivo autorizado for reinstalado, e registrar o
procedimento na ORQ-17.

## V4.10.8 Rollback da policy

Unico caminho: **restaurar a revisao anterior do policy file no admin console**. O console versiona
as edicoes de Access controls; nao existe rollback por CLI no host, e o `tailscale` do ORQ1 nao pode
reverter policy. Consequencia operacional a registrar antes de aplicar: **anotar a revisao vigente**
(data/hora e autor) antes da edicao, senao o rollback vira arqueologia. Rollback da **borda** e
independente e barato: `tailscale serve reset` (V4.5.4).

## V4.10.9 Ruling de custo - registrado

| item | ruling |
|---|---|
| Certificado HTTPS | Emitido via **Let's Encrypt** pela infraestrutura do Tailscale. **Sem taxa de certificado**, e a renovacao e automatica. Nao ha custo por dominio nem por emissao. |
| Plano Tailscale | O recurso HTTPS Certificates depende do **plano vigente do tailnet**. **Acao do owner**: verificar em Admin console > Settings > Billing qual e o plano atual e se ha cobranca por usuario/dispositivo. Eu **nao** li faturamento e **nao** afirmo qual e o plano. |
| Headscale | Alternativa **open-source** ao coordinator do Tailscale, sem custo de licenca. **FORA do Gate A** por risco de migracao e de operacao: exigiria migrar os 15 nos, reautenticar todos, operar o coordinator e refazer ACL - e derrubaria o tunel `ORQ2 -> ORQ1` e o daemon durante a migracao. Se o owner quiser avaliar, e problema separado, e hoje nem card pode ser criado (FREEZE). |

## V4.10.10 Passos que restam no console do owner (ORQ-17)

1. **Responder V4.10.3**: os tres dispositivos, por **DNSName ou Node ID**.
2. **Ler a Access controls atual** e buscar `"*"`, `*:*`, `autogroup:member`, `autogroup:self`,
   `orq1:*` (V4.10.5). Se houver, decidir remover/estreitar **antes** de tudo.
3. **Anotar a revisao vigente** da policy, para o rollback de V4.10.8.
4. **DNS > HTTPS Certificates > Enable HTTPS** (V4.6). Confirmar `CertDomains` no ORQ1.
5. **Aplicar** o fragmento de V4.10.4, na variante correta (`acls` ou `grants`).
6. Autorizar por escrito os passos G4-G8, que sao mutacao.

Passos 1-4 sao pre-condicao e nao dependem de nenhum agente. Nada em V4.10 foi aplicado.

## V4.10.11 Nada mutado

Nenhuma ACL, cert, serve, env, codigo, container, rede ou card tocado. Nenhum valor de segredo lido -
apenas `tailscale status --json` (hostnames, DNSNames, node IDs, IPs, online, OS) e `tailscale
serve --help`/`status`. Kanban em FREEZE respeitado: nenhum card criado, nenhum ORQ-31 citado como
referencia. As 9 issues preservadas seguem intactas; zero rerun.

**STOP registrado em V4.10.1. Re-review independente e resposta de V4.10.3 antes de qualquer execucao.**

---

## V4.11 - Verificacao de identidade do card (GTL Kanban Controlled Unfreeze)

- registrado por: Opus48#A - 2026-07-27T12:58Z - **somente leitura, nenhum POST**

Conforme a regra do unfreeze controlado, a citacao deste documento deixa de ser provisoria e passa a
ser um par UUID+numero **verificado**, nao previsto:

| campo | valor |
|---|---|
| UUID | `7d873133-16d5-42c6-8595-629d6fb16251` |
| number | `17` |
| identifier | `ORQ-17` |
| workspace_id | `20fce817-895d-447b-965a-49f5e279314a` (`orq2-dev`) |
| title | "Publicar frontend 13100 com acesso LAN estável" |

Metodo, duas chamadas read-only no backend do ORQ1:
1. `GET /api/issues?workspace_slug=orq2-dev` -> o item com `number: 17` tem
   `id = 7d873133-16d5-42c6-8595-629d6fb16251`;
2. `GET /api/issues/7d873133-16d5-42c6-8595-629d6fb16251?workspace_slug=orq2-dev` -> devolve o
   **mesmo par**, `"number":17` e `"identifier":"ORQ-17"`.

Nota de API, util para quem for registrar cards: o `GET` por UUID **exige** `workspace_slug` ou
`workspace_id`; sem isso responde `400 {"error":"workspace_id or workspace_slug is required"}`. Um
verificador que so olhe o status HTTP interpretaria isso como card inexistente.

Todas as secoes V3, V4 e V4.10 deste documento referem-se a este UUID. **Nao criei card**: nao sou o
registrar nomeado pelo GTL, e sigo proibido de `POST /api/issues`. O item de idle 4h (V4.4) e a
avaliacao de Headscale (V4.10.9) permanecem **enfileirados sem numero**, aguardando que o registrar
designado os crie um por vez, com o par UUID+numero reportado pelo proprio registrar.
