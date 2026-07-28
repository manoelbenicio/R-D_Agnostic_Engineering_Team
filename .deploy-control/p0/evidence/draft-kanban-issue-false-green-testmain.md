# DRAFT_NOT_POSTED — Rascunho de Proposta de Issue: Correção de Falso-Verde em TestMain de Pacotes com Dependência de Banco de Dados

- **Status da Proposta:** `DRAFT_NOT_POSTED` (Rascunho Local — NÃO POSTADO / NÃO ATRIBUÍDO NO QUADRO)
- **Autor do Rascunho:** Antigravity (wB:p1 / w8:p2)
- **Timestamp UTC:** `2026-07-27T16:18:50Z`
- **Modo:** RASCUNHO MECÂNICO LOCAL — Zero chamadas de rede, API, banco de dados ou execução de testes. Nenhuma atribuição de número de card, comentário ou mutação no quadro.

---

## 1. Inventário de Pacotes, Arquivos e Linhas do Falso-Verde

- **Pacote Primário Impactado**: `github.com/multica-ai/multica/server/internal/handler`
- **Arquivo Alvo Inspecionado**: `multica-auth-work/server/internal/handler/handler_test.go`
- **Intervalo de Linhas Crítico**: Linhas `38` a `54` (`func TestMain(m *testing.M)`)
- **Trecho de Código Mapeado**:
  ```go
  38: func TestMain(m *testing.M) {
  39: 	ctx := context.Background()
  40: 	dbURL := os.Getenv("DATABASE_URL")
  41: 	if dbURL == "" {
  42: 		dbURL = "postgres://multica:multica@localhost:5432/multica?sslmode=disable"
  43: 	}
  44: 
  45: 	pool, err := pgxpool.New(ctx, dbURL)
  46: 	if err != nil {
  47: 		fmt.Printf("Skipping tests: could not connect to database: %v\n", err)
  48: 		os.Exit(0) // <--- FALSO-VERDE: os.Exit(0) pré-m.Run()
  49: 	}
  50: 	if err := pool.Ping(ctx); err != nil {
  51: 		fmt.Printf("Skipping tests: database not reachable: %v\n", err)
  52: 		pool.Close()
  53: 		os.Exit(0) // <--- FALSO-VERDE: os.Exit(0) pré-m.Run()
  54: 	}
  ...
  ```

---

## 2. Conceito do Problema, Impacto e Reprodução

### Problema Diagnosticado:
Quando o banco de dados PostgreSQL não está acessível no ambiente de teste (ex: durante builds de CI sem container Postgres ativo), as linhas 48 e 53 de `handler_test.go` chamam `os.Exit(0)` **antes** de invocar `m.Run()`.

### Impacto Factual:
A ferramenta nativa `go test` interpreta `os.Exit(0)` como encerramento bem-sucedido do processo, imprimindo `ok github.com/multica-ai/multica/server/internal/handler 0.087s` (ou similar) no terminal com código de saída 0.
Isso cria um **FALSO-VERDE SILENCIOSO**: nenhum teste unitário do pacote é efetivamente executado (0 asserções rodadas), mas a suíte passa como "verde" nos portões de validação que dependem apenas do código de saída 0.

### Conceito de Reprodução:
1. Executar `go test ./internal/handler/...` em um ambiente sem serviço PostgreSQL escutando na porta 5432.
2. Observar a saída de `go test`: o utilitário imprime `Skipping tests:...` mas finaliza com `ok` e `exit 0`.
3. Nenhuma linha `--- PASS:` por função de teste é gerada.

---

## 3. Critérios de Aceite Propostos (Acceptance Criteria)

1. **Eliminação do `os.Exit(0)` Falso-Verde**:
   - `TestMain` **NUNCA** deve chamar `os.Exit(0)` pré-`m.Run()` quando a conectividade com o banco de dados falha.
   - Quando o banco de dados for exigido por um conjunto de testes, a ausência de conectividade deve resultar em falha explícita (`os.Exit(1)` ou `t.Fatalf` em testes individuais) OU pular testes explicitamente no nível de função de teste com aviso auditável, nunca mascarando a suíte com `os.Exit(0)` global.
2. **Exigência de Conectividade do Banco ou Falha Explícita (Fail-Closed)**:
   - Em pipelines de integração/CI, a ausência de `DATABASE_URL` válido e conectável deve falhar a execução do portão com erro não-zero (`Exit 1`).
3. **Prova Nominal de Testes (`Nominal Test Proof`)**:
   - As evidências de validação não podem aceitar a linha `ok <package>` isoladamente.
   - É obrigatória a verificação nominal da presença de eventos `--- PASS: <TestName>` no log detalhado (`go test -v`), com contagem explícita de funções de teste executadas.
4. **Isolamento e Segurança de Race Condition**:
   - Garantir que a inicialização de fixtures e pools de conexão permaneça segura contra concorrência (`-race`) e com limpeza adequada de recursos (`pool.Close()`).

---

## 4. Limites de Escopo e Arquivos Provavelmente Bloqueados (FILES_LOCKED)

- **Escopo Restrito**:
  - `multica-auth-work/server/internal/handler/handler_test.go`
  - `multica-auth-work/server/internal/handler/*_test.go` (apenas para adaptação de skips/asserções se necessário)
- **Cadeia de Arquivos Protegidos (Não-Sobreposição)**:
  - Zero alteração em código de produção (`server/internal/handler/*.go` não-testes).
  - Zero alteração em migrations (`server/migrations/**`).
  - Zero alteração em código gerado pelo sqlc (`server/pkg/db/generated/**`).

---

## 5. Dependências e Não-Sobreposição com Outras Cards

- **Dependência de Ordenação**: Deve ser alinhado com as raias de banco de dados (`ORQ-13` e `ORQ-26`) para reutilização do harness oficial de PostgreSQL epêmero em CI.
- **Não-Sobreposição**: Não sobrepõe refatorações de manipuladores HTTP nem alterações no router (`cmd/server/router.go`).

---

## 6. Estratégia de Rollback

- Em caso de regressão na suíte de testes de manipuladores, os arquivos de teste podem ser restaurados ao estado anterior via `git checkout -- multica-auth-work/server/internal/handler/handler_test.go`, preservando 100% o código de produção intocado.

---

## 7. Veredito do Rascunho
- **STATUS: DRAFT_NOT_POSTED (RASCUNHO LOCAL PRONTO PARA APRECIAÇÃO)**
- **Documento Gravado**: `.deploy-control/p0/evidence/draft-kanban-issue-false-green-testmain.md`
- *Rascunho estritamente local. Nenhuma publicação ou atribuição efetuada.*
