package daemon

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/rotation"
)

func TestCredentialIsolationTask53AutomaticRotation(t *testing.T) {
	now := time.Date(2026, 7, 18, 15, 0, 0, 0, time.UTC)

	t.Run("exhausted active account rotates within provider and tenant", func(t *testing.T) {
		store := newProducerSyntheticStore([]rotation.Account{
			{AccountID: "current", Vendor: "codex", TenantID: "tenant-1", Priority: 10, Status: rotation.StatusLeased},
			{AccountID: "wrong-provider", Vendor: "kiro", TenantID: "tenant-1", Priority: 1, Status: rotation.StatusAvailable},
			{AccountID: "wrong-tenant", Vendor: "codex", TenantID: "tenant-2", Priority: 2, Status: rotation.StatusAvailable},
			{AccountID: "replacement", Vendor: "codex", TenantID: "tenant-1", Priority: 20, Status: rotation.StatusAvailable},
		})
		store.assignments["agent-1"] = "current"
		auth := &producerSyntheticAuthenticator{}
		service := rotation.NewService(store, producerNoopDetector{}, auth)
		emitter := &producerLoopbackEmitter{daemon: &Daemon{rotationService: service}, now: now}
		producer := NewCredentialSessionDiscoveryProducer(emitter)

		emitted, err := producer.Produce(context.Background(), CredentialSessionDiscoveryObservation{
			AgentID: "agent-1", AccountID: "current", Provider: "codex", WorkspaceID: "tenant-1", Status: "exhausted",
		}, now)
		if err != nil || !emitted {
			t.Fatalf("Produce = (%v, %v), want true/nil", emitted, err)
		}
		if got := store.assignment("agent-1"); got != "replacement" {
			t.Fatalf("assignment = %q, want replacement", got)
		}
		if got := store.accountStatus("current"); got != rotation.StatusExhausted {
			t.Fatalf("current status = %q, want exhausted", got)
		}
		if got := store.accountStatus("wrong-provider"); got != rotation.StatusAvailable {
			t.Fatalf("wrong-provider status = %q, want available", got)
		}
		if got := store.accountStatus("wrong-tenant"); got != rotation.StatusAvailable {
			t.Fatalf("wrong-tenant status = %q, want available", got)
		}
		wantCalls := []string{"logout:current", "login:replacement", "wait:session-replacement"}
		if got := auth.snapshotCalls(); fmt.Sprint(got) != fmt.Sprint(wantCalls) {
			t.Fatalf("auth calls = %v, want %v", got, wantCalls)
		}
		if emitter.count() != 1 {
			t.Fatalf("event count = %d, want 1", emitter.count())
		}
		if got := task53RotationCount(store); got != 1 {
			t.Fatalf("rotation records = %d, want 1", got)
		}
	})

	t.Run("no same-provider same-tenant candidate fails closed", func(t *testing.T) {
		store := newProducerSyntheticStore([]rotation.Account{
			{AccountID: "current", Vendor: "codex", TenantID: "tenant-1", Priority: 10, Status: rotation.StatusLeased},
			{AccountID: "wrong-provider", Vendor: "kiro", TenantID: "tenant-1", Priority: 1, Status: rotation.StatusAvailable},
			{AccountID: "wrong-tenant", Vendor: "codex", TenantID: "tenant-2", Priority: 2, Status: rotation.StatusAvailable},
		})
		store.assignments["agent-1"] = "current"
		auth := &producerSyntheticAuthenticator{}
		service := rotation.NewService(store, producerNoopDetector{}, auth)
		emitter := &producerLoopbackEmitter{daemon: &Daemon{rotationService: service}, now: now}
		producer := NewCredentialSessionDiscoveryProducer(emitter)

		emitted, err := producer.Produce(context.Background(), CredentialSessionDiscoveryObservation{
			AgentID: "agent-1", AccountID: "current", Provider: "codex", WorkspaceID: "tenant-1", Status: "expired",
		}, now)
		if emitted || !errors.Is(err, rotation.ErrNoAccountAvailable) {
			t.Fatalf("Produce = (%v, %v), want false/ErrNoAccountAvailable", emitted, err)
		}
		if got := store.assignment("agent-1"); got != "current" {
			t.Fatalf("assignment = %q, want current", got)
		}
		if got := store.accountStatus("current"); got != rotation.StatusExhausted {
			t.Fatalf("current status = %q, want exhausted signal preserved", got)
		}
		if got := auth.snapshotCalls(); len(got) != 0 {
			t.Fatalf("auth calls = %v, want none before replacement selection", got)
		}
		if got := store.accountStatus("wrong-provider"); got != rotation.StatusAvailable {
			t.Fatalf("wrong-provider status = %q, want available", got)
		}
		if got := store.accountStatus("wrong-tenant"); got != rotation.StatusAvailable {
			t.Fatalf("wrong-tenant status = %q, want available", got)
		}
		if emitter.count() != 1 {
			t.Fatalf("dispatched event count = %d, want 1", emitter.count())
		}
		if got := task53RotationCount(store); got != 0 {
			t.Fatalf("rotation records = %d, want 0", got)
		}
	})
}

func task53RotationCount(store *producerSyntheticStore) int {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.rotations
}
