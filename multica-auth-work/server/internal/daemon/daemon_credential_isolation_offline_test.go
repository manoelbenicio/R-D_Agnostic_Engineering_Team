package daemon

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/execenv"
	"github.com/multica-ai/multica/server/internal/rotation"
)

func TestCredentialIsolationPerVendorOffline(t *testing.T) {
	for index, vendor := range allIsolationVendors {
		vendor := vendor
		foreignVendor := allIsolationVendors[(index+1)%len(allIsolationVendors)]
		t.Run(vendor, func(t *testing.T) {
			markerA := credentialMarker(vendor, "OFFLINE-A")
			markerB := credentialMarker(vendor, "OFFLINE-B")
			homeA := setupVendorAccountHome(t, vendor, markerA)
			homeB := setupVendorAccountHome(t, vendor, markerB)
			store := &offlineCredentialIsolationStore{
				accounts: map[string]rotation.Account{
					"account-a": {AccountID: "account-a", Vendor: vendor, HomeDir: homeA},
					"account-b": {AccountID: "account-b", Vendor: vendor, HomeDir: homeB},
					"account-foreign": {
						AccountID: "account-foreign", Vendor: foreignVendor, HomeDir: t.TempDir(),
					},
					"account-empty-home": {AccountID: "account-empty-home", Vendor: vendor},
				},
				assignments: map[string]string{
					"agent-a":          "account-a",
					"agent-b":          "account-b",
					"agent-foreign":    "account-foreign",
					"agent-empty-home": "account-empty-home",
					"agent-missing":    "account-missing",
				},
			}

			var output bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))
			daemon := &Daemon{rotationStore: store, logger: logger}
			ctx := context.Background()

			resolvedA, err := daemon.credentialAccountHomeForTask(ctx, Task{AgentID: "agent-a"}, vendor, logger)
			if err != nil || resolvedA != homeA {
				t.Fatal("account A did not resolve to its assigned vendor home")
			}
			resolvedB, err := daemon.credentialAccountHomeForTask(ctx, Task{AgentID: "agent-b"}, vendor, logger)
			if err != nil || resolvedB != homeB || resolvedA == resolvedB {
				t.Fatal("account B did not resolve to a distinct assigned vendor home")
			}

			envA := prepareIsolatedEnv(t, vendor, resolvedA, logger)
			defer envA.Cleanup(true)
			envB := prepareIsolatedEnv(t, vendor, resolvedB, logger)
			defer envB.Cleanup(true)
			assertOfflineCredentialEnvironment(t, vendor, envA, envB, markerA, markerB)
			assertOfflineCredentialEnvironment(t, vendor, envB, envA, markerB, markerA)

			for _, agentID := range []string{"agent-unassigned", "agent-foreign", "agent-empty-home", "agent-missing"} {
				home, gateErr := daemon.credentialAccountHomeForTask(ctx, Task{AgentID: agentID}, vendor, logger)
				if gateErr == nil || home != "" {
					t.Fatal("credential isolation did not fail closed for an invalid assignment")
				}
				if strings.Contains(gateErr.Error(), markerA) || strings.Contains(gateErr.Error(), markerB) {
					t.Fatal("credential isolation error exposed raw credential material")
				}
			}

			if strings.Contains(output.String(), markerA) || strings.Contains(output.String(), markerB) {
				t.Fatal("credential isolation logs exposed raw credential material")
			}
		})
	}

	daemon := &Daemon{}
	home, err := daemon.credentialAccountHomeForTask(context.Background(), Task{AgentID: "agent"}, allIsolationVendors[0], slog.Default())
	if err == nil || home != "" {
		t.Fatal("credential isolation accepted an unavailable store")
	}
}

func assertOfflineCredentialEnvironment(t *testing.T, vendor string, own, other *execenv.Environment, ownMarker, otherMarker string) {
	t.Helper()
	credentialEnv, err := isolatedCredentialEnv(vendor, "offline-assigned-home", own)
	if err != nil {
		t.Fatal("isolated credential environment is incomplete")
	}
	otherEnv := other.CredentialEnv(vendor)
	for key, value := range credentialEnv {
		if strings.Contains(value, ownMarker) || strings.Contains(value, otherMarker) {
			t.Fatal("child environment exposed raw credential material")
		}
		if key != "CLINE_SANDBOX" && value == otherEnv[key] {
			t.Fatal("two accounts share a provider credential environment value")
		}
	}

	credentialPath := vendorCredentialInEnv(own, vendor)
	content, err := os.ReadFile(credentialPath)
	if err != nil {
		t.Fatal("isolated credential fixture is unreadable")
	}
	if !bytes.Contains(content, []byte(ownMarker)) || bytes.Contains(content, []byte(otherMarker)) {
		t.Fatal("isolated credential content crossed account boundaries")
	}
}

type offlineCredentialIsolationStore struct {
	accounts    map[string]rotation.Account
	assignments map[string]string
}

func (s *offlineCredentialIsolationStore) ListAccounts(context.Context, string, string) ([]rotation.Account, error) {
	return nil, nil
}

func (s *offlineCredentialIsolationStore) GetAccount(_ context.Context, accountID string) (rotation.Account, error) {
	account, ok := s.accounts[accountID]
	if !ok {
		return rotation.Account{}, errors.New("offline account unavailable")
	}
	return account, nil
}

func (*offlineCredentialIsolationStore) UpdateAccountStatus(context.Context, string, rotation.AccountStatus, *time.Time) error {
	return nil
}

func (*offlineCredentialIsolationStore) RecordUsage(context.Context, string, int64, time.Time) error {
	return nil
}

func (s *offlineCredentialIsolationStore) Assign(_ context.Context, agentID, accountID string) error {
	s.assignments[agentID] = accountID
	return nil
}

func (s *offlineCredentialIsolationStore) CurrentAssignment(_ context.Context, agentID string) (string, error) {
	accountID, ok := s.assignments[agentID]
	if !ok {
		return "", rotation.ErrNoAssignment
	}
	return accountID, nil
}

func (*offlineCredentialIsolationStore) RecordRotation(context.Context, string, string, string, rotation.RotationReason, time.Time) error {
	return nil
}
