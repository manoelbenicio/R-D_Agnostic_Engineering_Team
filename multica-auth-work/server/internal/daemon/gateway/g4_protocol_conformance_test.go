package gateway

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func syntheticResponse(request *http.Request, status int, contentType, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{contentType}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    request,
	}
}

type protocolConformanceCase struct {
	name           string
	protocol       brain.ProtocolFamily
	cli            brain.CLIKind
	model          brain.RouteModel
	profile        ProfileID
	endpoint       string
	nonStreamBody  string
	streamBody     string
	nonStreamShape string
	streamEvents   []string
}

func anthropicMessagesConformanceCase(name string, model brain.RouteModel) protocolConformanceCase {
	return protocolConformanceCase{
		name: name, protocol: brain.ProtocolAnthropicMessages, cli: brain.CLIClaudeCode,
		model: model, profile: ProfileAnthropicMessages, endpoint: "/v1/messages", nonStreamShape: "message",
		nonStreamBody: `{"id":"synthetic-message","type":"message","role":"assistant","content":[{"type":"text","text":"synthetic response"}],"stop_reason":"end_turn","usage":{"input_tokens":4,"output_tokens":2}}`,
		streamBody: "event: message_start\ndata: {\"type\":\"message_start\"}\n\n" +
			"event: content_block_start\ndata: {\"type\":\"content_block_start\"}\n\n" +
			"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"synthetic response\"}}\n\n" +
			"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n",
		streamEvents: []string{"message_start", "content_block_start", "content_block_delta", "message_stop"},
	}
}

func protocolConformanceCases() []protocolConformanceCase {
	return []protocolConformanceCase{
		anthropicMessagesConformanceCase("anthropic_messages", "synthetic/anthropic-conformance"),
		anthropicMessagesConformanceCase("anthropic_messages_agy_claude_opus_4_6_thinking", "agy/claude-opus-4-6-thinking"),
		{
			name: "openai_responses", protocol: brain.ProtocolOpenAIResponses, cli: brain.CLICodex,
			model: "synthetic/responses-conformance", profile: ProfileOpenAIResponses, endpoint: "/v1/responses", nonStreamShape: "response",
			nonStreamBody: `{"id":"synthetic-response","object":"response","status":"completed","output":[{"type":"message","id":"synthetic-item","content":[{"type":"output_text","text":"synthetic response"}]}],"usage":{"input_tokens":4,"output_tokens":2,"total_tokens":6}}`,
			streamBody: "event: response.created\ndata: {\"type\":\"response.created\"}\n\n" +
				"event: response.output_item.added\ndata: {\"type\":\"response.output_item.added\"}\n\n" +
				"event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"synthetic response\"}\n\n" +
				"event: response.completed\ndata: {\"type\":\"response.completed\"}\n\n",
			streamEvents: []string{"response.created", "response.output_item.added", "response.output_text.delta", "response.completed"},
		},
		{
			name: "openai_chat", protocol: brain.ProtocolOpenAIChat, cli: brain.CLIOpenAICompatible,
			model: "synthetic/chat-conformance", profile: ProfileOpenAIChat, endpoint: "/v1/chat/completions", nonStreamShape: "chat.completion",
			nonStreamBody: `{"id":"synthetic-chat","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"synthetic response"},"finish_reason":"stop"}],"usage":{"prompt_tokens":4,"completion_tokens":2,"total_tokens":6}}`,
			streamBody: "data: {\"id\":\"synthetic-chat\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"synthetic response\"}}]}\n\n" +
				"data: [DONE]\n\n",
			streamEvents: []string{"chat.completion.chunk", "[DONE]"},
		},
		{
			name: "antigravity_compatible", protocol: brain.ProtocolAntigravity, cli: brain.CLIAntigravity,
			model: "synthetic/antigravity-conformance", profile: ProfileAntigravity, endpoint: "/v1/antigravity", nonStreamShape: "antigravity.response",
			nonStreamBody: `{"id":"synthetic-antigravity","object":"antigravity.response","status":"completed","output":"synthetic response","usage":{"input":4,"output":2,"total":6}}`,
			streamBody: "event: antigravity.response.start\ndata: {\"type\":\"antigravity.response.start\"}\n\n" +
				"event: antigravity.response.delta\ndata: {\"type\":\"antigravity.response.delta\",\"delta\":\"synthetic response\"}\n\n" +
				"event: antigravity.response.completed\ndata: {\"type\":\"antigravity.response.completed\"}\n\n",
			streamEvents: []string{"antigravity.response.start", "antigravity.response.delta", "antigravity.response.completed"},
		},
	}
}

