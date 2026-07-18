package daemon

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/rotation"
	"github.com/multica-ai/multica/server/pkg/protocol"
	"github.com/multica-ai/multica/server/pkg/redact"
)

func TestDispatchAndReportCredentialSessionDiscoveryEventReportsReassignmentWithoutCredentials(t *testing.T) {
	const credentialSentinel = "synthetic-credential-sentinel"
	var logs bytes.Buffer
	reassigner := &syntheticDiscoveryReassigner{
		account: rotation.Account{
			AccountID: "account-next",
			Vendor:    "codex",
			TenantID:  "workspace-1",
			HomeDir:   "/credential/" + credentialSentinel,
			ConfigDir: "/config/" + credentialSentinel,
			LastError: credentialSentinel,
		},
		reassigned: true,
	}
	d := &Daemon{
		rotationService: reassigner,
		logger:          credentialSessionAlertTestLogger(&logs),
	}
	payload := credentialSessionDiscoveryPayload{
		AgentID:     "agent-1",
		AccountID:   "account-current",
		Provider:    "codex",
		WorkspaceID: "workspace-1",
		Status:      "expired",
	}

	d.dispatchAndReportCredentialSessionDiscoveryEvent(context.Background(), protocol.Message{
		Type:    eventDaemonCredentialSessionDiscovery,
		Payload: syntheticDiscoveryPayload(t, payload),
	}, time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC))

	got := logs.String()
	for _, want := range []string{
		"automatic credential account reassignment completed",
		"agent_id=agent-1",
		"provider=codex",
		"tenant_id=workspace-1",
		"previous_account_id=account-current",
		"next_account_id=account-next",
		"reason=quota_exhausted_reactive",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("operator log missing %q: %s", want, got)
		}
	}
	if strings.Contains(got, credentialSentinel) {
		t.Fatalf("operator log leaked credential sentinel: %s", got)
	}
}

func TestDispatchAndReportCredentialSessionDiscoveryEventAlertsWithoutLeakingErrors(t *testing.T) {
	const credentialSentinel = "synthetic-error-credential-sentinel"
	tests := []struct {
		name      string
		err       error
		wantLevel string
		wantAlert string
	}{
		{name: "no account", err: rotation.ErrNoAccountAvailable, wantLevel: "level=WARN", wantAlert: "alert=no_account_available"},
		{name: "auth error", err: errors.New("provider rejected " + credentialSentinel), wantLevel: "level=ERROR", wantAlert: "alert=reassignment_failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			d := &Daemon{
				rotationService: &syntheticDiscoveryReassigner{err: tt.err},
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
			for _, want := range []string{tt.wantLevel, tt.wantAlert, "provider=codex", "tenant_id=workspace-1"} {
				if !strings.Contains(got, want) {
					t.Fatalf("operator alert missing %q: %s", want, got)
				}
			}
			if strings.Contains(got, credentialSentinel) {
				t.Fatalf("operator alert leaked credential error: %s", got)
			}
		})
	}
}

func TestDispatchAndReportCredentialSessionDiscoveryEventNoopIsDebugOnly(t *testing.T) {
	var logs bytes.Buffer
	d := &Daemon{
		rotationService: &syntheticDiscoveryReassigner{},
		logger:          credentialSessionAlertTestLogger(&logs),
	}
	d.dispatchAndReportCredentialSessionDiscoveryEvent(context.Background(), protocol.Message{
		Type: eventDaemonCredentialSessionDiscovery,
		Payload: syntheticDiscoveryPayload(t, credentialSessionDiscoveryPayload{
			AgentID:     "agent-1",
			AccountID:   "account-current",
			Provider:    "codex",
			WorkspaceID: "workspace-1",
			Status:      "available",
		}),
	}, time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC))

	got := logs.String()
	if !strings.Contains(got, "level=DEBUG") || !strings.Contains(got, "produced no reassignment") {
		t.Fatalf("no-op discovery was not recorded as debug: %s", got)
	}
	if strings.Contains(got, "level=WARN") || strings.Contains(got, "level=ERROR") {
		t.Fatalf("no-op discovery raised an operator alert: %s", got)
	}
}

func credentialSessionAlertTestLogger(output *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(output, &slog.HandlerOptions{
		Level:       slog.LevelDebug,
		ReplaceAttr: redact.SanitizeSlogAttr,
	}))
}
