# CORRECAO DE EVIDENCIA + PONTE registry(ORQ2) → Multica (PROPOSTA, NAO APLICADA)

Autor: Codex56#A (ORQ2 `ip-172-31-30-9`, pane `w7:p3`) · UTC 2026-07-26T23:20Z
Escopo: agy, codex, kiro. opencode fora. Nada editado, nada compilado, nada reiniciado.

## 1. CORRECAO: eu estava errado, e a causa raiz nao era 0700

Meu relatorio anterior ("`~/.agent-cred-homes/slots` nao existe em nenhum host") esta **RETRATADO**.
Causa exata, medida agora:

```text
$ whoami; hostname
ec2-user
ip-172-31-30-9.sa-east-1.compute.internal
$ echo $HOME
/home/ec2-user/.agent-cred-homes/slots/slot-139/home
$ ls -la ~/.agent-cred-homes
ls: cannot access '/home/ec2-user/.agent-cred-homes/slots/slot-139/home/.agent-cred-homes': No such file or directory
```

**O HOME do meu proprio pane JA E o slot-139.** Portanto `~` expandia para dentro do slot e o `~/...`
que eu usei nunca podia existir. Nao foi permissao 0700 nem sudo: foi til resolvido contra um HOME
isolado. A licao operacional para a frota: **nunca sondar o registry com `~`; usar sempre o caminho
absoluto `/home/ec2-user/.agent-cred-homes`.**

Confirmacao dos fatos do TL com caminho absoluto:

```text
$ ls -ld /home/ec2-user/.agent-cred-homes
drwx------. 5 ec2-user ec2-user 164 Jul 26 13:38        (FATO 1: existe, 0700)
$ ls -d /home/ec2-user/.agent-cred-homes/slots/slot-* | wc -l
22                                                      (FATO 1: 22 slots)
$ ls -la /home/ec2-user/.agent-cred-homes
-rw------- 13183 registry.json                          (FATO 2: 13183 bytes)
-rw------- 33009 registry.json.pre-orphan-cleanup.20260722T023208Z
-rw-rw-r--     0 registry.lock
drwx------      codex-logins/  fallback-terminals/  slots/
$ ls -la /home/ec2-user/.agent-cred-homes/slots/slot-140
cline  cline-sandbox  codex  home  xdg-config  xdg-data     (FATO 3: raizes por provider)
```

FATO 1, 2 e 3: **confirmados por mim**. Nao contesto nenhum.

Nota: no ORQ1 (`100.118.244.61`, onde roda o daemon PID 3244391) o caminho **absoluto**
`/home/ec2-user/.agent-cred-homes` realmente nao existe — a mensagem de erro que recebi la era com
caminho absoluto, nao com til. Isso e coerente com FATO 1 (o isolamento e do ORQ2) e com FATO 8 (no
ORQ1 funciona hoje porque o TL copiou tokens para o HOME do processo). Registro como dado de
localidade para a ponte (secao 4, questao Q1), nao como contestacao.

## 2. FATO 4 corroborado com evidencia de arquivo (a minha primeira medicao era inferior)

Contar diretorio e insuficiente: `xdg-data/kiro-cli` EXISTE em 10 slots, mas 5 estao vazios.
Contando arquivos (nomes e contagem apenas, sem ler conteudo):

```text
slot-139 files=7  [.refresh.lock,bun,bun.sha256,data.sqlite3,feed.json,knowledge_bases,tui.js,...]
slot-140 files=7  [idem]
slot-141 files=0  []
slot-142 files=1  [data.sqlite3]
slot-143 files=7  [idem]
slot-145 files=0  []      <= confirma: slot-145 NAO tem kiro
slot-146 files=0  []
slot-149 files=7  [idem]
slot-150 files=0  []
slot-152 files=0  []
```

Kiro com estado real: **139, 140, 142, 143, 149 — exatamente os 5 do FATO 4**. agy presente nos 22.
Consequencia direta para a ponte: **presenca de diretorio nao pode ser o criterio de elegibilidade**,
senao a rotacao escolhe um slot sem credencial e o kiro roda em silencio sem credencial por
`kiro_home.go:73-75 (if srcMissing return nil)`. O criterio tem de ser estado de credencial.

## 3. Forma real do registry (medida, chaves apenas — nenhum valor sensivel impresso)

```json
{
  "version": 1,
  "next_slot": 153,
  "slots":     { "<numero>": { "created_at": ..., "terminal_id": ... } },   // 48 registros
  "terminals": { "herdr:term_<id>": { "first_seen":..., "last_seen":..., "slot":... } }  // 48
}
```

48 registros de slot no registry x 22 diretorios em disco: o excedente e historico/orfao (coerente com
o backup `pre-orphan-cleanup`). FATO 5 confirmado: **a chave do registry e `terminal_id`
(`herdr:term_*`), efemera por terminal Herdr; a chave do Multica e conta de agente.** Sao dois modelos
de identidade distintos e e exatamente isso que a ponte precisa reconciliar.

## 4. PONTE PROPOSTA — registry(ORQ2) → accounts/approved_accounts/assignments

Principio: `slot-NNN` e a **unica chave estavel**. `terminal_id` NAO entra no Multica em nenhuma
hipotese (e efemero e nao e identidade de conta); ele continua servindo somente para deteccao de orfao
e GC dentro do registry.

