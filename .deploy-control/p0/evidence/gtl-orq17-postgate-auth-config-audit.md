# ORQ-17 — Auditoria READ-ONLY da config de auth pós-Gate A no ORQ1 — **BLOCK**

- Card: **ORQ-17** (`7d873133-16d5-42c6-8595-629d6fb16251`)
- Auditor: Kiro-Opus5 (sem poder de decisão, AA-001 §0.0)
- Data UTC: 2026-07-27T15:12Z
- Nó: ORQ1 `orq1.tail96e2c0.ts.net` / `100.118.244.61`
- Modo READ-ONLY. Não editei env, não reiniciei nem recriei container, não autentiquei, não li JWT,
  banco nem senha. **Nenhum valor de segredo foi impresso ou coletado.** Única escrita: este arquivo.

---

## VEREDITO: **BLOCK IMEDIATO**

O Gate A publicou a stack no tailnet **com o bypass de autenticação local ATIVO**. Toda requisição
a `https://orq1.tail96e2c0.ts.net/api/*` é executada como o operador configurado, **sem login**.

Isto é exatamente a dependência circular **CD1** que eu registrei no review do design de ORQ-17
(`gtl-orq17-auth-peer-review.md` §2): o bypass só se desliga quando `FRONTEND_ORIGIN` deixa de ser
loopback, e a publicação foi feita sem essa troca.

---

## 1. Configuração efetiva medida (container `multica-dev-transition-backend-1`)

Somente as chaves solicitadas, por inspeção de `docker inspect -f '{{range .Config.Env}}...'`
filtrada por allowlist de nomes:

| chave | valor efetivo | esperado pelo gate | veredito |
|---|---|---|---|
| `FRONTEND_ORIGIN` | `http://localhost:13100` | `https://orq1.tail96e2c0.ts.net` | **FALHA** |
| `MULTICA_APP_URL` | `http://localhost:13100` | `https://orq1.tail96e2c0.ts.net` | **FALHA** |
| `MULTICA_LOCAL_AUTH_BYPASS` | **`true`** | `false` | **FALHA CRÍTICA** |
| `MULTICA_LOCAL_AUTH_EMAIL` | **presente e não vazio** (só presença verificada; valor não lido nem impresso) | ausente/vazio | **FALHA** |
| `CORS_ALLOWED_ORIGINS` | vazio | origem HTTPS explícita | FALHA |
| `AUTH_TOKEN_TTL` | **ausente** ⇒ default do código = **30 dias** (`internal/auth/cookie.go:71-88`, `defaultAuthTokenTTL`) | 24 h | **FALHA** |
| `MULTICA_TRUSTED_PROXIES` | vazio | `127.0.0.1/32` (loopback) | FALHA |

Contexto adicional, medido nas mesmas chaves de allowlist: `APP_ENV=development`,
`MULTICA_PUBLIC_URL` vazio, `COOKIE_DOMAIN` vazio. O container de frontend
(`multica-dev-transition-frontend-1`) **não** define nenhuma das sete chaves.

Nenhuma das quatro confirmações pedidas se sustenta: bypass **não** está off, a origem **não** é
`https://orq1.tail96e2c0.ts.net`, o TTL **não** é 24 h, e os trusted proxies **não** estão em
loopback.

## 2. Por que isso é exploração ativa, não risco teórico

Cadeia, com âncoras de código já verificadas por mim em auditorias anteriores deste card:

1. `middleware.localAuthBypassEmail()` (`internal/middleware/auth.go:38-55`) habilita o bypass quando
   `MULTICA_LOCAL_AUTH_BYPASS=true` **e** o host de `FRONTEND_ORIGIN` é `localhost`/loopback. As duas
   condições estão satisfeitas agora.
2. Com o bypass ativo, `Auth()` (`auth.go:86-137`) injeta `X-User-ID`/`X-User-Email` do operador e
   chama o próximo handler **sem exigir token**.
3. O Gate A montou `/api`, `/uploads`, `/ws` e `/auth/*` em
   `https://orq1.tail96e2c0.ts.net` (tailnet-only), conforme eu documentei em
   `gtl-orq17-gate-a-postapply-independent-monitor.md` §1.
4. **Prova empírica já registrada:** na monitoria do Gate A, `GET /api/issues?workspace_id=…`
   partindo do ORQ2, **sem nenhuma credencial**, devolveu `200` com o payload real do board. Na
   época eu registrei isso como "a API de produção está exposta ao tailnet"; a causa é este bypass.

## 3. Alcance (blast radius)

- **Quem alcança:** qualquer nó do tailnet `tail96e2c0`. No momento da medição estavam online, além
  do ORQ1: `orq2`, `msi-laptop`, `ec2-jump-box`, `wsl-dataops-labs`. Os demais peers registrados
  (laptops, iPad, celulares) passam a ter o mesmo acesso ao entrarem online. Não verifiquei ACL
  (fora de leitura), então assumo full-mesh como o preflight declarou — o que significa **todos**.
- **O que consegue fazer, sem autenticar:** tudo o que o operador faz pela API — ler e escrever
  issues, criar e apagar cartões, workspaces, agentes, squads, tokens de acesso pessoal, anexos via
  `/uploads`, e abrir WebSocket. Ou seja, também **atribuir assignee**, que pela governança de hoje
  é gatilho de execução paga (ORQ-41).
