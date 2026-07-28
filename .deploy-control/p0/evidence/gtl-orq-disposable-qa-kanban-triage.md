# GTL-K01 — Kanban Triage & Card Assignment: Disposable Browser QA (ORQ-31)

- **Autor:** Antigravity (wB:p1 / w8:p2)
- **Data UTC:** 2026-07-27T12:36:00Z
- **Destinatários:** General-Tech-Lead (Codex56-TL w5:pC), KIRO-PRINCIPAL-TL (wB:p1)
- **Base de Evidência:** `.deploy-control/p0/evidence/gtl-browser-qa-disposable-plan.md`
- **Modo:** GOVERNANÇA DE KANBAN / TRIAGEM READ-ONLY — Zero edição de código, zero testes executados, zero instalações.

---

## 1. Evidência da Busca por Duplicatas e Decisão de Decisão

### 1.1 Auditoria do Catálogo Kanban (`ORQ-01` a `ORQ-30`)
- **`ORQ-26`**: Foco restrito ao contrato do servidor (`file.go` / `file_test.go`) e schemas do cliente TypeScript (`schema.test.ts`, `client.test.ts`). Não possui critérios de aceite cobrindo a infraestrutura de CI efêmera do Playwright, lockfile do Chromium, nem a suíte E2E de navegadores.
- **`ORQ-27` / `ORQ-28` / `ORQ-29`**: Testes de fumaça e liveness dos slots AGY no host ORQ2.
- **`ORQ-30`**: Persistência de ambiente e tokens JWT no host ORQ2.
- **Resultado da Busca**: Nenhuma card existente cobre a execução de testes E2E descartáveis via Playwright com stack efêmera completa e trava de supply chain.

### 1.2 Regra de Decisão Aplicada
Perante a ausência de cobertura nos critérios de aceite de `ORQ-26`, **NÃO se reusa `ORQ-26` como card principal**. É criada a **NOVA CARD `ORQ-31`**, devidamente encadeada com a `ORQ-26` como dependência de contrato.

---

## 2. Ficha Completa da Nova Card Kanban: `ORQ-31`

| Campo | Valor / Especificação |
|---|---|
| **ID do Card** | `ORQ-31` |
| **Título** | `Ephemeral Browser QA & Playwright Supply-Chain Pipeline` |
| **Status** | `PLANNED / TRIAGED` |
| **Reusada / Criada** | **CRIADA** (Nova card Kanban) |
| **Linkage / Dependências** | Vinculada a **`ORQ-26`** (Depende da fix de contrato `uploadFile` / `file.go`) |
| **Assignee (Responsável)** | `Opus48#B` (QA / Frontend Lead) |
| **Reviewer (Revisor)** | `Codex56-TL` (General-Tech-Lead) |

### 2.1 Evidências Medidas (`gtl-browser-qa-disposable-plan.md`)
- `playwright.config.ts:19-21`: `auto-start` desabilitado, exigindo servidor web `baseURL` já rodando.
- `e2e/fixtures.ts:29-51`: O teste de login executa `DELETE FROM verification_code` direto no banco via `DATABASE_URL`. Exige banco descartável efêmero para não destruir dados nem rodar `DELETE` em produções ORQ1/ORQ2.
- `package.json:43`: Especificador caret `^1.58.2` no Playwright exige validação estrita com `pnpm-lock.yaml` (versão 1.58.2) e a imagem Docker `mcr.microsoft.com/playwright:v1.58.2-noble`.

### 2.2 Impacto
Previne mutações acidentais e deleções na tabela `verification_code` em bancos de produção; bloqueia riscos de segurança de supply-chain entre a versão do pacote npm `@playwright/test` e o driver Chromium em runners de CI.

### 2.3 Critérios de Aceite Obrigatórios
1. **Isolamento de Ambiente**: Execução exclusiva em runner efêmero de CI (GitHub Actions `ubuntu-24.04` ou container dedicado com `--rm`), utilizando container de serviço `postgres:17` efêmero (`localhost:5432`). Zero acesso a ORQ1/ORQ2 ou segredos de produção.
2. **Supply-Chain Locking**: Uso estrito de `pnpm install --frozen-lockfile`, validando a coincidência exata da versão do `@playwright/test` (1.58.2) com a tag da imagem Docker `mcr.microsoft.com/playwright:v1.58.2-noble`.
3. **Cobertura E2E de Superfícies UI**: Sucesso da suíte Playwright Chromium testando as superfícies da UI (Chat, Upload de Arquivos da `ORQ-26`, Quadro Kanban, Níveis de Reasoning, Squads e Exclusão de Runtimes).
4. **Zero Segredos Produtivos**: `DATABASE_URL` apontando exclusivamente para o container de teste local `postgres://postgres:postgres@localhost:5432/test_db`.

### 2.4 Invariante de Dados de Produção (No-Production-Data)
Stack 100% isolada e efêmera que morre ao final do job. Nenhuma credencial de produção, nenhuma chave privada e nenhum banco live são expostos ou alterados.

### 2.5 Riscos & Plano de Rollback
- **Risco**: Conflito de portas ou timeout de compilação Next.js no runner de CI.
- **Rollback**: Falha na CI cancela o workflow sem afetar o repositório, branches principais ou servidores de produção.

---

## 3. Veredito Final
- **STATUS: PASS (CARD ORQ-31 TRIADA E CRIADA NO KANBAN)**
- **Documento Gravado**: `.deploy-control/p0/evidence/gtl-orq-disposable-qa-kanban-triage.md`
- *Operação 100% Read-Only de Governança. Zero código editado, zero testes executados.*
