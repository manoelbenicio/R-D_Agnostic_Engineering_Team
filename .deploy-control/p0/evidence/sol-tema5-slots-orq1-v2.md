# TEMA 5 v2 — Solução HOJE para agy+kiro+codex (opencode fora de escopo)
**Data:** 2026-07-26T23:07Z  **Revisor:** Antigravity Opus48#C / w8:p1

## Diagnóstico definitivo (literal do código)

### execenv.go — comportamento com CredentialAccountHome=""

```go
// L300-301: kiro usa HOME global quando vazio
// Empty CredentialAccountHome = shared/global behavior (no isolated home).
if params.Provider == "kiro" && params.CredentialAccountHome != "" {
    // só isola se NÃO vazio — se vazio, kiro usa o HOME do processo (=HOME global do ORQ1)

// L311-312: antigravity idem
// Empty CredentialAccountHome = shared/global behavior (no isolated home).
if params.Provider == "antigravity" && params.CredentialAccountHome != "" {
    // só isola se NÃO vazio — se vazio, agy usa HOME do processo
```

**CONCLUSÃO: com `CredentialAccountHome=""` (estado atual), kiro e agy usam o HOME global do ORQ1 (`/home/ec2-user`) automaticamente.** O TL já logou agy e kiro nesse HOME. Logo, as tasks JÁ DEVERIAM funcionar com o binário atual — sem patch, sem recompilação.

### O que bloqueia HOJE não é o CredentialAccountHome

`CredentiallessGateway: true` (L3476/3493) para codex chama `prepareCredentiallessCodexHome` (L284-285) em vez de `prepareCodexHomeWithOpts`. Para kiro e agy, `CredentiallessGateway` não tem branch especial — eles caem no caminho normal acima.

**Caminho real do bloqueio para kiro/agy:**  
A condição `CredentialAccountHome != ""` em L301/L312 é `false` (vazio), então os blocos `prepareKiroHome`/`prepareAntigravityHome` são **pulados** — mas isso é intencional: o comentário diz que vazio = comportamento global. Kiro e agy **vão usar o HOME do processo**, que é `/home/ec2-user` no ORQ1, que já tem os tokens.

### Patch mínimo em daemon.go (1 linha, requer recompilação)

**ANTES (L3448):**
```go
credentialAccountHome := ""
```

**DEPOIS — para garantir isolamento futuro sem quebrar hoje:**
```go
credentialAccountHome := os.Getenv("MULTICA_CREDENTIAL_ACCOUNT_HOME")
// Vazio = comportamento global (usa HOME do processo). Não quebra nada.
// Quando slots forem provisionados no ORQ1, basta setar a env var por runtime.
```

Isso não muda nada no comportamento atual (valor continua vazio = HOME global) mas abre o caminho para slots sem recompilar de novo.

## Resposta à pergunta: funciona SEM recompilar?

**Para kiro e agy: SIM — já funciona com o binário atual.**  
O HOME global do ORQ1 já tem as credenciais. A condição `CredentialAccountHome=""` faz o código usar o HOME do processo (`/home/ec2-user`). Se as tasks ainda falham, o problema está em outro lugar (modelo não mapeado em config.go:145-154, não em credential home).

**Para codex: SIM — `CredentiallessGateway: true` já existe no código.**  
Codex usa `prepareCredentiallessCodexHome` (L284). Funciona sem credencial.

## Provável causa real das falhas de task com kiro/agy

`config.go:145-154` — `agentBrainBuiltInCLIFor` mapeia só 3 providers. Se kiro e agy não estão no mapa, as tasks chegam ao execenv com `provider=""` ou com provider que não bate. Isso é o patch de outro pacote — **não é problema de CredentialAccountHome**.

## O que fazer HOJE (sem recompilar)

1. Verificar se uma task de kiro/agy realmente falha e qual é o erro literal no log.
2. Se o erro for "unsupported CLI kind" ou "provider not found" → é config.go, não credential home.
3. Se o erro for "no credentials" → aí credentialAccountHome precisa ser preenchido com `/home/ec2-user`.

**Em qualquer caso, não há nova infra de slots para provisionar hoje.** O HOME global já está pronto.
