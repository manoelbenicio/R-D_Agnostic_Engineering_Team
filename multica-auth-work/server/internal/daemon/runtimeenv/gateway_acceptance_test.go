package runtimeenv

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

const g4SyntheticGatewayValue = "synthetic-g4-gateway-value"

type syntheticProtocolResult struct {
	toolCall  bool
	reasoning bool
	input     int64
	output    int64
}

type syntheticGatewayMock struct {
	server *httptest.Server
	secret string

	mu             sync.Mutex
	toolRequests   map[string]int
	requestStarted chan string
	cancelObserved chan string
	forceStop      chan struct{}
}

func newSyntheticGatewayMock(t *testing.T) *syntheticGatewayMock {
	t.Helper()
	mock := &syntheticGatewayMock{
		secret:         g4SyntheticGatewayValue,
		toolRequests:   make(map[string]int),
		requestStarted: make(chan string, 4),
		cancelObserved: make(chan string, 4),
		forceStop:      make(chan struct{}),
	}
	mock.server = httptest.NewServer(http.HandlerFunc(mock.handle))
	t.Cleanup(func() {
		close(mock.forceStop)
		mock.server.Close()
	})
	return mock
}

func (m *syntheticGatewayMock) handle(w http.ResponseWriter, r *http.Request) {
	protocol := ""
	switch r.URL.Path {
	case "/v1/messages":
		protocol = "claude"
	case "/v1/responses":
		protocol = "codex"
	default:
		http.NotFound(w, r)
		return
	}
	if r.Header.Get("Authorization") != "Bearer "+m.secret {
		w.Header().Set("X-OmniRoute-Error-Class", "gateway-authentication")
		http.Error(w, "synthetic authentication rejected", http.StatusUnauthorized)
		return
	}
	if r.Header.Get("X-Task-ID") == "" || r.Header.Get("X-Session-ID") == "" || r.Header.Get("X-Request-ID") == "" {
		http.Error(w, "synthetic correlation rejected", http.StatusBadRequest)
		return
	}

	scenario := r.Header.Get("X-Synthetic-Scenario")
	if scenario == "cancel" {
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
		m.requestStarted <- protocol
		select {
		case <-r.Context().Done():
			m.cancelObserved <- protocol
		case <-m.forceStop:
		}
		return
	}
	if scenario == "error" {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-OmniRoute-Error-Class", "capability-rejected")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = io.WriteString(w, `{"error":{"code":"capability_rejected","type":"invalid_request_error"}}`)
		return
	}

	var request map[string]any
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid synthetic request", http.StatusBadRequest)
		return
	}
	if tools, ok := request["tools"].([]any); ok && len(tools) > 0 {
		m.mu.Lock()
		m.toolRequests[protocol]++
		m.mu.Unlock()
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	if protocol == "claude" {
		writeSyntheticSSE(w,
			`{"type":"message_start","message":{"usage":{"input_tokens":11}}}`,
			`{"type":"content_block_start","content_block":{"type":"thinking"}}`,
			`{"type":"content_block_start","content_block":{"type":"tool_use","id":"call_fixture","name":"fixture_tool"}}`,
			`{"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":7}}`,
			`{"type":"message_stop"}`,
		)
		return
	}
	writeSyntheticSSE(w,
		`{"type":"response.created","response":{"id":"resp_fixture"}}`,
		`{"type":"response.reasoning_summary_text.delta","delta":"fixture"}`,
		`{"type":"response.output_item.added","item":{"type":"function_call","call_id":"call_fixture","name":"fixture_tool"}}`,
		`{"type":"response.completed","response":{"usage":{"input_tokens":13,"output_tokens":9}}}`,
	)
}

func writeSyntheticSSE(w io.Writer, events ...string) {
	for _, event := range events {
		_, _ = io.WriteString(w, "data: "+event+"\n\n")
	}
}

func (m *syntheticGatewayMock) sawToolRequest(protocol string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.toolRequests[protocol] > 0
}

