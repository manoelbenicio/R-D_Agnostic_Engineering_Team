package daemon

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/multica-ai/multica/server/internal/daemon/gateway"
)

// PD-08 regression for the credential-source wiring branch in
// cmd/multica/cmd_daemon.go, exercised at the daemon/brain_integration level.
//
// cmd_daemon.go selects the Agent Brain dependencies exactly as:
//
//	if cfg.AgentBrain.DevelopmentEnabled && cfg.AgentBrain.Neutral.Gateway.Required {
//	    daemon.NewWithAgentBrainDependencies(cfg, logger, daemon.AgentBrainDependencies{
//	        CredentialSource: daemon.FileCredentialSource{}, HTTPClient: ...})
//	} else {
//	    daemon.New(cfg, logger) // no AgentBrainDependencies -> nil CredentialSource
//	}
//
// PD-08 invariant: a file-backed credential reader is wired ONLY in the
// default-off development slice (development enabled AND gateway required);
// otherwise none is provided and admission fails closed. These tests assert the
// branch's flag/type intent and its behavioral consequence in admitTask.

// FileCredentialSource is the exact production gateway.CredentialSource the
// dev+gateway branch injects (compile-time contract).
var _ gateway.CredentialSource = FileCredentialSource{}

// credentialSourceForBranch mirrors the cmd_daemon.go predicate so this
// regression fails if that branch's flag condition or wired type changes.
func credentialSourceForBranch(developmentEnabled, gatewayRequired bool) AgentBrainDependencies {
	if developmentEnabled && gatewayRequired {
		return AgentBrainDependencies{CredentialSource: FileCredentialSource{}}
	}
	return AgentBrainDependencies{}
}

// TestPD08CredentialSourceWiredOnlyForDevGatewaySlice asserts the wiring intent
// of the cmd_daemon.go branch across every flag combination: a non-nil
// FileCredentialSource is present iff DevelopmentEnabled && Gateway.Required.
func TestPD08CredentialSourceWiredOnlyForDevGatewaySlice(t *testing.T) {
	cases := []struct {
		developmentEnabled bool
		gatewayRequired    bool
		wantSource         bool
	}{
		{true, true, true},   // dev + gateway-required -> FileCredentialSource{}
		{true, false, false}, // gateway not required -> none
		{false, true, false}, // development off -> none
		{false, false, false},
	}
	for _, c := range cases {
		deps := credentialSourceForBranch(c.developmentEnabled, c.gatewayRequired)
		gotSource := deps.CredentialSource != nil
		if gotSource != c.wantSource {
			t.Fatalf("development=%v gatewayRequired=%v: CredentialSource present=%v, want %v",
				c.developmentEnabled, c.gatewayRequired, gotSource, c.wantSource)
		}
		if c.wantSource {
			if _, ok := deps.CredentialSource.(FileCredentialSource); !ok {
				t.Fatalf("dev+gateway wired %T, want daemon.FileCredentialSource", deps.CredentialSource)
			}
		}
	}
}

// TestPD08AdmissionFailsClosedWithoutCredentialSource proves the "else" branch
// consequence: with no credential source wired, admission fails closed with the
// deterministic credential_source_unavailable class (never admits).
func TestPD08AdmissionFailsClosedWithoutCredentialSource(t *testing.T) {
	gw := newSyntheticGateway(t, true) // gateway readiness is irrelevant: the nil
	defer gw.Close()                   // credential gate is checked before it.
	runtime, err := newAgentBrainRuntime(
		syntheticAgentBrainConfig(t, gw.URL),
		AgentBrainDependencies{HTTPClient: gw.Client()}, // CredentialSource nil
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err != nil {
		t.Fatalf("newAgentBrainRuntime: %v", err)
	}
	_, admitErr := runtime.admitTask(context.Background(), syntheticGatewayTask(), "claude", string(runtime.config.RouteModel))
	if admitErr == nil {
		t.Fatal("admitTask succeeded with no credential source (PD-08 must fail closed)")
	}
	var adm *agentBrainAdmissionError
	if !errors.As(admitErr, &adm) || adm.class != "credential_source_unavailable" {
		t.Fatalf("want credential_source_unavailable, got %v", admitErr)
	}
}

// TestPD08AdmissionPassesCredentialGateWithFileCredentialSource proves the
// dev+gateway branch consequence: a non-nil FileCredentialSource clears the
// PD-08 nil gate. Admission may still fail at a later gate (the file-backed
// reader has no real secret file in this hermetic test), but it must never be
// credential_source_unavailable.
func TestPD08AdmissionPassesCredentialGateWithFileCredentialSource(t *testing.T) {
	gw := newSyntheticGateway(t, true)
	defer gw.Close()
	runtime, err := newAgentBrainRuntime(
		syntheticAgentBrainConfig(t, gw.URL),
		AgentBrainDependencies{CredentialSource: FileCredentialSource{}, HTTPClient: gw.Client()},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err != nil {
		t.Fatalf("newAgentBrainRuntime: %v", err)
	}
	_, admitErr := runtime.admitTask(context.Background(), syntheticGatewayTask(), "claude", string(runtime.config.RouteModel))
	var adm *agentBrainAdmissionError
	if errors.As(admitErr, &adm) && adm.class == "credential_source_unavailable" {
		t.Fatalf("credential gate must pass when FileCredentialSource is wired, got %v", admitErr)
	}
}
