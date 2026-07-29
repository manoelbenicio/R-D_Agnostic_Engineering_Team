package commitledger

import (
	"errors"
	"log/slog"
	"testing"
)

func TestReplayGate_NilLedger_Blocked(t *testing.T) {
	result := ReplayGate(nil)
	if result.Decision != ReplayBlocked {
		t.Errorf("nil ledger should block, got %s", result.Decision)
	}
}

func TestReplayGate_FailClosed_Blocked(t *testing.T) {
	l := NewFailClosed("task-1")
	result := ReplayGate(l)
	if result.Decision != ReplayBlocked {
		t.Errorf("fail-closed should block, got %s", result.Decision)
	}
	if !result.FailClosed {
		t.Error("expected FailClosed flag")
	}
}

func TestReplayGate_EmptyLedger_Allowed(t *testing.T) {
	l := mustNew(t, "task-1")
	result := ReplayGate(l)
	if result.Decision != ReplayAllowed {
		t.Errorf("empty ledger (no tool activity) should allow, got %s", result.Decision)
	}
}

func TestReplayGate_AnyToolUse_Blocked(t *testing.T) {
	l := mustNew(t, "task-1")
	l.RecordToolUse("call-1", 1) // just tool_use, no result

	result := ReplayGate(l)
	if result.Decision != ReplayBlocked {
		t.Errorf("any tool_use should block replay, got %s", result.Decision)
	}
}

func TestReplayGate_AllDefinite_Blocked(t *testing.T) {
	l := mustNew(t, "task-1")
	tok, _ := l.RecordToolUse("call-1", 1)
	l.RecordToolResult(tok) // committed

	result := ReplayGate(l)
	if result.Decision != ReplayBlocked {
		t.Errorf("definite commits should BLOCK replay (side effects occurred), got %s", result.Decision)
	}
}

func TestReplayGate_WithAmbiguous_Blocked(t *testing.T) {
	l := mustNew(t, "task-1")
	tok, _ := l.RecordToolUse("call-1", 1)
	l.MarkAmbiguous(tok)

	result := ReplayGate(l)
	if result.Decision != ReplayBlocked {
		t.Errorf("ambiguous should block, got %s", result.Decision)
	}
}

func TestReplayGate_Saturated_Blocked(t *testing.T) {
	l := mustNew(t, "task-1")
	// Force saturation via seq violation
	l.RecordToolUse("call-1", 10)
	l.RecordToolUse("call-2", 5) // out of order → saturates

	result := ReplayGate(l)
	if result.Decision != ReplayBlocked {
		t.Errorf("saturated should block, got %s", result.Decision)
	}
}

func TestLedgerRegistry_Basic(t *testing.T) {
	reg := NewLedgerRegistry()
	l := mustNew(t, "task-1")
	reg.Register("corr-1", l)

	got := reg.Get("corr-1")
	if got != l {
		t.Error("expected same ledger from registry")
	}
	if reg.Get("nonexistent") != nil {
		t.Error("expected nil for nonexistent")
	}

	reg.Unregister("corr-1")
	if reg.Get("corr-1") != nil {
		t.Error("expected nil after unregister")
	}
}

func TestLedgerRegistry_CheckReplayAllowed_NoLedger_Blocked(t *testing.T) {
	reg := NewLedgerRegistry()
	err := reg.CheckReplayAllowed("missing")
	if err == nil {
		t.Fatal("missing ledger should block")
	}
	if !errors.Is(err, ErrReplayBlocked) {
		t.Errorf("expected ErrReplayBlocked, got: %v", err)
	}
}

func TestLedgerRegistry_CheckReplayAllowed_Empty_Allowed(t *testing.T) {
	reg := NewLedgerRegistry()
	l := mustNew(t, "task-1")
	reg.Register("corr-1", l)

	err := reg.CheckReplayAllowed("corr-1")
	if err != nil {
		t.Fatalf("empty ledger should allow: %v", err)
	}
}

func TestLedgerRegistry_CheckReplayAllowed_ToolUse_Blocked(t *testing.T) {
	reg := NewLedgerRegistry()
	l := mustNew(t, "task-1")
	l.RecordToolUse("call-1", 1)
	reg.Register("corr-1", l)

	err := reg.CheckReplayAllowed("corr-1")
	if err == nil {
		t.Fatal("tool_use should block")
	}
}

func TestReplayGateHook_NilHook_FailClosed(t *testing.T) {
	err := CheckOrAllow(nil, "corr-1")
	if err == nil {
		t.Fatal("nil hook should fail closed (block)")
	}
	if !errors.Is(err, ErrReplayBlocked) {
		t.Errorf("expected ErrReplayBlocked, got: %v", err)
	}
}

func TestReplayGateHook_NilRegistry_FailClosed(t *testing.T) {
	hook := &ReplayGateHook{Registry: nil, Logger: slog.Default()}
	err := hook.Check("corr-1")
	if err == nil {
		t.Fatal("nil registry should fail closed")
	}
}

func TestReplayGateHook_EmptyLedger_Allows(t *testing.T) {
	reg := NewLedgerRegistry()
	l := mustNew(t, "task-1")
	reg.Register("corr-1", l)

	hook := NewReplayGateHook(reg, slog.Default())
	err := hook.Check("corr-1")
	if err != nil {
		t.Fatalf("empty ledger should allow: %v", err)
	}
}

func TestReplayGateHook_Definite_Blocks(t *testing.T) {
	reg := NewLedgerRegistry()
	l := mustNew(t, "task-1")
	tok, _ := l.RecordToolUse("call-1", 1)
	l.RecordToolResult(tok)
	reg.Register("corr-1", l)

	hook := NewReplayGateHook(reg, slog.Default())
	err := hook.Check("corr-1")
	if err == nil {
		t.Fatal("definite should block")
	}
}

// TestReplayGateHook_IntegrationScenario exercises the full crash-recovery flow.
func TestReplayGateHook_IntegrationScenario(t *testing.T) {
	reg := NewLedgerRegistry()

	// Scenario 1: Task completed no tool calls → retry allowed
	l1 := mustNew(t, "task-clean")
	reg.Register("task-clean", l1)

	// Scenario 2: Task ran tools, all committed → retry BLOCKED
	l2 := mustNew(t, "task-committed")
	tok, _ := l2.RecordToolUse("call-1", 1)
	l2.RecordToolResult(tok)
	reg.Register("task-committed", l2)

	// Scenario 3: Task crashed mid-tool → retry BLOCKED
	l3 := mustNew(t, "task-crashed")
	l3.RecordToolUse("call-1", 1)
	l3.MarkAllUnresolvedAmbiguous()
	reg.Register("task-crashed", l3)

	// Scenario 4: No ledger exists (unknown state) → BLOCKED
	// (not registered)

	hook := NewReplayGateHook(reg, slog.Default())

	if err := hook.Check("task-clean"); err != nil {
		t.Errorf("task-clean should allow: %v", err)
	}
	if err := hook.Check("task-committed"); err == nil {
		t.Error("task-committed should block (definite side effects)")
	}
	if err := hook.Check("task-crashed"); err == nil {
		t.Error("task-crashed should block (ambiguous)")
	}
	if err := hook.Check("task-unknown"); err == nil {
		t.Error("task-unknown should block (no ledger, fail closed)")
	}
}

func TestReplayDecision_String(t *testing.T) {
	if ReplayAllowed.String() != "allowed" {
		t.Errorf("got %s", ReplayAllowed.String())
	}
	if ReplayBlocked.String() != "blocked" {
		t.Errorf("got %s", ReplayBlocked.String())
	}
}
