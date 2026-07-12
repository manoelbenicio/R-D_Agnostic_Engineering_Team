package agent

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClineToolNameFromTitle(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"Read file: /tmp/foo.go": "read_file",
		"Write: /tmp/bar.go":     "write_file",
		"Patch: /tmp/x":          "edit_file",
		"Run command: pwd":       "terminal",
		"Search: foo":            "search_files",
		"Browser Action":         "browser_action",
		"":                       "",
	}
	for title, want := range tests {
		if got := clineToolNameFromTitle(title); got != want {
			t.Errorf("clineToolNameFromTitle(%q) = %q, want %q", title, got, want)
		}
	}
}

func fakeClineACPScript() string {
	return `#!/bin/sh
for arg in "$@"; do printf '%s\n' "$arg" >> "$CLINE_ARGS_FILE"; done
printf '%s|%s\n' "$CLINE_DATA_DIR" "$CLINE_SANDBOX_DATA_DIR" > "$CLINE_ENV_FILE"
while IFS= read -r line; do
  printf '%s\n' "$line" >> "$CLINE_REQUESTS_FILE"
  id=$(printf '%s' "$line" | sed -n 's/.*"id":\([0-9]*\).*/\1/p')
  case "$line" in
    *'"method":"initialize"'*)
      printf '{"jsonrpc":"2.0","id":%s,"result":{"protocolVersion":1,"agentCapabilities":{}}}\n' "$id"
      ;;
    *'"method":"session/new"'*)
      printf '{"jsonrpc":"2.0","id":%s,"result":{"sessionId":"cline-session"}}\n' "$id"
      ;;
    *'"method":"session/set_model"'*)
      printf '{"jsonrpc":"2.0","id":%s,"result":{}}\n' "$id"
      ;;
    *'"method":"session/prompt"'*)
      printf '{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"cline-session","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"done"}}}}\n'
      printf '{"jsonrpc":"2.0","id":%s,"result":{"stopReason":"end_turn","usage":{"inputTokens":7,"outputTokens":3}}}\n' "$id"
      exit 0
      ;;
  esac
done
`
}

func TestClineBackendRunsNativeACPAndPropagatesIsolation(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	fakePath := filepath.Join(tmp, "cline")
	writeTestExecutable(t, fakePath, []byte(fakeClineACPScript()))
	argsFile := filepath.Join(tmp, "args")
	envFile := filepath.Join(tmp, "env")
	requestsFile := filepath.Join(tmp, "requests")
	dataDir := filepath.Join(tmp, "cline-data")
	sandboxDir := filepath.Join(tmp, "cline-sandbox")

	backend := &clineBackend{cfg: Config{
		ExecutablePath: fakePath,
		Logger:         slog.Default(),
		Env: map[string]string{
			"CLINE_ARGS_FILE":        argsFile,
			"CLINE_ENV_FILE":         envFile,
			"CLINE_REQUESTS_FILE":    requestsFile,
			"CLINE_DATA_DIR":         dataDir,
			"CLINE_SANDBOX_DATA_DIR": sandboxDir,
		},
	}}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	session, err := backend.Execute(ctx, "implement it", ExecOptions{
		Cwd:           tmp,
		Model:         "cline-model",
		SystemPrompt:  "system rules",
		Timeout:       5 * time.Second,
		ThinkingLevel: "high",
		CustomArgs:    []string{"--acp", "--json", "--thinking", "low", "--verbose"},
		McpConfig:     json.RawMessage(`{"mcpServers":{}}`),
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var messages []Message
	for msg := range session.Messages {
		messages = append(messages, msg)
	}
	result := <-session.Result
	if result.Status != "completed" || result.Output != "done" || result.SessionID != "cline-session" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if got := result.Usage["cline-model"]; got.InputTokens != 7 || got.OutputTokens != 3 {
		t.Fatalf("unexpected usage: %+v", got)
	}
	if len(messages) != 1 || messages[0].Type != MessageText || messages[0].Content != "done" {
		t.Fatalf("unexpected messages: %+v", messages)
	}

	args := strings.Fields(string(mustReadFile(t, argsFile)))
	wantArgs := []string{"--acp", "--json", "--thinking", "high", "--verbose"}
	if strings.Join(args, "|") != strings.Join(wantArgs, "|") {
		t.Fatalf("argv = %q, want %q", args, wantArgs)
	}
	if got := strings.TrimSpace(string(mustReadFile(t, envFile))); got != dataDir+"|"+sandboxDir {
		t.Fatalf("isolation env = %q", got)
	}
	requests := string(mustReadFile(t, requestsFile))
	for _, want := range []string{
		`"method":"initialize"`,
		`"method":"session/new"`,
		`"method":"session/set_model"`,
		`"modelId":"cline-model"`,
		`"text":"system rules\n\n---\n\nimplement it"`,
	} {
		if !strings.Contains(requests, want) {
			t.Errorf("requests missing %s:\n%s", want, requests)
		}
	}
}

func TestClineBackendRejectsMalformedMCPConfig(t *testing.T) {
	t.Parallel()
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	backend := &clineBackend{cfg: Config{ExecutablePath: executable, Logger: slog.Default()}}
	_, err = backend.Execute(context.Background(), "prompt", ExecOptions{McpConfig: json.RawMessage(`{`)})
	if err == nil || !strings.Contains(err.Error(), "cline: invalid mcp_config") {
		t.Fatalf("error = %v, want invalid mcp_config", err)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return b
}
