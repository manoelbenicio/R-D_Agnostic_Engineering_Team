package rotation

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestServiceOnExhaustionRotatesHappyPath(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	store := newFakeStore([]Account{
		{AccountID: "current", Vendor: "codex", TenantID: "tenant-1", Priority: 10, Status: StatusLeased},
		{AccountID: "next", Vendor: "codex", TenantID: "tenant-1", Priority: 20, Status: StatusAvailable},
	})
	store.assignments["agent-1"] = "current"
	auth := &fakeAuthenticator{}
	service := NewService(store, fakeDetector{}, auth)

	got, err := service.OnExhaustion(context.Background(), "agent-1", "codex", "tenant-1", ReasonQuotaReactive, now)
	if err != nil {
		t.Fatalf("OnExhaustion: %v", err)
	}
	if got.AccountID != "next" {
		t.Fatalf("rotated account = %s, want next", got.AccountID)
	}
	assertSequence(t, auth.calls, []string{"logout:current", "login:next", "wait:session-next"})
	if assigned := store.assignments["agent-1"]; assigned != "next" {
		t.Fatalf("assignment = %s, want next", assigned)
	}
	if len(store.rotations) != 1 {
		t.Fatalf("rotation count = %d, want 1", len(store.rotations))
	}
	rotation := store.rotations[0]
	if rotation.agentID != "agent-1" || rotation.fromAccountID != "current" || rotation.toAccountID != "next" || rotation.reason != ReasonQuotaReactive || !rotation.at.Equal(now) {
		t.Fatalf("rotation record = %+v, want agent-1 current->next %s at %s", rotation, ReasonQuotaReactive, now)
	}
}

func TestServiceOnExhaustionMarksFailedLoginDegradedAndTriesNext(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	store := newFakeStore([]Account{
		{AccountID: "current", Vendor: "codex", TenantID: "tenant-1", Priority: 10, Status: StatusLeased},
		{AccountID: "first", Vendor: "codex", TenantID: "tenant-1", Priority: 20, Status: StatusAvailable},
		{AccountID: "second", Vendor: "codex", TenantID: "tenant-1", Priority: 30, Status: StatusAvailable},
	})
	store.assignments["agent-1"] = "current"
	auth := &fakeAuthenticator{loginErrs: map[string]error{"first": errors.New("login failed")}}
	service := NewService(store, fakeDetector{}, auth)

	got, err := service.OnExhaustion(context.Background(), "agent-1", "codex", "tenant-1", ReasonQuotaReactive, now)
	if err != nil {
		t.Fatalf("OnExhaustion: %v", err)
	}
	if got.AccountID != "second" {
		t.Fatalf("rotated account = %s, want second", got.AccountID)
	}
	if status := store.accounts["first"].Status; status != StatusDegraded {
		t.Fatalf("failed account status = %s, want degraded", status)
	}
	assertSequence(t, auth.calls, []string{"logout:current", "login:first", "login:second", "wait:session-second"})
	if assigned := store.assignments["agent-1"]; assigned != "second" {
		t.Fatalf("assignment = %s, want second", assigned)
	}
}

func assertSequence(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("call count = %d, want %d: got %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("call[%d] = %s, want %s: got %v", i, got[i], want[i], got)
		}
	}
}

type fakeDetector struct{}

func (fakeDetector) Detect(vendor, screenText string, httpStatus int) DetectionResult {
	return DetectionResult{}
}

type rotationRecord struct {
	agentID       string
	fromAccountID string
	toAccountID   string
	reason        RotationReason
	at            time.Time
}

type fakeStore struct {
	accounts    map[string]Account
	order       []string
	assignments map[string]string
	rotations   []rotationRecord
}

func newFakeStore(accounts []Account) *fakeStore {
	store := &fakeStore{
		accounts:    map[string]Account{},
		assignments: map[string]string{},
	}
	for _, account := range accounts {
		store.accounts[account.AccountID] = account
		store.order = append(store.order, account.AccountID)
	}
	return store
}

func (s *fakeStore) ListAccounts(ctx context.Context, vendor, tenantID string) ([]Account, error) {
	var out []Account
	for _, accountID := range s.order {
		account := s.accounts[accountID]
		if account.Vendor == vendor && account.TenantID == tenantID {
			out = append(out, account)
		}
	}
	return out, nil
}

func (s *fakeStore) GetAccount(ctx context.Context, accountID string) (Account, error) {
	account, ok := s.accounts[accountID]
	if !ok {
		return Account{}, errors.New("account not found")
	}
	return account, nil
}

func (s *fakeStore) UpdateAccountStatus(ctx context.Context, accountID string, status AccountStatus, cooldownUntil *time.Time) error {
	account, ok := s.accounts[accountID]
	if !ok {
		return errors.New("account not found")
	}
	account.Status = status
	account.CooldownUntil = cooldownUntil
	s.accounts[accountID] = account
	return nil
}

func (s *fakeStore) RecordUsage(ctx context.Context, accountID string, tokensUsed int64, windowStart time.Time) error {
	account, ok := s.accounts[accountID]
	if !ok {
		return errors.New("account not found")
	}
	account.TokensUsed = tokensUsed
	account.WindowStart = &windowStart
	s.accounts[accountID] = account
	return nil
}

func (s *fakeStore) Assign(ctx context.Context, agentID, accountID string) error {
	if _, ok := s.accounts[accountID]; !ok {
		return errors.New("account not found")
	}
	s.assignments[agentID] = accountID
	return nil
}

func (s *fakeStore) CurrentAssignment(ctx context.Context, agentID string) (string, error) {
	return s.assignments[agentID], nil
}

func (s *fakeStore) RecordRotation(ctx context.Context, agentID, fromAccountID, toAccountID string, reason RotationReason, at time.Time) error {
	s.rotations = append(s.rotations, rotationRecord{
		agentID:       agentID,
		fromAccountID: fromAccountID,
		toAccountID:   toAccountID,
		reason:        reason,
		at:            at,
	})
	return nil
}

type fakeAuthenticator struct {
	calls     []string
	loginErrs map[string]error
}

func (a *fakeAuthenticator) Login(ctx context.Context, acc Account) (string, error) {
	a.calls = append(a.calls, "login:"+acc.AccountID)
	if err := a.loginErrs[acc.AccountID]; err != nil {
		return "", err
	}
	return "session-" + acc.AccountID, nil
}

func (a *fakeAuthenticator) Logout(ctx context.Context, acc Account) error {
	a.calls = append(a.calls, "logout:"+acc.AccountID)
	return nil
}

func (a *fakeAuthenticator) WaitAuthenticated(ctx context.Context, sessionID string, timeout time.Duration) (bool, error) {
	a.calls = append(a.calls, "wait:"+sessionID)
	return true, nil
}
