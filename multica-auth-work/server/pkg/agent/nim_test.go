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

	backend := &nimBackend{cfg: Config{Env: map[string]string{"OMNIROUTE_API_KEY": "test-key"}}, client: server.Client(), baseURL: server.URL + "/v1"}
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
		cfg:     Config{Env: map[string]string{"OMNIROUTE_API_KEY": "test-key"}},
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

func TestNIMExecuteRequiresOmniRouteCredentialAndWorkspace(t *testing.T) {
	backend := &nimBackend{cfg: Config{Env: map[string]string{"OMNIROUTE_API_KEY": ""}}}
	t.Setenv("OMNIROUTE_API_KEY", "")
	if _, err := backend.Execute(context.Background(), "prompt", ExecOptions{Cwd: t.TempDir()}); err == nil || !strings.Contains(err.Error(), "OMNIROUTE_API_KEY") {
		t.Fatalf("credential error = %v", err)
	}
	backend.cfg.Env["OMNIROUTE_API_KEY"] = "key"
	// With credential but missing endpoint (nimDefaultBaseURL is empty):
	// the fail-closed endpoint guard triggers before the workspace check
	// only if Cwd is set; test workspace-missing first with baseURL injected.
	backend.baseURL = "http://gateway.test"
	if _, err := backend.Execute(context.Background(), "prompt", ExecOptions{}); err == nil || !strings.Contains(err.Error(), "workspace") {
		t.Fatalf("workspace error = %v", err)
	}
}

func TestNIMExecuteRejectsNvidiaAPIKey(t *testing.T) {
	// NIM must NEVER read NVIDIA_API_KEY. Even if it is set, the adapter
	// must require OMNIROUTE_API_KEY and fail closed without it.
	backend := &nimBackend{cfg: Config{Env: map[string]string{"NVIDIA_API_KEY": "leaked-key"}}}
	t.Setenv("NVIDIA_API_KEY", "leaked-key")
	t.Setenv("OMNIROUTE_API_KEY", "")
	_, err := backend.Execute(context.Background(), "prompt", ExecOptions{Cwd: t.TempDir()})
	if err == nil {
		t.Fatal("expected error when only NVIDIA_API_KEY is set, got nil")
	}
	if !strings.Contains(err.Error(), "OMNIROUTE_API_KEY") {
		t.Fatalf("error should mention OMNIROUTE_API_KEY, got: %v", err)
	}
}

func TestNIMExecuteFailsClosedWithoutOmniRouteBaseURL(t *testing.T) {
	// When no OmniRoute base URL is configured (no struct field, no env var),
	// Execute must fail closed — no silent fallback to a direct provider.
	backend := &nimBackend{cfg: Config{Env: map[string]string{"OMNIROUTE_API_KEY": "key"}}}
	t.Setenv("OMNIROUTE_BASE_URL", "")
	_, err := backend.Execute(context.Background(), "prompt", ExecOptions{Cwd: t.TempDir()})
	if err == nil {
		t.Fatal("expected error when OMNIROUTE_BASE_URL is empty, got nil")
	}
	if !strings.Contains(err.Error(), "OMNIROUTE_BASE_URL") {
		t.Fatalf("error should mention OMNIROUTE_BASE_URL, got: %v", err)
	}
}

