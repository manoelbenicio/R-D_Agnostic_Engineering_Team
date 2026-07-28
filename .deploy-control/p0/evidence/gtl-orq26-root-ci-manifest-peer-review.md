# Peer Review Adversarial: Manifesto do Commit CI Temporário ORQ-26 (GTL-69)

**auditor**: Antigravity w8:p2  
**timestamp**: 2026-07-27T12:07Z  
**solicitante**: General-Tech-Lead (Codex56-TL w5:pC)  
**documento revisado**: `.deploy-control/p0/evidence/gtl-orq26-ci-commit-manifest.md` (autor: Kiro-Opus5)  
**modo**: READ-ONLY / AUDITORIA ADVERSARIAL — NENHUM commit, push, edit ou trigger executado  

---

## 1. Veredito Final de Peer Review

### **VEREDITO: BLOCK** ❌

**Resumo da Avaliação**:
O manifesto `gtl-orq26-ci-commit-manifest.md` é tecnicamente brilhante no tratamento de falso-verde, isolamento de diretórios temporários (`0700`), verificação do banco pgvector e validação dos 8 testes de contrato. No entanto, o veredito é **BLOCK por 2 violações estritas de política de CI**:
1. **Gatilho Proibido no YAML**: O arquivo YAML proposto em `.github/workflows/orq26-db-gate.yml` (linhas 97-100) incluiu o gatilho automático `push: branches: [agent/kiro-opus5/orq-26-contract-fix]`. A regra estrita exige **`workflow_dispatch` EXCLUSIVAMENTE**, sem gatilhos automáticos de `push` ou `pull_request`.
2. **Pinning de Actions**: Os passos utilizam tags mutáveis de versão (`actions/checkout@v6`, `actions/setup-go@v5`) em vez de hashes SHA de 40 caracteres para mitigar riscos de supply-chain.

---

## 2. Auditoria Item a Item dos Critérios Obrigatórios

| # | Item Auditado | Resultado da Auditoria | Evidência & Linha Literal | Status |
|---|---|---|---|---|
| **1** | **Workflows da Raiz e Inertes** | Confirmado que a raiz do repo **não possui** `.github/workflows/`. Workflows em subdiretórios (`multica-auth-work/...` e `node_modules/...`) são totalmente **inertes**. | `ls .github/workflows/` -> Inexistente na raiz. GitHub Actions lê apenas a raiz. | ✅ PASS |
| **2** | **Gatilho `workflow_dispatch` Único** | **FALHA CRÍTICA**. O YAML proposto inclui gatilho `push:` em branch, violando a regra de execução manual estrita. | Lines 97-100: `on: workflow_dispatch: push: branches: [...]` | ❌ **BLOCK** |
| **3** | **Permissões Mínimas (`contents: read`)** | Aplicadas permissões mínimas globais no job. | Line 106-107: `permissions: { contents: read }` | ✅ PASS |
| **4** | **Ausência de Secrets ou DSN Privado** | Utiliza credenciais descartáveis locais `orq26_gate:orq26_gate` no container de serviço. Zero secrets de repositório. | Lines 117-124: `POSTGRES_USER: orq26_gate` / `POSTGRES_PASSWORD: orq26_gate` | ✅ PASS |
| **5** | **Paths e `working-directory` Corretos** | Aponta corretamente para `multica-auth-work/server` nos passos de build, vet, migrate e test. | Lines 165, 172, 195, 199, 203, 218: `working-directory: multica-auth-work/server` | ✅ PASS |
| **6** | **Service Container `pgvector/pg17`** | Configurado com healthcheck `pg_isready` e porta 5432 exposta para o runner efêmero. | Lines 113-131: `image: pgvector/pgvector:pg17` | ✅ PASS |
| **7** | **Verificação Completa de Migrations** | Aplica todas as migrations `.up.sql` e valida ausência de migrations pendentes (`missing-migrations` vazio). | Lines 202-216: `go run ./cmd/migrate up` + `comm -23` | ✅ PASS |
| **8** | **Filtro JSON e Prova de 8 Testes Folhas** | Valida os 8 testes folhas exatos com `run`+`pass` e falha se houver marcador de falso-verde ou `skip`. | Lines 220-254: Validação de 8 testes exatos com `count -eq 8`. | ✅ PASS |
| **9** | **Diretórios Privados `0700` (`RUNNER_TEMP`)** | Cria diretórios com `umask 077` e `mkdir -m 700` para isolamento no runner. | Lines 153-163: `mkdir -m 700 -p "$gate_dir/tmp" ...` | ✅ PASS |
| **10** | **Pinning de Actions (Supply-Chain)** | Usa tags major (`@v6`, `@v5`, `@v4`). Recomenda-se converter para hashes SHA256 de 40 caracteres para imunidade a supply-chain attacks. | Lines 145, 148, 295: `actions/checkout@v6`, `actions/setup-go@v5` | 🟡 RESSALVA |
| **11** | **Sem Vazamento de DSN nos Artifacts** | Artifacts contêm apenas JSONs de resultados (`targeted.json`, `pkg.json`) e listas de migrations. Logs com DSN são omitidos. | Lines 293-304: Upload apenas de `.json` e listas de migrations. | ✅ PASS |
| **12** | **Lista Exata dos 3 Arquivos no Commit** | Confirma exatamente 3 arquivos para o commit temporário. | Section 2.1: `file.go`, `file_test.go`, `.github/workflows/orq26-db-gate.yml`. | ✅ PASS |
| **13** | **Status dos Consumer Tests (GTL-48)** | Consumer tests (`packages/core/api/schema.test.ts`) continuam com **PASS garantido em evidência separada** (GTL-48). Não exigem inclusão neste commit Go server. | Evidência `gtl-orq26-consumer-tests-peer-review.md` (PASS 95/95). | ✅ PASS |
| **14** | **Viabilidade de Push (`workflow` Scope)** | O envio de arquivos em `.github/workflows/` exige que o Token/PAT do git possua o escopo de OAuth `workflow`. Sem esse escopo, o GitHub recusa o push com erro 403. | Análise de viabilidade realizada sem disparar comando de push. | 🟡 ALERTA SCOPE |

---

## 3. Correção Mínima Requerida no YAML Proposto

O bloco `on:` do YAML deve ser corrigido para aceitar **EXCLUSIVAMENTE `workflow_dispatch`**:

```yaml
# CORREÇÃO OBRIGATÓRIA em .github/workflows/orq26-db-gate.yml
name: ORQ-26 DB Gate (temporary)

on:
  workflow_dispatch:  # Apenas acionamento manual; NENHUM gatilho de push ou pull_request
```

---

## 4. Plano de Autorização Mínima Corrigido para o Owner

Para que o General-TL (`Codex56-TL` w5:pC) solicite autorização ao Owner, as 4 etapas devem seguir esta ordem de liberação:

1. **A1 (Commit Local na Branch)**: Commitar localmente os 3 arquivos ajustados na branch isolada `agent/kiro-opus5/orq-26-contract-fix`.
2. **A2 (Push com Verificação de Scope)**: Executar `git push -u origin agent/kiro-opus5/orq-26-contract-fix` (requer aprovação explícita do Owner por ser envio externo e requerer escopo `workflow`).
3. **A3 (Acionamento Manual `workflow_dispatch`)**: Disparar manualmente o workflow efêmero via GitHub CLI / Interface Web.
4. **A4 (Reversão & Cleanup)**: Após a geração do artefato de evidência do gate, reverter o workflow na raiz e deletar a branch remota.

*Auditoria 100% READ-ONLY. Nenhum commit, push, merge ou dispatch foi realizado.*
