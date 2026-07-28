# ORQ-17 — (A) Auditoria da janela do Gate A + (B) Plano de cutover atômico de auth

- Card: **ORQ-17** (`7d873133-16d5-42c6-8595-629d6fb16251`)
- Autor: Kiro-Opus5 (sem poder de decisão, AA-001 §0.0)
- Data UTC: 2026-07-27T15:17Z
- Modo READ-ONLY. Não editei env, container, serve nem board; não autentiquei; **não li corpo de
  requisição, cookie, token, JWT, senha ou qualquer segredo**. A auditoria usou apenas o access log
  estruturado, agregado por contagem. Única escrita: este arquivo.

---

# PARTE A — Auditoria da janela do Gate A

## A.0 Fonte e método

Access log do middleware `RequestLogger` (`internal/middleware/request_logger.go:119-195`), que emite
apenas `method`, `path` (com webhook redigido), `status`, `duration`, `request_id`, `user_id` e
`client_*`. **Não há corpo, cookie, header de autorização ou token no log** — verifiquei no código
antes de olhar o log. Todo o resultado abaixo é agregação por contagem; nenhuma linha crua foi
publicada.

Janela: `docker logs --since 2026-07-27T14:00:00Z --until 2026-07-27T15:15:00Z` no container
`multica-dev-transition-backend-1` (ORQ1). Primeira linha `14:00:11.129`, última `15:14:59.338`.

## A.1 Volume por método e status

| método | requests |
|---|---:|
| GET | 498 |
| POST | 451 |
| OPTIONS | 1 |
| **total** | **955** |

| status | requests |
|---|---:|
| 200 | 929 |
| 404 | 8 |
| 405 | 6 |
| 400 | 4 |
| 201 | 3 |

Nenhum 5xx. Nenhum 401/403 — coerente com o bypass ativo: **nada foi recusado por falta de
autenticação durante a janela**.

## A.2 Composição do tráfego

| classe | requests | leitura |
|---|---:|---|
| `/api/daemon/**` (polling do daemon local) | 763 | tráfego normal de loop: `tasks/claim` em 3 runtimes, `runtime-profiles`, `repos`, `gc-check` |
| `/api/workspaces` | 150 | polling do daemon/frontend |
| **restante (não-daemon)** | **42** | detalhado em A.3 |

Ou seja **95,6% da janela é o loop do daemon local**, e a superfície realmente interessante são 42
requests em 75 minutos.

## A.3 As 42 requests não-daemon, item por item

| # | método · rota · status | atribuição |
|---:|---|---|
| 10 | GET `/api/issues` 200 | probes de monitoria e leituras de board |
| 2 | GET `/api/issues` 400 | probe sem `workspace_id` (minha, Gate A monitor) |
| 6 | GET `/healthz` 200 | health checks |
| **3** | **POST `/api/issues` 201** | **as únicas mutações não-daemon: ORQ-42, ORQ-43, ORQ-44** |
| 5 | GET `/api/issues/<uuid>` 200 | GET-back de verificação dos cartões criados |
| 2 | GET `/ws` 400 | probe (sem upgrade) |
| 2+1+1 | GET/OPTIONS `/uploads`, `/uploads/nonexistent.png` 404 | probes |
| 2+2+2 | GET `/auth/login`, `/auth/google`, `/auth/logout` 405 | probes com GET em rota POST |
| 1 | GET `/api` 404 | probe |
| 1 | GET `/api/v1/issues` 404 | probe de rota inexistente |
| 1 | GET `/api/healthz` 404 | probe |
| 1 | GET `/api/auth/me` 404 | probe |

**Cada uma das 42 é explicável** por (a) monitoria read-only minha e do executor, ou (b) as três
criações autorizadas de cartão com seu GET-back. Não há acesso não explicado.

## A.4 Mutações

- **451 POST**, das quais **448 são `/api/daemon/**/tasks/claim`** (200) — loop normal do daemon.
- **3 POST `/api/issues` → 201**: ORQ-42 `64bfcae0-b867-4812-ad33-ae03ef7f25ae` (15:13:32Z),
  ORQ-43 `d1149dd3-9da8-4678-a4b7-d98f3eddca14` (15:14:02Z),
  ORQ-44 `abd12d6a-16a5-439b-bc52-74ec4bb6b231` (15:14:34Z). Todos com `assignee_type=null`, o que
  **respeita o freeze de atribuição** (ORQ-41).
