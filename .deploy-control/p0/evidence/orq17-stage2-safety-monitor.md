# ORQ-17 STAGE 2 - monitor de seguranca independente (READ-ONLY)

- **Monitor:** Opus48#B (ORQ2, pane w6:p2) · **Cartao:** `ORQ-17` stage 2
- **Host observado:** ORQ1 — `ip-172-31-18-217.sa-east-1.compute.internal`, tailscale `100.118.244.61`
- **Janela:** 2026-07-27T15:53:58Z (S2-T0) -> 15:54:47Z (S2-T1)
- **Modo:** READ-ONLY. Nao buildei, nao reiniciei, nao recriei, nao editei, nao apliquei/resetei
  `serve`, nao fiz login, nao li segredo, nao mutei DB, board nem AWS.

# VEREDITO: **SEM DRIFT. NENHUM STOP.**
# Mais **UMA CORRECAO FACTUAL** (secao 3) e **uma regra de conferencia de compose** (secao 2),
# ambas medidas ANTES de qualquer recriacao.

---

## 1. QUEUE-ZERO - DUAS LEITURAS AGREGADAS, AMBAS `0`

Predicado com os **4 estados ativos**, derivado de
`migrations/109_agent_task_waiting_local_directory.up.sql:13-15`:
```sql
SELECT count(*) FROM agent_task_queue
WHERE status IN ('queued','dispatched','running','waiting_local_directory');
```
| leitura | UTC | resultado |
|---|---|---|
| 1 | 15:53:58Z | **0** |
| 2 | 15:54:47Z | **0** |

Contexto por status na leitura 2: `completed 137 · failed 71 · cancelled 7` — **nenhuma** linha em
estado ativo. Gate de fila **PASS** nas duas leituras independentes.

## 2. CAMINHOS DE COMPOSE - QUATRO NO LABEL VIVO, **CINCO ESPERADOS NO STAGE 2**

Medido no label do container **em execucao** (estado anterior ao stage 2):
```
project = multica-dev-transition
service = backend
com.docker.compose.project.config_files:
  1 /home/ec2-user/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.yml
  2 /home/ec2-user/R-D_Agnostic_Engineering_Team/multica-auth-work/docker-compose.selfhost.build.yml
  3 /home/ec2-user/.config/multica-transition/images.yml
  4 /home/ec2-user/.config/multica-transition/backend-env.override.yml
```
**Quatro** no label vivo. Esclarecimento recebido do TL e incorporado: o stage 2 deve usar **esses
quatro exatos mais um** override de auth novo e **nao-secreto**, totalizando **cinco**. Isso e
**esperado, nao drift** — retiro a duvida que eu havia levantado.

Regra de conferencia que passo a aplicar, e ela e mais estrita do que so contar arquivos:
```
ACEITO   : exatamente os 4 caminhos acima, na mesma grafia absoluta, MAIS 1 novo override de auth
           nao-secreto, e project = multica-dev-transition, service = backend
STOP      : qualquer caminho diferente dos 4 conhecidos
STOP      : ausencia de qualquer um dos 4
STOP      : project ou service diferente
STOP      : mais de um arquivo novo, ou novo arquivo que nao seja o override de auth
STOP      : qualquer container nao-backend alterado
```
Verificacao pos-recriacao: reler
`com.docker.compose.project.config_files` do **novo** container e conferir que os 4 primeiros sao
identicos, byte a byte na grafia, e que o quinto e o override de auth declarado.

Ponto que ainda registro como **a confirmar**, sem trata-lo como drift: o quinto arquivo **ainda nao
existe** no label vivo, por definicao, e eu **nao** verifiquei seu caminho nem seu conteudo. Quando o
executor o declarar, confiro apenas **caminho, modo e ausencia de segredo por metadado** — nao abro
conteudo. Se o override trouxer valor de segredo em claro, isso e materia de STOP por outra razao
(exposicao), nao por contagem.

## 3. CORRECAO FACTUAL - `/api/me` ANONIMO RETORNA **200**, E ISSO E O ESPERADO NESTE HOST

