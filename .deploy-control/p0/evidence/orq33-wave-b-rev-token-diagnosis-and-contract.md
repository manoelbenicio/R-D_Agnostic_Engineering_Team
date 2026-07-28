# ORQ-33 — diagnóstico factual, correção de escopo e contrato executável (READ-ONLY)

- Card: **ORQ-33** — "Security Wave B: Rev Token Rotation & Validation"
- Executor: Codex56#A (`w7:p3`) · UTC 2026-07-28T16:18Z
- Skill obrigatória lida **integralmente** antes de qualquer trabalho: `.agents/skills/aws-secrets-manager/SKILL.md`
  (178 linhas) — regras aplicadas: nunca `get-secret-value`/`batch-get-secret-value`, nunca acesso ao
  SMA em loopback, e uso exclusivo de `{{resolve:secretsmanager:...}}` via `asm-exec`.
- Fase: **zero rotação, zero AWS, zero segredo, zero runtime.** Nenhum valor foi lido; classificação só
  por nome de chave, caminho e metadado.

## 1. A correção de escopo pedida **já ocorreu ontem** — e eu fui o revisor que a forçou

O card nasceu como rotação de um "rev token". A trilha, em ordem:

| artefato | autor | veredito |
|---|---|---|
| `orq33-rev-token-rotation-runbook.md` (V1) | Agy-P0-A8 | auto-PASS |
| `gtl-orq33-rev-token-rotation-peer-review.md` (GTL-87) | GTL | **BLOCK** |
| `orq33-rev-token-independent-review.md` | Codex56#A (eu) | **BLOCK ampliado** |
| `orq33-jwt-secret-rotation-runbook-v2.md` | — | **substitui** a V1 e redefine o alvo real como `JWT_SECRET` |
| `orq33-jwt-rotation-v2-independent-peer-review.md` | — | **BLOCK** por defeito de segurança nos Passos 2+3 |
| `orq33-jwt-rotation-remediation-v3-design.md` | Antigravity | emenda com 9 ajustes |

Portanto **não há escopo fictício pendente de correção**: a V2 já declara, no cabeçalho, que substitui
a V1 de "rev token". O que faltava era registrar o fechamento factual, que é o que este documento faz.

## 2. Diagnóstico factual do "rev token": **não existe como credencial de produto**

Busca em todo o código (Go, TS/TSX, SQL, YAML, JSON), excluindo `node_modules`, `.deploy-control`,
`e2e` e testes:

```text
grep -rniE "rev[-_ ]?token|REV_TOKEN|revToken"   -> ZERO ocorrências em código de produto
grep -rniE "revoke_token|rev_token|handshake_token" (Go, sem testes) -> ZERO
```

O termo só aparece em artefatos de controle sob `.deploy-control` (check-outs e evidências), ou seja em
texto de coordenação, nunca em código. O artefato físico que originou o card,
`/tmp/rev-token.txt` no host LOCAL, foi medido por mim hoje na verificação do ORQ-31: **ausente**,
junto com os outros 15 alvos da Wave A.

Isto reproduz exatamente o padrão que eu já havia provado no ORQ-32: o "handshake token" era **falso
positivo**, string de teste em pane, com zero referência em código. Conclusão para o revisor: **não há
emissor, não há consumidor e não há validade a verificar** para um "rev token". Rotacionar algo que
nenhum componente emite ou aceita não é possível nem útil.

Ressalva importante, e é a razão de o card não ser simplesmente cancelado: **ausência de artefato não é
prova de invalidação de credencial**. Se o valor que estava naquele arquivo era um token real emitido
por algum sistema externo, ele continua válido até ser rotacionado na origem. O que se pode afirmar com
os dados é que **este produto** não o emite nem o consome.

## 3. Alvo real, remedido hoje: `JWT_SECRET`, chave única, cinco consumidores

Emissor e cache:

```text
internal/auth/jwt.go:27  jwtSecretOnce sync.Once
internal/auth/jwt.go:30  func JWTSecret() []byte      <- resolvido uma única vez por processo
internal/auth/jwt.go:46  func ValidateJWTConfiguration(appEnv, secret string) error
```

Consumidores de `JWTSecret()` em produção, contagem atual **cinco**, idêntica à da V2:

```text
internal/handler/auth.go:213        assina token (emissão)
internal/handler/auth.go:226        assina token (emissão)
internal/middleware/auth.go:295     valida token de usuário
internal/middleware/daemon_auth.go:234  valida token de daemon
internal/realtime/hub.go:691        valida token em WebSocket/realtime
```

Consequência estrutural, que define todo o contrato: **não há `kid`, não há chave secundária, não há
conjunto de chaves**, e o valor é memoizado por `sync.Once` no processo. Logo a troca é **disruptiva por
natureza** — todo token emitido com o segredo antigo passa a falhar, e o processo precisa ser recriado
para reler o valor. Não existe rotação sem invalidação.

## 4. Estado dos dois BLOCKs da V2, e o que a V3 endereça

