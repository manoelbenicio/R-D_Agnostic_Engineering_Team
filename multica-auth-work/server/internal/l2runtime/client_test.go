package l2runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestHealthAndReadyFailClosed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		path    string
		body    any
		call    func(context.Context, *Client) error
		wantErr string
	}{
		{
			name: "health accepts alive sidecar",
			path: "/healthz",
			body: HealthResponse{ContractVersion: ContractVersion, Status: "alive", Sidecar: SidecarBuild{Name: "prodex"}},
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Health(ctx)
				return err
			},
		},
		{
			name: "health rejects wrong contract",
			path: "/healthz",
			body: HealthResponse{ContractVersion: "rpp.l2.v2", Status: "alive"},
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Health(ctx)
				return err
			},
			wantErr: "l2 health failed closed",
		},
		{
			name: "ready accepts passing checks",
			path: "/readyz",
			body: ReadyResponse{ContractVersion: ContractVersion, Status: "ready", Checks: []ReadyCheck{{Name: "kill_switch", Status: "pass"}}},
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Ready(ctx)
				return err
			},
		},
		{
			name: "ready rejects failing kill switch check",
			path: "/readyz",
			body: ReadyResponse{ContractVersion: ContractVersion, Status: "ready", Checks: []ReadyCheck{{Name: "kill_switch", Status: "fail"}}},
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Ready(ctx)
				return err
			},
			wantErr: "l2 readiness check \"kill_switch\" failed closed",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := newJSONClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tt.path {
					t.Fatalf("path = %s, want %s", r.URL.Path, tt.path)
				}
				_ = json.NewEncoder(w).Encode(tt.body)
			})
			err := tt.call(context.Background(), client)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("call: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("call error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestApplyKillSwitchConfirmsAppliedState(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	client := newJSONClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/v1/killswitch/apply" {
			t.Fatalf("path = %s, want /v1/killswitch/apply", r.URL.Path)
		}
		var req KillSwitch
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.ContractVersion != ContractVersion || req.Feature != "smart_context" || req.State != "disabled" {
			t.Fatalf("unexpected request: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(KillSwitchResponse{
			ContractVersion: ContractVersion,
			RequestID:       req.RequestID,
			Applied:         true,
			EffectiveAt:     "next_request",
		})
	})

	resp, err := client.ApplyKillSwitch(context.Background(), validKillSwitch())
	if err != nil {
		t.Fatalf("ApplyKillSwitch: %v", err)
	}
	if resp.EffectiveAt != "next_request" {
		t.Fatalf("EffectiveAt = %q, want next_request", resp.EffectiveAt)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("requests = %d, want 1", got)
	}
}

func TestApplyKillSwitchFailsClosed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		req       KillSwitch
		response  KillSwitchResponse
		wantLocal bool
	}{
		{
			name: "invalid feature rejected before HTTP",
			req: func() KillSwitch {
				req := validKillSwitch()
				req.Feature = "unknown_feature"
				return req
			}(),
			wantLocal: true,
		},
		{
			name: "unapplied response rejected",
			req:  validKillSwitch(),
			response: KillSwitchResponse{
				ContractVersion: ContractVersion,
				Applied:         false,
				EffectiveAt:     "next_request",
			},
		},
		{
			name: "unconfirmed effective_at rejected",
			req:  validKillSwitch(),
			response: KillSwitchResponse{
				ContractVersion: ContractVersion,
				Applied:         true,
				EffectiveAt:     "",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var requests atomic.Int32
			client := newJSONClient(t, func(w http.ResponseWriter, _ *http.Request) {
				requests.Add(1)
				_ = json.NewEncoder(w).Encode(tt.response)
			})
			_, err := client.ApplyKillSwitch(context.Background(), tt.req)
			if err == nil {
				t.Fatal("ApplyKillSwitch error = nil, want fail-closed error")
			}
			if tt.wantLocal {
				if !errors.Is(err, ErrInvalidControlRequest) {
					t.Fatalf("ApplyKillSwitch error = %v, want ErrInvalidControlRequest", err)
				}
				if got := requests.Load(); got != 0 {
					t.Fatalf("requests = %d, want 0 for local validation failure", got)
				}
			}
		})
	}
}

func TestRegisterAccountsInvalidAuthFailsClosedBeforeHTTP(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	client := newJSONClient(t, func(http.ResponseWriter, *http.Request) {
		requests.Add(1)
	})
	req := validAccountRegistration()
	req.Profiles[0].AuthMode = "shared_bearer_token"

	_, err := client.RegisterAccounts(context.Background(), req)
	if !errors.Is(err, ErrInvalidControlRequest) {
		t.Fatalf("RegisterAccounts error = %v, want ErrInvalidControlRequest", err)
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("requests = %d, want 0 for invalid auth", got)
	}
}

