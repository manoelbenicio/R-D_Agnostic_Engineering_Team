package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/multica-ai/multica/server/internal/rotation"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

const eventDaemonCredentialSessionDiscovery = "daemon:credential_session_discovery"

var errDiscoveryReassignmentUnavailable = errors.New("daemon: discovery reassignment service unavailable")

// credentialSessionDiscoveryPayload is the non-secret daemon event emitted by
// session discovery. AccountID is the assignment observed when discovery ran;
// the rotation service uses it as a stale-event compare value.
type credentialSessionDiscoveryPayload struct {
	AgentID     string `json:"agent_id"`
	AccountID   string `json:"account_id"`
	Provider    string `json:"provider"`
	WorkspaceID string `json:"workspace_id"`
	Status      string `json:"status"`
	ExpiresAt   string `json:"expires_at"`
}

// discoverySessionReassigner is deliberately narrower than RotationService so
// the daemon monitor can be tested without a database or vendor authenticator.
type discoverySessionReassigner interface {
	ReassignDiscoverySession(
		ctx context.Context,
		agentID string,
		expectedAccountID string,
		provider string,
		tenantID string,
		session rotation.DiscoverySession,
		now time.Time,
	) (rotation.Account, bool, error)
}

var _ discoverySessionReassigner = (*rotation.Service)(nil)

// credentialSessionDiscoveryOutcome contains only non-secret assignment
// metadata. The rotation service records the durable rotation event before a
// successful reassignment is returned to this bridge.
type credentialSessionDiscoveryOutcome struct {
	Handled           bool
	Reassigned        bool
	AgentID           string
	PreviousAccountID string
	NextAccountID     string
	Provider          string
	TenantID          string
}

// dispatchCredentialSessionDiscoveryEvent is the daemon/session-monitor bridge.
// It accepts only the dedicated discovery event and forwards only non-secret
// status metadata to the concrete rotation service.
func (d *Daemon) dispatchCredentialSessionDiscoveryEvent(ctx context.Context, message protocol.Message, now time.Time) (bool, error) {
	outcome, err := d.dispatchCredentialSessionDiscoveryEventWithOutcome(ctx, message, now)
	return outcome.Handled, err
}

func (d *Daemon) dispatchCredentialSessionDiscoveryEventWithOutcome(ctx context.Context, message protocol.Message, now time.Time) (credentialSessionDiscoveryOutcome, error) {
	if message.Type != eventDaemonCredentialSessionDiscovery {
		return credentialSessionDiscoveryOutcome{}, nil
	}
	outcome := credentialSessionDiscoveryOutcome{Handled: true}
	if d == nil {
		return outcome, errDiscoveryReassignmentUnavailable
	}
	reassigner, ok := d.rotationService.(discoverySessionReassigner)
	if !ok || reassigner == nil {
		return outcome, errDiscoveryReassignmentUnavailable
	}

	var payload credentialSessionDiscoveryPayload
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		return outcome, err
	}
	outcome.AgentID = payload.AgentID
	outcome.PreviousAccountID = payload.AccountID
	outcome.Provider = payload.Provider
	outcome.TenantID = payload.WorkspaceID

	next, reassigned, err := reassigner.ReassignDiscoverySession(
		ctx,
		payload.AgentID,
		payload.AccountID,
		payload.Provider,
		payload.WorkspaceID,
		rotation.DiscoverySession{
			Provider:  payload.Provider,
			Status:    payload.Status,
			ExpiresAt: payload.ExpiresAt,
		},
		now,
	)
	if err != nil {
		return outcome, err
	}
	outcome.Reassigned = reassigned
	if reassigned {
		outcome.NextAccountID = next.AccountID
	}
	return outcome, nil
}