Medido, **antes** de qualquer recriacao, em S2-T0 e S2-T1:
```
anon_api_me = 200   (15:53:58Z)
anon_api_me = 200   (15:54:47Z)
```
O dispatch exige `401/403`. **Esse criterio nao pode ser satisfeito neste host como configurado**, e
o motivo esta no codigo, nao em falha de seguranca introduzida agora.

A rota **esta** no grupo autenticado, `cmd/server/router.go:552` dentro de:
```go
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(queries, patCache, cloudPATVerifier))
		r.Use(middleware.RefreshCloudFrontCookies(cfSigner))
		...
		r.Get("/api/me", h.GetMe)
```
Existe um **bypass local documentado**, `internal/middleware/auth.go:34-55`, com duas guardas:
```go
// localAuthBypassEmail returns the configured local user only when the
// operator explicitly enabled the bypass and the frontend is loopback-only.
// The second check prevents a copied production configuration from silently
// turning a public deployment into an unauthenticated instance.
func localAuthBypassEmail() string {
	if !strings.EqualFold(strings.TrimSpace(os.Getenv(localAuthBypassEnv)), "true") {
		return ""
	}
	frontendOrigin := strings.TrimSpace(os.Getenv("FRONTEND_ORIGIN"))
	parsed, err := url.Parse(frontendOrigin)
	...
	host := parsed.Hostname()
	ip := net.ParseIP(host)
	if !strings.EqualFold(host, "localhost") && (ip == nil || !ip.IsLoopback()) {
		return ""
	}
	return strings.TrimSpace(os.Getenv(localAuthBypassEmailEnv))
}
```
Constatei, **lendo apenas nomes de variavel** e nunca valores, que o container do backend define
`MULTICA_LOCAL_AUTH_BYPASS=`, `FRONTEND_ORIGIN=` e `APP_ENV=`. O `200` anonimo e, portanto,
consistente com o bypass ativo sob origem loopback — comportamento **deliberado** desta implantacao
self-host, com a segunda guarda existindo justamente para impedir que uma copia de configuracao de
producao virasse instancia sem autenticacao.

**Classificacao:** condicao **pre-existente**, medida **antes** da recriacao. **Nao e drift** causado
pelo stage 2. Mas o criterio "anonymous /api/me must fail 401/403" produziria **falso BLOCK** contra
um executor correto. Recomendacao ao TL: substituir por um dos dois abaixo, que sao verificaveis aqui:
- **A** `GET /api/me` com `Authorization: Bearer <token invalido sintetico>` deve devolver `401`;
- **B** `anon /api/me` deve permanecer **exatamente igual** antes e depois (`200` -> `200`), provando
  que a recriacao **nao alterou** a postura de auth.
Prefiro **B** como invariante de monitor: nao exige gerar token e detecta tanto afrouxamento quanto
endurecimento acidental.

Registro de risco, sem alarme: o bypass depende de `FRONTEND_ORIGIN` ser loopback. Se a recriacao
mudar `FRONTEND_ORIGIN` para um host nao-loopback, o bypass se desliga (`return ""`) e o `200` viraria
`401` — o que **tambem** seria drift de configuracao, detectavel pelo criterio B.

## 4. INVARIANTES - 2 AMOSTRAS, TODAS NO ESTADO EXIGIDO

| amostra | UTC | serve | funnel | :443 | 18080 /health | /readyz | 13100 | anon /api/me |
|---|---|---|---|---|---|---|---|---|
| S2-T0 | 15:53:58 | `No serve config` | `No serve config` | 0 | 200 | 200 | 200 | 200 |
| S2-T1 | 15:54:47 | `No serve config` | `No serve config` | 0 | 200 | 200 | 200 | 200 |

- **Serve vazio** nas duas. **Funnel vazio** nas duas. **Zero listeners em `:443`** nas duas.
- Backend `18080` `/health` e `/readyz` em `200` nas duas — **ainda nao houve janela de indisponibilidade**.

## 5. IDs DE CONTAINER - SOMENTE O BACKEND PODE MUDAR

