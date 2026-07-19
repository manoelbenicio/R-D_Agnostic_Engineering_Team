package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/rotation"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

func TestDispatchCredentialSessionDiscoveryEventForwardsExpiredObservation(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	reassigner := &syntheticDiscoveryReassigner{
		account:    rotation.Account{AccountID: "account-next", Vendor: "codex", TenantID: "workspace-1"},
		reassigned: true,
	}
	d := &Daemon{rotationService: reassigner}
	payload := credentialSessionDiscoveryPayload{
		AgentID:     "agent-1",
		AccountID:   "account-current",
		Provider:    "codex",
		WorkspaceID: "workspace-1",
		Status:      "expired",
		ExpiresAt:   "2026-07-18T11:59:59Z",
	}

	handled, err := d.dispatchCredentialSessionDiscoveryEvent(context.Background(), protocol.Message{
		Type:    eventDaemonCredentialSessionDiscovery,
		Payload: syntheticDiscoveryPayload(t, payload),
	}, now)
	if err != nil {
		t.Fatalf("dispatchCredentialSessionDiscoveryEvent: %v", err)
	}
	if !handled {
		t.Fatal("discovery event was not handled")
	}
	if reassigner.calls != 1 {
		t.Fatalf("reassignment calls = %d, want 1", reassigner.calls)
	}
	if reassigner.agentID != payload.AgentID || reassigner.expectedAccountID != payload.AccountID {
		t.Fatalf("assignment identity = (%q, %q), want (%q, %q)", reassigner.agentID, reassigner.expectedAccountID, payload.AgentID, payload.AccountID)
	}
	if reassigner.provider != payload.Provider || reassigner.tenantID != payload.WorkspaceID {
		t.Fatalf("provider boundary = (%q, %q), want (%q, %q)", reassigner.provider, reassigner.tenantID, payload.Provider, payload.WorkspaceID)
	}
	if reassigner.session != (rotation.DiscoverySession{Provider: payload.Provider, Status: payload.Status, ExpiresAt: payload.ExpiresAt}) {
		t.Fatalf("discovery session = %+v, want payload metadata", reassigner.session)
	}
	if !reassigner.now.Equal(now) {
		t.Fatalf("observation time = %s, want %s", reassigner.now, now)
	}
}

func TestDispatchCredentialSessionDiscoveryEventPreservesReassignmentErrors(t *testing.T) {
	wantErr := rotation.ErrNoAccountAvailable
	reassigner := &syntheticDiscoveryReassigner{err: wantErr}
	d := &Daemon{rotationService: reassigner}

	handled, err := d.dispatchCredentialSessionDiscoveryEvent(context.Background(), protocol.Message{
		Type: eventDaemonCredentialSessionDiscovery,
		Payload: syntheticDiscoveryPayload(t, credentialSessionDiscoveryPayload{
			AgentID: "agent-1", AccountID: "account-current", Provider: "codex", WorkspaceID: "workspace-1", Status: "exhausted",
		}),
	}, time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC))
	if !handled {
		t.Fatal("discovery event was not handled")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want ErrNoAccountAvailable", err)
	}
	if reassigner.calls != 1 {
		t.Fatalf("reassignment calls = %d, want 1", reassigner.calls)
	}
}

func TestDispatchCredentialSessionDiscoveryEventRejectsMalformedOrUnavailable(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	t.Run("malformed payload", func(t *testing.T) {
		reassigner := &syntheticDiscoveryReassigner{}
		d := &Daemon{rotationService: reassigner}
		handled, err := d.dispatchCredentialSessionDiscoveryEvent(context.Background(), protocol.Message{
			Type: eventDaemonCredentialSessionDiscovery, Payload: json.RawMessage(`{"agent_id":`),
		}, now)
		if !handled || err == nil {
			t.Fatalf("handled/error = (%v, %v), want true/non-nil", handled, err)
		}
		if reassigner.calls != 0 {
			t.Fatalf("malformed event made %d reassignment calls", reassigner.calls)
		}
	})

	t.Run("service unavailable", func(t *testing.T) {
		d := &Daemon{}
		handled, err := d.dispatchCredentialSessionDiscoveryEvent(context.Background(), protocol.Message{
			Type: eventDaemonCredentialSessionDiscovery, Payload: json.RawMessage(`{}`),
		}, now)
		if !handled || !errors.Is(err, errDiscoveryReassignmentUnavailable) {
			t.Fatalf("handled/error = (%v, %v), want true/unavailable", handled, err)
		}
	})

	t.Run("unrelated event", func(t *testing.T) {
		d := &Daemon{}
		handled, err := d.dispatchCredentialSessionDiscoveryEvent(context.Background(), protocol.Message{Type: "daemon:heartbeat_ack"}, now)
		if handled || err != nil {
			t.Fatalf("handled/error = (%v, %v), want false/nil", handled, err)
		}
	})
}

func syntheticDiscoveryPayload(t *testing.T, payload credentialSessionDiscoveryPayload) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal synthetic discovery payload: %v", err)
	}
	return raw
}

type syntheticDiscoveryReassigner struct {
	calls             int
	agentID           string
	expectedAccountID string
	provider          string
	tenantID          string
	session           rotation.DiscoverySession
	now               time.Time
	account           rotation.Account
	reassigned        bool
	err               error
}

func (r *syntheticDiscoveryReassigner) SelectNext(context.Context, string, string, time.Time) (rotation.Account, error) {
	return rotation.Account{}, errors.New("unexpected SelectNext call")
}

func (r *syntheticDiscoveryReassigner) OnExhaustion(context.Context, string, string, string, rotation.RotationReason, time.Time) (rotation.Account, error) {
	return rotation.Account{}, errors.New("unexpected OnExhaustion call")
}

func (r *syntheticDiscoveryReassigner) ReassignDiscoverySession(
	_ context.Context,
	agentID string,
	expectedAccountID string,
	provider string,
	tenantID string,
	session rotation.DiscoverySession,
	now time.Time,
) (rotation.Account, bool, error) {
	r.calls++
	r.agentID = agentID
	r.expectedAccountID = expectedAccountID
	r.provider = provider
	r.tenantID = tenantID
	r.session = session
	r.now = now
	return r.account, r.reassigned, r.err
}
