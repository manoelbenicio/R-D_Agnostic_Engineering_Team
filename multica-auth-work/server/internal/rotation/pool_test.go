package rotation

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPoolSelectNextByPriority(t *testing.T) {
	store := newFakeStore([]Account{
		{AccountID: "antigravity", Vendor: "codex", TenantID: "tenant-1", Priority: 30, Status: StatusAvailable},
		{AccountID: "opus", Vendor: "codex", TenantID: "tenant-1", Priority: 20, Status: StatusAvailable},
		{AccountID: "codex", Vendor: "codex", TenantID: "tenant-1", Priority: 10, Status: StatusAvailable},
	})

	got, err := NewPool(store).SelectNext(context.Background(), "codex", "tenant-1", time.Now())
	if err != nil {
		t.Fatalf("SelectNext: %v", err)
	}
	if got.AccountID != "codex" {
		t.Fatalf("selected account = %s, want codex", got.AccountID)
	}
}

func TestPoolSelectNextRespectsCooldown(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	future := now.Add(time.Minute)
	past := now.Add(-time.Minute)
	store := newFakeStore([]Account{
		{AccountID: "codex", Vendor: "codex", TenantID: "tenant-1", Priority: 10, Status: StatusCooldown, CooldownUntil: &future},
		{AccountID: "opus", Vendor: "codex", TenantID: "tenant-1", Priority: 20, Status: StatusAvailable},
	})

	got, err := NewPool(store).SelectNext(context.Background(), "codex", "tenant-1", now)
	if err != nil {
		t.Fatalf("SelectNext before cooldown: %v", err)
	}
	if got.AccountID != "opus" {
		t.Fatalf("selected before cooldown = %s, want opus", got.AccountID)
	}

	store.accounts["codex"] = Account{AccountID: "codex", Vendor: "codex", TenantID: "tenant-1", Priority: 10, Status: StatusCooldown, CooldownUntil: &past}
	got, err = NewPool(store).SelectNext(context.Background(), "codex", "tenant-1", now)
	if err != nil {
		t.Fatalf("SelectNext after cooldown: %v", err)
	}
	if got.AccountID != "codex" {
		t.Fatalf("selected after cooldown = %s, want codex", got.AccountID)
	}
}

func TestPoolSelectNextAllExhausted(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	future := now.Add(time.Minute)
	store := newFakeStore([]Account{
		{AccountID: "exhausted", Vendor: "codex", TenantID: "tenant-1", Priority: 10, Status: StatusExhausted},
		{AccountID: "cooldown", Vendor: "codex", TenantID: "tenant-1", Priority: 20, Status: StatusCooldown, CooldownUntil: &future},
		{AccountID: "degraded", Vendor: "codex", TenantID: "tenant-1", Priority: 30, Status: StatusDegraded},
	})

	_, err := NewPool(store).SelectNext(context.Background(), "codex", "tenant-1", now)
	if !errors.Is(err, ErrNoAccountAvailable) {
		t.Fatalf("SelectNext error = %v, want ErrNoAccountAvailable", err)
	}
}
