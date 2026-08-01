package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validRuntimeManagerAuthority = `{
  "providers": {
    "provider-a": {
      "version": "v1",
      "provider": "provider-a",
      "models": {
        "model-a": {
          "reasoning_efforts": ["low", "high"],
          "max_input_tokens": 1000,
          "max_output_tokens": 500,
          "context_window_tokens": 1500,
          "max_tool_calls": 10
        }
      }
    }
  }
}`

func TestRuntimeManagerOptionsFromEnvDisabled(t *testing.T) {
	t.Setenv(runtimeManagerEnabledEnv, "false")
	t.Setenv(runtimeManagerAuthorityFileEnv, "")
	options, err := runtimeManagerOptionsFromEnv()
	if err != nil || options != nil {
		t.Fatalf("options=%+v err=%v", options, err)
	}
}

func TestRuntimeManagerOptionsFromEnvFailsClosed(t *testing.T) {
	t.Setenv(runtimeManagerEnabledEnv, "true")
	t.Setenv(runtimeManagerAuthorityFileEnv, "")
	if _, err := runtimeManagerOptionsFromEnv(); err == nil {
		t.Fatal("enabled runtime manager accepted missing authority")
	}

	path := filepath.Join(t.TempDir(), "authority.json")
	if err := os.WriteFile(path, []byte(`{"providers":{},"unknown":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(runtimeManagerAuthorityFileEnv, path)
	if _, err := runtimeManagerOptionsFromEnv(); err == nil {
		t.Fatal("authority with unknown field accepted")
	}
}

func TestRuntimeManagerOptionsFromEnvLoadsValidatedShape(t *testing.T) {
	path := filepath.Join(t.TempDir(), "authority.json")
	if err := os.WriteFile(path, []byte(validRuntimeManagerAuthority), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(runtimeManagerEnabledEnv, "true")
	t.Setenv(runtimeManagerAuthorityFileEnv, path)
	options, err := runtimeManagerOptionsFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if options == nil || len(options.Capabilities) != 1 || string(options.Capabilities["provider-a"].Provider) != "provider-a" {
		t.Fatalf("options=%+v", options)
	}
	if _, err := newRuntimeManagerComposition(nil, nil, options); err == nil {
		t.Fatal("composition accepted missing postgres pool")
	}
}

func TestRuntimeManagerOwnerMiddlewareClassification(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
	request := httptest.NewRequest(http.MethodPost, "/api/runtime-standards", nil)
	request.Header.Set("X-User-ID", "01972f7e-7e8d-77ef-a13d-1b0ce3e9c001")
	request.Header.Set("X-Actor-Source", "task_token")
	response := httptest.NewRecorder()
	runtimeManagerOwnerMiddleware(true)(next).ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || called || !strings.Contains(response.Body.String(), `"code":"forbidden"`) {
		t.Fatalf("admin response=%d body=%s called=%v", response.Code, response.Body.String(), called)
	}

	called = false
	response = httptest.NewRecorder()
	runtimeManagerOwnerMiddleware(false)(next).ServeHTTP(response, request)
	if response.Code != http.StatusOK || !called {
		t.Fatalf("read response=%d body=%s called=%v", response.Code, response.Body.String(), called)
	}
}
