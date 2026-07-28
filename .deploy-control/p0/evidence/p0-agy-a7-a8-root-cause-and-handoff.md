# P0 AGY — handoff de criação para o Kiro + issue separada de duplicidade

- Autor: Codex56#A (`w7:p3`) · UTC 2026-07-27T17:56Z · READ-ONLY (nenhum INSERT, nenhum POST meu)
- Estado no momento da escrita: `SELECT count(*) FROM agent WHERE name ILIKE '%agy%'` → **0** (17:55:07Z)

## 1. Causa raiz registrada

`Agy-P0-A7` e `Agy-P0-A8` **nunca existiram como agentes do produto**. A busca por
`name ILIKE '%agy%' OR '%a7%' OR '%a8%' OR '%p0-a%'` retorna zero linhas, com 25 agentes no total e 15
ativos. Não é falha de eligibility, readiness, `archived_at`, `runtime_id`, catálogo de modelos,
Leader assignment nem admissão de fila: não há registro para selecionar.

O runtime que eles precisam já está saudável:

```text
405b751d-e831-4da3-8fd5-bb3744c49334  Antigravity (ORQ2 Credential Runtime)
  visibility=public  status=online  last_seen=2026-07-27 17:52  daemon=orq2-credential-runtime-v1
  workspace_id=20fce817-895d-447b-965a-49f5e279314a          <-- workspace correto

38994e92-f7dc-46ad-93b6-3c08c4c52304  Antigravity (ORQ2 Credential Runtime)
  visibility=private workspace_id=6733441a-13f6-4a27-b393-d3c66d0d3425  <-- NÃO USAR
```

Prova de que o caminho antigravity executa: `Gemini-3.6-Flash-A` tem 3 tasks, a última `completed` em
07-27 10:42; `Gemini-3.6-Flash-B` tem 1. Unit do daemon com
`MULTICA_ANTIGRAVITY_PATH=/home/ec2-user/.local/bin/agy`.

## 2. Payload exato para o Kiro (dois POST, um por agente, sem repetir)

`POST /api/agents?workspace_id=20fce817-895d-447b-965a-49f5e279314a`

```json
{
  "name": "Agy-P0-A7",
  "runtime_id": "405b751d-e831-4da3-8fd5-bb3744c49334",
  "model": "gemini-3.6-flash-high",
  "visibility": "workspace",
  "max_concurrent_tasks": 6,
  "description": "Agente AGY P0 A7 (antigravity/agy) no runtime público do ORQ2",
  "instructions": ""
}
```

Idêntico para `"name": "Agy-P0-A8"`.

Regras que o campo de criação impõe, medidas no código:

- **não enviar `thinking_level`** — `handler/agent.go:768` valida contra o enum do provider e
  `antigravity` não está em `providerThinkingEnums` (`pkg/agent/thinking.go:875-935`), então qualquer
  valor devolve 400. Reasoning em `agy` é model-embedded.
