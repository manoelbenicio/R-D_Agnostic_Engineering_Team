# ORQ-17 — Monitor independente de contenção de emergência (READ-ONLY) — **PASS**

- Card: **ORQ-17** (`7d873133-16d5-42c6-8595-629d6fb16251`)
- Monitor: Kiro-Opus5 (sem poder de decisão, AA-001 §0.0) — independente do executor
- Nó: ORQ1 `orq1.tail96e2c0.ts.net` / `100.118.244.61`
- Amostras UTC: **T0 = 2026-07-27T15:11:49Z** e **T1 = 2026-07-27T15:12:46Z**
- Modo READ-ONLY. **Não executei reset, não reiniciei nada, não editei env, não mutei config.**
  Só leitura: `serve status`, `funnel status`, `serve status --json`, `ss -ltn`, e requisições
  idempotentes. Única escrita: este arquivo.

## VEREDITO: **PASS** — contenção confirmada e estável em duas amostras

A exposição sem autenticação que eu reportei em
`gtl-orq17-postgate-auth-config-audit.md` está **fechada**. Os serviços locais e o caminho
administrativo continuam intactos.

## 1. Serve e Funnel ausentes no ORQ1

| verificação | T0 | T1 |
|---|---|---|
| `tailscale serve status` | `No serve config` | `No serve config` |
| `tailscale funnel status` | `No serve config` | `No serve config` |
| `tailscale serve status --json` | — | **`{}`** (config vazia, sem `TCP`, sem `Web`, sem `AllowFunnel`) |

O JSON vazio é a confirmação mais forte: não há handler, não há bind de 443 e não há flag de Funnel
remanescente.

## 2. Porta 443 fechada

| verificação | resultado |
|---|---|
| TCP direto `100.118.244.61:443` de ORQ2 (T0 e T1) | **`443_CLOSED`** nas duas amostras |
| `HEAD https://orq1.tail96e2c0.ts.net/` | `http_code=000` (sem conexão) |
| `GET https://orq1.tail96e2c0.ts.net/` (T1) | `000` |
| `GET https://orq1.tail96e2c0.ts.net/api/issues` (T1) | `000` — a rota que antes devolvia `200` com dados reais **não responde mais** |
| Listener local em ORQ1 na porta 443 | `ss -ltnH | grep -E ':443$'` → **`NO_LISTENER_ON_PORT_443`** |

Nota de precisão: um grep ingênuo por `:443` casa também com `[fd7a:…]:44364`, que é porta efêmera
alta e **não** é o serviço. Confirmei com âncora de fim de string que **não existe** listener na
porta 443.

## 3. Serviços loopback intactos

| serviço | T0 | T1 |
|---|---|---|
| Frontend `127.0.0.1:13100/` | `200` | `200` |
| Backend `127.0.0.1:18080/healthz` | `200` | `200` |

Containers no ORQ1, sem restart durante a contenção — os uptimes seguem os mesmos que eu medi na
auditoria anterior: `multica-dev-transition-backend-1` `Up 4 hours`,
`multica-dev-transition-frontend-1` `Up 12 hours`, `multica-dev-transition-postgres-1`
`Up 5 days (healthy)`, `omniroute` `Up 2 days (healthy)`. Ou seja, a contenção foi feita **apenas no
serve**, sem tocar em processo, o que é exatamente o comportamento desejado.

## 4. Túnel ORQ2 intacto — caminho administrativo preservado

| verificação | T0 | T1 |
|---|---|---|
| Listener `127.0.0.1:18080` em ORQ2 (processo `ssh`) | presente (v4 e v6) | presente |
| `GET http://127.0.0.1:18080/health` via túnel | `200` | `200` |
| `GET http://127.0.0.1:18080/api/issues?workspace_id=…` via túnel | — | `200` |

O acesso administrativo não depende do serve, como eu havia registrado no monitor do Gate A. Isso
significa que a contenção **não causou lockout**.

Observação de segurança que permanece aberta e não é deste escopo: o túnel continua chegando a um
backend cujo `MULTICA_LOCAL_AUTH_BYPASS` ainda é `true` (medido às 15:12Z). Isso agora está restrito
a quem tem o túnel, não ao tailnet — mas a correção de env do item 2 da minha contenção recomendada
**segue pendente**, e o serve não deve ser religado antes dela.

## 5. Situação consolidada

| item pedido | resultado |
|---|---|
| `No serve config` | **PASS** (T0 e T1, mais `--json` = `{}`) |
| FQDN 443 fechado | **PASS** (TCP fechado, HTTPS `000`, sem listener em 443) |
| Funnel ausente | **PASS** (T0 e T1, e sem `AllowFunnel` no JSON) |
| Túnel ORQ2 intacto | **PASS** (listener presente, `/health` e `/api` respondendo `200`) |
| Serviços loopback intactos | **PASS** (13100 e 18080 em `200`, containers sem restart) |

## Check-out — ORQ-17

- Veredito: **PASS**. Contenção efetiva e estável em duas amostras separadas por ~57 s. A superfície
  de tailnet sem autenticação está fechada; nada foi derrubado no processo.
- Pendência que continua bloqueando o religamento do serve: `MULTICA_LOCAL_AUTH_BYPASS=false` junto
  com `FRONTEND_ORIGIN`/`MULTICA_APP_URL` em `https://orq1.tail96e2c0.ts.net`, num único movimento,
  com credencial do owner previamente provada; depois `AUTH_TOKEN_TTL=24h`, `CORS_ALLOWED_ORIGINS`,
  `MULTICA_TRUSTED_PROXIES=127.0.0.1/32` e revisão de `APP_ENV`. Sessões emitidas na janela de
  exposição vivem 30 dias e precisam de revisão.
- Não executei reset, restart, recreate, edição de env, autenticação nem leitura de banco, JWT ou
  senha. Nenhum segredo impresso. Worktrees ORQ-26 e ORQ-17 intocados.
