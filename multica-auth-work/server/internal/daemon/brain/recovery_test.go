package brain

import "testing"

func TestRecoveryModeGatewayOutageFailsClosedWithoutFallback(t *testing.T) {
	mode := NewRecoveryMode()
	if mode.State() != RecoveryNormal || mode.RouterOwner() != RecoveryRouterOmniRoute {
		t.Fatalf("unexpected normal mode: state=%q owner=%q", mode.State(), mode.RouterOwner())
	}
	degraded, err := mode.Transition(RecoveryGatewayUnavailable, RecoveryGates{SessionBoundary: true})
	if err != nil {
		t.Fatalf("degrade: %v", err)
	}
	if degraded.State() != RecoveryDegraded || degraded.RouterOwner() != RecoveryRouterNone {
		t.Fatalf("outage did not fail closed: state=%q owner=%q", degraded.State(), degraded.RouterOwner())
	}
	if _, err := degraded.Transition(RecoveryGatewayUnavailable, RecoveryGates{SessionBoundary: true}); err == nil {
		t.Fatal("repeated outage promoted or changed the fail-closed state")
	}
}

func TestRecoveryModeTransitionsOnlyAtSessionBoundary(t *testing.T) {
	if _, err := NewRecoveryMode().Transition(RecoveryGatewayUnavailable, RecoveryGates{}); err == nil {
		t.Fatal("mid-session transition succeeded")
	}
}

func TestRecoveryModeZeroValueFailsClosed(t *testing.T) {
	var mode RecoveryMode
	if mode.RouterOwner() != RecoveryRouterNone {
		t.Fatalf("zero value owner=%q", mode.RouterOwner())
	}
	if _, err := mode.Transition(RecoveryGatewayUnavailable, RecoveryGates{SessionBoundary: true}); err == nil {
		t.Fatal("invalid zero-value mode accepted a transition")
	}
}

func TestRecoveryModeRestoresOnlyReadyOmniRoute(t *testing.T) {
	degraded, err := NewRecoveryMode().Transition(RecoveryGatewayUnavailable, RecoveryGates{SessionBoundary: true})
	if err != nil {
		t.Fatalf("degrade: %v", err)
	}
	if _, err := degraded.Transition(RecoveryGatewayRestored, RecoveryGates{SessionBoundary: true}); err == nil {
		t.Fatal("restore succeeded without OmniRoute readiness")
	}
	normal, err := degraded.Transition(RecoveryGatewayRestored, RecoveryGates{SessionBoundary: true, GatewayReady: true})
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if normal != NewRecoveryMode() {
		t.Fatalf("restore result=%#v", normal)
	}
}
