package runtimeenv

import (
	"errors"
	"fmt"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

type AdapterState string

const (
	AdapterReady      AdapterState = "ready"
	AdapterFailClosed AdapterState = "fail-closed"
)

type AdapterGate string

const (
	GateOpenAICompatibleUnaccepted  AdapterGate = "openai-compatible-contract-unaccepted"
	GateNativeKimiUnaccepted        AdapterGate = "native-kimi-registry-unaccepted"
	GateNativeNIMUnaccepted         AdapterGate = "native-nim-gateway-adapter-unaccepted"
	GateNativeAntigravityUnaccepted AdapterGate = "native-antigravity-endpoint-unaccepted"
)

var ErrAdapterFailClosed = errors.New("runtime adapter is not accepted for gateway-required execution")

// AdapterContract is a no-secret decision record. FallbackFrontends are
// candidates only; callers must select one explicitly through an approved
// model policy. The package never performs an automatic fallback.
type AdapterContract struct {
	CLI               brain.CLIKind
	State             AdapterState
	Protocol          brain.ProtocolFamily
	Gate              AdapterGate
	FallbackFrontends []brain.CLIKind
}

type AdapterGateError struct {
	Gate AdapterGate
}

func (e *AdapterGateError) Error() string {
	return fmt.Sprintf("%v: %s", ErrAdapterFailClosed, e.Gate)
}

func (e *AdapterGateError) Unwrap() error { return ErrAdapterFailClosed }

// CredentiallessAdapterContract implements Claude and Codex contracts and
// returns explicit fail-closed stubs for the credential-bearing adapters whose
// native gateway contracts are not accepted by the G1 model matrix.
func CredentiallessAdapterContract(cli brain.CLIKind) (AdapterContract, error) {
	switch cli {
	case brain.CLIClaudeCode:
		return AdapterContract{CLI: cli, State: AdapterReady, Protocol: brain.ProtocolAnthropicMessages}, nil
	case brain.CLICodex:
		return AdapterContract{CLI: cli, State: AdapterReady, Protocol: brain.ProtocolOpenAIResponses}, nil
	case brain.CLIOpenAICompatible:
		contract := AdapterContract{CLI: cli, State: AdapterFailClosed, Gate: GateOpenAICompatibleUnaccepted}
		return contract, &AdapterGateError{Gate: contract.Gate}
	case brain.CLIKimi:
		contract := AdapterContract{
			CLI: cli, State: AdapterFailClosed, Gate: GateNativeKimiUnaccepted,
			FallbackFrontends: []brain.CLIKind{brain.CLIClaudeCode, brain.CLICodex},
		}
		return contract, &AdapterGateError{Gate: contract.Gate}
	case brain.CLINIM:
		contract := AdapterContract{CLI: cli, State: AdapterFailClosed, Gate: GateNativeNIMUnaccepted}
		return contract, &AdapterGateError{Gate: contract.Gate}
	case brain.CLIAntigravity:
		contract := AdapterContract{
			CLI: cli, State: AdapterFailClosed, Gate: GateNativeAntigravityUnaccepted,
			FallbackFrontends: []brain.CLIKind{brain.CLIClaudeCode, brain.CLICodex},
		}
		return contract, &AdapterGateError{Gate: contract.Gate}
	default:
		return AdapterContract{}, &AdapterGateError{Gate: GateOpenAICompatibleUnaccepted}
	}
}
