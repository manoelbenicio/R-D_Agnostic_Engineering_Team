package runtimeenv

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

// Cline 3.0.44 stores custom OpenAI-compatible providers in the carrier file
// ~/.cline/data/settings/providers.json. The exact schema was captured from the
// installed CLI (`cline auth openai-compatible`): a top-level
// {version, lastUsedProvider, providers}, where providers[<id>] =
// {settings:{provider, apiKey, model, baseUrl}, updatedAt, tokenSource}.
//
// Unlike the Codex custom provider (which references the stable OmniRoute secret
// through an env_key and never embeds it), Cline holds the credential INLINE in
// settings.apiKey. To keep the single stable OmniRoute secret out of the
// generated configuration entirely, this contract writes a non-secret reference
// sentinel into apiKey; the central launch injector (W1) substitutes the value
// resolved from the single OmniRoute secret file at spawn time. This package
// never reads, embeds, hashes, or logs a secret value.
//
// This contract serves the Agent Brain-selectable Cline routes only
// (Cline -> Kimi-K2.7 and Cline -> GLM-5.2). NVIDIA is an OmniRoute-owned
// fallback and is never Agent Brain-selected; NVIDIA-namespaced routes are
// rejected here.
const (
	// ClineOpenAICompatibleProviderID is the exact provider key and
	// settings.provider discriminator Cline uses for a custom OpenAI-compatible
	// endpoint.
	ClineOpenAICompatibleProviderID = "openai-compatible"
	// ClineOmniRouteAPIKeyEnv names the environment variable that carries the
	// single stable OmniRoute inference secret at launch. The value is never
	// embedded by this package.
	ClineOmniRouteAPIKeyEnv = "CLINE_OMNIROUTE_API_KEY"
	// ClineSecretReferenceSentinel is the non-secret placeholder written into
	// settings.apiKey. The central launch injector replaces it with the value
	// resolved from the single OmniRoute secret file. It is not a credential.
	ClineSecretReferenceSentinel = "${" + ClineOmniRouteAPIKeyEnv + "}"
	// ClineTokenSourceOmniRoute marks the provider entry as OmniRoute-injected.
	ClineTokenSourceOmniRoute = "omniroute-stable-secret"
	// clineProvidersSchemaVersion mirrors the installed Cline providers.json
	// schema version.
	clineProvidersSchemaVersion = 1
	// clineNVIDIARouteNamespace is the OmniRoute-owned fallback namespace the
	// Agent Brain must never select directly for a Cline task.
	clineNVIDIARouteNamespace = "nvidia"
)

// ErrClineConfigContract is returned when a generated or supplied Cline
// configuration violates the controlled OmniRoute provider contract. It never
// wraps a JSON parser error that could echo configuration content.
var ErrClineConfigContract = errors.New("Cline configuration violates the controlled OmniRoute provider contract")

// ErrClineRouteNotAgentBrainSelectable rejects OmniRoute-owned fallback routes
// (e.g. the NVIDIA GLM-5.2 fallback) that the Agent Brain must never select
// directly for a Cline task; NVIDIA is reached only as an OmniRoute-internal
// fallback.
var ErrClineRouteNotAgentBrainSelectable = errors.New("route is an OmniRoute-owned fallback and is not Agent Brain-selectable for Cline")

type clineProviderSettings struct {
	Provider string `json:"provider"`
	APIKey   string `json:"apiKey"`
	Model    string `json:"model"`
	BaseURL  string `json:"baseUrl"`
}

type clineProviderEntry struct {
	Settings    clineProviderSettings `json:"settings"`
	UpdatedAt   string                `json:"updatedAt"`
	TokenSource string                `json:"tokenSource"`
}

type clineProvidersDocument struct {
	Version          int                           `json:"version"`
	LastUsedProvider string                        `json:"lastUsedProvider"`
	Providers        map[string]clineProviderEntry `json:"providers"`
}

// ClineConfigContract contains generated non-secret JSON for the Cline
// providers.json carrier. The stable OmniRoute key value is never embedded;
// settings.apiKey holds ClineSecretReferenceSentinel until the central launch
// injector substitutes the resolved value.
type ClineConfigContract struct {
	raw       []byte
	model     brain.RouteModel
	baseURL   string
	updatedAt string
}