func TestRegisterAccountsRejectsInvalidProfileResponse(t *testing.T) {
	t.Parallel()

	client := newJSONClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(RegisterAccountsResponse{
			ContractVersion:  ContractVersion,
			RejectedProfiles: []string{"profile-1"},
		})
	})
	_, err := client.RegisterAccounts(context.Background(), validAccountRegistration())
	if err == nil || !strings.Contains(err.Error(), "l2 account registration failed closed") {
		t.Fatalf("RegisterAccounts error = %v, want fail-closed rejection", err)
	}
}

func TestStreamEventsValidatesBeforeHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		event   string
		wantErr error
	}{
		{
			name:    "unknown event_type",
			event:   strings.Replace(validSelectionRuntimeEvent(), `"event_type":"selection"`, `"event_type":"rotate_now"`, 1),
			wantErr: ErrInvalidEvent,
		},
		{
			name:    "wrong contract_version",
			event:   strings.Replace(validSelectionRuntimeEvent(), `"contract_version":"rpp.l2.v1"`, `"contract_version":"rpp.l2.v2"`, 1),
			wantErr: ErrInvalidEvent,
		},
		{
			name:    "secrets present",
			event:   strings.Replace(validSelectionRuntimeEvent(), `"secrets_present":false`, `"secrets_present":true`, 1),
			wantErr: ErrSecretEvent,
		},
		{
			name:    "missing per event type runtime_request_id",
			event:   strings.Replace(validSelectionRuntimeEvent(), `"runtime_request_id":"runtime-request-1",`, ``, 1),
			wantErr: ErrInvalidEvent,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, streamURL := newRuntimeEventStreamClient(t, tt.event)
			var handled atomic.Int32
			err := client.StreamEvents(context.Background(), streamURL, func(context.Context, RuntimeEvent) error {
				handled.Add(1)
				return nil
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("StreamEvents error = %v, want %v", err, tt.wantErr)
			}
			if got := handled.Load(); got != 0 {
				t.Fatalf("handler calls = %d, want 0 for invalid event", got)
			}
		})
	}
}

func TestStreamEventsAcceptsSchemaRequiredSelectionEvent(t *testing.T) {
	t.Parallel()

	client, streamURL := newRuntimeEventStreamClient(t, validSelectionRuntimeEvent())
	var handled atomic.Int32
	err := client.StreamEvents(context.Background(), streamURL, func(_ context.Context, event RuntimeEvent) error {
		handled.Add(1)
		if event.ContractVersion != ContractVersion || event.EventType != "selection" || event.RuntimeRequestID != "runtime-request-1" {
			t.Fatalf("unexpected event: %+v", event)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("StreamEvents: %v", err)
	}
	if got := handled.Load(); got != 1 {
		t.Fatalf("handler calls = %d, want 1", got)
	}
}

func newRuntimeEventStreamClient(t *testing.T, event string) (*Client, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = io.WriteString(w, event+"\n")
	}))
	t.Cleanup(srv.Close)
	client, err := NewClient(srv.URL, "test-token", time.Second)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client, srv.URL + "/v1/events/stream"
}

func validSelectionRuntimeEvent() string {
	return fmt.Sprintf(`{"contract_version":"%s","event_id":"event-0001","event_type":"selection","occurred_at":"2026-07-04T19:55:00Z","severity":"info","producer":{"plane":"rust_l2","component":"event_stream","version":"0.246.0"},"tenant_id":"tenant-1","session_id":"session-1","runtime_request_id":"runtime-request-1","selection":{"decision_phase":"pre_commit","selected_profile_id":"profile-1","selected_provider":"codex","reason":"fresh_request","committed":true},"redaction":{"secrets_present":false,"scrubber_version":"test"}}`, ContractVersion)
}

func newJSONClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client, err := NewClient(srv.URL, "test-token", time.Second)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

func validKillSwitch() KillSwitch {
	return KillSwitch{
		ControlEnvelope: ControlEnvelope{
			RequestID: "req-kill-1",
			TenantID:  "tenant-1",
		},
		Scope: KillSwitchScope{
			Provider:  "codex",
			ProfileID: "profile-1",
		},
		Feature:     "smart_context",
		State:       "disabled",
		Reason:      "operator_guardrail",
		EffectiveAt: "next_request",
	}
}

func validAccountRegistration() AccountRegistration {
	return AccountRegistration{
		ControlEnvelope: ControlEnvelope{
			RequestID: "req-accounts-1",
			TenantID:  "tenant-1",
		},
		Profiles: []AccountProfile{{
			ProfileID:     "profile-1",
			Provider:      "codex",
			ProfileHome:   "$PRODEX_HOME/profiles/profile-1",
			AuthMode:      "oauth_profile",
			Status:        "approved",
			CapabilityRef: "codex.oauth_profile.v1",
		}},
	}
}