- **Zero PUT, zero PATCH, zero DELETE** em toda a janela.
- Nota temporal relevante: as três criações ocorreram **depois** do reset do serve (confirmado
  `No serve config` às 15:11:49Z), logo entraram pelo caminho loopback/túnel do registrar
  autorizado, **não** pela superfície de tailnet.

## A.5 Sessões e cookies emitidos — **zero**, por contagem segura

| sinal | contagem |
|---|---:|
| `POST /auth/login` bem-sucedido | **0** (só 2 GETs → 405) |
| `POST /auth/verify-code` | **0** |
| Linhas de evento `user logged in with password` | **0** |
| Linhas casando `verification code|logged in|signup` | **0** |

Portanto **nenhuma sessão foi criada e nenhum cookie de autenticação foi emitido durante a janela**.
Isto é a boa notícia da auditoria: o token de 30 dias que eu havia sinalizado como agravante
**não foi materializado** — não há sessão de longa duração para revogar. Consegui isso sem contar
nem inspecionar cookie algum: a contagem é de eventos de emissão no log, não de cookies.

## A.6 Identidade

`user_id` distintos no log da janela: **1**. Nenhum valor foi impresso. Coerente com um único
principal ativo (o operador do bypass, que é também o dono dos tokens de daemon).

## A.7 Origem Tailscale — **indisponível**, declarado

Não é possível atribuir origem por peer:

1. o access log **não registra `RemoteAddr`**;
2. `MULTICA_TRUSTED_PROXIES` está vazio, logo `X-Forwarded-For` é ignorado e nem seria logado;
3. o `tailscale serve` proxia a partir de loopback, então mesmo com `RemoteAddr` tudo apareceria como
   `127.0.0.1`;
4. `journalctl -u tailscaled` na janela contém apenas auditoria de **Tailscale SSH** (sessões
   `ssh-session(...)`, inclusive as minhas de leitura), **não** log por requisição HTTP do serve.

Consequência honesta: **não posso provar nem refutar** que algum peer do tailnet fez requisições pelo
serve durante a janela. O que posso afirmar é que o total de requests não-daemon é 42 e que **todas**
casam com monitoria conhecida e com as 3 criações autorizadas — não sobra volume para atividade de
terceiro. Fechar essa lacuna é exatamente para que serve `MULTICA_TRUSTED_PROXIES=127.0.0.1/32` mais
log de IP na próxima janela.

## A.8 Veredito da Parte A

**Sem evidência de abuso.** Exposição real existiu (é fato, e está em
`gtl-orq17-postgate-auth-config-audit.md`), mas na janela auditada: nenhuma sessão emitida, nenhuma
mutação além de 3 criações autorizadas e do loop do daemon, nenhum 5xx, nenhum acesso não explicado,
um único principal. Não há incidente de dados a declarar; há uma lacuna de observabilidade (A.7) a
corrigir.

---

# PARTE B — Plano de cutover atômico de auth (nada executado)

Princípio: **uma única troca de env, um único recreate, e o serve só volta depois da prova de
login.** A ordem existe porque `FRONTEND_ORIGIN` governa três coisas ao mesmo tempo — o bypass
(`middleware/auth.go:38-55`), o flag `Secure` do cookie (`auth/cookie.go:114-118`) e a origem que o
front usa.

## B.0 Pré-condições, antes de tocar qualquer coisa

| # | pré-condição | como provar (executor, não eu) |
|---|---|---|
| P1 | **Caminho de login do owner existe** | ver B.1; sem isto o cutover é lockout garantido |
| P2 | `JWT_SECRET` atual é válido para produção | ≥32 bytes e não é o default conhecido (`auth/jwt.go:46-58`). Provar **sem imprimir**: `docker exec … sh -c 'test ${#JWT_SECRET} -ge 32 && echo OK_LEN'` |
| P3 | Túnel SSH de ORQ2 ativo como caminho de contingência | já medido: listener `127.0.0.1:18080`, `/health` 200 |
| P4 | Serve continua resetado durante todo o procedimento | `tailscale serve status` = `No serve config` |
| P5 | Snapshot dos 7 mounts para reapply | já registrado em `gtl-orq17-gate-a-postapply-independent-monitor.md` §1 |