func TestG4ClaudeAndCodexTrustedGatewayProtocols(t *testing.T) {
	mock := newSyntheticGatewayMock(t)
	secret, err := NewStableSecret(g4SyntheticGatewayValue)
	if err != nil {
		t.Fatal("synthetic stable value was rejected")
	}

	tests := []struct {
		name          string
		cli           brain.CLIKind
		protocol      brain.ProtocolFamily
		model         string
		thinking      string
		path          string
		credentialKey string
	}{
		{
			name: "claude", cli: brain.CLIClaudeCode, protocol: brain.ProtocolAnthropicMessages,
			model: "anthropic/claude-g4-fixture", thinking: "enabled",
			path: "/v1/messages", credentialKey: "ANTHROPIC_AUTH_TOKEN",
		},
		{
			name: "codex", cli: brain.CLICodex, protocol: brain.ProtocolOpenAIResponses,
			model: "openai/codex-g4-fixture", thinking: "high",
			path: "/v1/responses", credentialKey: CodexOmniRouteAPIKeyEnv,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			executionRoot, taskHome, codexHome := controlledTestDirectories(t)
			model, err := brain.ParseRouteModel(test.model)
			if err != nil {
				t.Fatal("synthetic route model was rejected")
			}
			policy, err := NewGatewayModelPolicy([]ApprovedGatewayModel{{
				Model: model, Protocol: test.protocol, CLIs: []brain.CLIKind{test.cli}, ThinkingLevels: []string{test.thinking},
			}})
			if err != nil || policy.ValidateSelection(test.cli, model, test.thinking) != nil {
				t.Fatal("gateway-aware model/thinking validation failed")
			}

			environment, _, err := BuildGatewayEnvironment(ComposeOptions{
				Inherited: []string{"PATH=/usr/bin"},
				Adapter: AdapterEnvironment{
					CLI: test.cli, GatewayRoot: mock.server.URL, StableSecret: secret,
					TaskHome: taskHome, CodexHome: codexHome,
				},
			})
			if err != nil {
				t.Fatal("trusted gateway environment construction failed")
			}
			if err := AssertPreLaunch(LaunchPlan{
				Environment: environment, ExecutionRoot: executionRoot,
			}); test.cli == brain.CLIClaudeCode && err != nil {
				t.Fatal("Claude pre-launch assertion failed")
			}
			if test.cli == brain.CLICodex {
				config, err := NewCodexConfigContract(mock.server.URL, model, testCorrelation())
				if err != nil || config.Validate() != nil {
					t.Fatal("controlled Codex provider contract failed")
				}
				if err := AssertPreLaunch(LaunchPlan{
					Environment: environment, CodexConfig: &config, ExecutionRoot: executionRoot,
				}); err != nil {
					t.Fatal("Codex pre-launch assertion failed")
				}
			}

			credential, ok := environmentValue(environment, test.credentialKey)
			if !ok {
				t.Fatal("trusted gateway credential projection missing")
			}
			body, err := json.Marshal(map[string]any{
				"model":     test.model,
				"tools":     []any{map[string]any{"name": "fixture_tool"}},
				"reasoning": map[string]any{"effort": test.thinking},
			})
			if err != nil {
				t.Fatal("synthetic request construction failed")
			}
			req, err := http.NewRequest(http.MethodPost, mock.server.URL+test.path, bytes.NewReader(body))
			if err != nil {
				t.Fatal("synthetic request creation failed")
			}
			setSyntheticGatewayHeaders(req, credential)
			resp, err := mock.server.Client().Do(req)
			if err != nil {
				t.Fatal("synthetic gateway request failed")
			}
			result, err := parseSyntheticProtocol(resp.Body, test.name)
			_ = resp.Body.Close()
			if err != nil || resp.StatusCode != http.StatusOK {
				t.Fatal("synthetic gateway response was invalid")
			}
			if !result.toolCall || !result.reasoning || result.input == 0 || result.output == 0 || !mock.sawToolRequest(test.name) {
				t.Fatal("tools, reasoning, or usage semantics were not preserved")
			}
		})
	}
}