- `visibility: "workspace"` é o valor usado pelos agentes antigravity que funcionam hoje
  (Gemini-3.6-Flash-A/B, Opus-46#A/B), não `public` nem `private`.
- `max_concurrent_tasks: 6` espelha os mesmos agentes.
- um POST por agente; **não** repetir em caso de dúvida — verificar por GET antes de reenviar, porque
  nomes duplicados são exatamente o problema descrito na §4.

## 3. Verificação que eu executo depois (READ-ONLY)

1. `id`, `name`, `runtime_id`, `workspace_id`, `visibility`, `status`, `model`, `thinking_level` e
   `archived_at` das duas linhas, exigindo: runtime `405b751d…`, workspace `20fce817…`,
   `thinking_level` NULL, `archived_at` nulo, status inicial esperado.
2. unicidade: `count(*) = 1` por nome entre agentes ativos, para não repetir o padrão Codex-A/B.
3. uma task real admitida por agente, acompanhando a transição de status na fila e conferindo que as
   duas não colidem (agente distinto por task, sem disputa de slot).

## 5. RCA atualizada — execução: allowlist do daemon aponta para um slot sem credencial

A criação resolveu visibilidade e atribuição, mas as duas tasks foram despachadas e falharam em
milissegundos. Causa exata, do log do daemon:

```text
17:56:48 picked task task=234e51bc agent=Agy-P0-A7 provider=antigravity
17:56:48 ERR task failed error="credential isolation: inspect
         /home/****/.agent-cred-homes/slots/slot-150: lstat ...: no such file or directory"
(idem task=1ee00841 agent=Agy-P0-A8)
```

**Quem escolhe o slot:** não é o registry do herdr — é uma variável de ambiente da unit do daemon,
lida em `internal/daemon/credential_home.go:57-75`:

```text
Environment=MULTICA_CREDENTIAL_SLOT_ALLOWLIST_ANTIGRAVITY=141,145,146,150
Environment=MULTICA_CREDENTIAL_SLOT_ALLOWLIST_KIRO=139,140,143,149
Environment=MULTICA_CREDENTIAL_SLOT_ALLOWLIST_CODEX=152
```

Estado de cada slot da allowlist antigravity, medido por existência e tamanho, **sem ler conteúdo**:

| slot | diretório | token `home/.gemini/antigravity-cli/antigravity-oauth-token` |
|---|---|---|
| 141 | sim | **presente** |
| 145 | sim | **presente** |
| 146 | sim | **presente** |
| 150 | sim (recriado por mim) | **ausente** |

O caminho e o nome do arquivo vêm de `credential_home.go:48-49`
(`<slot>/home` + `.gemini/antigravity-cli/antigravity-oauth-token`) e de
`execenv/antigravity_home.go:11-19`, que documenta que o isolamento do `agy` é por **HOME**.

## 6. Auditoria: qual rotina removeu slot-150/151

Verificado read-only, e a conclusão honesta é que **nenhuma rotina auditada apaga diretórios de slot**:

- `scripts/ops/agent-cred-isolation.sh` cria o slot completo em `:201-209`
  (`codex`, `cline`, `cline-sandbox`, `home/.gemini/antigravity-cli`, `xdg-data/kiro-cli`,
  `xdg-data/opencode`, `xdg-config/opencode`, todos 0700). Seus `rm -rf` (`:325-350`, `:369`) atuam
  **somente** em `codex-logins/login-*`, nunca em `slots/*`.
- `orq2-agent-cache-lifecycle.service` roda com `InaccessiblePaths` cobrindo os homes de credencial; a
  própria evidência dele registra que "credential-home metadata changed concurrently, but the service
  could not access those paths". Fica excluído.
- Não há `auditd` nem entrada de journal na janela 17:05–17:12 mencionando slot, cred-home ou remoção.

O padrão real é mais antigo e mais amplo: o registry tem **53 entradas de slot** e o disco tinha **26
diretórios**. Faltam 26 números, não dois: `1,2,3,4,5,13,121,124..138,144,147,148,151` (mais o 150,
que eu recriei). Entre os terminais registrados uma única vez, apenas 27 de 51 têm diretório. Isso é
consistente com **alocação de número sem materialização de diretório**, e não com deleção em massa.

Conclusão de atribuição: **NÃO PROVADO** que algo removeu slot-150/151. A hipótese que os dados
sustentam é que o slot 150 foi acrescentado à allowlist do daemon para uma quarta conta `agy` cujo
login nunca aconteceu, então o diretório nunca existiu. A entrada 150 no registry é de 2026-07-25 com
o terminal visto uma única vez (`first_seen == last_seen`).

**Mapeamento pedido**, do `registry.json` (apenas identificadores):

```text
slot 150 -> terminal herdr:term_6576f4e596efd7   first_seen = last_seen = 2026-07-25T13:17:09Z
slot 151 -> terminal herdr:term_65771075a3d4ad   first_seen = last_seen = 2026-07-25T15:20:26Z
```

Nenhum agente do produto está mapeado a esses slots: o vínculo agente→slot não existe no registry, e o
daemon resolve por allowlist de provider. Portanto `slot-151` **não** afeta AGY: ele não está em
nenhuma allowlist.

## 7. Ação executada por mim (mínima, reversível, sem segredo)

Recriei o esqueleto de isolamento do `slot-150` no layout de referência do próprio script, tudo 0700 e
**vazio**: `codex`, `cline`, `cline-sandbox`, `home`, `home/.gemini`,
`home/.gemini/antigravity-cli`, `xdg-config`, `xdg-config/opencode`, `xdg-data`, `xdg-data/kiro-cli`,
`xdg-data/opencode`.

```text
snapshot 0600  ~/.private-tmp/agy-slot150-fix-20260727T175847Z/{slots-before.txt,registry-before.json,rollback.txt}
rollback       rm -rf /home/ec2-user/.agent-cred-homes/slots/slot-150   (diretório vazio, zero arquivos)
health daemon  200 antes / 200 depois   unit active antes / depois
registry.json  comparado por cmp: IDÊNTICO (não toquei)
```

Isso remove o erro de isolamento. A próxima task AGY passa a falhar por **autenticação de provider**,
porque o token do slot não existe.

## 8. Comando exato de login para o Kiro (não executo — é operação de credencial)

`agy` **não tem subcomando `login`** (`agy --help` em 1.1.7 lista apenas
`agent, agents, changelog, help, install, models, plugin, plugins, update`; `agy help login` responde
`unknown subcommand: login`). O OAuth acontece na primeira execução interativa, e o token é gravado em
`HOME/.gemini/antigravity-cli/antigravity-oauth-token`. Com HOME e XDG isolados no slot-150:

```bash
HOME=/home/ec2-user/.agent-cred-homes/slots/slot-150/home \
XDG_CONFIG_HOME=/home/ec2-user/.agent-cred-homes/slots/slot-150/xdg-config \
XDG_DATA_HOME=/home/ec2-user/.agent-cred-homes/slots/slot-150/xdg-data \
  agy
```

Verificação de sucesso **sem ler o token**:

```bash
test -s /home/ec2-user/.agent-cred-homes/slots/slot-150/home/.gemini/antigravity-cli/antigravity-oauth-token \
  && stat -c '%a %s' /home/ec2-user/.agent-cred-homes/slots/slot-150/home/.gemini/antigravity-cli/antigravity-oauth-token
```

Regras: usar uma conta `agy` **distinta** das que já ocupam 141, 145 e 146; nunca copiar, mover ou
linkar token de outro slot; manter modo 0600 no arquivo e 0700 nos diretórios; e não reutilizar slot
alternativo para AGY, porque isso quebra o isolamento por conta.

**Alternativa que o owner pode preferir, e que eu não executo:** remover `150` da
`MULTICA_CREDENTIAL_SLOT_ALLOWLIST_ANTIGRAVITY`, deixando `141,145,146`. AGY passaria a executar
imediatamente com três contas válidas, sem nenhuma operação de credencial. O custo é que a unit do
daemon serve **também** codex e kiro, então o restart necessário não é isolado ao runtime AGY — está
fora do que me foi autorizado e precisa de decisão explícita.

## 9. Correção de um fato que eu próprio afirmei antes

Na auditoria P1 eu escrevi que `agy` "não expõe flag de effort". **Está errado no nível de CLI**:
`agy --help` em 1.1.7 mostra `--effort  Reasoning effort for the current CLI session (low|medium|high)`.
O que é verdade é que o **produto não liga** esse flag: `pkg/agent/antigravity.go` não referencia
`ThinkingLevel` e `antigravity` não está em `providerThinkingEnums`. Portanto a decisão de manter
Antigravity fora da allowlist do gateway continua correta como descrição do estado atual, mas a
justificativa deve ser "não implementado no produto", não "o CLI não suporta". O commit `d9d569d` está
congelado; registro isto como correção pendente de comentário/teste para quando P1 for retomado.


## 10. Issue separada, pronta para o Leader — **não corrijo agora**

**Título**: Agentes ativos com nome duplicado tornam a atribuição ambígua (Codex-A ×2, Codex-B ×2)

**Fato medido**: entre os 15 agentes ativos existem quatro linhas com nome repetido:

```text
Codex-A x2
Codex-B x2
```

e o par não é simétrico: uma das linhas `Codex-A` e uma das `Codex-B` estão ligadas ao runtime
**Kiro (ORQ2 Credential Runtime)**, provider `kiro`, enquanto as outras estão no runtime **Codex**,
provider `codex`. Ou seja, dois agentes com o mesmo nome executam em CLIs diferentes.

**Impacto**: qualquer atribuição feita por nome (humano na UI, mensagem de coordenação, automação)
pode acertar o agente errado, e o CLI que executa muda com ele. Também polui a leitura de fila e de
histórico por agente.

**Escopo sugerido**: decidir qual linha é canônica por nome, renomear ou arquivar a outra, e avaliar
uma restrição de unicidade de nome por workspace entre agentes não arquivados. Isso mexe em dados de
produto e possivelmente em schema, então precisa de dono próprio — provavelmente a fila do LANE-DB se
a restrição entrar.

**Evidência**: esta seção e a consulta
`SELECT name, count(*) FROM agent WHERE archived_at IS NULL GROUP BY name HAVING count(*) > 1`.

## 11. Não-alegações

- Não executei `INSERT`, `UPDATE`, `DELETE` nem `POST`; nada foi criado por mim.
- Não publiquei card no Kanban: a API responde 401 para mim desde 15:56Z e o Kiro é quem publica.
- Não reiniciei runtime, daemon ou container; não foi necessário, o runtime AGY já está online.
- Não corrigi a duplicidade Codex-A/B, por ordem explícita.
- Nenhum segredo lido; só colunas nomeadas e não secretas.
