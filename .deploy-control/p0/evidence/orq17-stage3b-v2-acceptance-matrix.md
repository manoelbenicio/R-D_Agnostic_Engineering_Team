# ORQ-17 Stage3B — matriz de aceite para a V2 corrigida (READ-ONLY, pré-chegada)

- Card: **ORQ-17** · revisor independente: Codex56#A (`w7:pC` dispatch, pane `w7:p3`)
- Base: `.deploy-control/p0/evidence/orq17-stage3b-independent-adversarial-review.md` (veredito
  **BLOCK**, defeitos C1–C8), sobre o pacote
  `orq17-stage3b-owner-credential-provisioning-runbook.md` + `…-bootstrap-helper.go.txt` +
  `…-helper-go.mod.txt` + `CHECKOUT__Codex__ORQ-17-STAGE-3B-…__20260727T161040Z.json`
- UTC: 2026-07-27T16:45Z
- Modo: **READ-ONLY**. Não editei artefato de autor, não resolvi referência de segredo, não chamei
  `GetSecretValue`/`BatchGetSecretValue`, não toquei banco, Docker/Compose, Serve/Funnel, fila ou board.
- Uso: cada linha é **binária** e verificável por leitura ou comando read-only. Qualquer `FAIL` = BLOCK
  da V2 inteira; não há aceite parcial, porque C1–C8 se compõem (build não pinado invalida o gate de
  saída fixa, fila não congelada invalida a prova de `member=0`, etc.).

## 0. Fatos que eu já medi e que a V2 tem de casar exatamente

Estes valores foram verificados agora, na fonte e por consulta read-only. Se a V2 divergir, é FAIL.

| fato | valor medido | fonte |
|---|---|---|
| hostname ORQ1 | `ip-172-31-18-217.sa-east-1.compute.internal` | `ssh 100.118.244.61 hostname -f` |
| IPv4 Tailscale | `100.118.244.61` | `tailscale ip -4` |
| Go no host | `go1.26.1 linux/amd64` | `go version` |
| Docker Compose | `5.3.1` | `docker compose version --short` |
| tabela da fila | **`agent_task_queue`** | `migrations/109_agent_task_waiting_local_directory.up.sql:13-15` |
| estados possíveis | `queued, dispatched, running, waiting_local_directory, completed, failed, cancelled` | mesma migration, `CHECK` |
| estados **ativos** | os 4 primeiros | idem |
| `asm-exec` resolve argv | `args = [resolve_string(a) … for a in cmd_args]` | `references/asm-exec:373` |
| `asm-exec` resolve o próprio env para o filho | `child_env = {k: resolve_string(v) …}` sobre `os.environ` | `references/asm-exec:378-381`, comentário “documented behavior” em `:375-377` |
| execução do filho | `subprocess.run(args, env=child_env)` | `:383` |
| build atual do runbook | `GOWORK=off CGO_ENABLED=0 go build -trimpath -o orq17-stage3b-helper .` | runbook `:106` |
| host aceito hoje pelo runbook | só `100.118.244.61` | runbook `:82` |

**Catálogo de FKs para `user(id)`** — extraído das migrations, **15 tabelas** (o C6 do review nomeia
menos; duas ausentes na lista dele estão marcadas ▲):

| tabela | ON DELETE | efeito no rollback |
|---|---|---|
| `user_password_credential` | CASCADE | é o único alvo legítimo de deleção |
| `member` | CASCADE | apagaria membership em silêncio |
| `chat_session` | CASCADE | idem |
| `feedback` | CASCADE | idem |
| `notification_preference` | CASCADE | idem |
| `personal_access_token` | CASCADE | idem |
| `pinned_item` | CASCADE | idem |
| `task_token` | CASCADE | idem |
| `daemon_pairing_session` ▲ | SET NULL | perde autoria silenciosamente |
| `github_installation` | SET NULL | idem |
| `lark_installation` | RESTRICT | a deleção falha (protetor) |
| `agent` | NO ACTION | falha se houver linha (protetor) |
| `agent_runtime` | NO ACTION | idem |
| `skill` ▲ | NO ACTION | idem |
| `workspace_invitation` | NO ACTION | idem |

## 1. C1 — transporte seguro do segredo (sem plaintext em argv)