## B.1 Ruling de `APP_ENV` — e por que ele decide o caminho de login

Hoje: `APP_ENV=development`. As consequências são acopladas:

| APP_ENV | `JWT_SECRET` | código de verificação de dev | login viável |
|---|---|---|---|
| `development` / `test` | **dispensado** (`jwt.go:46-58`) | **habilitado** (`handler/auth.go:176-190`) | e-mail + `MULTICA_DEV_VERIFICATION_CODE` |
| `production` | **exigido**, ≥32 bytes e não-placeholder | **desabilitado** | senha (`POST /api/auth/login`) **ou** código real por SMTP/Resend |

Fato que fecha a escolha: sem `RESEND_API_KEY`/`SMTP_HOST`, o `EmailService` cai no *dev tier* que
**registra a linha com o código redigido** por `redact.Text` (`service/email.go:345-352`) — ou seja,
**o código não é recuperável do log**. Logo, com `APP_ENV=production` e sem SMTP, o único login
possível é **senha**.

**Recomendação:** `APP_ENV=production` **mais** uma das duas provas de P1:

- **B.1a (preferida)** credencial de senha do owner existente e testada — a tabela existe desde
  `125_user_password_credential` e o provider está wired (`cmd/server/router.go:487`,
  `router.go:165-168`); ou
- **B.1b** configurar `SMTP_HOST`/`RESEND_API_KEY` para que o código chegue de fato.

Se nenhuma das duas estiver pronta, **não fazer o cutover**: manter serve resetado e o acesso pelo
túnel, e tratar P1 como bloqueio. Alternativa explicitamente **não** recomendada: manter
`APP_ENV=development` só para preservar o código de dev — isso mantém viva uma via de autenticação de
desenvolvimento num serviço exposto ao tailnet.

## B.2 Conjunto de env do cutover — aplicado de uma vez

```text
MULTICA_LOCAL_AUTH_BYPASS = false
MULTICA_LOCAL_AUTH_EMAIL  = (remover ou deixar vazio)
FRONTEND_ORIGIN           = https://orq1.tail96e2c0.ts.net
MULTICA_APP_URL           = https://orq1.tail96e2c0.ts.net
MULTICA_PUBLIC_URL        = https://orq1.tail96e2c0.ts.net
CORS_ALLOWED_ORIGINS      = https://orq1.tail96e2c0.ts.net
AUTH_TOKEN_TTL            = 24h
MULTICA_TRUSTED_PROXIES   = 127.0.0.1/32
APP_ENV                   = production        (ruling B.1)
COOKIE_DOMAIN             = (permanecer VAZIO)
GOOGLE_REDIRECT_URI       = https://orq1.tail96e2c0.ts.net/auth/callback   (se Google OAuth for mantido)
```

Justificativas ancoradas, para não haver mudança sem motivo:

- `COOKIE_DOMAIN` **vazio** é obrigatório: o host é um nome, mas `cookieDomain()`
  (`auth/cookie.go:91-113`) ignora IP e a RFC 6265 proíbe literal; com host único, cookie
  host-only é o correto.
- `AUTH_TOKEN_TTL=24h` substitui o default de 30 dias (`cookie.go:71-88`).
- `MULTICA_TRUSTED_PROXIES=127.0.0.1/32` é o que faz o `X-Real-IP` do serve ser aceito e devolve
  sentido ao rate limit por IP — e fecha a lacuna A.7.
- `SameSite` permanece `Strict` (código, `cookie.go:164`); nada a configurar.
- Frontend: `NEXT_PUBLIC_API_URL` precisa estar no ambiente **do build** do Next, porque
  `NEXT_PUBLIC_*` é inlinado (`apps/web/config/runtime-urls.ts:7`). Se a imagem atual foi construída
  apontando para outra origem, **recreate do container não basta** — exige rebuild. Isto precisa ser
  verificado antes, e é o item com maior chance de surpresa.

## B.3 Sequência atômica