func TestG4ClaudeAndCodexCancellationAndDeterministicErrors(t *testing.T) {
	mock := newSyntheticGatewayMock(t)

	for _, test := range []struct {
		name string
		path string
	}{
		{name: "claude", path: "/v1/messages"},
		{name: "codex", path: "/v1/responses"},
	} {
		t.Run(test.name, func(t *testing.T) {
			firstStatus, firstClass, firstBody := requestSyntheticError(t, mock, test.path)
			secondStatus, secondClass, secondBody := requestSyntheticError(t, mock, test.path)
			if firstStatus != secondStatus || firstClass != secondClass || !bytes.Equal(firstBody, secondBody) {
				t.Fatal("synthetic error classification was not deterministic")
			}
			if firstStatus != http.StatusUnprocessableEntity || firstClass != "capability-rejected" {
				t.Fatal("synthetic error classification did not match the contract")
			}

			ctx, cancel := context.WithCancel(context.Background())
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, mock.server.URL+test.path, strings.NewReader(`{"tools":[]}`))
			if err != nil {
				t.Fatal("synthetic cancellation request creation failed")
			}
			setSyntheticGatewayHeaders(req, g4SyntheticGatewayValue)
			req.Header.Set("X-Synthetic-Scenario", "cancel")
			result := make(chan error, 1)
			go func() {
				resp, requestErr := mock.server.Client().Do(req)
				if resp != nil {
					_ = resp.Body.Close()
				}
				result <- requestErr
			}()
			select {
			case protocol := <-mock.requestStarted:
				if protocol != test.name {
					t.Fatal("synthetic cancellation reached the wrong protocol")
				}
			case <-time.After(2 * time.Second):
				t.Fatal("synthetic gateway did not observe request start")
			}
			cancel()
			select {
			case requestErr := <-result:
				if !errors.Is(requestErr, context.Canceled) {
					t.Fatal("client cancellation was not preserved")
				}
			case <-time.After(2 * time.Second):
				t.Fatal("synthetic client cancellation timed out")
			}
			select {
			case protocol := <-mock.cancelObserved:
				if protocol != test.name {
					t.Fatal("synthetic cancellation reached the wrong handler")
				}
			case <-time.After(2 * time.Second):
				t.Fatal("synthetic gateway did not observe cancellation")
			}
		})
	}
}

func requestSyntheticError(t *testing.T, mock *syntheticGatewayMock, path string) (int, string, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, mock.server.URL+path, strings.NewReader(`{"tools":[]}`))
	if err != nil {
		t.Fatal("synthetic error request creation failed")
	}
	setSyntheticGatewayHeaders(req, g4SyntheticGatewayValue)
	req.Header.Set("X-Synthetic-Scenario", "error")
	resp, err := mock.server.Client().Do(req)
	if err != nil {
		t.Fatal("synthetic deterministic error request failed")
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatal("synthetic deterministic error response read failed")
	}
	return resp.StatusCode, resp.Header.Get("X-OmniRoute-Error-Class"), body
}

func setSyntheticGatewayHeaders(req *http.Request, credential string) {
	req.Header.Set("Authorization", "Bearer "+credential)
	req.Header.Set("X-Task-ID", "task-g4-fixture")
	req.Header.Set("X-Session-ID", "session-g4-fixture")
	req.Header.Set("X-Request-ID", "request-g4-fixture")
}

func parseSyntheticProtocol(body io.Reader, protocol string) (syntheticProtocolResult, error) {
	result := syntheticProtocolResult{}
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &event); err != nil {
			return syntheticProtocolResult{}, err
		}
		eventType, _ := event["type"].(string)
		if protocol == "claude" {
			parseSyntheticClaudeEvent(eventType, event, &result)
		} else {
			parseSyntheticCodexEvent(eventType, event, &result)
		}
	}
	if err := scanner.Err(); err != nil {
		return syntheticProtocolResult{}, err
	}
	return result, nil
}

func parseSyntheticClaudeEvent(eventType string, event map[string]any, result *syntheticProtocolResult) {
	if eventType == "content_block_start" {
		block, _ := event["content_block"].(map[string]any)
		switch blockType, _ := block["type"].(string); blockType {
		case "thinking":
			result.reasoning = true
		case "tool_use":
			result.toolCall = true
		}
	}
	if message, ok := event["message"].(map[string]any); ok {
		result.input += syntheticUsageValue(message, "input_tokens")
	}
	result.output += syntheticUsageValue(event, "output_tokens")
}

func parseSyntheticCodexEvent(eventType string, event map[string]any, result *syntheticProtocolResult) {
	if eventType == "response.reasoning_summary_text.delta" {
		result.reasoning = true
	}
	if item, ok := event["item"].(map[string]any); ok && item["type"] == "function_call" {
		result.toolCall = true
	}
	if response, ok := event["response"].(map[string]any); ok {
		result.input += syntheticUsageValue(response, "input_tokens")
		result.output += syntheticUsageValue(response, "output_tokens")
	}
}

func syntheticUsageValue(container map[string]any, key string) int64 {
	usage, _ := container["usage"].(map[string]any)
	value, _ := usage[key].(float64)
	return int64(value)
}

func environmentValue(environment ChildEnvironment, key string) (string, bool) {
	for _, item := range environment.Exec() {
		candidate, value, ok := strings.Cut(item, "=")
		if ok && strings.EqualFold(candidate, key) {
			return value, true
		}
	}
	return "", false
}
