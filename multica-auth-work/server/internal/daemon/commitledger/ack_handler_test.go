package commitledger

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func TestAckHandler_ProcessOutputAck(t *testing.T) {
	reg := NewLedgerRegistry()
	l := mustNew(t, "task-1")
	l.RecordToolUse("call-1", 1)
	l.RecordToolUse("call-2", 2)
	reg.Register("corr-1", l)

	handler := NewAckHandler(reg, slog.Default())
	handler.ProcessOutputAck("corr-1", 1)
	if l.OutputPersistedSeq() != 1 {
		t.Errorf("expected 1, got %d", l.OutputPersistedSeq())
	}

	handler.ProcessOutputAck("corr-1", 2)
	if l.OutputPersistedSeq() != 2 {
		t.Errorf("expected 2, got %d", l.OutputPersistedSeq())
	}
}

func TestAckHandler_UnknownCorrelation(t *testing.T) {
	reg := NewLedgerRegistry()
	handler := NewAckHandler(reg, slog.Default())
	// Should not panic
	handler.ProcessOutputAck("nonexistent", 5)
}

func TestAckHandler_BackwardAck_NoOp(t *testing.T) {
	reg := NewLedgerRegistry()
	l := mustNew(t, "task-1")
	l.AcknowledgeOutputPersisted(5)
	reg.Register("corr-1", l)

	handler := NewAckHandler(reg, slog.Default())
	handler.ProcessOutputAck("corr-1", 3) // backward
	if l.OutputPersistedSeq() != 5 {
		t.Errorf("expected 5 (no regress), got %d", l.OutputPersistedSeq())
	}
}

func TestAckHandler_ZeroSeq_NoOp(t *testing.T) {
	reg := NewLedgerRegistry()
	l := mustNew(t, "task-1")
	reg.Register("corr-1", l)

	handler := NewAckHandler(reg, slog.Default())
	handler.ProcessOutputAck("corr-1", 0)
	if l.OutputPersistedSeq() != 0 {
		t.Errorf("expected 0, got %d", l.OutputPersistedSeq())
	}
}

func TestDrainOwner_SingleStart(t *testing.T) {
	l := mustNew(t, "task-1")
	owner := NewDrainOwner(l, 2*time.Second, nil)

	if !owner.Start() {
		t.Fatal("first Start should succeed")
	}
	if owner.Start() {
		t.Fatal("second Start should fail (single-owner)")
	}
}

func TestDrainOwner_JoinSuccess(t *testing.T) {
	l := mustNew(t, "task-1")
	owner := NewDrainOwner(l, 2*time.Second, nil)
	owner.Start()

	go func() {
		time.Sleep(10 * time.Millisecond)
		owner.Finish()
	}()

	err := owner.JoinOrTimeout(context.Background())
	if err != nil {
		t.Fatalf("expected successful join, got %v", err)
	}
}

func TestDrainOwner_FinishIdempotent(t *testing.T) {
	l := mustNew(t, "task-1")
	owner := NewDrainOwner(l, 2*time.Second, nil)
	owner.Start()

	// Multiple Finish calls should not panic (sync.Once)
	owner.Finish()
	owner.Finish()
	owner.Finish()
}

func TestDrainOwner_JoinTimeout_CancelsAndMarksAmbiguous(t *testing.T) {
	l := mustNew(t, "task-1")
	l.RecordToolUse("call-1", 1)
	l.RecordToolUse("call-2", 2)

	cancelled := false
	cancelFn := func() { cancelled = true }

	owner := NewDrainOwner(l, 50*time.Millisecond, cancelFn)
	owner.Start()
	// Don't call Finish — simulate hung drain

	err := owner.JoinOrTimeout(context.Background())
	if err == nil {
		t.Fatal("expected timeout error")
	}

	// Verify cancel was called BEFORE marking ambiguous
	if !cancelled {
		t.Error("cancelFn should have been called")
	}

	// Verify unresolved entries were marked ambiguous
	tok1 := l.TokenizeCallID("call-1")
	tok2 := l.TokenizeCallID("call-2")
	if l.Lookup(tok1) != CommitAmbiguous {
		t.Errorf("call-1 should be ambiguous after timeout, got %s", l.Lookup(tok1))
	}
	if l.Lookup(tok2) != CommitAmbiguous {
		t.Errorf("call-2 should be ambiguous after timeout, got %s", l.Lookup(tok2))
	}
}

func TestDrainOwner_ContextCancel_CancelsAndMarksAmbiguous(t *testing.T) {
	l := mustNew(t, "task-1")
	l.RecordToolUse("call-1", 1)

	cancelled := false
	cancelFn := func() { cancelled = true }

	owner := NewDrainOwner(l, 10*time.Second, cancelFn)
	owner.Start()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately

	err := owner.JoinOrTimeout(ctx)
	if err == nil {
		t.Fatal("expected context error")
	}

	if !cancelled {
		t.Error("cancelFn should have been called")
	}

	tok1 := l.TokenizeCallID("call-1")
	if l.Lookup(tok1) != CommitAmbiguous {
		t.Errorf("should be ambiguous after ctx cancel, got %s", l.Lookup(tok1))
	}
}

func TestDrainOwner_DefaultJoinBudget(t *testing.T) {
	l := mustNew(t, "task-1")
	owner := NewDrainOwner(l, 0, nil)
	if owner.joinBudget != 5*time.Second {
		t.Errorf("expected default 5s, got %v", owner.joinBudget)
	}
}

func TestDrainOwner_DoneChannel(t *testing.T) {
	l := mustNew(t, "task-1")
	owner := NewDrainOwner(l, 2*time.Second, nil)
	owner.Start()

	select {
	case <-owner.Done():
		t.Fatal("should not be done yet")
	default:
	}

	owner.Finish()

	select {
	case <-owner.Done():
		// expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("should be done after Finish")
	}
}
