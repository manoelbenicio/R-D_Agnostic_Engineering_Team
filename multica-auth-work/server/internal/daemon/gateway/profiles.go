package gateway

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

type ProfileID string

const (
	ProfileAnthropicMessages ProfileID = "omniroute-anthropic-messages"
	ProfileOpenAIResponses   ProfileID = "omniroute-openai-responses"
	ProfileOpenAIChat        ProfileID = "omniroute-openai-chat"
	ProfileAntigravity       ProfileID = "omniroute-antigravity-compatible"
)

type BaseURLForm string

const (
	BaseURLRoot BaseURLForm = "root"
	BaseURLV1   BaseURLForm = "v1"
)

type RuntimeProfile struct {
	ID               ProfileID
	Protocol         brain.ProtocolFamily
	Endpoint         string
	BaseURLForm      BaseURLForm
	WireAPI          string
	StreamTransport  string
	AllowedCLI       []brain.CLIKind
	EvidenceRequired bool
}

func TrustedRuntimeProfiles() map[brain.ProtocolFamily]RuntimeProfile {
	return map[brain.ProtocolFamily]RuntimeProfile{
		brain.ProtocolAnthropicMessages: {
			ID: ProfileAnthropicMessages, Protocol: brain.ProtocolAnthropicMessages,
			Endpoint: "/v1/messages", BaseURLForm: BaseURLRoot, WireAPI: "messages",
			StreamTransport: "sse", AllowedCLI: []brain.CLIKind{brain.CLIClaudeCode}, EvidenceRequired: true,
		},
		brain.ProtocolOpenAIResponses: {
			ID: ProfileOpenAIResponses, Protocol: brain.ProtocolOpenAIResponses,
			Endpoint: "/v1/responses", BaseURLForm: BaseURLV1, WireAPI: "responses",
			StreamTransport: "sse", AllowedCLI: []brain.CLIKind{brain.CLICodex}, EvidenceRequired: true,
		},
		brain.ProtocolOpenAIChat: {
			ID: ProfileOpenAIChat, Protocol: brain.ProtocolOpenAIChat,
			Endpoint: "/v1/chat/completions", BaseURLForm: BaseURLV1, WireAPI: "chat",
			StreamTransport: "sse", AllowedCLI: []brain.CLIKind{brain.CLIKimi, brain.CLIOpenAICompatible, brain.CLINIM}, EvidenceRequired: true,
		},
		brain.ProtocolAntigravity: {
			ID: ProfileAntigravity, Protocol: brain.ProtocolAntigravity,
			Endpoint: "/v1/antigravity", BaseURLForm: BaseURLV1, WireAPI: "antigravity-compatible",
			StreamTransport: "sse", AllowedCLI: []brain.CLIKind{brain.CLIAntigravity}, EvidenceRequired: true,
		},
	}
}

func LookupRuntimeProfile(protocol brain.ProtocolFamily, cli brain.CLIKind) (RuntimeProfile, error) {
	profile, ok := TrustedRuntimeProfiles()[protocol]
	if !ok {
		return RuntimeProfile{}, &GatewayError{Operation: "profile.lookup", Class: ErrorCapability}
	}
	for _, allowed := range profile.AllowedCLI {
		if allowed == cli {
			return cloneProfile(profile), nil
		}
	}
	return RuntimeProfile{}, &GatewayError{Operation: "profile.lookup", Class: ErrorCapability}
}

func (p RuntimeProfile) Validate() error {
	if p.ID == "" || p.Protocol == "" || validateEndpointPath(p.Endpoint) != nil || p.StreamTransport != "sse" || len(p.AllowedCLI) == 0 || !p.EvidenceRequired {
		return &GatewayError{Operation: "profile.validate", Class: ErrorInvalidConfiguration}
	}
	switch p.BaseURLForm {
	case BaseURLRoot:
		if p.Protocol != brain.ProtocolAnthropicMessages {
			return &GatewayError{Operation: "profile.validate", Class: ErrorInvalidConfiguration}
		}
	case BaseURLV1:
	default:
		return &GatewayError{Operation: "profile.validate", Class: ErrorInvalidConfiguration}
	}
	return nil
}

func (p RuntimeProfile) AdapterBaseURL(root string) (string, error) {
	if err := p.Validate(); err != nil {
		return "", err
	}
	parsed, err := url.Parse(strings.TrimRight(root, "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", &GatewayError{Operation: "profile.base_url", Class: ErrorInvalidConfiguration}
	}
	if p.BaseURLForm == BaseURLV1 {
		parsed.Path = strings.TrimRight(parsed.Path, "/") + "/v1"
	}
	return parsed.String(), nil
}

func (p RuntimeProfile) String() string {
	return fmt.Sprintf("gateway.RuntimeProfile{id:%q, protocol:%q, evidence_required:%t}", p.ID, p.Protocol, p.EvidenceRequired)
}

func cloneProfile(source RuntimeProfile) RuntimeProfile {
	result := source
	result.AllowedCLI = append([]brain.CLIKind(nil), source.AllowedCLI...)
	return result
}
