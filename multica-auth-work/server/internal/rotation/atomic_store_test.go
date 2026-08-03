package rotation

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPGStoreRotateAssignmentAtomicCommitsAssignmentAndAudit(t *testing.T) {
	store, _, agentID, fromID, nextID := setupAtomicRotation(t)
	now := time.Now().UTC().Truncate(time.Second)
	var prepared atomic.Int32

	err := store.RotateAssignmentAtomic(context.Background(), agentID, fromID, nextID, ReasonQuotaReactive, now, func(context.Context) error {
		prepared.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("RotateAssignmentAtomic: %v", err)
	}
	assertAtomicRotationState(t, store, agentID, nextID, 1)
	if prepared.Load() != 1 {
		t.Fatalf("prepare count = %d, want 1", prepared.Load())
	}
}

func TestPGStoreRotateAssignmentAtomicAuditFailureRollsBack(t *testing.T) {
	store, _, agentID, fromID, nextID := setupAtomicRotation(t)

	err := store.RotateAssignmentAtomic(context.Background(), agentID, fromID, nextID, RotationReason("invalid"), time.Now(), func(context.Context) error {
		return nil
	})
	if err == nil {
		t.Fatal("RotateAssignmentAtomic error = nil, want audit constraint failure")
	}
	assertAtomicRotationState(t, store, agentID, fromID, 0)
}

func TestPGStoreRotateAssignmentAtomicStaleSkipsPrepare(t *testing.T) {
	store, _, agentID, fromID, nextID := setupAtomicRotation(t)
	var prepared atomic.Int32

	err := store.RotateAssignmentAtomic(context.Background(), agentID, uuid.NewString(), nextID, ReasonQuotaReactive, time.Now(), func(context.Context) error {
		prepared.Add(1)
		return nil
	})
	if !errors.Is(err, ErrStaleAssignment) {
		t.Fatalf("RotateAssignmentAtomic error = %v, want %v", err, ErrStaleAssignment)
	}
	assertAtomicRotationState(t, store, agentID, fromID, 0)
	if prepared.Load() != 0 {
		t.Fatalf("prepare count = %d, want 0", prepared.Load())
	}
}

func TestPGStoreRotateAssignmentAtomicConcurrentExpectedCurrentHasOneWinner(t *testing.T) {
	store, tenantID, agentID, fromID, nextID := setupAtomicRotation(t)
	otherID := seedPGAccount(t, store.pool, Account{Vendor: "codex", TenantID: tenantID, Priority: 3, Status: StatusAvailable})
	var prepared atomic.Int32
	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, candidateID := range []string{nextID, otherID} {
		candidateID := candidateID
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs <- store.RotateAssignmentAtomic(context.Background(), agentID, fromID, candidateID, ReasonQuotaReactive, time.Now(), func(context.Context) error {
				prepared.Add(1)
				return nil
			})
		}()
	}
	close(start)
	wg.Wait()
	close(errs)

	var success, stale int
	for err := range errs {
		switch {
		case err == nil:
			success++
		case errors.Is(err, ErrStaleAssignment):
			stale++
		default:
			t.Fatalf("unexpected concurrent error: %v", err)
		}
	}
	if success != 1 || stale != 1 {
		t.Fatalf("success/stale = %d/%d, want 1/1", success, stale)
	}
	if prepared.Load() != 1 {
		t.Fatalf("prepare count = %d, want 1", prepared.Load())
	}
	assertAtomicRotationState(t, store, agentID, "", 1)
}

func setupAtomicRotation(t *testing.T) (*PGStore, string, string, string, string) {
	t.Helper()
	store := setupPGStore(t)
	tenantID := uuid.NewString()
	agentID := uuid.NewString()
	t.Cleanup(func() { cleanupPGTenant(t, store.pool, tenantID, agentID) })
	fromID := seedPGAccount(t, store.pool, Account{Vendor: "codex", TenantID: tenantID, Priority: 1, Status: StatusExhausted})
	nextID := seedPGAccount(t, store.pool, Account{Vendor: "codex", TenantID: tenantID, Priority: 2, Status: StatusAvailable})
	if err := store.Assign(context.Background(), agentID, fromID); err != nil {
		t.Fatalf("seed assignment: %v", err)
	}
	return store, tenantID, agentID, fromID, nextID
}

func assertAtomicRotationState(t *testing.T, store *PGStore, agentID, wantAssignment string, wantEvents int) {
	t.Helper()
	gotAssignment, err := store.CurrentAssignment(context.Background(), agentID)
	if err != nil {
		t.Fatalf("CurrentAssignment: %v", err)
	}
	if wantAssignment != "" && gotAssignment != wantAssignment {
		t.Fatalf("assignment = %s, want %s", gotAssignment, wantAssignment)
	}
	var events int
	if err := store.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM rotation_events WHERE agent_id = $1`, agentID).Scan(&events); err != nil {
		t.Fatalf("count rotation events: %v", err)
	}
	if events != wantEvents {
		t.Fatalf("rotation events = %d, want %d", events, wantEvents)
	}
}

func TestNativeRotationRejectsCrossLinkedIdentityBeforeDatabase(t *testing.T) {
	request := nativeRotationTestRequest()
	request.Target.WorkspaceID = uuid.NewString()
	err := request.validate()
	if !errors.Is(err, ErrInvalidNativeIdentity) {
		t.Fatalf("validate error = %v, want ErrInvalidNativeIdentity", err)
	}
}

func TestNativeRotationRejectsAccountLikeReducedIdentity(t *testing.T) {
	request := nativeRotationTestRequest()
	request.Target.RuntimeSessionID = ""
	err := request.validate()
	if !errors.Is(err, ErrInvalidNativeIdentity) {
		t.Fatalf("validate error = %v, want ErrInvalidNativeIdentity", err)
	}
}
