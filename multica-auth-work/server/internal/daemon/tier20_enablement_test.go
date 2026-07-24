package daemon

import (
	"testing"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

func tierCfg(dev, gate bool, tier brain.CapacityTier) AgentBrainIntegrationConfig {
	return AgentBrainIntegrationConfig{
		DevelopmentEnabled: dev,
		Neutral: brain.Config{
			Gateway:      brain.GatewayConfig{Required: true, Readiness: brain.StrictReadinessPolicy()},
			CapacityTier: tier,
		},
		CapacityGateEnabled: gate,
	}
}

func TestTier20HonoredOnlyWhenGateEnabledAndTier20(t *testing.T) {
	// Default (gate off): fail-closed at one development task.
	if got := effectiveTaskAdmissionLimit(tierCfg(true, false, brain.CapacityTier20), 20); got != agentBrainDevelopmentMaxTasks {
		t.Fatalf("gate off must stay fail-closed at %d, got %d", agentBrainDevelopmentMaxTasks, got)
	}
	// Gate on + tier 20: authorized to 20.
	if got := effectiveTaskAdmissionLimit(tierCfg(true, true, brain.CapacityTier20), 20); got != 20 {
		t.Fatalf("gate on + tier20 must authorize 20, got %d", got)
	}
	// Gate on but tiers 50/100 remain blocked (evidence-required).
	for _, tier := range []brain.CapacityTier{brain.CapacityTier50, brain.CapacityTier100} {
		if got := effectiveTaskAdmissionLimit(tierCfg(true, true, tier), 20); got != agentBrainDevelopmentMaxTasks {
			t.Fatalf("gate on + tier %d must stay blocked at %d, got %d", tier, agentBrainDevelopmentMaxTasks, got)
		}
	}
	// Gate flag alone (tier not 20) does not raise the limit.
	if got := effectiveAgentBrainCapacity(tierCfg(true, true, brain.CapacityTier50)); got != agentBrainDevelopmentMaxTasks {
		t.Fatalf("gate on with non-20 tier must not raise limit, got %d", got)
	}
	// Disabled development slice preserves legacy requested limit.
	if got := effectiveTaskAdmissionLimit(tierCfg(false, true, brain.CapacityTier20), 7); got != 7 {
		t.Fatalf("disabled slice must return requested legacy limit, got %d", got)
	}
}
