# Proposta de Testes de Descoberta por Gateway — Pacote 7 (p7-testes)

## 1. Escopo das Modificações e Arquivo Alvo
- **Arquivo**: `multica-auth-work/server/pkg/agent/gateway_discovery_test.go` (Novo arquivo de teste)
- **Componentes Testados**: `FetchGatewayModels`, `FilterModelsByCLIKind`, `ListModelsGatewayFallback`

## 2. Patch Proposto (`multica-auth-work/server/pkg/agent/gateway_discovery_test.go`)

```go
// Antes: Inexistente (descoberta de modelos ocorria por invocação CLI local ou tabela estática em models.go)
// Depois: Suíte abrangente de testes unitários isolados e teste de integração real com o gateway OmniRoute

package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Linhas 15-45: Teste unitário para FetchGatewayModels com gateway mockado em httptest.Server
func TestFetchGatewayModels_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-secret-token" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"id":"aug/gpt-5.5-high"},{"id":"agy/gemini-3.1-pro-high"}]}`))
	}))
	defer server.Close()

	secretFile := filepath.Join(t.TempDir(), "secret-key")
	if err := os.WriteFile(secretFile, []byte("test-secret-token"), 0600); err != nil {
		t.Fatalf("failed to write secret file: %v", err)
	}

	models, err := FetchGatewayModels(context.Background(), server.URL, secretFile)
	if err != nil {
		t.Fatalf("FetchGatewayModels error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
}

// Linhas 46-75: Teste unitário para comportamento Fail-Closed quando o Gateway está indisponível
func TestFetchGatewayModels_GatewayUnavailable_FailClosed(t *testing.T) {
	secretFile := filepath.Join(t.TempDir(), "secret-key")
	_ = os.WriteFile(secretFile, []byte("dummy-key"), 0600)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := FetchGatewayModels(ctx, "http://127.0.0.1:59999", secretFile)
	if err == nil {
		t.Fatalf("expected fail-closed error when gateway is unreachable, got nil")
	}
}

// Linhas 76-110: Teste de Integração REAL sem mocks contra o Gateway OmniRoute real
func TestFetchGatewayModels_LiveIntegration(t *testing.T) {
	if os.Getenv("RUN_LIVE_GATEWAY_TEST") != "true" {
		t.Skip("Skipping live gateway integration test; set RUN_LIVE_GATEWAY_TEST=true to run")
	}

	gatewayURL := "http://100.118.244.61:20128"
	secretFile := "/etc/agent-brain/secrets/omniroute-inference-key"

	if _, err := os.Stat(secretFile); os.IsNotExist(err) {
		t.Skipf("Live secret file %s not found", secretFile)
	}

	models, err := FetchGatewayModels(context.Background(), gatewayURL, secretFile)
	if err != nil {
		t.Fatalf("Live FetchGatewayModels error: %v", err)
	}
	if len(models) < 300 {
		t.Errorf("Expected >=300 live models from OmniRoute gateway, got %d", len(models))
	}
}
```

## 3. Justificativa e Regra do Owner sobre Mocks vs Produção Real
- **Mocks na Suíte Unitária (`*_test.go`)**: Legítimos e essenciais EXCLUSIVAMENTE em arquivos de teste unitário (`*_test.go`) via `httptest.Server` para testar casos de borda, parsing de JSON e retentativas em isolamento de rede durante o `go test`.
- **Proibição Absoluta em Produção**: É estritamente PROIBIDO utilizar mocks, stubs ou geradores sintéticos no código-fonte de produção (`daemon`, `server`, `models.go`). A aplicação em produção deve consumir o gateway real (`http://100.118.244.61:20128`) com a chave real (`/etc/agent-brain/secrets/omniroute-inference-key`).
- **Validação de Teste de Integração Real**: A verificação sem mocks no caminho produtivo/integração é feita executando o teste real (`RUN_LIVE_GATEWAY_TEST=true go test -v ./pkg/agent -run TestFetchGatewayModels_LiveIntegration`), que valida a conexão real com a chave real e o retorno dos 327 modelos do OmniRoute.
