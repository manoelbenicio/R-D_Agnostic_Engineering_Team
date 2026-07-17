package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	nimDefaultBaseURL = "https://integrate.api.nvidia.com/v1"
	nimDefaultModel   = "z-ai/glm-5.2"
	nimMaxFileBytes   = 4 << 20
	nimDefaultTurns   = 24
)

type nimBackend struct {
	cfg     Config
	client  *http.Client
	baseURL string
}

type nimMessage struct {
	Role       string        `json:"role"`
	Content    string        `json:"content,omitempty"`
	ToolCalls  []nimToolCall `json:"tool_calls,omitempty"`
	ToolCallID string        `json:"tool_call_id,omitempty"`
}

type nimToolCall struct {
	Index    int             `json:"index,omitempty"`
	ID       string          `json:"id,omitempty"`
	Type     string          `json:"type,omitempty"`
	Function nimFunctionCall `json:"function"`
}

type nimFunctionCall struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type nimRequest struct {
	Model         string          `json:"model"`
	Messages      []nimMessage    `json:"messages"`
	Stream        bool            `json:"stream"`
	StreamOptions map[string]bool `json:"stream_options,omitempty"`
	Tools         []nimTool       `json:"tools"`
	ToolChoice    string          `json:"tool_choice"`
}

type nimTool struct {
	Type     string            `json:"type"`
	Function nimToolDefinition `json:"function"`
}

type nimToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type nimChunk struct {
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Content   string        `json:"content"`
			ToolCalls []nimToolCall `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
		PromptDetails    struct {
			CachedTokens int64 `json:"cached_tokens"`
		} `json:"prompt_tokens_details"`
	} `json:"usage"`
	UsageMetadata struct {
		PromptTokenCount        int64 `json:"promptTokenCount"`
		CandidatesTokenCount    int64 `json:"candidatesTokenCount"`
		CachedContentTokenCount int64 `json:"cachedContentTokenCount"`
		InputTokens             int64 `json:"inputTokens"`
		OutputTokens            int64 `json:"outputTokens"`
	} `json:"usageMetadata"`
}

type nimTurn struct {
	content   string
	toolCalls []nimToolCall
	usage     TokenUsage
	model     string
}

func (b *nimBackend) Execute(ctx context.Context, prompt string, opts ExecOptions) (*Session, error) {
	apiKey := b.env("NVIDIA_API_KEY")
	if apiKey == "" {
		return nil, errors.New("NVIDIA_API_KEY is required for the NIM backend")
	}
	if strings.TrimSpace(prompt) == "" {
		return nil, errors.New("NIM prompt must not be empty")
	}
	if opts.Cwd == "" {
		return nil, errors.New("NIM backend requires a workspace directory")
	}
	root, err := filepath.Abs(opts.Cwd)
	if err != nil {
		return nil, fmt.Errorf("resolve NIM workspace: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("NIM workspace is not a directory: %s", root)
	}

	runCtx, cancel := runContext(ctx, opts.Timeout)
	msgCh := make(chan Message, 256)
	resCh := make(chan Result, 1)
	go b.run(runCtx, cancel, prompt, opts, root, apiKey, msgCh, resCh)
	return &Session{Messages: msgCh, Result: resCh}, nil
}

func (b *nimBackend) run(ctx context.Context, cancel context.CancelFunc, prompt string, opts ExecOptions, root, apiKey string, msgCh chan Message, resCh chan Result) {
	defer cancel()
	defer close(msgCh)
	defer close(resCh)
	started := time.Now()
	model := opts.Model
	if model == "" {
		model = nimDefaultModel
	}
	messages := make([]nimMessage, 0, 8)
	if opts.SystemPrompt != "" {
		messages = append(messages, nimMessage{Role: "system", Content: opts.SystemPrompt})
	}
	messages = append(messages, nimMessage{Role: "user", Content: prompt})
	maxTurns := opts.MaxTurns
	if maxTurns <= 0 {
		maxTurns = nimDefaultTurns
	}
	var output strings.Builder
	var usage TokenUsage
	status, errText := "completed", ""

	for turnNo := 0; turnNo < maxTurns; turnNo++ {
		turn, err := b.streamTurn(ctx, model, messages, apiKey, msgCh, &output)
		if err != nil {
			status, errText = nimContextStatus(ctx, err)
			break
		}
		usage = addTokenUsage(usage, turn.usage)
		if turn.model != "" {
			model = turn.model
		}
		assistant := nimMessage{Role: "assistant", Content: turn.content, ToolCalls: turn.toolCalls}
		messages = append(messages, assistant)
		if len(turn.toolCalls) == 0 {
			break
		}
		for _, call := range turn.toolCalls {
			result := executeNIMTool(root, call)
			trySend(msgCh, Message{Type: MessageToolResult, Tool: call.Function.Name, CallID: call.ID, Output: result})
			messages = append(messages, nimMessage{Role: "tool", ToolCallID: call.ID, Content: result})
		}
		if turnNo == maxTurns-1 {
			status, errText = "failed", fmt.Sprintf("NIM exceeded maximum of %d agent turns", maxTurns)
		}
	}
	usageMap := map[string]TokenUsage(nil)
	if usage != (TokenUsage{}) {
		usageMap = map[string]TokenUsage{model: usage}
	}
	resCh <- Result{Status: status, Output: output.String(), Error: errText, DurationMs: time.Since(started).Milliseconds(), Usage: usageMap}
}

func (b *nimBackend) streamTurn(ctx context.Context, model string, messages []nimMessage, apiKey string, msgCh chan<- Message, output *strings.Builder) (nimTurn, error) {
	payload := nimRequest{Model: model, Messages: messages, Stream: true, StreamOptions: map[string]bool{"include_usage": true}, Tools: nimTools(), ToolChoice: "auto"}
	body, err := json.Marshal(payload)
	if err != nil {
		return nimTurn{}, fmt.Errorf("marshal NIM request: %w", err)
	}
	endpoint := strings.TrimRight(b.endpoint(), "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nimTurn{}, fmt.Errorf("create NIM request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	client := b.client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nimTurn{}, fmt.Errorf("NIM request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		return nimTurn{}, fmt.Errorf("NIM API returned %s: %s", resp.Status, strings.TrimSpace(string(detail)))
	}

	var turn nimTurn
	toolCalls := make(map[int]*nimToolCall)
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64<<10), 8<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") || !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var chunk nimChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return nimTurn{}, fmt.Errorf("decode NIM SSE chunk: %w", err)
		}
		if chunk.Model != "" {
			turn.model = chunk.Model
		}
		if chunkUsage := usageFromNIMChunk(chunk); chunkUsage != (TokenUsage{}) {
			turn.usage = chunkUsage
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				turn.content += choice.Delta.Content
				output.WriteString(choice.Delta.Content)
				trySend(msgCh, Message{Type: MessageText, Content: choice.Delta.Content})
			}
			for _, delta := range choice.Delta.ToolCalls {
				call := toolCalls[delta.Index]
				if call == nil {
					call = &nimToolCall{Index: delta.Index, Type: "function"}
					toolCalls[delta.Index] = call
				}
				call.ID += delta.ID
				call.Function.Name += delta.Function.Name
				call.Function.Arguments += delta.Function.Arguments
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nimTurn{}, fmt.Errorf("read NIM SSE stream: %w", err)
	}
	indices := make([]int, 0, len(toolCalls))
	for index := range toolCalls {
		indices = append(indices, index)
	}
	sort.Ints(indices)
	for _, index := range indices {
		if call := toolCalls[index]; call != nil {
			turn.toolCalls = append(turn.toolCalls, *call)
			var input map[string]any
			_ = json.Unmarshal([]byte(call.Function.Arguments), &input)
			trySend(msgCh, Message{Type: MessageToolUse, Tool: call.Function.Name, CallID: call.ID, Input: input})
		}
	}
	return turn, nil
}

func (b *nimBackend) endpoint() string {
	if b.baseURL != "" {
		return b.baseURL
	}
	if value := b.env("NIM_BASE_URL"); value != "" {
		return value
	}
	return nimDefaultBaseURL
}

func (b *nimBackend) env(key string) string {
	if value := b.cfg.Env[key]; value != "" {
		return value
	}
	return os.Getenv(key)
}

func nimTools() []nimTool {
	object := func(properties map[string]any, required ...string) map[string]any {
		return map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
	}
	path := map[string]any{"type": "string", "description": "Path relative to the workspace root"}
	return []nimTool{
		{Type: "function", Function: nimToolDefinition{Name: "read_file", Description: "Read a UTF-8 text file in the workspace", Parameters: object(map[string]any{"path": path}, "path")}},
		{Type: "function", Function: nimToolDefinition{Name: "write_file", Description: "Create or replace a UTF-8 text file in the workspace", Parameters: object(map[string]any{"path": path, "content": map[string]any{"type": "string"}}, "path", "content")}},
	}
}

func executeNIMTool(root string, call nimToolCall) string {
	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
		return "error: invalid tool arguments: " + err.Error()
	}
	path, err := confinedNIMPath(root, args.Path)
	if err != nil {
		return "error: " + err.Error()
	}
	switch call.Function.Name {
	case "read_file":
		file, err := os.Open(path)
		if err != nil {
			return "error: " + err.Error()
		}
		defer file.Close()
		data, err := io.ReadAll(io.LimitReader(file, nimMaxFileBytes+1))
		if err != nil {
			return "error: " + err.Error()
		}
		if len(data) > nimMaxFileBytes {
			return "error: file exceeds 4 MiB limit"
		}
		return string(data)
	case "write_file":
		if len(args.Content) > nimMaxFileBytes {
			return "error: content exceeds 4 MiB limit"
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return "error: " + err.Error()
		}
		if err := os.WriteFile(path, []byte(args.Content), 0o644); err != nil {
			return "error: " + err.Error()
		}
		return fmt.Sprintf("wrote %d bytes to %s", len(args.Content), args.Path)
	default:
		return "error: unsupported tool: " + call.Function.Name
	}
}

func confinedNIMPath(root, name string) (string, error) {
	if name == "" {
		return "", errors.New("path must not be empty")
	}
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return "", err
	}
	candidate := name
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(root, candidate)
	}
	candidate, err = filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, candidate)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("path escapes workspace")
	}
	// Refuse symlinked path components. This prevents a workspace symlink from
	// redirecting reads or writes outside the root between lexical checks.
	current := root
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if os.IsNotExist(statErr) {
			break
		}
		if statErr != nil {
			return "", statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("symlink paths are not allowed for NIM tools")
		}
	}
	return candidate, nil
}

func usageFromNIMChunk(chunk nimChunk) TokenUsage {
	input := chunk.UsageMetadata.PromptTokenCount
	if input == 0 {
		input = chunk.UsageMetadata.InputTokens
	}
	output := chunk.UsageMetadata.CandidatesTokenCount
	if output == 0 {
		output = chunk.UsageMetadata.OutputTokens
	}
	if input == 0 {
		input = chunk.Usage.PromptTokens
	}
	if output == 0 {
		output = chunk.Usage.CompletionTokens
	}
	cacheRead := chunk.UsageMetadata.CachedContentTokenCount
	if cacheRead == 0 {
		cacheRead = chunk.Usage.PromptDetails.CachedTokens
	}
	return TokenUsage{InputTokens: input, OutputTokens: output, CacheReadTokens: cacheRead}
}

func addTokenUsage(a, b TokenUsage) TokenUsage {
	return TokenUsage{InputTokens: a.InputTokens + b.InputTokens, OutputTokens: a.OutputTokens + b.OutputTokens, CacheReadTokens: a.CacheReadTokens + b.CacheReadTokens, CacheWriteTokens: a.CacheWriteTokens + b.CacheWriteTokens}
}

func nimContextStatus(ctx context.Context, err error) (string, string) {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return "timeout", err.Error()
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return "aborted", err.Error()
	}
	return "failed", err.Error()
}