// NewClineConfigContract builds the controlled Cline OpenAI-compatible provider
// document that targets the OmniRoute Chat Completions surface at gatewayRoot
// for the supplied Agent Brain route model (the Cline Kimi-K2.7 or GLM-5.2
// routes). NVIDIA-namespaced routes are rejected. No secret value is read or
// embedded; settings.apiKey carries only the reference sentinel.
func NewClineConfigContract(gatewayRoot string, model brain.RouteModel, updatedAt string) (ClineConfigContract, error) {
	parsedModel, err := brain.ParseRouteModel(string(model))
	if err != nil {
		return ClineConfigContract{}, err
	}
	if isNVIDIAOwnedRoute(parsedModel) {
		return ClineConfigContract{}, ErrClineRouteNotAgentBrainSelectable
	}
	if !validHeaderValue(updatedAt) {
		return ClineConfigContract{}, ErrClineConfigContract
	}
	root, err := normalizeGatewayRoot(gatewayRoot)
	if err != nil {
		return ClineConfigContract{}, err
	}
	baseURL := root + "/v1"
	document := clineProvidersDocument{
		Version:          clineProvidersSchemaVersion,
		LastUsedProvider: ClineOpenAICompatibleProviderID,
		Providers: map[string]clineProviderEntry{
			ClineOpenAICompatibleProviderID: {
				Settings: clineProviderSettings{
					Provider: ClineOpenAICompatibleProviderID,
					APIKey:   ClineSecretReferenceSentinel,
					Model:    string(parsedModel),
					BaseURL:  baseURL,
				},
				UpdatedAt:   updatedAt,
				TokenSource: ClineTokenSourceOmniRoute,
			},
		},
	}
	raw, err := json.Marshal(document)
	if err != nil {
		return ClineConfigContract{}, ErrClineConfigContract
	}
	contract := ClineConfigContract{raw: raw, model: parsedModel, baseURL: baseURL, updatedAt: updatedAt}
	if err := contract.Validate(); err != nil {
		return ClineConfigContract{}, err
	}
	return contract, nil
}

// Bytes returns a copy of the generated providers.json document.
func (c ClineConfigContract) Bytes() []byte {
	return append([]byte(nil), c.raw...)
}

// Model returns the validated Agent Brain route model.
func (c ClineConfigContract) Model() brain.RouteModel { return c.model }

// BaseURL returns the OmniRoute Chat Completions base URL (gateway root + /v1).
func (c ClineConfigContract) BaseURL() string { return c.baseURL }

// CredentialEnvKey returns the environment variable name the central launch
// injector uses to resolve the single stable OmniRoute secret. The value is
// never handled by this package.
func (c ClineConfigContract) CredentialEnvKey() string { return ClineOmniRouteAPIKeyEnv }

// Validate re-validates the generated document against the frozen contract.
func (c ClineConfigContract) Validate() error {
	return ValidateClineConfigBytes(c.raw, c.model, c.baseURL, c.updatedAt)
}

// ValidateClineConfigBytes performs structural validation without surfacing a
// JSON parser error that could echo configuration content. It enforces the
// exact key set, the single OmniRoute openai-compatible provider, the
// no-secret sentinel invariant (settings.apiKey must be the reference sentinel,
// never a value), the /v1 base-URL shape, and the NVIDIA non-selectability rule.
func ValidateClineConfigBytes(raw []byte, model brain.RouteModel, baseURL string, updatedAt string) error {
	if len(raw) == 0 || len(raw) > 64<<10 {
		return ErrClineConfigContract
	}
	if isNVIDIAOwnedRoute(model) {
		return ErrClineRouteNotAgentBrainSelectable
	}
	var document clineProvidersDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		return ErrClineConfigContract
	}
	if document.Version != clineProvidersSchemaVersion ||
		document.LastUsedProvider != ClineOpenAICompatibleProviderID ||
		len(document.Providers) != 1 {
		return ErrClineConfigContract
	}
	entry, ok := document.Providers[ClineOpenAICompatibleProviderID]
	if !ok {
		return ErrClineConfigContract
	}
	if entry.Settings.Provider != ClineOpenAICompatibleProviderID ||
		entry.Settings.APIKey != ClineSecretReferenceSentinel ||
		entry.Settings.Model != string(model) ||
		entry.Settings.BaseURL != baseURL ||
		entry.TokenSource != ClineTokenSourceOmniRoute ||
		entry.UpdatedAt != updatedAt {
		return ErrClineConfigContract
	}
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return ErrClineConfigContract
	}
	if !exactMapKeys(generic, "version", "lastUsedProvider", "providers") {
		return ErrClineConfigContract
	}
	providers, ok := generic["providers"].(map[string]any)
	if !ok || !exactMapKeys(providers, ClineOpenAICompatibleProviderID) {
		return ErrClineConfigContract
	}
	providerMap, ok := providers[ClineOpenAICompatibleProviderID].(map[string]any)
	if !ok || !exactMapKeys(providerMap, "settings", "updatedAt", "tokenSource") {
		return ErrClineConfigContract
	}
	settings, ok := providerMap["settings"].(map[string]any)
	if !ok || !exactMapKeys(settings, "provider", "apiKey", "model", "baseUrl") {
		return ErrClineConfigContract
	}
	parsed, err := url.Parse(entry.Settings.BaseURL)
	if err != nil || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "/v1" {
		return ErrClineConfigContract
	}
	return nil
}

// isNVIDIAOwnedRoute reports whether the route's leading namespace segment is
// the OmniRoute-owned NVIDIA fallback namespace.
func isNVIDIAOwnedRoute(model brain.RouteModel) bool {
	value := strings.ToLower(strings.TrimSpace(string(model)))
	segment := value
	if i := strings.IndexByte(value, '/'); i >= 0 {
		segment = value[:i]
	}
	return segment == clineNVIDIARouteNamespace
}