### 4.1 Consequencia de schema que precisa de decisao

`accounts` tem UM `home_dir` e UM `config_dir` por linha (migration 123), mas o FATO 3 diz que a raiz e
**por provider**:

| vendor | AccountHome correto | fonte |
|---|---|---|
| antigravity (agy) | `/home/ec2-user/.agent-cred-homes/slots/slot-NNN/home` | FATO 3 |
| kiro | `.../slots/slot-NNN/xdg-data` | FATO 3 |
| codex | `.../slots/slot-NNN/codex` | FATO 3 |
| cline (fora do escopo obrigatorio) | `.../slots/slot-NNN/cline` | FATO 3 |

Logo a granularidade correta e **1 linha de `accounts` por par (vendor, slot)**, nao por slot. Com 22
slots: 22 linhas agy + 5 linhas kiro (so as com estado) + N linhas codex (a medir em `codex-logins/` e
`slots/*/codex`). `config_dir` recebe `slots/slot-NNN/xdg-config` quando o provider usa XDG.

### 4.2 Elegibilidade (status) derivada de estado, nao de diretorio

- `status='available'` somente quando o provider tem estado de credencial no slot (kiro: arquivos em
  `xdg-data/kiro-cli`; agy: `home/.gemini` populado; codex: idem em `slots/*/codex`).
- `status='degraded'` quando o diretorio existe vazio. Isso impede o modo silencioso do
  `kiro_home.go:73-75`.
- `tenant_id` = workspace/tenant do owner; `priority` = ordem de rotacao desejada pelo owner.

### 4.3 Aprovacao e vinculo

- `approved_accounts(tenant_id, account_id, allowed=true, worktype_scope)` — o owner aprova quais das 4
  contas agy entram, e em qual classe de trabalho (`GENERAL|HEAVY|CHEAP|REVIEW`).
- `assignments(agent_id → account_id)` — vinculo 1 agente : 1 conta, ja com PK em `agent_id`, que e
  exatamente "isolamento por conta".
- `credentials(account_id, vendor, secret_ref, format)` — apenas **referencia** (o valor nunca entra no
  banco nem no daemon). Ha unico parcial `uq_credentials_active_account` em `expires_at IS NULL`.
- `rotation_events` ja tem o CHECK de razao (`quota_exhausted_reactive|quota_forecast_proactive|
  login_failed|manual`) para a auditoria de rotacao e atribuicao de custo.

### 4.4 Ferramenta da ponte (nova, read-only sobre o registry)

`scripts/bridge/agent-cred-bridge.py` (proposta, ~120 linhas, sem dependencia externa):

1. le `/home/ec2-user/.agent-cred-homes/registry.json` sob `registry.lock` (compartilhado, so leitura);
2. inventaria `slots/slot-*/{home,xdg-data,codex,xdg-config}` e classifica cada par (vendor, slot) em
   `available|degraded` pelo criterio 4.2 — **contando arquivos, nunca lendo conteudo**;
3. emite SQL idempotente (`INSERT ... ON CONFLICT DO UPDATE`) para `accounts` e `approved_accounts`,
   com `home_dir`/`config_dir` por provider, e um relatorio texto do diff;
4. NAO executa o SQL. A aplicacao e ato do owner (seed de dados).
5. Reexecucao e segura: chave natural `(vendor, home_dir)` para o upsert, o que torna a ponte
   reconciliavel a cada novo slot criado pelo `next_slot`.

O consumo pelo daemon e o item que ja propus no TEMA 3 (`daemon.go:3448` volta a resolver o valor;
FATO 6 mostra que isso e restauracao do `aa62401`, nao feature nova).

## 5. Questoes abertas que travam a ponte (para o TL/owner, nao decido)

- **Q1 localidade**: o registry esta no ORQ2 e o daemon no ORQ1. `home_dir` absoluto so e valido no
  host que tem o slot. A ponte precisa escopo de host (`accounts` nao tem coluna de host/daemon) ou o
  daemon passa a rodar no ORQ2. Qual dos dois o owner quer?
- **Q2 tenant_id**: qual UUID usar (`workspace` do owner?) — `accounts.tenant_id` e NOT NULL.
- **Q3 mapa conta→slot**: quais slots correspondem as 4 contas agy (`a2a7860c`, `965277db`, `aebdffce`,
  `199009bc`)? O registry indexa por terminal, nao por conta, entao esse mapa **nao existe em nenhum
  arquivo** — e o dado que a ponte precisa receber do TL/owner (ou derivar de `codex-logins/`).
- **Q4 codex**: confirmar a raiz real por slot (`slots/*/codex`) versus `codex-logins/` (20 entradas no
  topo do registry), que parece um pool separado.

## 6. Nao-alegacoes

- Nao editei codigo, nao compilei, nao troquei binario, nao reiniciei daemon/container, nao commitei.
- Nao li conteudo de nenhum arquivo de credencial: apenas `ls`, `find -type f | wc -l` e chaves de
  primeiro nivel do `registry.json`. Nenhum token, `secret_ref` ou valor foi impresso.
- Nao contesto FATO 1-8. As unicas correcoes sao **as minhas**: retratacao da ausencia dos slots e
  substituicao da minha medicao de kiro por contagem de arquivos.
- Nao mapeei conta→slot (Q3) porque esse dado nao existe no registry.