| # | critério de aceite | como verifico (read-only) | FAIL se |
|---|---|---|---|
| A1.1 | nenhuma referência `{{resolve:…}}` aparece como **argumento** de comando; só em variáveis de ambiente do próprio `asm-exec` | grep no runbook por `{{resolve` e checar que cada ocorrência está à esquerda de `asm-exec`, na forma `NAME='{{…}}' asm-exec -- …` | qualquer `env NAME={{…}}` ou `--flag={{…}}` |
| A1.2 | o filho que recebe o valor resolvido **não** é o Docker/Compose: o valor vai por **stdin** ao container | ler o pipeline: `printf … \| env -u OWNER_EMAIL -u OWNER_PASSWORD docker compose run …` | `docker compose` herdando as variáveis resolvidas |
| A1.3 | separadores NUL e leitura exata de 2 campos no helper; sem `echo`, sem heredoc com valor | grep `printf "%s\0%s\0"` no runbook e leitura de `os.Stdin` no helper | uso de `echo`, argv ou arquivo temporário |
| A1.4 | `set +x` antes de qualquer expansão e nenhum `set -x`/`PS4` ativo | grep no bloco `sh -eu -c` | ausência de `set +x` |
| A1.5 | risco residual **declarado**: `/proc/<pid>/environ` do shell filho é legível por mesmo-UID/root; exige janela selada e nenhum outro processo do UID do operador | procurar o parágrafo de risco residual | risco omitido ou minimizado |
| A1.6 | **uma única** resolução de `AWSCURRENT` para todo o ciclo (sem dry-run separado que resolve de novo) | contar invocações de `asm-exec` que carregam referência | 2 ou mais resoluções |
| A1.7 | zero `GetSecretValue`/`BatchGetSecretValue`/acesso ao SMA em qualquer passo | grep por `get-secret-value`, `BatchGetSecretValue`, `:2773` | qualquer ocorrência |

## 2. C2 — fechamento de fonte e pin de build

| # | critério | verificação | FAIL se |
|---|---|---|---|
| A2.1 | pins SHA-256 **esperados** para helper, `go.mod` do helper, `go.sum` do helper e `asm-exec`, escritos como valor esperado, não “observado” | comparar com os hashes que já registrei: helper `24777e12…`, go.mod `b3411a64…`, asm-exec `d55eb38a…` | hash ausente, ou apresentado só como leitura pós-fato |
| A2.2 | o módulo do helper **não** aponta para checkout mutável absoluto; usa snapshot read-only verificado por hash | procurar `replace … => /home/...` no template | `replace` para caminho vivo |
| A2.3 | manifesto de **fechamento de compilação**: toda a árvore que entra no binário, não só `auth_provider.go` (importar `internal/handler` arrasta muito mais) | conferir se há lista de pacotes/arquivos com hash agregado | manifesto ausente ou só os 6 arquivos de aplicação |
| A2.4 | flags exatas `GOWORK=off GOFLAGS=-mod=readonly CGO_ENABLED=0 go build -trimpath` | comparar com runbook `:106`, que hoje **falta** `-mod=readonly` | qualquer flag faltando |
| A2.5 | Go pinado por versão **e** hash do executável, com `GOOS`/`GOARCH` explícitos; deve casar `go1.26.1 linux/amd64` que medi no host | ler o pin e comparar | versão diferente da do host, ou sem hash do binário Go |
| A2.6 | hash **esperado** do binário compilado, estabelecido por build independente **antes** da janela, e igualdade exigida na execução | ler o valor esperado e o passo de comparação | “calcular o hash e registrar” sem valor esperado prévio |
| A2.7 | prova de que as entradas não mudaram **durante** o build (hash antes e depois, ou snapshot imutável) | ler o passo | ausente |
| A2.8 | `gofmt`/`go vet` executados e registrados (o check-out anterior declarou `gofmt` indisponível) | ler evidência de execução | “não executado” |

## 3. C3 — identidade exata de host/stack/banco e congelamento de fila

