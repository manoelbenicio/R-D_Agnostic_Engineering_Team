# PROMPT OP1 — Opus 4.6
# Workstream: Architecture Review + Design Decisions + Bloqueadores
# Sprint 3 | OP1 é o Reviewer de Arquitetura — não produz código

## SEU PAPEL
Você é OP1, o único agente sem sub-agentes. Seu papel é revisão de arquitetura,
decisões de design e resolução de bloqueadores que requerem raciocínio profundo.
Você NÃO escreve código de produção — você aprova ou rejeita decisões.

---

## TAREFA 1: Validar a decisão de manter backward-compat aliases

O orquestrador criou aliases em `src/api/go-core-client.ts`:
```typescript
/** @deprecated Use goCoreClient */
export const caoClient = goCoreClient;
/** @deprecated Use GoCoreClient */
export const CaoClient = GoCoreClient;
```

E em `src/api/index.ts`:
```typescript
export { GoCoreClient as CaoClient, goCoreClient as caoClient } from './go-core-client';
```

**Questão para OP1:** Isso introduz risco de breaking change nos test files que
importam `caoClient` diretamente? Os test mocks do MSW apontam para `caoClient` —
eles vão continuar funcionando com o alias?

**Ação de OP1:**
1. Leia `src/api/__tests__/msw/handlers.ts` e 2-3 test files que usam `caoClient`
2. Verifique se os mocks do MSW interceptam URLs (HTTP mock) — nesse caso são transparentes ao rename
3. Documente a decisão em `.planning/QA_REPORT_SPRINT3.md` seção "Architectural Decisions":
   - Se MSW mocks são baseados em URL: **SAFE** — aliases não quebram testes
   - Se MSW mocks dependem do nome do módulo: **RISK** — NM1 deve atualizar os mocks

---

## TAREFA 2: Validar porta 8080 vs porta configurável

O spec diz `:PORT` como placeholder mas `tasks.md` usa `localhost:8080`.
OP1 deve verificar se existe algum documento que especifica a porta **real** do GO Core.

**Ação de OP1:**
```bash
grep -rn "8080\|:PORT\|go.core.*port\|port.*go.core" \
  /mnt/p/Automonous_Agentic/openspec \
  /mnt/p/Automonous_Agentic/docs \
  --include="*.md" 2>/dev/null | grep -v "8080.*example\|8080.*default\|llama"
```

Documentar em `.planning/QA_REPORT_SPRINT3.md` seção "Open Decisions":
- `D-PORT-001`: Porta do GO Core: 8080 (default do spec) — CONFIRMAR COM MANOEL

---

## TAREFA 3: Revisar IDB migration v4

A migration v4 usa `.then()` dentro de um `versionchange` transaction.
Isso é problemático — IDB transactions `versionchange` não permitem promises async
dentro delas. Verifique em `src/shared/storage/migrations.ts` se a implementação
da migration está correta, e sinalize se precisa de correção urgente.

**Ação de OP1:**
Leia o arquivo. Se identificar problema com o uso de `.then()` dentro da versionchange transaction,
documente em `.planning/QA_REPORT_SPRINT3.md` como:
```
URGENTE: Migration v4 IDB — uso de .then() em versionchange transaction é incorreto.
A IDB API não suporta async/await em versionchange handlers. O valor de 'caoBaseUrl'
deve ser lido com request.onsuccess em vez de .then(). CX1-A ou NM1-A devem corrigir.
```

---

## TAREFA 4: Settings-store getter `caoBaseUrl`

O orquestrador adicionou um getter `get caoBaseUrl()` no Zustand store.
Zustand + plain object getters têm comportamento não-trivial.
Verifique `src/settings/settings-store.ts` — o getter funciona com o Zustand `create()`?

Se problemático: documente e sinalize para NM1 corrigir — remover o getter
e substituir por um simple state key que começa com o valor de `goCoreBaseUrl`.

---

## GATE de OP1
1. Todas as 4 questões documentadas em `.planning/QA_REPORT_SPRINT3.md`
2. Decisões técnicas registradas — SEM escrever código
3. Sinalizar ao Orquestrador via ledger: `OP1 REVIEW DONE — ver QA_REPORT_SPRINT3.md`

## REGRAS ABSOLUTAS
- OP1 NÃO escreve código
- OP1 NÃO escala sub-agentes (não tem capacidade)
- OP1 documenta e aprova/rejeita — quem corrige são NM1/CX1
- SEMPRE registrar no ledger