func TestG4ProtocolFamiliesNonStreamingAndStreaming(t *testing.T) {
	for _, test := range protocolConformanceCases() {
		t.Run(test.name, func(t *testing.T) {
			profile, err := LookupRuntimeProfile(test.protocol, test.cli)
			if err != nil {
				t.Fatalf("LookupRuntimeProfile: %v", err)
			}
			if profile.ID != test.profile || profile.Endpoint != test.endpoint {
				t.Fatalf("unexpected runtime profile: id=%q endpoint=%q", profile.ID, profile.Endpoint)
			}
			transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
				if request.Method != http.MethodPost || request.URL.Path != test.endpoint || request.Header.Get("Content-Type") != "application/json" {
					return nil, fmt.Errorf("synthetic protocol request contract mismatch")
				}
				body, readErr := io.ReadAll(io.LimitReader(request.Body, 4097))
				if readErr != nil || len(body) == 0 || len(body) > 4096 {
					return nil, fmt.Errorf("synthetic request body invalid")
				}
				var document struct {
					Model  string `json:"model"`
					Stream bool   `json:"stream"`
				}
				if json.Unmarshal(body, &document) != nil || document.Model != string(test.model) {
					return nil, fmt.Errorf("synthetic model contract mismatch")
				}
				if document.Stream {
					return syntheticResponse(request, http.StatusOK, "text/event-stream", test.streamBody), nil
				}
				return syntheticResponse(request, http.StatusOK, "application/json", test.nonStreamBody), nil
			})
			client := &http.Client{Transport: transport, Timeout: time.Second}

			nonStreamResponse := executeSyntheticProtocolRequest(t, client, profile, test.model, false)
			var nonStream struct {
				Object string `json:"object"`
				Type   string `json:"type"`
			}
			decodeBoundedJSON(t, nonStreamResponse.Body, &nonStream)
			_ = nonStreamResponse.Body.Close()
			shape := nonStream.Object
			if shape == "" {
				shape = nonStream.Type
			}
			if shape != test.nonStreamShape {
				t.Fatalf("unexpected non-stream shape %q", shape)
			}

			streamResponse := executeSyntheticProtocolRequest(t, client, profile, test.model, true)
			events, err := parseSyntheticSSE(streamResponse.Body, test.protocol)
			_ = streamResponse.Body.Close()
			if err != nil {
				t.Fatalf("parseSyntheticSSE: %v", err)
			}
			if strings.Join(events, ",") != strings.Join(test.streamEvents, ",") {
				t.Fatalf("unexpected stream events: %v", events)
			}
			if test.protocol != brain.ProtocolAntigravity {
				if err := ValidateSSESequence(test.protocol, events); err != nil {
					t.Fatalf("ValidateSSESequence: %v", err)
				}
			}
		})
	}
}

func TestG4InitialModelSetRouteConformanceDriftGuard(t *testing.T) {
	type routeKey struct {
		model    brain.RouteModel
		cli      brain.CLIKind
		protocol brain.ProtocolFamily
	}

	covered := make(map[routeKey]struct{}, len(protocolConformanceCases()))
	for _, protocolCase := range protocolConformanceCases() {
		covered[routeKey{model: protocolCase.model, cli: protocolCase.cli, protocol: protocolCase.protocol}] = struct{}{}
	}
	for _, route := range brain.InitialModelSet() {
		key := routeKey{model: route.Model, cli: route.CLI, protocol: route.Protocol}
		if _, ok := covered[key]; !ok {
			t.Errorf("initial route lacks synthetic conformance coverage: model=%q cli=%q protocol=%q", route.Model, route.CLI, route.Protocol)
		}
	}
}

func TestG4SSEContractRejectsEmptyAndEarlyTerminal(t *testing.T) {
	for protocol, contract := range ProtocolSSEContracts() {
		if err := ValidateSSESequence(protocol, nil); !IsErrorClass(err, ErrorProtocol) {
			t.Fatalf("known protocol %s did not classify an empty stream as malformed", protocol)
		}
		events := append([]string(nil), contract.RequiredPrefix...)
		events = append(events, contract.TerminalEvents[0], contract.TerminalEvents[0])
		if err := ValidateSSESequence(protocol, events); !IsErrorClass(err, ErrorProtocol) {
			t.Fatalf("known protocol %s accepted an early terminal event", protocol)
		}
	}
	if err := ValidateSSESequence(brain.ProtocolFamily("synthetic-unknown"), nil); !IsErrorClass(err, ErrorCapability) {
		t.Fatal("unknown protocol did not retain capability-error classification")
	}
}

func executeSyntheticProtocolRequest(t *testing.T, client *http.Client, profile RuntimeProfile, model brain.RouteModel, stream bool) *http.Response {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"model":  model,
		"stream": stream,
		"input":  "synthetic protocol input",
	})
	if err != nil {
		t.Fatalf("Marshal synthetic request: %v", err)
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://synthetic.invalid"+profile.Endpoint, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(HeaderTaskID, "task-synthetic-g4")
	request.Header.Set(HeaderSessionID, "session-synthetic-g4")
	request.Header.Set(HeaderRequestID, "request-synthetic-g4")
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("synthetic protocol request failed: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		_ = response.Body.Close()
		t.Fatalf("unexpected synthetic status %d", response.StatusCode)
	}
	return response
}