| # | critério | verificação | FAIL se |
|---|---|---|---|
| A3.1 | igualdade exigida em **hostname** `ip-172-31-18-217.sa-east-1.compute.internal` **e** IPv4 `100.118.244.61` | grep; hoje o runbook `:82` só pina o IP | hostname ausente |
| A3.2 | labels Compose `project=multica-dev-transition` e `service=backend`, mais IDs de imagem/container comparados com a evidência do Stage2 | ler o passo de derivação por label | comparação por nome de arquivo em vez de label |
| A3.3 | `Serve={}`, `Funnel={}`, health `200`, readiness `200`, todos como pré-condição de bloqueio | grep | qualquer um só como “observação” |
| A3.4 | o **helper** parseia o DSN herdado e falha antes de `BeginTx` se host/database/usuário ≠ tupla aprovada, **sem imprimir o DSN** | ler o helper V2 | aceitar qualquer `DATABASE_URL` não vazio |
| A3.5 | gate de fila com o nome de tabela **correto** `agent_task_queue` e os 4 estados ativos exatos | comparar com a `CHECK` da migration 109 | tabela/estado errado, ou lista de estados incompleta |
| A3.6 | **duas leituras** com ≥3 s de intervalo, ambas `0`, e **congelamento de admissão** mantido até fim da verificação | ler o mecanismo de freeze (não prosa) | só uma leitura, ou freeze descrito como intenção |
| A3.7 | o freeze é reversível e tem passo explícito de descongelamento com verificação | ler | sem passo de reversão |

## 4. C4 — rollback transacional explícito, sem `os.Exit` no meio

| # | critério | verificação | FAIL se |
|---|---|---|---|
| A4.1 | nenhuma função abaixo de `main` chama `os.Exit`; `stop()` deixa de existir ou vira erro tipado | grep `os.Exit` no helper V2 → só em `main`, após cleanup | qualquer `os.Exit` fora de `main` |
| A4.2 | erros tipados com código fixo (`E_*`) retornados, não impressos no ponto de falha | ler assinaturas | strings soltas |
| A4.3 | `defer tx.Rollback`, `defer pool.Close` e limpeza de bytes **executam** em todo caminho de erro | seguir o fluxo | caminho que pula defer |
| A4.4 | `Rollback` é chamado **e verificado** explicitamente antes de retornar erro (não só via defer) | ler | apenas defer |
| A4.5 | `memberRefs` retorna `(int64, error)` e nunca termina o processo | ler assinatura | mantém contrato booleano/terminação |
| A4.6 | limpeza de bytes das credenciais em memória em todos os caminhos, inclusive sucesso | ler os defers | limpeza só no caminho felizes |

## 5. C5 — FK/ausência de membership e sessão descrita com honestidade

| # | critério | verificação | FAIL se |
|---|---|---|---|
| A5.1 | `member` é travado junto com `user` e `user_password_credential` (mesmo `SHARE ROW EXCLUSIVE`) | ler os `LOCK TABLE` | `member` fora do lock |
| A5.2 | `member=0` exigido **três vezes**: antes da criação, antes do commit e depois do commit | contar as verificações | menos de três |
| A5.3 | a linha de sucesso afirma **somente** o que foi medido | comparar campo por campo com as consultas feitas | qualquer campo não medido |
| A5.4 | `SESSION=NONE` substituído por `HTTP_LOGIN=NOT_CALLED`, ancorado no source pinado do handler/provider | grep | mantém `SESSION=NONE` |
| A5.5 | declaração explícita de que JWT é **stateless** e por isso não detectável no banco | ler | alega detecção |
| A5.6 | a janela pós-commit está coberta pelo freeze de A3.6 (senão `member=0` pós-commit é vazio) | cruzar C3×C5 | prova pós-commit sem freeze ativo |

## 6. C6 — rollback não-cascateante e simétrico

| # | critério | verificação | FAIL se |
|---|---|---|---|
| A6.1 | enumeração do **catálogo FK vivo** (`pg_constraint`) de referências a `user(id)`, comparada com allowlist revisada | ler o passo | allowlist estática sem consulta ao catálogo |
| A6.2 | a allowlist cobre as **15** tabelas que medi, incluindo `daemon_pairing_session` e `skill`, ausentes do C6 original | comparar item a item com a tabela da §0 | qualquer tabela faltando |
| A6.3 | FK nova/desconhecida ⇒ **BLOCK** automático | ler a regra | tratada como aviso |
| A6.4 | zero linhas para o usuário-alvo em **toda** tabela referenciante exceto `user_password_credential`, sob lock | ler as consultas | verificação só de `member` |
| A6.5 | deleção explícita de **1** linha de credencial e **1** de usuário, na mesma transação serializável, sem depender de `ON DELETE CASCADE` | ler o SQL e a checagem de `RowsAffected == 1` | confiar em cascade, ou sem checar contagem |
| A6.6 | correção da afirmação falsa do V1 de que workspace/sessão produz `E_ROLLBACK_REFERENCED` | grep pela frase antiga | mantida |
| A6.7 | rollback repete **todos** os gates: host, build, fila, segredo, saída | ler | gates só no provision |
| A6.8 | se qualquer login pode ter ocorrido, **retém** o usuário e escala; deleção só na janela selada pré-login | ler a regra | deleção incondicional |

