package deploy

import (
	"testing"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

func TestDefaultSecretReferenceContract(t *testing.T) {
	contract := DefaultSecretReferenceContract()
	if err := contract.Validate(); err != nil {
		t.Fatalf("default secret reference contract: %v", err)
	}
	if contract.ConfigEnvironment != brain.EnvGatewaySecretFile {
		t.Fatal("secret reference does not use frozen configuration name")
	}
	if contract.FileMode.Perm()&0o007 != 0 {
		t.Fatal("secret reference permits world access")
	}
}

func TestSecretReferenceRejectsUnsafeMode(t *testing.T) {
	contract := DefaultSecretReferenceContract()
	contract.FileMode = 0o444
	if err := contract.Validate(); err == nil {
		t.Fatal("expected world-readable mode to fail")
	}
}

func TestEndpointPlansMatchFrozenTopology(t *testing.T) {
	for _, plan := range []EndpointPlan{HostWSLEndpointPlan(), ContainerEndpointPlan()} {
		if err := plan.Validate(); err != nil {
			t.Fatalf("endpoint plan %q: %v", plan.Topology, err)
		}
	}
	host := HostWSLEndpointPlan()
	host.BaseURL = brain.DefaultContainerGatewayURL
	if err := host.Validate(); err == nil {
		t.Fatal("host topology accepted container DNS")
	}
}

func TestOperationsCatalogComplete(t *testing.T) {
	catalog := DefaultOperationsCatalog()
	if err := catalog.Validate(); err != nil {
		t.Fatalf("operations catalog: %v", err)
	}
	if len(catalog.Runbooks) != 8 {
		t.Fatalf("runbook count = %d, want 8", len(catalog.Runbooks))
	}
}

func TestRolloutPlanIsDefaultOffAndTier20Bounded(t *testing.T) {
	plan := DefaultRolloutPlan()
	if err := plan.Validate(); err != nil {
		t.Fatalf("rollout plan: %v", err)
	}
	for _, gate := range plan.Gates {
		if gate.DefaultEnabled {
			t.Fatalf("gate %q is unexpectedly enabled", gate.ID)
		}
	}
	for _, cohort := range plan.Cohorts {
		if cohort.MaxTaskTier > 20 {
			t.Fatalf("cohort %q exceeds authorized tier", cohort.ID)
		}
	}
}
