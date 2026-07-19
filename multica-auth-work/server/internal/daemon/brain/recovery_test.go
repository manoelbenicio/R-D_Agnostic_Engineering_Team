package brain

import "testing"

func TestRecoveryModeFailsClosedAndRequiresExplicitOperatorGate(t *testing.T) {
	mode := NewRecoveryMode()
	if mode.State() != RecoveryNormal || mode.RouterOwner() != RecoveryRouterOmniRoute || mode.ProdexEnabled() {
		t.Fatalf("unexpected default recovery mode: state=%q owner=%q prodex=%t", mode.State(), mode.RouterOwner(), mode.ProdexEnabled())
	}

	degraded, err := mode.Transition(RecoveryGatewayUnavailable, RecoveryGates{SessionBoundary: true})
	if err != nil {
		t.Fatalf("degrade: %v", err)
	}
	if degraded.State() != RecoveryDegraded || degraded.RouterOwner() != RecoveryRouterNone || degraded.ProdexEnabled() {
		t.Fatalf("degraded mode must fail closed: state=%q owner=%q prodex=%t", degraded.State(), degraded.RouterOwner(), degraded.ProdexEnabled())
	}

	if _, err := degraded.Transition(RecoveryEnable, RecoveryGates{SessionBoundary: true, OmniRouteQuiesced: true}); err == nil {
		t.Fatal("recovery entry succeeded without operator authorization")
	}
	if _, err := degraded.Transition(RecoveryEnable, RecoveryGates{SessionBoundary: true, OperatorAuthorized: true}); err == nil {
		t.Fatal("recovery entry succeeded while OmniRoute was not quiesced")
	}

	recovery, err := degraded.Transition(RecoveryEnable, RecoveryGates{
		SessionBoundary: true, OperatorAuthorized: true, OmniRouteQuiesced: true,
	})
	if err != nil {
		t.Fatalf("enter recovery: %v", err)
	}
	if recovery.State() != RecoveryActive || recovery.RouterOwner() != RecoveryRouterRustL2 || !recovery.ProdexEnabled() {
		t.Fatalf("unexpected recovery mode: state=%q owner=%q prodex=%t", recovery.State(), recovery.RouterOwner(), recovery.ProdexEnabled())
	}
}

func TestRecoveryModeTransitionsOnlyAtSessionBoundaries(t *testing.T) {
	mode := NewRecoveryMode()
	if _, err := mode.Transition(RecoveryGatewayUnavailable, RecoveryGates{}); err == nil {
		t.Fatal("mid-session transition succeeded")
	}
}

func TestRecoveryModeZeroValueFailsClosed(t *testing.T) {
	var mode RecoveryMode
	if mode.RouterOwner() != RecoveryRouterNone || mode.ProdexEnabled() {
		t.Fatalf("invalid recovery mode did not fail closed: owner=%q prodex=%t", mode.RouterOwner(), mode.ProdexEnabled())
	}
	if _, err := mode.Transition(RecoveryGatewayUnavailable, RecoveryGates{SessionBoundary: true}); err == nil {
		t.Fatal("invalid recovery mode accepted a transition")
	}
}

func TestRecoveryModeNeverAutomaticallyPromotesProdex(t *testing.T) {
	degraded, err := NewRecoveryMode().Transition(RecoveryGatewayUnavailable, RecoveryGates{SessionBoundary: true})
	if err != nil {
		t.Fatalf("degrade: %v", err)
	}
	if _, err := degraded.Transition(RecoveryGatewayUnavailable, RecoveryGates{SessionBoundary: true}); err == nil {
		t.Fatal("repeated outage unexpectedly changed the degraded state")
	}
	if degraded.ProdexEnabled() {
		t.Fatal("gateway outage automatically promoted Prodex")
	}
}

func TestRecoveryModeRestoreRequiresDrainReadyAndOperator(t *testing.T) {
	degraded, err := NewRecoveryMode().Transition(RecoveryGatewayUnavailable, RecoveryGates{SessionBoundary: true})
	if err != nil {
		t.Fatalf("degrade: %v", err)
	}
	recovery, err := degraded.Transition(RecoveryEnable, RecoveryGates{
		SessionBoundary: true, OperatorAuthorized: true, OmniRouteQuiesced: true,
	})
	if err != nil {
		t.Fatalf("enter recovery: %v", err)
	}

	for name, gates := range map[string]RecoveryGates{
		"operator": {SessionBoundary: true, ProdexDrained: true, GatewayReady: true},
		"drain":    {SessionBoundary: true, OperatorAuthorized: true, GatewayReady: true},
		"gateway":  {SessionBoundary: true, OperatorAuthorized: true, ProdexDrained: true},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := recovery.Transition(RecoveryRestore, gates); err == nil {
				t.Fatal("unsafe restore succeeded")
			}
		})
	}

	normal, err := recovery.Transition(RecoveryRestore, RecoveryGates{
		SessionBoundary: true, OperatorAuthorized: true, ProdexDrained: true, GatewayReady: true,
	})
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if normal != NewRecoveryMode() {
		t.Fatalf("restore did not return to the default NORMAL state: %#v", normal)
	}
}

func TestDegradedModeReturnsToNormalOnlyWhenGatewayReady(t *testing.T) {
	degraded, err := NewRecoveryMode().Transition(RecoveryGatewayUnavailable, RecoveryGates{SessionBoundary: true})
	if err != nil {
		t.Fatalf("degrade: %v", err)
	}
	if _, err := degraded.Transition(RecoveryGatewayRestored, RecoveryGates{SessionBoundary: true}); err == nil {
		t.Fatal("gateway restore succeeded without readiness")
	}
	normal, err := degraded.Transition(RecoveryGatewayRestored, RecoveryGates{SessionBoundary: true, GatewayReady: true})
	if err != nil {
		t.Fatalf("gateway restore: %v", err)
	}
	if normal != NewRecoveryMode() {
		t.Fatalf("gateway restore did not return NORMAL: %#v", normal)
	}
}
