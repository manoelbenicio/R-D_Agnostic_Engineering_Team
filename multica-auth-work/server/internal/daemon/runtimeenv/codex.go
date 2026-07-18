package runtimeenv

import (
	"errors"
	"net/url"
	"strings"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
	"github.com/pelletier/go-toml/v2"
)

const (
	CodexOmniRouteProviderID = "omniroute"
	CodexResponsesWireAPI    = "responses"
)

var ErrCodexConfigContract = errors.New("Codex configuration violates the controlled OmniRoute provider contract")

type codexProviderDocument struct {
	Name               string            `toml:"name"`
	BaseURL            string            `toml:"base_url"`
	EnvKey             string            `toml:"env_key"`
	WireAPI            string            `toml:"wire_api"`
	SupportsWebSockets bool              `toml:"supports_websockets"`
	HTTPHeaders        map[string]string `toml:"http_headers"`
}

type codexProviderTables struct {
	OmniRoute codexProviderDocument `toml:"omniroute"`
}

type codexConfigDocument struct {
	Model          string              `toml:"model"`
	ModelProvider  string              `toml:"model_provider"`
	ModelProviders codexProviderTables `toml:"model_providers"`
}

// CodexConfigContract contains generated non-secret TOML. The stable key value
// is never embedded; Codex looks it up exclusively through EnvKey.
type CodexConfigContract struct {
	raw         []byte
	model       brain.RouteModel
	baseURL     string
	correlation brain.Correlation
}

func NewCodexConfigContract(gatewayRoot string, model brain.RouteModel, correlation brain.Correlation) (CodexConfigContract, error) {
	parsedModel, err := brain.ParseRouteModel(string(model))
	if err != nil {
		return CodexConfigContract{}, err
	}
	if err := correlation.Validate(); err != nil {
		return CodexConfigContract{}, err
	}
	for _, value := range []string{correlation.TaskID, correlation.SessionID, correlation.RequestID} {
		if !validHeaderValue(value) {
			return CodexConfigContract{}, ErrCodexConfigContract
		}
	}
	root, err := normalizeGatewayRoot(gatewayRoot)
	if err != nil {
		return CodexConfigContract{}, err
	}
	baseURL := root + "/v1"
	document := codexConfigDocument{
		Model:         string(parsedModel),
		ModelProvider: CodexOmniRouteProviderID,
		ModelProviders: codexProviderTables{OmniRoute: codexProviderDocument{
			Name:               "OmniRoute",
			BaseURL:            baseURL,
			EnvKey:             CodexOmniRouteAPIKeyEnv,
			WireAPI:            CodexResponsesWireAPI,
			SupportsWebSockets: false,
			HTTPHeaders: map[string]string{
				"X-Task-Id":    correlation.TaskID,
				"X-Session-Id": correlation.SessionID,
				"X-Request-Id": correlation.RequestID,
			},
		}},
	}
	raw, err := toml.Marshal(document)
	if err != nil {
		return CodexConfigContract{}, ErrCodexConfigContract
	}
	contract := CodexConfigContract{raw: raw, model: parsedModel, baseURL: baseURL, correlation: correlation}
	if err := contract.Validate(); err != nil {
		return CodexConfigContract{}, err
	}
	return contract, nil
}

func (c CodexConfigContract) Bytes() []byte {
	return append([]byte(nil), c.raw...)
}

func (c CodexConfigContract) Validate() error {
	return ValidateCodexConfigBytes(c.raw, c.model, c.baseURL, c.correlation)
}

// ValidateCodexConfigBytes performs structural validation without returning a
// TOML parser error that could echo configuration content.
func ValidateCodexConfigBytes(raw []byte, model brain.RouteModel, baseURL string, correlation brain.Correlation) error {
	if len(raw) == 0 || len(raw) > 64<<10 {
		return ErrCodexConfigContract
	}
	var document codexConfigDocument
	if err := toml.Unmarshal(raw, &document); err != nil {
		return ErrCodexConfigContract
	}
	provider := document.ModelProviders.OmniRoute
	if document.Model != string(model) || document.ModelProvider != CodexOmniRouteProviderID ||
		provider.Name != "OmniRoute" || provider.BaseURL != baseURL ||
		provider.EnvKey != CodexOmniRouteAPIKeyEnv || provider.WireAPI != CodexResponsesWireAPI ||
		provider.SupportsWebSockets {
		return ErrCodexConfigContract
	}
	expectedHeaders := map[string]string{
		"X-Task-Id": correlation.TaskID, "X-Session-Id": correlation.SessionID, "X-Request-Id": correlation.RequestID,
	}
	if !equalStringMap(provider.HTTPHeaders, expectedHeaders) {
		return ErrCodexConfigContract
	}
	var generic map[string]any
	if err := toml.Unmarshal(raw, &generic); err != nil {
		return ErrCodexConfigContract
	}
	if !exactMapKeys(generic, "model", "model_provider", "model_providers") {
		return ErrCodexConfigContract
	}
	providers, ok := generic["model_providers"].(map[string]any)
	if !ok || !exactMapKeys(providers, CodexOmniRouteProviderID) {
		return ErrCodexConfigContract
	}
	providerMap, ok := providers[CodexOmniRouteProviderID].(map[string]any)
	if !ok || !exactMapKeys(providerMap, "name", "base_url", "env_key", "wire_api", "supports_websockets", "http_headers") {
		return ErrCodexConfigContract
	}
	parsed, err := url.Parse(provider.BaseURL)
	if err != nil || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "/v1" {
		return ErrCodexConfigContract
	}
	return nil
}

func validHeaderValue(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && len(value) <= 256 && !strings.ContainsAny(value, "\x00\r\n")
}

func equalStringMap(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

func exactMapKeys(values map[string]any, expected ...string) bool {
	if len(values) != len(expected) {
		return false
	}
	for _, key := range expected {
		if _, ok := values[key]; !ok {
			return false
		}
	}
	return true
}