```
             S2-T0          S2-T1          veredito
backend      dc719a1bf4e7   dc719a1bf4e7   inalterado (recriacao AINDA NAO ocorreu)
frontend     1b3b6c9ee32a   1b3b6c9ee32a   inalterado  <- exigido
postgres     2a4a84897363   2a4a84897363   inalterado  <- exigido
omniroute    2fb3fd57e885   2fb3fd57e885   inalterado  <- exigido
```
`frontend`, `postgres` e `omniroute` **intactos**. O `backend` tambem esta intacto, o que significa que
a recriacao autorizada **ainda nao aconteceu** na janela observada. Quando ocorrer, o unico ID que
pode mudar e o do backend; qualquer outro ID mudando e **STOP imediato**.

Baseline de referencia para a comparacao pos-recriacao:
`backend started = 2026-07-27T10:55:19Z`, `frontend started = 2026-07-27T03:11:24Z`,
`postgres started = 2026-07-21T17:10:49Z`, `omniroute started = 2026-07-24T18:36:12Z`.

## 6. METADADO DE BACKUP - PRESENTE E CONFORME

```
/home/ec2-user/.local/state/orq17-stage1-frontend-recovery-20260727T155000Z  mode=700
```
Verificado no stage 1 e reconferido agora por metadado: diretorio `0700`, quatro copias de
compose/override em `0600` correspondendo **exatamente** aos 4 `config_files` do label, mais
`tailscale-serve-status.before.json`, `frontend-safe-metadata.before.txt` e `SHA256SUMS`, todos `0600`;
`sha256sum -c` = **6 OK / 0 FAILED**. **Nao abri conteudo de backup algum.**

Ressalva de nomenclatura: o diretorio se chama `...stage1-frontend-recovery...`. Se o stage 2 recria o
**backend**, convem um diretorio proprio de stage 2 — ou a declaracao explicita de que este cobre os
dois. **Nao existe** diretorio de backup especifico de stage 2 nesta janela.

## 7. CONDICOES DE STOP - NENHUMA ACIONADA

| condicao | estado |
|---|---|
| `serve` ou `funnel` deixar de estar vazio | nao ocorreu |
| listener em `:443` | nao ocorreu |
| ID de `frontend`, `postgres` ou `omniroute` mudar | nao ocorreu |
| mais de um container recriado | nao ocorreu |
| backend nao recuperar `200` apos recriacao | nao aplicavel ainda: recriacao nao ocorreu |
| fila sair de `0` | nao ocorreu (2 leituras, ambas `0`) |
| backup ausente no momento de mutar | nao ocorreu |
| alvo no ORQ2/tunel em vez do Docker do ORQ1 | nao ocorreu: labels confirmam ORQ1 |

## 8. NAO-AFIRMACOES
- READ-ONLY: nenhuma mutacao partiu de mim.
- **Nao inspecionei `Config.Env` por completo.** Extraí apenas a **presenca de nomes** de tres
  variaveis (`MULTICA_LOCAL_AUTH_BYPASS`, `FRONTEND_ORIGIN`, `APP_ENV`) com `grep -oE '^NOME='`, que
  descarta o valor. **Nao li o valor de nenhuma variavel** e nao listei o env inteiro.
- **Nao abri conteudo de backup**: apenas `stat`, `find` e `sha256sum -c`.
- Nao verifiquei o caminho nem o conteudo do quinto arquivo de override de auth: ele ainda nao existe no label vivo, e quando existir eu confiro apenas caminho, modo e ausencia de segredo por metadado.
- Nao testei `/api/me` com token invalido sintetico, porque isso exigiria fabricar credencial; a
  recomendacao B da secao 3 evita essa necessidade.
- A janela e curta (~50 s, 2 amostras) e **anterior** a recriacao do backend. Nao afirmo nada sobre o
  comportamento durante ou depois dela; sigo monitorando.
- Nao avaliei o merito do plano de recuperacao; sou monitor.
- Nao criei, atribui nem comentei issue alguma, e nao enfileirei task.