| passo | ação | verificação de saída | quem autoriza |
|---|---|---|---|
| 1 | Confirmar P1-P5 | todas verdes | — |
| 2 | Serve permanece resetado | `No serve config` | — |
| 3 | Aplicar **todo** o bloco B.2 no `.env`/compose do ORQ1 | diff do env revisado, sem imprimir valores de segredo | owner (§0.1 credencial/infra) |
| 4 | `docker compose up -d --force-recreate` **só do backend** | container `Up`, `127.0.0.1:18080/healthz` 200 | owner (§0.1) |
| 5 | **Prova de bypass off** | `GET http://127.0.0.1:18080/api/issues?workspace_id=…` **sem credencial** deve devolver **401**, não 200 | — |
| 6 | **Prova de login sem lockout** | login pelo caminho de B.1 devolve 200 e `Set-Cookie` presente; **não** inspecionar o valor do cookie, só a presença e o `Max-Age`≈86400 | owner |
| 7 | Frontend: se `NEXT_PUBLIC_API_URL` estiver embutido errado, rebuild da imagem web | `GET 127.0.0.1:13100/` 200 e a página falando com a origem HTTPS | owner (§0.1 build) |
| 8 | Reaplicar o serve com os **7** mounts do snapshot | `serve status` com 7 mounts, `funnel status` "(tailnet only)", `AllowFunnel=null` | owner (§0.1) |
| 9 | Reverificação externa desde ORQ2 | `/` 200 HTML; `/api/issues` sem credencial **401**; `/auth/callback` 200; cert com CN/SAN do FQDN | monitor independente |
| 10 | Registrar evidência com contagens da nova janela | — | — |

O passo 5 é o que transforma "achamos que desligou" em prova: hoje a mesma requisição devolve 200 com
dados. Se depois do recreate ela continuar 200, **parar** e não seguir para o passo 8.

## B.4 Rollback

| cenário | ação | efeito |
|---|---|---|
| Falha em qualquer passo 3-7 | restaurar o bloco de env anterior e `--force-recreate` | volta ao estado atual (bypass on, serve **resetado**) |
| Falha no passo 8-9 | `tailscale serve reset` | fecha a superfície em segundos; túnel SSH mantém administração |
| Lockout (login não funciona após bypass off) | reativar **temporariamente** `MULTICA_LOCAL_AUTH_BYPASS=true` **com** `FRONTEND_ORIGIN` de volta a loopback **e** serve resetado | recupera acesso sem reabrir o tailnet; é a combinação que já existe hoje |
| Necessidade de invalidar sessões | rotação de `JWT_SECRET` é o mecanismo, e já tem cartão próprio (**ORQ-42**) | não misturar com este cutover |

Regra: **nunca** reativar o bypass com o serve ativo. A combinação proibida é exatamente
`bypass=true` + `FRONTEND_ORIGIN` loopback + serve publicado, que foi o incidente.

## B.5 O que este plano deliberadamente não faz

- Não rotaciona `JWT_SECRET` (ORQ-42), não toca token de daemon (ORQ-43) nem chave do OmniRoute
  (ORQ-44).
- Não altera ACL do tailnet.
- Não habilita Funnel.
- Não atribui assignee nem comenta em cartão (freeze ORQ-41).
- Não mexe em `playwright`/CI (ORQ-39) nem nos worktrees congelados ORQ-26 e ORQ-17.

## Check-out — ORQ-17

- **Parte A: PASS sem evidência de abuso.** 955 requests na janela, 95,6% loop do daemon, 42
  não-daemon todas explicadas, 3 mutações (criação de ORQ-42/43/44, todas unassigned, todas
  **após** o reset do serve), zero PUT/PATCH/DELETE, zero 5xx, zero 401/403, **zero sessão ou cookie
  emitido**, 1 principal. Origem por peer **indisponível** e declarada como lacuna (A.7).
- **Parte B: plano entregue, nada executado.** Bloqueio real do cutover é **P1** (caminho de login do
  owner) combinado ao ruling de `APP_ENV`; o item de maior risco escondido é
  `NEXT_PUBLIC_API_URL` embutido no build do frontend.
- Não editei env, container, serve ou board; não autentiquei; não li corpo, cookie, token, JWT, senha
  ou segredo; nenhuma linha crua de log publicada.
