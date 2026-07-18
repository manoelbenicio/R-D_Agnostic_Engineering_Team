package brain

import (
	"fmt"
	"strings"
)

// ApprovedRoutePolicy is a versioned policy reference approved outside the
// coordinator. It contains routing intent only and never account or credential
// data.
type ApprovedRoutePolicy struct {
	ID       string         `json:"id"`
	Revision string         `json:"revision"`
	Protocol ProtocolFamily `json:"protocol"`
	Approved bool           `json:"approved"`
}

func (p ApprovedRoutePolicy) validateFor(request TaskRequest) error {
	if strings.TrimSpace(p.ID) == "" {
		return fmt.Errorf("approved route policy id is required")
	}
	if p.ID != request.RoutePolicyID {
		return fmt.Errorf("approved route policy does not match task policy reference")
	}
	if strings.TrimSpace(p.Revision) == "" {
		return fmt.Errorf("approved route policy revision is required")
	}
	switch p.Protocol {
	case ProtocolAnthropicMessages, ProtocolOpenAIResponses, ProtocolOpenAIChat, ProtocolAntigravity:
	default:
		return fmt.Errorf("approved route policy protocol is unsupported")
	}
	if request.GatewayRequired && !p.Approved {
		return fmt.Errorf("route policy is not approved for gateway-required admission")
	}
	return nil
}

// Task combines the frozen task request with its approved route-policy
// reference. TaskRequest carries CLI/model/owner/correlation fields directly.
// LifecycleBindings carries opaque cold-plane references required by the
// existing task lifecycle. Values are identifiers only: task content,
// provider credentials, and secret material do not belong in this contract.
type LifecycleBindings struct {
	WorkspaceRef      string   `json:"workspace_ref,omitempty"`
	RepositoryRefs    []string `json:"repository_refs,omitempty"`
	WorktreeRef       string   `json:"worktree_ref,omitempty"`
	ContextRef        string   `json:"context_ref,omitempty"`
	SkillRefs         []string `json:"skill_refs,omitempty"`
	RecoveryRef       string   `json:"recovery_ref,omitempty"`
	WatchdogPolicyRef string   `json:"watchdog_policy_ref,omitempty"`
	StreamPolicyRef   string   `json:"stream_policy_ref,omitempty"`
	TerminalPolicyRef string   `json:"terminal_policy_ref,omitempty"`
}

type Task struct {
	Request     TaskRequest         `json:"request"`
	RoutePolicy ApprovedRoutePolicy `json:"route_policy"`
	Lifecycle   LifecycleBindings   `json:"lifecycle,omitempty"`
}

func (t Task) Validate() error {
	if err := t.Request.Validate(); err != nil {
		return err
	}
	return t.RoutePolicy.validateFor(t.Request)
}