- **BLOQUEADOR 1**: `asm-exec` **não** resolve `{{resolve:...}}` dentro de **arquivo**; ele resolve em
  argv e em variáveis de ambiente do próprio processo (confirmei no código do wrapper em revisão
  anterior: `args = [resolve_string(a) …]` e `child_env = {k: resolve_string(v) …}`). O Passo 2 da V2
  escrevia o placeholder num arquivo de configuração e o Passo 3 esperava resolução — o resultado seria
  **sucesso aparente com o placeholder literal persistido**, que é pior que falhar.
- **BLOQUEADOR 2**: os `-f` do Passo 3 eram relativos e apontariam para a árvore errada.

A emenda V3 declara nove ajustes cobrindo exatamente isso: wrapper de shell com `trap` para resolver via
ambiente (ajuste 1), trava que **rejeita** `{{resolve:` literal remanescente (ajuste 2), mapeamento dos
quatro caminhos absolutos de Compose com projeto explícito (ajuste 3), validação de estado autenticado
em `GET /api/me` (ajuste 4), isolamento `0700`/`0600` (ajuste 5), gate de tamanho e não-vazio do segredo
(ajuste 6), disparo de rollback com preservação de dados (ajuste 7), separação da autorização de
re-pair do daemon no ORQ2 (ajuste 8) e pedido de revisão independente (ajuste 9). **Não revisei a V3
nesta task** — se o GTL quiser, faço em seguida, e é o passo que fecha a cadeia.

## 5. Contrato executável (metadata-only), para execução em fase futura autorizada

Cada linha é verificável e nenhuma expõe valor.

**Pré-condições, todas obrigatórias e verificáveis antes de tocar qualquer coisa**

1. segredo existe no Secrets Manager e é referenciado por **ARN completo** com conta e região
   `sa-east-1`; nunca por nome nu, para não herdar `AWS_REGION` ambiente;
2. exatamente uma versão com `AWSCURRENT`, e **congelamento de rotação** declarado da autorização até o
   fim da janela;
3. `ValidateJWTConfiguration` satisfeita pelo candidato: `APP_ENV=production` exige ≥32 bytes e proíbe o
   valor default — o gate roda **dentro** do processo, nunca com o valor em argv;
4. fila de tarefas ativas em **zero** nos quatro estados `queued`, `dispatched`, `running`,
   `waiting_local_directory`, e admissão congelada até o fim;
5. host e stack confirmados por igualdade: `ip-172-31-18-217.sa-east-1.compute.internal`,
   `100.118.244.61`, projeto Compose `multica-dev-transition`, serviço `backend`;
6. autorização one-time do owner, com `AUTH_ID` e expiração, recibo `0600` consumido atomicamente.

**Execução**

7. o valor **nunca** entra em argv, arquivo, log ou histórico: só como variável de ambiente resolvida
   por `asm-exec` no processo filho, com `set +x` e `umask 077`;
8. trava obrigatória: após a substituição, falhar se qualquer arquivo de configuração contiver a string
   literal `{{resolve:` — é o remédio direto do Bloqueador 1;
9. recriação do container do backend com caminhos de Compose **absolutos** e projeto explícito;
10. gate de sucesso em duas metades, e ambas necessárias: token antigo passa a responder **401** e um
    token novo responde **200** em rota autenticada barata (`GET /api/me`).

**Rollback**

11. rollback é **igualmente disruptivo**: reverter o segredo invalida os tokens emitidos no intervalo;
    isso precisa estar escrito na autorização, não descoberto durante o incidente;
12. gatilho de rollback: qualquer pré-condição violada, `{{resolve:` literal detectado, ou o gate 10
    falhando em qualquer metade;
13. procedimento: restaurar a referência da versão anterior por ARN, recriar o container, repetir o
    gate 10 invertido, e **preservar dados** — nenhum volume é removido;
14. re-pair do daemon no ORQ2 é ação **separada**, com autorização própria, e não faz parte do rollback
    automático;
15. saída content-free em todos os passos: apenas códigos fixos `E_*`, com stdout comparado byte a byte
    contra a linha aprovada e stderr exigido vazio.

## 6. Recomendação de escopo do card

1. **corrigir o título/escopo do ORQ-33** para nomear `JWT_SECRET`, encerrando formalmente a
   nomenclatura "rev token", que não corresponde a nenhuma credencial do produto;
2. registrar como **fechado sem ação** o item de rotação do "rev token" (B5 do plano da Wave A), com a
   ressalva do §2 sobre validade na origem;
3. manter em aberto, sob este card ou sucessor, apenas a rotação de `JWT_SECRET` conforme V3 + contrato
   acima, dependente de revisão independente da V3 e de janela com fila zero.

## 7. Não-alegações

- Não rotacionei nada, não chamei AWS, não usei `aws-cli`, não invoquei `asm-exec`, não acessei o SMA e
  não li nenhum valor de segredo.
- Não verifiquei **existência** do segredo no Secrets Manager: exigiria chamada AWS, proibida nesta fase.
- Não revisei a emenda V3 nem re-executei os gates da V2; o §4 descreve o que cada documento declara e o
  que eu havia medido antes sobre o comportamento do `asm-exec`.
- Não afirmo que o token do arquivo ausente foi invalidado: apenas que este produto não o emite nem o
  consome.
- Não toquei board, runtime, container, daemon, fila ou qualquer host além de leitura.