func decodeBoundedJSON(t *testing.T, reader io.Reader, target any) {
	t.Helper()
	body, err := io.ReadAll(io.LimitReader(reader, 8193))
	if err != nil || len(body) > 8192 {
		t.Fatal("synthetic response exceeded bound")
	}
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("decode synthetic response: %v", err)
	}
}

func parseSyntheticSSE(reader io.Reader, protocol brain.ProtocolFamily) ([]string, error) {
	var events []string
	scanner := bufio.NewScanner(io.LimitReader(reader, 64<<10))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			events = append(events, strings.TrimPrefix(line, "event: "))
			continue
		}
		if protocol == brain.ProtocolOpenAIChat && strings.HasPrefix(line, "data: ") {
			payload := strings.TrimPrefix(line, "data: ")
			if payload == "[DONE]" {
				events = append(events, payload)
				continue
			}
			var chunk struct {
				Object string `json:"object"`
			}
			if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
				return nil, err
			}
			events = append(events, chunk.Object)
		}
	}
	return events, scanner.Err()
}

func TestG4AuthenticatedSyntheticModelsAndCapabilities(t *testing.T) {
	models := make([]ModelDocument, 0, len(protocolConformanceCases()))
	for _, protocolCase := range protocolConformanceCases() {
		models = append(models, ModelDocument{
			ID: string(protocolCase.model), Protocol: string(protocolCase.protocol),
			Streaming: boolPointer(true), Tools: boolPointer(true), Reasoning: boolPointer(true), StructuredOutput: boolPointer(true),
			ContextLimit: 32768, AccountPool: "pool-synthetic-g4", Rotation: string(RotationStrictIndependentRequest),
			Affinity: string(AffinityOriginAccount), Available: boolPointer(true),
		})
	}
	documentBody, err := json.Marshal(ModelsDocument{Object: "list", RegistryVersion: "synthetic-g4-v1", Models: models})
	if err != nil {
		t.Fatalf("Marshal models document: %v", err)
	}
	var pathsMu sync.Mutex
	var paths []string
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		pathsMu.Lock()
		paths = append(paths, request.URL.Path)
		pathsMu.Unlock()
		if request.URL.Path == "/health/live" {
			if request.Header.Get("Authorization") != "" {
				return nil, fmt.Errorf("synthetic liveness unexpectedly authenticated")
			}
			return syntheticResponse(request, http.StatusOK, "application/json", `{"status":"synthetic-live"}`), nil
		}
		if request.Header.Get("Authorization") != "Bearer "+syntheticCredentialValue {
			return syntheticResponse(request, http.StatusUnauthorized, "application/json", `{}`), nil
		}
		switch request.URL.Path {
		case "/health/ready":
			return syntheticResponse(request, http.StatusOK, "application/json", `{"status":"synthetic-ready"}`), nil
		case "/v1/models":
			response := syntheticResponse(request, http.StatusOK, "application/json", string(documentBody))
			response.Header.Set(HeaderRegistryVersion, "synthetic-g4-v1")
			return response, nil
		default:
			return syntheticResponse(request, http.StatusNotFound, "application/json", `{}`), nil
		}
	})
	credential := &syntheticCredentialSource{}
	client, err := NewClient(ClientOptions{
		Gateway:        testGatewayConfig(t, "http://synthetic.invalid"),
		Endpoints:      EndpointSet{Liveness: "/health/live", Readiness: "/health/ready"},
		Credential:     credential,
		HTTPClient:     &http.Client{Transport: transport},
		RequestTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := client.CheckLiveness(context.Background(), testCorrelation()); err != nil {
		t.Fatalf("CheckLiveness: %v", err)
	}
	if _, err := client.CheckReadiness(context.Background(), testCorrelation()); err != nil {
		t.Fatalf("CheckReadiness: %v", err)
	}
	registry, err := NewRegistry(ModelsFetchFunc(func(ctx context.Context) (ModelsDocument, error) {
		return client.FetchModels(ctx, testCorrelation())
	}), time.Minute)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	for _, protocolCase := range protocolConformanceCases() {
		err := registry.ValidateCapability(context.Background(), protocolCase.model, CapabilityRequirement{
			Protocol: protocolCase.protocol, Streaming: true, Tools: true, Reasoning: true, StructuredOutput: true, MinimumContext: 16384,
		})
		if err != nil {
			t.Fatalf("ValidateCapability(%s): %v", protocolCase.protocol, err)
		}
	}
	if credential.calls.Load() != 2 {
		t.Fatalf("expected two authenticated operations, got %d", credential.calls.Load())
	}
	pathsMu.Lock()
	sort.Strings(paths)
	gotPaths := strings.Join(paths, ",")
	pathsMu.Unlock()
	if gotPaths != "/health/live,/health/ready,/v1/models" {
		t.Fatalf("unexpected synthetic paths %q", gotPaths)
	}
}