- **O que NÃO está exposto:** Funnel está ausente (confirmado por `funnel status` "(tailnet only)" e
  `AllowFunnel=null`), então não há exposição à internet pública. Postgres segue em
  `127.0.0.1:15433` e os upstreams em loopback. O OmniRoute escuta em `100.118.244.61:20128`, que é
  exposição de tailnet pré-existente e fora deste escopo.
- **Agravantes de configuração:**
  - `AUTH_TOKEN_TTL` ausente ⇒ token de 30 dias em vez de 24 h; qualquer sessão emitida agora vive
    30 dias.
  - `FRONTEND_ORIGIN` em `http://` ⇒ `isSecureCookie()` (`internal/auth/cookie.go:114-118`) devolve
    `false`, logo cookies de sessão sairiam **sem o atributo `Secure`** mesmo servidos por HTTPS.
  - `MULTICA_TRUSTED_PROXIES` vazio ⇒ XFF ignorado; como o `tailscale serve` origina de loopback,
    todo tráfego aparece com o mesmo IP e o rate limit por IP colapsa num único bucket.
  - `APP_ENV=development` ⇒ dispensa a validação de `JWT_SECRET` (`auth/jwt.go:46-58`) e habilita o
    código de verificação de desenvolvimento (`handler/auth.go:176-190`).

## 4. Contenção recomendada (nenhuma ação executada por mim)

Ordem importa. Os itens 1 e 2 são o mínimo para fechar a exposição.

| # | ação | efeito | classe |
|---|---|---|---|
| 1 | `tailscale serve reset` no ORQ1 | remove a exposição ao tailnet em segundos; o túnel SSH continua como acesso administrativo (medido, ainda ativo) | §0.1 infra — decisão do owner |
| 2 | Definir `MULTICA_LOCAL_AUTH_BYPASS=false` **e** `FRONTEND_ORIGIN`/`MULTICA_APP_URL` = `https://orq1.tail96e2c0.ts.net`, num único movimento | desliga o bypass e liga `Secure` nos cookies; exige recreate do container | §0.1 credencial/infra |
| 3 | `AUTH_TOKEN_TTL=24h`, `CORS_ALLOWED_ORIGINS=https://orq1.tail96e2c0.ts.net`, `MULTICA_TRUSTED_PROXIES=127.0.0.1/32`, `MULTICA_PUBLIC_URL=https://orq1.tail96e2c0.ts.net`, `APP_ENV` revisto | fecha os agravantes | §0.1 |
| 4 | Pré-condição antes de 2: provar credencial do owner utilizável, senão há **lockout** | `POST /api/auth/login` existe e o provider de senha é wired; sem `RESEND_API_KEY`/`SMTP_HOST` o código de verificação vai para o log do container | verificação |
| 5 | Após 2: rever tokens/sessões emitidos durante a janela de exposição (TTL de 30 dias) | qualquer sessão criada agora sobrevive à correção | decisão do owner |

**Não executei nenhuma delas.** Reset de serve, edição de env e recreate de container são
STOP-AND-WAIT (AA-001 §0.1) e o próprio dispatch me proíbe de editar env ou reiniciar.

## 5. Janela de exposição

- Serve aplicado com certificado emitido em **2026-07-27T14:07:29Z** (notBefore do cert), e o
  backend está `Up 4 hours` com esta configuração. Portanto a exposição sem autenticação existe
  desde, no mínimo, o momento do apply do Gate A — cerca de **1 hora** até esta medição
  (15:12Z), e **continua ativa agora**.
- O container de frontend está `Up 12 hours`, anterior ao gate.

## 6. Limites desta auditoria

- Não li nem imprimi valor de `MULTICA_LOCAL_AUTH_EMAIL`, `JWT_SECRET`, `DATABASE_URL`,
  `POSTGRES_PASSWORD` ou qualquer credencial; a inspeção usou allowlist de nomes e, para o e-mail,
  reportei apenas presença e o fato de ser não vazio.
- Não autentiquei, não abri sessão, não li banco.
- Não verifiquei a ACL do tailnet (exige admin API), então o alcance de §3 assume o full-mesh que o
  preflight declarou.
- Não inspecionei se algum outro nó do tailnet replica a mesma exposição.

## Check-out — ORQ-17

- Veredito: **BLOCK IMEDIATO**. Bypass de auth ativo em serviço exposto ao tailnet; quatro das
  quatro confirmações pedidas falharam; sete de sete chaves auditadas estão fora do esperado.
- Alcance: todo o tailnet `tail96e2c0`, com poder equivalente ao do operador na API, incluindo
  atribuição de assignee, que é gatilho de execução paga (ORQ-41).
- Contenção mínima: `tailscale serve reset` agora, e só religar o serve depois de
  `MULTICA_LOCAL_AUTH_BYPASS=false` com `FRONTEND_ORIGIN` HTTPS aplicados no mesmo movimento, com
  credencial do owner previamente provada para não haver lockout.
- Nada editado, reiniciado, recriado, autenticado ou lido de banco/JWT/senha por mim.
