# PROMPT CX1 — Codex (Instância 1)
# Workstream: Reconciler GO Core Migration + Orphan Cleanup
# Sprint 3 | CRIT-003.6 + HIGH-001 + MED-002

## SEU PAPEL
Você é CX1, responsável por migrar o reconciler para GO Core e adicionar resiliência (orphan cleanup).
Você pode escalar 2 sub-agentes: **CX1-A** e **CX1-B**.

## DEPENDÊNCIA
**AGUARDE NM1 finalizar** (verificar GATE 1 no `.planning/AGENT_LEDGER_S3.md` = DONE).
Só então inicie. O reconciler depende de `goCoreClient` que NM1 está consolidando.

## OBRIGATORIO antes de qualquer edição
1. Leia `.planning/AGENT_LEDGER_S3.md`
2. Confirme GATE 1 = aprovado
3. Registre CHECK-IN para cada arquivo

---

### CX1-A: CRIT-003.6 — Migrar reconciler.ts para GO Core

**Arquivo:** `src/canvas-reconciler/reconciler.ts`

1. Troque o import:
   ```typescript
   // DE:
   import { caoClient } from '@/api/cao-client';
   // PARA:
   import { goCoreClient } from '@/api/go-core-client';
   ```

2. Troque TODAS as ocorrências de `caoClient.` por `goCoreClient.` dentro do arquivo.
   Verifique com:
   ```bash
   grep -n "caoClient" src/canvas-reconciler/reconciler.ts
   ```

3. Verifique se `resolveSessionEnv()` continua compatível com GO Core (a assinatura não muda — só o endpoint chamado).

4. Após editar:
   ```bash
   npx tsc --noEmit
   ```
   Deve ser 0 errors. Documente no ledger.

---

### CX1-B: HIGH-001 + MED-002 — Orphan Cleanup + tearDown 404

**CONTEXTO:** Quando a sequência `installProfile() → createSession() → addTerminal()` falha na metade,
o profile instalado fica "orfão" no GO Core. O reconciler deve limpar. (RCA-2026-05-31-001 FINDING-007)

**Arquivo:** `src/canvas-reconciler/reconciler.ts`

**Padrão a aplicar em cada deploy path:**
```typescript
// PADRÃO CORRETO — compensation transaction
let profileInstalled = false;
let sessionName: string | null = null;

try {
  await goCoreClient.installProfile(profileMarkdown);
  profileInstalled = true;

  const session = await goCoreClient.createSession({ ... });
  sessionName = session.name;

  await goCoreClient.addTerminalToSession(session.name, { ... });

} catch (err) {
  // Compensation — limpar o que foi criado
  if (sessionName) {
    try { await goCoreClient.deleteSession(sessionName); } catch { /* best-effort */ }
  }
  if (profileInstalled && !sessionName) {
    // Profile orfão — GO Core não tem DELETE /agents/profiles/:name em v1
    // Documente como known limitation se o endpoint não existir
    console.warn('[reconciler] Profile orphan after session creation failure:', profileName);
  }
  throw err;
}
```

**Aplique este padrão nos 4 caminhos de deploy** no reconciler (busque por `installProfile` — há 4 call sites):
```bash
grep -n "installProfile" src/canvas-reconciler/reconciler.ts
```

**MED-002:** `tearDownCanvas()` — adicionar try/catch ao deletar sessão:
```typescript
// DE:
await goCoreClient.deleteSession(sessionName);

// PARA:
try {
  await goCoreClient.deleteSession(sessionName);
} catch (err) {
  // 404 = sessão já removida (tolerável)
  if (!(err instanceof CaoApiError) || err.status !== 404) throw err;
  console.warn('[reconciler] Session already gone on teardown:', sessionName);
}
```

**Verificação CX1-B:**
```bash
npx tsc --noEmit
npx vitest run src/canvas-reconciler/ --reporter=verbose 2>&1 | tail -20
```

---

## GATE de CX1
1. Confirmar `npx tsc --noEmit` = 0 errors
2. Confirmar testes do reconciler passam
3. Registrar CHECK-OUT no ledger
4. Commit: `fix(reconciler): CRIT-003.6 GO Core + HIGH-001 orphan cleanup — Sprint-3`

## REGRAS ABSOLUTAS
- NUNCA deletar `cao-client.ts` ou `base-url.ts`
- NUNCA rodar `npm install` ou `npm audit fix`
- SEMPRE registrar no ledger antes/depois
