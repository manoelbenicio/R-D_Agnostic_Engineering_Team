# PROMPT NM1 — Nemotron 3 Ultra 550B A55B (Instância 1)
# Workstream: GO Core Migration Sweep + ESLint Fix
# Sprint 3 | CRIT-002 + CRIT-003 (completar)

## SEU PAPEL
Você é NM1, responsável por completar a migração CAO → GO Core no frontend e corrigir o ESLint.
Você pode escalar 2 sub-agentes: **NM1-A** e **NM1-B** que trabalham em paralelo.

## CONTEXTO DO QUE JÁ FOI FEITO (pelo orquestrador — nao refaça)
Os seguintes arquivos JÁ foram modificados:
- `src/api/go-core-base-url.ts` — NOVO (porta 8080)
- `src/api/go-core-client.ts` — NOVO (GoCoreClient + alias caoClient)
- `src/api/types.ts` — `provider: string` adicionado a CreateSessionInput
- `src/api/index.ts` — re-exports atualizados para GO Core
- `src/api/connect-terminal-socket.ts` — usa GO_CORE_BASE_URL
- `src/api/session-discovery.ts` — primeira linha trocada (import), mas usos de caoClient dentro do arquivo ainda precisam ser trocados para goCoreClient
- `src/settings/settings-store.ts` — goCoreBaseUrl + goCoreClient
- `src/vite-env.d.ts` — VITE_GO_CORE_BASE_URL adicionado
- `src/shared/storage/migrations.ts` — schema v4 com migração caoBaseUrl→goCoreBaseUrl
- `.env.local` e `.env.example` — porta 8080

## SUAS TAREFAS

### NM1-A: Completar sweep de migração (CRIT-003)

**OBRIGATORIO antes de qualquer edição:**
1. Leia `.planning/AGENT_LEDGER_S3.md`
2. Registre CHECK-IN com agentID=NM1-A para CADA arquivo antes de editar
3. Após finalizar, registre CHECK-OUT

**Arquivos a verificar e corrigir:**

#### `src/api/session-discovery.ts`
- O import já foi trocado para `goCoreClient`
- Verifique se dentro do arquivo existem chamadas a `caoClient.*` — substitua por `goCoreClient.*`
- Run `npx tsc --noEmit` após cada arquivo

#### `src/api/health-store.ts`
- Troque: `import { caoClient } from './cao-client'` → `import { goCoreClient } from './go-core-client'`
- Troque todas as chamadas `caoClient.` → `goCoreClient.`

#### `src/api/session-store.ts`
- Verifique se importa `caoClient` diretamente — se sim, troque para `goCoreClient`
- Se importa via `session-discovery`, nada a fazer

#### `src/shell/app-fetch.ts`
- Verifique se usa `base-url.ts` ou `CAO_BASE_URL` diretamente
- Se sim, troque para `go-core-base-url.ts` / `GO_CORE_BASE_URL`

#### Busca global de remanescentes:
```bash
grep -rn "from.*cao-client\|from.*base-url\|CAO_BASE_URL\|caoClient\b" \
  src/ --include="*.ts" --include="*.tsx" | \
  grep -v "go-core-client\|go-core-base-url\|deprecated\|@deprecated\|// " | \
  grep -v "__tests__"
```
Para cada resultado encontrado: corrigir e registrar no ledger.

**Verificação final NM1-A:**
```bash
npx tsc --noEmit
```
Deve retornar 0 errors. Documente o resultado no ledger.

---

### NM1-B: CRIT-002 — Corrigir ESLint Plugin (paralelo com NM1-A)

**OBRIGATORIO antes de qualquer edição:**
1. Leia `.planning/AGENT_LEDGER_S3.md`
2. Verifique que `package.json` e `.eslintrc.cjs` estao Available
3. Registre CHECK-IN

**Contexto do problema:**
- `package.json` tem: `"eslint-plugin-agentverse": "file:./eslint-rules"`
- Node.js em UNC path (`//21LAPGLMVPJ4/...`) resolve errado → ENOENT
- Fix: Option B — absolute path para o drive mapeado

**Tarefa:**

1. Verifique qual letra de drive o projeto está mapeado. O path Windows é `C:\VMs\Projetos\Automonous_Agentic` OU `M:\Automonous_Agentic`. Tente:
```bash
ls /mnt/c/VMs/Projetos/Automonous_Agentic/package.json 2>/dev/null && echo "C drive" || echo "nao C"
ls /mnt/m/Automonous_Agentic/package.json 2>/dev/null && echo "M drive" || echo "nao M"
```

2. No `package.json`, na seção `"devDependencies"`, troque:
```json
"eslint-plugin-agentverse": "file:./eslint-rules"
```
Para (use o drive correto encontrado no passo 1):
```json
"eslint-plugin-agentverse": "file:///M:/Automonous_Agentic/eslint-rules"
```

3. **IMPORTANTE:** NÃO rodar `npm install` nem `npm ci` — apenas editar o arquivo.

4. Atualize `eslint-rules/` — a regra `no-direct-cao-fetch` deve ser renomeada para `no-direct-go-core-fetch` (ou mantida com alias). Verifique o arquivo:
```bash
ls /mnt/p/Automonous_Agentic/eslint-rules/
cat /mnt/p/Automonous_Agentic/eslint-rules/package.json
```
Se a regra referencia "cao" internamente, atualize para "go-core".

5. Em `.eslintrc.cjs`, verifique se a rule `agentverse/no-direct-cao-fetch` precisa ser renomeada.

**Verificação NM1-B:**
```bash
# Em Windows PowerShell (não no WSL) — NM1-B documenta o resultado esperado mas não executa
echo "npm run lint deve retornar 0 errors apos ajuste do drive correto"
```
Documente no ledger: arquivo editado, drive usado, resultado esperado.

---

## GATE de NM1 (após NM1-A e NM1-B finalizarem)

NM1-LEAD verifica:
1. Ler o ledger — NM1-A e NM1-B fizeram CHECK-OUT com DONE
2. Documentar no ledger: resultado de `npx tsc --noEmit` (deve ser 0 errors)
3. Confirmar: `grep -r "from.*cao-client\|caoClient\b" src/ | grep -v deprecated` = vazio
4. Sinalizar ao Orquestrador que GATE 1 está pronto para revisão

## REGRAS ABSOLUTAS
- NUNCA rodar `npm install` ou `npm audit fix`
- NUNCA deletar `cao-client.ts` ou `base-url.ts` (são legados mantidos por segurança)
- SEMPRE rodar `npx tsc --noEmit` após cada arquivo modificado
- SEMPRE registrar no AGENT_LEDGER_S3.md
- Commit obrigatório: `fix(api): CRIT-003 GO Core migration sweep — Sprint-3`