func TestNIMEndpointResolvesOmniRouteBaseURL(t *testing.T) {
	// 1. Struct field takes priority.
	b := &nimBackend{baseURL: "http://struct.test/v1"}
	if got := b.endpoint(); got != "http://struct.test/v1" {
		t.Errorf("struct baseURL: got %q", got)
	}
	// 2. OMNIROUTE_BASE_URL env var via Config.Env.
	b = &nimBackend{cfg: Config{Env: map[string]string{"OMNIROUTE_BASE_URL": "http://envmap.test/v1"}}}
	if got := b.endpoint(); got != "http://envmap.test/v1" {
		t.Errorf("Config.Env OMNIROUTE_BASE_URL: got %q", got)
	}
	// 3. OMNIROUTE_BASE_URL from os.Getenv.
	t.Setenv("OMNIROUTE_BASE_URL", "http://osenv.test/v1")
	b = &nimBackend{}
	if got := b.endpoint(); got != "http://osenv.test/v1" {
		t.Errorf("os.Getenv OMNIROUTE_BASE_URL: got %q", got)
	}
	// 4. Empty when nothing is configured (no direct-provider default).
	t.Setenv("OMNIROUTE_BASE_URL", "")
	b = &nimBackend{}
	if got := b.endpoint(); got != "" {
		t.Errorf("expected empty endpoint when unconfigured, got %q", got)
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
	backend := &nimBackend{cfg: Config{Env: map[string]string{"OMNIROUTE_API_KEY": "secret-key"}}, client: server.Client(), baseURL: server.URL}
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

// --- Brain 8.6 adapter-boundary coverage (cancellation propagation + no replay) ---
//
// These exercise the NIM adapter's half of OpenSpec task 8.6 at the CLI/backend
// boundary, disjoint from the gateway Coordinator retry/replay/dedup tests:
//   - cancellation mid-stream propagates to a single terminal "aborted" Result
//     with partial output preserved and NO re-issued upstream request;
//   - a committed-partial turn that then breaks is surfaced terminally and is
//     NOT replayed (the adapter never re-sends a turn after partial output).
// NIM is chosen because it is fully synthetic (httptest, no spawned CLI) and is
// the only owned adapter with a multi-turn upstream loop where "no replay of a
// committed turn" is a testable property. Claude/Codex cancellation is already
// covered (claude_deadlock_test.go, codex_test.go:933/1660) and is not duplicated.

func TestNIMCancellationMidStreamAbortsOnceWithoutReplay(t *testing.T) {
	var requests atomic.Int32
	root := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// Emit one partial text delta and flush so the client observes output,
		// then block until the client cancels — no finish_reason, no [DONE].
		fmt.Fprintln(w, `data: {"model":"test/model","choices":[{"delta":{"content":"partial-"}}]}`)
		if fl, ok := w.(http.Flusher); ok {
			fl.Flush()
		}
		<-r.Context().Done()
	}))
	defer server.Close()

	backend := &nimBackend{
		cfg:     Config{Env: map[string]string{"OMNIROUTE_API_KEY": "test-key"}},
		client:  server.Client(),
		baseURL: server.URL,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session, err := backend.Execute(ctx, "prompt", ExecOptions{Cwd: root, Model: "test/model"})
	if err != nil {
		t.Fatal(err)
	}

	gotPartial := false
	for msg := range session.Messages {
		if msg.Type == MessageText && strings.Contains(msg.Content, "partial-") {
			gotPartial = true
			cancel() // cancel AFTER partial output is delivered
		}
	}
	result := <-session.Result

	if !gotPartial {
		t.Fatal("did not observe partial output before cancellation")
	}
	if result.Status != "aborted" {
		t.Fatalf("status = %q, want aborted (cancellation must propagate to a terminal aborted result)", result.Status)
	}
	if !strings.Contains(result.Output, "partial-") {
		t.Fatalf("partial output not preserved on cancel: %q", result.Output)
	}
	if n := requests.Load(); n != 1 {
		t.Fatalf("upstream requests = %d, want 1 (a cancelled turn must not be replayed)", n)
	}
}

func TestNIMBrokenStreamAfterPartialOutputIsNotReplayed(t *testing.T) {
	var requests atomic.Int32
	root := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		// Commit partial output, then break the stream mid-turn with a malformed
		// SSE chunk. The adapter must surface this terminally, NOT re-issue the turn.
		fmt.Fprintln(w, `data: {"model":"test/model","choices":[{"delta":{"content":"partial-only"}}]}`)
		fmt.Fprintln(w, `data: {not-valid-json`)
	}))
	defer server.Close()

	backend := &nimBackend{
		cfg:     Config{Env: map[string]string{"OMNIROUTE_API_KEY": "test-key"}},
		client:  server.Client(),
		baseURL: server.URL,
	}
	session, err := backend.Execute(context.Background(), "prompt", ExecOptions{Cwd: root, Model: "test/model"})
	if err != nil {
		t.Fatal(err)
	}
	for range session.Messages {
	}
	result := <-session.Result

	if result.Status != "failed" {
		t.Fatalf("status = %q, want failed (a broken stream after partial output is terminal)", result.Status)
	}
	if !strings.Contains(result.Output, "partial-only") {
		t.Fatalf("partial output not preserved after mid-stream break: %q", result.Output)
	}
	if n := requests.Load(); n != 1 {
		t.Fatalf("upstream requests = %d, want 1 (a committed-partial turn must not be replayed)", n)
	}
}
