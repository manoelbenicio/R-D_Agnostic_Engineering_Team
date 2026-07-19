package daemon

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/rotation"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

// recordRotationFailingStore embeds the existing synthetic store fixture and
// overrides ONLY RecordRotation so it fails closed with a synthetic sentinel.
// Every other Store operation (list/select/assign/status/login bookkeeping)
// runs through the real fixture, so the wrapped store drives the genuine
// rotation.Service reassignment path up to the durable-record step.
type recordRotationFailingStore struct {
	*producerSyntheticStore
	recordErr error
}

func (s *recordRotationFailingStore) RecordRotation(
	context.Context, string, string, string, rotation.RotationReason, time.Time,
) error {
	return s.recordErr
}

// TestDispatchCredentialSessionRecordRotationFailureAlertsWithoutSuccessOrLeak
// drives a REAL rotation.Service (not a stub reassigner) through
// dispatchAndReportCredentialSessionDiscoveryEvent with a store whose
// RecordRotation returns a synthetic sentinel error. It asserts the operator
// sees an ERROR alert=reassignment_failed, never a success alert or
// next_account_id, and that no credential/error sentinel leaks into the log.
//
// It also documents — without changing — the current non-atomic
// assignment-before-record behavior in rotation.Service.onExhaustionLocked:
// Assign is persisted before RecordRotation, so a failed record leaves the
// assignment written and NOT rolled back. No rollback is claimed.
func TestDispatchCredentialSessionRecordRotationFailureAlertsWithoutSuccessOrLeak(t *testing.T) {
	const credentialSentinel = "synthetic-credential-home-sentinel"
	const recordSentinel = "synthetic-record-rotation-sentinel"

	base := newProducerSyntheticStore([]rotation.Account{
		{AccountID: "account-current", Vendor: "codex", TenantID: "workspace-1", Priority: 10, Status: rotation.StatusLeased},
		{
			AccountID: "account-next", Vendor: "codex", TenantID: "workspace-1", Priority: 20,
			Status:    rotation.StatusAvailable,
			HomeDir:   "/credential/" + credentialSentinel,
			ConfigDir: "/config/" + credentialSentinel,
			LastError: credentialSentinel,
		},
	})
	base.assignments["agent-1"] = "account-current"

	store := &recordRotationFailingStore{
		producerSyntheticStore: base,
		recordErr:              errors.New("persist rotation failed: " + recordSentinel),
	}
	auth := &producerSyntheticAuthenticator{}
	service := rotation.NewService(store, producerNoopDetector{}, auth)

	var logs bytes.Buffer
	d := &Daemon{
		rotationService: service,
		logger:          credentialSessionAlertTestLogger(&logs),
	}

	d.dispatchAndReportCredentialSessionDiscoveryEvent(context.Background(), protocol.Message{
		Type: eventDaemonCredentialSessionDiscovery,
		Payload: syntheticDiscoveryPayload(t, credentialSessionDiscoveryPayload{
			AgentID:     "agent-1",
			AccountID:   "account-current",
			Provider:    "codex",
			WorkspaceID: "workspace-1",
			Status:      "exhausted",
		}),
	}, time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC))

	got := logs.String()

	// The real service must have executed the rotation up to the failing
	// record step: it logs out the current account and logs in the next one.
	authCalls := strings.Join(auth.snapshotCalls(), ",")
	if !strings.Contains(authCalls, "login:account-next") {
		t.Fatalf("real rotation.Service did not reach login of the next account: calls=%q", authCalls)
	}

	// Failure alert surface: ERROR + reassignment_failed + non-secret metadata.
	for _, want := range []string{
		"level=ERROR",
		"alert=reassignment_failed",
		"automatic credential account reassignment failed",
		"agent_id=agent-1",
		"provider=codex",
		"tenant_id=workspace-1",
		"previous_account_id=account-current",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("failure alert missing %q: %s", want, got)
		}
	}

	// No success alert and no success metadata may be emitted on the failure path.
	for _, forbidden := range []string{
		"automatic credential account reassignment completed",
		"next_account_id=",
		"level=WARN",
	} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("failure path emitted forbidden success marker %q: %s", forbidden, got)
		}
	}

	// No credential material or raw record-rotation error text may leak.
	for _, secret := range []string{credentialSentinel, recordSentinel, "persist rotation failed"} {
		if strings.Contains(got, secret) {
			t.Fatalf("operator log leaked sensitive text %q: %s", secret, got)
		}
	}

	// Document (do not change) the non-atomic assignment-before-record behavior:
	// Assign persisted before the failed RecordRotation and is NOT rolled back.
	// This is an observation of current behavior; no rollback is asserted.
	if assigned := base.assignment("agent-1"); assigned != "account-next" {
		t.Fatalf("expected non-atomic assignment to persist as account-next (assignment-before-record), got %q", assigned)
	}
}