## 7. C7 — saída fixa por allowlist mecânica

| # | critério | verificação | FAIL se |
|---|---|---|---|
| A7.1 | stdout/stderr capturados em arquivos `0600` privados da tarefa | ler o redirecionamento e o modo | sem captura, ou modo default |
| A7.2 | comparação **byte a byte** do stdout com a única linha aprovada | ler o comparador (`cmp`/`sha256`) | comparação por `grep`/substring |
| A7.3 | stderr vazio, ou allowlist separada e explícita | ler | stderr livre |
| A7.4 | nada de saída bruta de Compose/runtime chega ao contexto do agente ou do chat | ler o passo de supressão | forward direto |
| A7.5 | em falha, só código fixo; nenhum diagnóstico contendo argv/env/DSN | ler os caminhos de erro | mensagem com detalhe do ambiente |
| A7.6 | os arquivos capturados são apagados ou movidos para quarentena `0700` ao fim | ler | permanecem `644` em `/tmp` |

## 8. C8 — identidade exata do segredo e autorização do owner

| # | critério | verificação | FAIL se |
|---|---|---|---|
| A8.1 | **ARN completo** do segredo, com conta e região; nenhum ID nu | grep por `arn:aws:secretsmanager:` e ausência de ID curto | aceita ID e depende de `AWS_REGION` ambiente |
| A8.2 | região do ARN coerente com a região de workload `sa-east-1`, e divergência declarada se houver | ler | região implícita |
| A8.3 | versão estável: `AWSCURRENT` resolvido **uma vez** (A1.6) **e** congelamento de rotação da autorização até o fim da janela | ler a cláusula de no-rotation | rotação permitida no meio |
| A8.4 | exatamente **dois** inputs humanos, e nenhum deles é e-mail, senha, JSON, hash ou conteúdo de versão | ler a lista | terceiro input, ou input com valor sensível |
| A8.5 | frase de autorização única, literal, amarrada a host/stack/banco pinados e à janela selada | comparar com o texto exigido em C8 | autorização genérica de escopo |
| A8.6 | pré-requisitos fora do Stage3B (criação do segredo, rotação, permissão do resolvedor) declarados como já satisfeitos, com evidência apontada | ler | assumidos |

## 9. Regras de julgamento que vou aplicar na chegada da V2

1. **Composição, não média**: C1–C8 são conjuntivos. 7 PASS e 1 FAIL = BLOCK.
2. **Prosa não é gate**: “apenas a saída aceita”, “fila deve estar vazia”, “rollback automático” só
   contam se houver comando, comparação e código de erro.
3. **Hash esperado ≠ hash observado**: um valor calculado no momento da execução não é procedência.
4. **Nada de execução minha**: vou revisar por leitura + verificações read-only (hash de arquivo,
   catálogo de migrations, `hostname`, labels). Não resolvo segredo, não abro transação, não rodo
   Compose, não mexo em Serve/Funnel.
5. **Se a V2 exigir execução para ser avaliada**, isso já é FAIL de desenho: o pacote precisa ser
   auditável antes de qualquer criação de credencial.
6. Emissão de **PASS ou BLOCK** com evidência linha a linha, e no caso de PASS a lista explícita do que
   permanece não provado até a execução.

## 10. Não-alegações

- Não editei nenhum artefato do autor (runbook, helper, `go.mod`, check-out dele).
- Não chamei `GetSecretValue`/`BatchGetSecretValue`, não acessei o SMA, não executei `asm-exec`, não
  resolvi nenhuma referência dinâmica, não li e-mail, senha, hash ou DSN.
- Não consultei nem mutei banco algum: o catálogo de FKs vem das **migrations em disco**, não de
  `pg_constraint` — a V2 ainda precisa comparar com o catálogo vivo (A6.1), e é possível que o banco
  real tenha FK que a leitura de migrations não capture.
- Não rodei Docker/Compose, não alterei Serve/Funnel, não criei usuário, credencial, sessão ou membro,
  não mexi em fila nem em board.
- As únicas ações remotas foram leituras em ORQ1: `hostname -f`, `tailscale ip -4`, `go version`,
  `docker compose version --short`.
- Não vi a V2: esta matriz é anterior a ela e não presume que qualquer C tenha sido corrigido.
