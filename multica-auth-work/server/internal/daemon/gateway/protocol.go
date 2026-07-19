package gateway

import (
	"fmt"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

type SSEContract struct {
	Protocol       brain.ProtocolFamily
	RequiredPrefix []string
	TerminalEvents []string
}

func ProtocolSSEContracts() map[brain.ProtocolFamily]SSEContract {
	return map[brain.ProtocolFamily]SSEContract{
		brain.ProtocolAnthropicMessages: {
			Protocol:       brain.ProtocolAnthropicMessages,
			RequiredPrefix: []string{"message_start", "content_block_start"},
			TerminalEvents: []string{"message_stop", "error"},
		},
		brain.ProtocolOpenAIResponses: {
			Protocol:       brain.ProtocolOpenAIResponses,
			RequiredPrefix: []string{"response.created", "response.output_item.added"},
			TerminalEvents: []string{"response.completed", "response.failed", "response.cancelled"},
		},
		brain.ProtocolOpenAIChat: {
			Protocol:       brain.ProtocolOpenAIChat,
			RequiredPrefix: []string{"chat.completion.chunk"},
			TerminalEvents: []string{"[DONE]"},
		},
	}
}

func ValidateSSESequence(protocol brain.ProtocolFamily, events []string) error {
	contract, ok := ProtocolSSEContracts()[protocol]
	if !ok {
		return &GatewayError{Operation: "sse_contract.validate", Class: ErrorCapability}
	}
	if len(events) == 0 {
		return &GatewayError{Operation: "sse_contract.validate", Class: ErrorProtocol}
	}
	if len(events) < len(contract.RequiredPrefix)+1 {
		return &GatewayError{Operation: "sse_contract.validate", Class: ErrorProtocol}
	}
	for index, required := range contract.RequiredPrefix {
		if events[index] != required {
			return &GatewayError{Operation: "sse_contract.validate", Class: ErrorProtocol}
		}
	}
	for _, event := range events[:len(events)-1] {
		for _, terminal := range contract.TerminalEvents {
			if event == terminal {
				return &GatewayError{Operation: "sse_contract.validate", Class: ErrorProtocol}
			}
		}
	}
	terminal := events[len(events)-1]
	for _, allowed := range contract.TerminalEvents {
		if terminal == allowed {
			return nil
		}
	}
	return &GatewayError{Operation: "sse_contract.validate", Class: ErrorProtocol}
}

func (c SSEContract) String() string {
	return fmt.Sprintf("gateway.SSEContract{protocol:%q, required_prefix:%d, terminal_events:%d}", c.Protocol, len(c.RequiredPrefix), len(c.TerminalEvents))
}
