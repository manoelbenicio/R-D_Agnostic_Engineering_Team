package runtimeenv

import (
	"errors"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

var (
	ErrModelNotApproved      = errors.New("route model is not approved by the OmniRoute registry")
	ErrCLIModelNotApproved   = errors.New("CLI is not approved for the selected OmniRoute model")
	ErrProtocolNotCompatible = errors.New("CLI protocol is not compatible with the selected OmniRoute model")
	ErrThinkingNotApproved   = errors.New("thinking level is not approved for the selected OmniRoute model")
)

type ApprovedGatewayModel struct {
	Model          brain.RouteModel
	Protocol       brain.ProtocolFamily
	CLIs           []brain.CLIKind
	ThinkingLevels []string
}

type GatewayModelPolicy struct {
	models map[brain.RouteModel]approvedModel
}

type approvedModel struct {
	protocol brain.ProtocolFamily
	clis     map[brain.CLIKind]struct{}
	thinking map[string]struct{}
}

func NewGatewayModelPolicy(models []ApprovedGatewayModel) (GatewayModelPolicy, error) {
	policy := GatewayModelPolicy{models: make(map[brain.RouteModel]approvedModel, len(models))}
	for _, candidate := range models {
		model, err := brain.ParseRouteModel(string(candidate.Model))
		if err != nil {
			return GatewayModelPolicy{}, err
		}
		if _, exists := policy.models[model]; exists || len(candidate.CLIs) == 0 {
			return GatewayModelPolicy{}, ErrModelNotApproved
		}
		entry := approvedModel{protocol: candidate.Protocol, clis: map[brain.CLIKind]struct{}{}, thinking: map[string]struct{}{}}
		for _, cli := range candidate.CLIs {
			if _, err := brain.ParseCLIKind(string(cli)); err != nil {
				return GatewayModelPolicy{}, err
			}
			entry.clis[cli] = struct{}{}
		}
		for _, level := range candidate.ThinkingLevels {
			if level == "" {
				return GatewayModelPolicy{}, ErrThinkingNotApproved
			}
			entry.thinking[level] = struct{}{}
		}
		policy.models[model] = entry
	}
	return policy, nil
}

// ValidateSelection is gateway-aware and pure. It consumes only an approved
// OmniRoute registry snapshot; it has no provider catalog callback and cannot
// discover provider credentials or invoke a native CLI.
func (p GatewayModelPolicy) ValidateSelection(cli brain.CLIKind, model brain.RouteModel, thinking string) error {
	contract, err := CredentiallessAdapterContract(cli)
	if err != nil {
		return err
	}
	entry, ok := p.models[model]
	if !ok {
		return ErrModelNotApproved
	}
	if _, ok := entry.clis[cli]; !ok {
		return ErrCLIModelNotApproved
	}
	if entry.protocol != contract.Protocol {
		return ErrProtocolNotCompatible
	}
	if thinking != "" {
		if _, ok := entry.thinking[thinking]; !ok {
			return ErrThinkingNotApproved
		}
	}
	return nil
}
