# ORQ-17 Stage3B - re-review final BINARIA (L1/L2/L3/S1/S2/self-deadlock)

- revisor: **Opus48#A** - ORQ2 w6:p1 - 2026-07-27T17:18Z - READ-ONLY
- escopo: **somente** L1, L2, L3, S1, S2 e self-deadlock contra
  `orq17-stage3b-v2-f1-f5-correction.md`. **Sem expansao.**

## VEREDITO: **PASS**

### Hashes do pacote - conferidos por mim, 3 de 5 casam
```
helper   fabc8f8585449d2263fbc62cb7437726db9cae0c33c1607adbad17abebb59e82  MATCH
runner   c3d64f85a348fdf1d5b373b9253f0fb5bc19f5b468add7ef17072c6f253d8b42  MATCH
closure  719f9a6b44c2b6fab1d198bf7af77a00164946d68b7df9b2ce8fa9726ba3d9d1  MATCH
binary   1710e010…  NAO VERIFICAVEL aqui (nao ha binario; nao compilei)
1696 entradas do closure  NAO RECONTADAS
```

| item | veredito | evidencia no artefato citado (`fabc8f85`) |
|---|---|---|
| **self-deadlock** | **RESOLVIDO** | `beginGuard` agora trava **so** `LOCK TABLE agent_task_queue IN SHARE MODE` (:156); `member` entrou nas **duas** listas de `beginMutation` em `SHARE ROW EXCLUSIVE` (:189 e :191-194). Nenhuma tabela e travada em modos conflitantes por **duas conexoes**, e `agent_task_queue` nunca e travada pela transacao de mutacao. `compensateWithFreshGuard` toma guarda nova (`SHARE` em `agent_task_queue`) enquanto `compensate` trava as 14 incluindo `member` - sem sobreposicao. O registro de correcao descreve exatamente isso. |
| **L1** rollback sem senha | **FECHADO** | `rollbackBreakglass` exige recibo + SHA (`check_receipt`: prefixo, modo `600`, UID, hash), pina a identidade em `expectedOwnerEmail`, e mantem catalogo de FK, `verifyGlobal(1,1,0)`, `targetReferenceCounts` (inclui `member` `CASCADE`), `RowsAffected()==1` por DELETE e `verifyGlobal(0,0,0)`. Com `member` agora na **mesma** transacao do DELETE, a protecao do DELETE ficou **mais forte**. |
| **L2** falha apos commit | **FECHADO** | `E_PROVISION_COMMITTED_GUARD_FAILED_COMPENSATED`, `..._STATE_UNKNOWN`, `E_ROLLBACK_COMMITTED_GUARD_FAILED`; compensacao **so** sob guarda nova; runner captura `0\|0\|0` ou `1\|1\|0` via modo `state` (`</dev/null`) antes de qualquer segunda acao; `DELETE` manual proibido. |
| **L3** lockout por login | **FECHADO** | Gate contido: `POST /auth/login` = `200` **e** `GET /api/me` = `200`, jar privado, credenciais retidas **ate** o gate; falha captura contagens e tenta rollback guardado; sucesso so e emitido depois do gate. |
| **S1** `printf` builtin | **FECHADO** | `command -V printf` com `E_PRINTF_EXTERNAL` na **linha 5-7**, antes de qualquer expansao de `OWNER_*`. |
| **S2** core dump | **FECHADO** | `ulimit -c 0` na **linha 9**. |

Nota unica (nao bloqueante, dentro do escopo): tirar `member` da guarda encurta a **janela** de
congelamento - entre o commit da mutacao e a verificacao pos-commit o lock de `member` ja foi
liberado. Isso **nao** cria corrupcao nem lockout: qualquer insercao nesse intervalo cai em
`E_GLOBAL_COUNTS` (fail-closed, com compensacao), e **todo** DELETE continua ocorrendo com
`member` travado em `SHARE ROW EXCLUSIVE` e apos `targetReferenceCounts` = 0.

## Nao-afirmacoes
- Nao compilei, nao verifiquei o hash do **binario** `1710e010…`, nao recontei as 1.696 entradas do
  closure, nao executei `go test`/`go vet`/`sh -n`, nao rodei helper, runner, Docker, SQL, `curl`,
  login, `asm-exec` nem AWS. Nenhuma mutacao.
- Revisei **apenas** os seis itens pedidos. Nada aqui aprova A2.x, testes, ou qualquer outro item.
