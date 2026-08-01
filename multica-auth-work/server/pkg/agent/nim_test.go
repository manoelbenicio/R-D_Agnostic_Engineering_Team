package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestNIMExecuteStreamsToolsAndUsage(t *testing.T) {
	var requests atomic.Int32
	root := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("Accept"); got != "text/event-stream" {
			t.Errorf("Accept = %q", got)
		}
		var request nimRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Model != "test/model" {
			t.Errorf("model = %q, want test/model", request.Model)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		if requests.Add(1) == 1 {
			fmt.Fprintln(w, `data: {"model":"test/model","choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"write_file","arguments":"{\"path\":\"nested/"}}]}}]}`)
			fmt.Fprintln(w, `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"result.txt\",\"content\":\"created\"}"}}]},"finish_reason":"tool_calls"}],"usageMetadata":{"promptTokenCount":11,"candidatesTokenCount":3,"cachedContentTokenCount":2}}`)
			fmt.Fprintln(w, "data: [DONE]")
			return
		}
		if len(request.Messages) < 3 || request.Messages[len(request.Messages)-1].Role != "tool" {
			t.Errorf("second request did not contain tool result: %#v", request.Messages)
		}
		fmt.Fprintln(w, `data: {"model":"test/model","choices":[{"delta":{"content":"Finished"},"finish_reason":"stop"}],"usage":{"prompt_tokens":7,"completion_tokens":2,"prompt_tokens_details":{"cached_tokens":1}}}`)
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer server.Close()

	backend := &nimBackend{cfg: Config{Env: map[string]string{"NVIDIA_API_KEY": "test-key"}}, client: server.Client(), baseURL: server.URL + "/v1"}
	session, err := backend.Execute(context.Background(), "create the file", ExecOptions{Cwd: root, Model: "test/model"})
	if err != nil {
		t.Fatal(err)
	}
	var messages []Message
	for message := range session.Messages {
		messages = append(messages, message)
	}
	result := <-session.Result
	if result.Status != "completed" || result.Output != "Finished" {
		t.Fatalf("result = %#v", result)
	}
	if got := result.Usage["test/model"]; got != (TokenUsage{InputTokens: 18, OutputTokens: 5, CacheReadTokens: 3}) {
		t.Fatalf("usage = %#v", got)
	}
	if data, err := os.ReadFile(filepath.Join(root, "nested", "result.txt")); err != nil || string(data) != "created" {
		t.Fatalf("written file = %q, %v", data, err)
	}
	if len(messages) != 3 || messages[0].Type != MessageToolUse || messages[1].Type != MessageToolResult || messages[2].Content != "Finished" {
		t.Fatalf("messages = %#v", messages)
	}
}

func TestNIMUsesGLM52AsRuntimeDefault(t *testing.T) {
	root := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request nimRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Model != "z-ai/glm-5.2" {
			t.Errorf("model = %q, want z-ai/glm-5.2", request.Model)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"model":"z-ai/glm-5.2","choices":[{"delta":{"content":"ok"},"finish_reason":"stop"}]}`)
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer server.Close()

	backend := &nimBackend{
		cfg:     Config{Env: map[string]string{"NVIDIA_API_KEY": "test-key"}},
		client:  server.Client(),
		baseURL: server.URL,
	}
	session, err := backend.Execute(context.Background(), "test", ExecOptions{Cwd: root})
	if err != nil {
		t.Fatal(err)
	}
	for range session.Messages {
	}
	result := <-session.Result
	if result.Status != "completed" || result.Output != "ok" {
		t.Fatalf("result = %+v", result)
	}
}

func TestNIMExecuteRequiresCredentialAndWorkspace(t *testing.T) {
	backend := &nimBackend{cfg: Config{Env: map[string]string{"NVIDIA_API_KEY": ""}}}
	t.Setenv("NVIDIA_API_KEY", "")
	if _, err := backend.Execute(context.Background(), "prompt", ExecOptions{Cwd: t.TempDir()}); err == nil || !strings.Contains(err.Error(), "NVIDIA_API_KEY") {
		t.Fatalf("credential error = %v", err)
	}
	backend.cfg.Env["NVIDIA_API_KEY"] = "key"
	if _, err := backend.Execute(context.Background(), "prompt", ExecOptions{}); err == nil || !strings.Contains(err.Error(), "workspace") {
		t.Fatalf("workspace error = %v", err)
	}
}

func TestExecuteNIMToolConfinesPaths(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(root, "..", "outside.txt")
	call := nimToolCall{Function: nimFunctionCall{Name: "write_file", Arguments: `{"path":"../outside.txt","content":"bad"}`}}
	if got := executeNIMTool(root, call); !strings.Contains(got, "escapes workspace") {
		t.Fatalf("tool result = %q", got)
	}
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		t.Fatalf("outside path was written: %v", err)
	}
}

func TestExecuteNIMToolRejectsWorkspaceSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	call := nimToolCall{Function: nimFunctionCall{Name: "write_file", Arguments: `{"path":"linked/outside.txt","content":"bad"}`}}
	if got := executeNIMTool(root, call); !strings.Contains(got, "symlink") {
		t.Fatalf("tool result = %q", got)
	}
	if _, err := os.Stat(filepath.Join(outside, "outside.txt")); !os.IsNotExist(err) {
		t.Fatalf("outside path was written: %v", err)
	}
}

func TestNIMAPIErrorsAreReturnedWithoutLeakingKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"detail":"bad model"}`, http.StatusUnprocessableEntity)
	}))
	defer server.Close()
	backend := &nimBackend{cfg: Config{Env: map[string]string{"NVIDIA_API_KEY": "secret-key"}}, client: server.Client(), baseURL: server.URL}
	session, err := backend.Execute(context.Background(), "prompt", ExecOptions{Cwd: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	for range session.Messages {
	}
	result := <-session.Result
	if result.Status != "failed" || !strings.Contains(result.Error, "422") || strings.Contains(result.Error, "secret-key") {
		t.Fatalf("result = %#v", result)
	}
}
