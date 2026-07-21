package commitledger

import (
	"fmt"
	"testing"
	"time"
)

var testSecret = []byte("test-secret-stable-32-bytes-xxxx")

func testConfig(taskID string) Config {
	return Config{HMACSecret: testSecret, TaskID: taskID}
}

func mustNew(t *testing.T, taskID string) *Ledger {
	t.Helper()
	l, err := New(testConfig(taskID))
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	return l
}

func TestNew_RequiresHMACSecret(t *testing.T) {
	_, err := New(Config{TaskID: "task-1", HMACSecret: nil})
	if err == nil {
		t.Fatal("expected error for nil HMAC secret")
	}
	_, err = New(Config{TaskID: "task-1", HMACSecret: []byte("short")})
	if err == nil {
		t.Fatal("expected error for short HMAC secret")
	}
}

func TestNew_RequiresTaskID(t *testing.T) {
	_, err := New(Config{TaskID: "", HMACSecret: testSecret})
	if err == nil {
		t.Fatal("expected error for empty TaskID")
	}
}

func TestNew_ValidConfig(t *testing.T) {
	l := mustNew(t, "task-1")
	stats := l.Stats()
	if stats.Version != SchemaVersion {
		t.Errorf("expected version %d, got %d", SchemaVersion, stats.Version)
	}
	if stats.ActiveEntries != 0 {
		t.Errorf("expected 0 entries, got %d", stats.ActiveEntries)
	}
}

func TestTokenizeCallID_Deterministic(t *testing.T) {
	l := mustNew(t, "task-1")
	tok1 := l.TokenizeCallID("call-abc-123")
	tok2 := l.TokenizeCallID("call-abc-123")
	if tok1 != tok2 {
		t.Errorf("expected deterministic tokens: %s != %s", tok1, tok2)
	}
	if len(tok1) != 32 {
		t.Errorf("expected 32-char token, got %d", len(tok1))
	}
}

func TestTokenizeCallID_DomainSeparation(t *testing.T) {
	l1 := mustNew(t, "task-1")
	l2 := mustNew(t, "task-2")
	tok1 := l1.TokenizeCallID("same-call-id")
	tok2 := l2.TokenizeCallID("same-call-id")
	if tok1 == tok2 {
		t.Error("different taskIDs should produce different tokens (domain separation)")
	}
}

func TestTokenizeCallID_DifferentCallIDs(t *testing.T) {
	l := mustNew(t, "task-1")
	tok1 := l.TokenizeCallID("call-1")
	tok2 := l.TokenizeCallID("call-2")
	if tok1 == tok2 {
		t.Error("different call IDs should produce different tokens")
	}
}

func TestRecordToolUse_Basic(t *testing.T) {
	l := mustNew(t, "task-1")
	token, err := l.RecordToolUse("call-1", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(token) != 32 {
		t.Errorf("expected 32-char token, got %d", len(token))
	}
	state := l.Lookup(token)
	if state != CommitNone {
		t.Errorf("expected CommitNone for tool_use, got %s", state)
	}
	summary := l.Summary()
	if !summary.EverHadToolUse {
		t.Error("expected EverHadToolUse after RecordToolUse")
	}
}

func TestRecordToolUse_EmptyCallID(t *testing.T) {
	l := mustNew(t, "task-1")
	_, err := l.RecordToolUse("", 1)
	if err == nil {
		t.Fatal("expected error for empty callID")
	}
}

func TestRecordToolUse_NonPositiveSeq(t *testing.T) {
	l := mustNew(t, "task-1")
	_, err := l.RecordToolUse("call-1", 0)
	if err == nil {
		t.Fatal("expected error for seq=0")
	}
	_, err = l.RecordToolUse("call-1", -1)
	if err == nil {
		t.Fatal("expected error for negative seq")
	}
}

func TestRecordToolUse_StrictIncreasingSeq(t *testing.T) {
	l := mustNew(t, "task-1")
	_, err := l.RecordToolUse("call-1", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Same seq (non-idempotent different callID) → saturates
	_, err = l.RecordToolUse("call-2", 5)
	if err == nil {
		t.Fatal("expected error for non-increasing seq")
	}
	if !l.IsFailClosed() {
		t.Error("ledger should be fail-closed after seq violation")
	}
}

func TestRecordToolUse_IdempotentSameCallID(t *testing.T) {
	l := mustNew(t, "task-1")
	tok1, _ := l.RecordToolUse("call-1", 1)
	tok2, _ := l.RecordToolUse("call-1", 1) // same callID, same seq → idempotent
	if tok1 != tok2 {
		t.Error("idempotent record should return same token")
	}
	stats := l.Stats()
	if stats.ActiveEntries != 1 {
		t.Errorf("expected 1 entry after dedup, got %d", stats.ActiveEntries)
	}
}

func TestRecordToolUse_ClosedLedger(t *testing.T) {
	l := mustNew(t, "task-1")
	l.Close()
	_, err := l.RecordToolUse("call-1", 1)
	if err == nil {
		t.Fatal("expected error recording to closed ledger")
	}
}

func TestRecordToolResult_CommitsImmediately(t *testing.T) {
	l := mustNew(t, "task-1")
	token, _ := l.RecordToolUse("call-1", 1)
	err := l.RecordToolResult(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	state := l.Lookup(token)
	if state != CommitDefinite {
		t.Errorf("expected CommitDefinite after tool_result, got %s", state)
	}
	summary := l.Summary()
	if !summary.EverDefinite {
		t.Error("expected EverDefinite after RecordToolResult")
	}
}

func TestRecordToolResult_InvalidToken(t *testing.T) {
	l := mustNew(t, "task-1")
	err := l.RecordToolResult("short")
	if err == nil {
		t.Fatal("expected error for short token")
	}
}

func TestRecordToolResult_UnknownToken_MarksSummary(t *testing.T) {
	l := mustNew(t, "task-1")
	// 32-char token that was never recorded (simulates evicted)
	err := l.RecordToolResult("abcdef0123456789abcdef0123456789")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Summary should mark EverDefinite
	summary := l.Summary()
	if !summary.EverDefinite {
		t.Error("unknown token tool_result should set EverDefinite in summary")
	}
}

func TestMarkAmbiguous_FromNone(t *testing.T) {
	l := mustNew(t, "task-1")
	token, _ := l.RecordToolUse("call-1", 1)
	err := l.MarkAmbiguous(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l.Lookup(token) != CommitAmbiguous {
		t.Error("expected CommitAmbiguous")
	}
}

func TestMarkAmbiguous_FromDefinite(t *testing.T) {
	l := mustNew(t, "task-1")
	token, _ := l.RecordToolUse("call-1", 1)
	l.RecordToolResult(token) // now Definite
	err := l.MarkAmbiguous(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Definite → Ambiguous is allowed (crash after commit)
	if l.Lookup(token) != CommitAmbiguous {
		t.Error("expected Definite → Ambiguous transition to succeed")
	}
}

func TestMarkAmbiguous_Idempotent(t *testing.T) {
	l := mustNew(t, "task-1")
	token, _ := l.RecordToolUse("call-1", 1)
	l.MarkAmbiguous(token)
	err := l.MarkAmbiguous(token) // second call
	if err != nil {
		t.Fatalf("expected idempotent MarkAmbiguous, got: %v", err)
	}
}

func TestMarkAmbiguous_ShortToken(t *testing.T) {
	l := mustNew(t, "task-1")
	err := l.MarkAmbiguous("abc") // too short
	if err == nil {
		t.Fatal("expected error for very short token")
	}
}

func TestMarkAmbiguous_UnknownToken_MarksSummary(t *testing.T) {
	l := mustNew(t, "task-1")
	err := l.MarkAmbiguous("unknown-token-long-enough")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !l.Summary().EverAmbiguous {
		t.Error("unknown token should set EverAmbiguous in summary")
	}
}

func TestLookup_Unknown_ReturnsAmbiguous(t *testing.T) {
	l := mustNew(t, "task-1")
	// Unknown token should return Ambiguous (fail-closed safe)
	state := l.Lookup("nonexistent-token-32-chars-xxxxx")
	if state != CommitAmbiguous {
		t.Errorf("expected Ambiguous for unknown token, got %s", state)
	}
}

func TestLookup_FailClosed_ReturnsAmbiguous(t *testing.T) {
	l := NewFailClosed("task-1")
	state := l.Lookup("any-token-32-chars-xxxxxxxxxxxxxxx")
	if state != CommitAmbiguous {
		t.Errorf("fail-closed should return Ambiguous, got %s", state)
	}
}

func TestMarkAllUnresolvedAmbiguous(t *testing.T) {
	l := mustNew(t, "task-1")
	l.RecordToolUse("call-1", 1)
	tok2, _ := l.RecordToolUse("call-2", 2)
	l.RecordToolResult(tok2) // call-2 is Definite
	l.RecordToolUse("call-3", 3)

	n := l.MarkAllUnresolvedAmbiguous()
	if n != 2 { // call-1 and call-3 were None
		t.Errorf("expected 2 marked ambiguous, got %d", n)
	}
	if !l.Summary().EverAmbiguous {
		t.Error("expected EverAmbiguous after MarkAllUnresolvedAmbiguous")
	}
}

func TestAcknowledgeOutputPersisted_GapSafe(t *testing.T) {
	l := mustNew(t, "task-1")
	l.AcknowledgeOutputPersisted(5)
	if l.OutputPersistedSeq() != 5 {
		t.Errorf("expected 5, got %d", l.OutputPersistedSeq())
	}
	// Backward is no-op
	l.AcknowledgeOutputPersisted(3)
	if l.OutputPersistedSeq() != 5 {
		t.Errorf("expected 5 (no regress), got %d", l.OutputPersistedSeq())
	}
	// Forward advances
	l.AcknowledgeOutputPersisted(10)
	if l.OutputPersistedSeq() != 10 {
		t.Errorf("expected 10, got %d", l.OutputPersistedSeq())
	}
	// Zero/negative is no-op
	l.AcknowledgeOutputPersisted(0)
	l.AcknowledgeOutputPersisted(-1)
	if l.OutputPersistedSeq() != 10 {
		t.Errorf("expected 10, got %d", l.OutputPersistedSeq())
	}
}

func TestEviction_NeverEvictsNoneOrAmbiguous(t *testing.T) {
	// Fill ledger to capacity with unresolved entries
	l := mustNew(t, "task-1")
	for i := 1; i <= MaxEntriesPerTask; i++ {
		_, err := l.RecordToolUse(fmt.Sprintf("call-%d", i), int64(i))
		if err != nil {
			t.Fatalf("record %d failed: %v", i, err)
		}
	}
	// Next record should saturate (no evictable entries)
	_, err := l.RecordToolUse("overflow", int64(MaxEntriesPerTask+1))
	if err == nil {
		t.Fatal("expected error when at capacity with no evictable entries")
	}
	if !l.IsFailClosed() {
		t.Error("should be fail-closed after saturation")
	}
}

func TestEviction_EvictsOldDefiniteOnly(t *testing.T) {
	l := mustNew(t, "task-1")
	// Record and commit an entry, then age it
	token, _ := l.RecordToolUse("call-old", 1)
	l.RecordToolResult(token)

	// Manually age the entry past TTL
	l.mu.Lock()
	entry := l.entries[token]
	entry.ResolvedAt = time.Now().Add(-TerminalTTL - time.Hour)
	l.mu.Unlock()

	// Fill to capacity-1 (we already have 1)
	for i := 2; i <= MaxEntriesPerTask; i++ {
		tok, _ := l.RecordToolUse(fmt.Sprintf("call-%d", i), int64(i))
		l.RecordToolResult(tok) // make definite
	}

	// Next record should evict the old one
	_, err := l.RecordToolUse("call-new", int64(MaxEntriesPerTask+1))
	if err != nil {
		t.Fatalf("expected successful eviction, got: %v", err)
	}
	if l.IsFailClosed() {
		t.Error("should not be fail-closed after successful eviction")
	}
}

func TestFailClosed_FromNew(t *testing.T) {
	l := NewFailClosed("task-1")
	if !l.IsFailClosed() {
		t.Fatal("expected fail-closed")
	}
	if !l.Summary().EverSaturated {
		t.Error("fail-closed should mark EverSaturated")
	}
}

func TestSnapshot_Roundtrip(t *testing.T) {
	l := mustNew(t, "task-1")
	tok1, _ := l.RecordToolUse("call-1", 1)
	l.RecordToolResult(tok1) // definite
	l.RecordToolUse("call-2", 2)
	l.AcknowledgeOutputPersisted(1)

	snap := l.Snapshot()
	if snap.Version != SchemaVersion {
		t.Errorf("expected version %d, got %d", SchemaVersion, snap.Version)
	}

	restored, err := RestoreFromSnapshot(snap, testConfig("task-1"))
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	// Unresolved entries (call-2) should be marked ambiguous on restore
	tok2 := restored.TokenizeCallID("call-2")
	if restored.Lookup(tok2) != CommitAmbiguous {
		t.Error("unresolved entry should be ambiguous after restore")
	}
	if !restored.Summary().EverAmbiguous {
		t.Error("restored should have EverAmbiguous from unresolved→ambiguous")
	}
}

func TestSnapshot_VersionMismatch_FailClosed(t *testing.T) {
	snap := LedgerSnapshot{
		Version: 999,
		Entries: []EntrySnapshot{{Token: "abcdef0123456789abcdef0123456789", Seq: 1, State: CommitDefinite, CreatedAt: time.Now()}},
	}
	restored, err := RestoreFromSnapshot(snap, testConfig("task-1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !restored.IsFailClosed() {
		t.Fatal("version mismatch should produce fail-closed ledger")
	}
}

func TestSnapshot_CorruptState_FailClosed(t *testing.T) {
	snap := LedgerSnapshot{
		Version: SchemaVersion,
		Entries: []EntrySnapshot{{Token: "abcdef0123456789abcdef0123456789", Seq: 1, State: CommitState(42), CreatedAt: time.Now()}},
	}
	restored, err := RestoreFromSnapshot(snap, testConfig("task-1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !restored.IsFailClosed() {
		t.Fatal("corrupt state should produce fail-closed")
	}
}

func TestSnapshot_DuplicateToken_FailClosed(t *testing.T) {
	snap := LedgerSnapshot{
		Version: SchemaVersion,
		Entries: []EntrySnapshot{
			{Token: "abcdef0123456789abcdef0123456789", Seq: 1, State: CommitDefinite, CreatedAt: time.Now(), ResolvedAt: time.Now()},
			{Token: "abcdef0123456789abcdef0123456789", Seq: 2, State: CommitDefinite, CreatedAt: time.Now(), ResolvedAt: time.Now()},
		},
	}
	restored, _ := RestoreFromSnapshot(snap, testConfig("task-1"))
	if !restored.IsFailClosed() {
		t.Fatal("duplicate token should produce fail-closed")
	}
}

func TestSnapshot_InvalidSeqOrder_FailClosed(t *testing.T) {
	snap := LedgerSnapshot{
		Version: SchemaVersion,
		Entries: []EntrySnapshot{
			{Token: "abcdef0123456789abcdef0123456789", Seq: 5, State: CommitDefinite, CreatedAt: time.Now(), ResolvedAt: time.Now()},
			{Token: "0123456789abcdef0123456789abcdef", Seq: 3, State: CommitDefinite, CreatedAt: time.Now(), ResolvedAt: time.Now()},
		},
	}
	restored, _ := RestoreFromSnapshot(snap, testConfig("task-1"))
	if !restored.IsFailClosed() {
		t.Fatal("out-of-order seq should produce fail-closed")
	}
}

func TestSnapshot_InvalidTokenLength_FailClosed(t *testing.T) {
	snap := LedgerSnapshot{
		Version: SchemaVersion,
		Entries: []EntrySnapshot{
			{Token: "too-short", Seq: 1, State: CommitDefinite, CreatedAt: time.Now()},
		},
	}
	restored, _ := RestoreFromSnapshot(snap, testConfig("task-1"))
	if !restored.IsFailClosed() {
		t.Fatal("invalid token length should produce fail-closed")
	}
}

func TestSnapshot_ZeroTimestamp_FailClosed(t *testing.T) {
	snap := LedgerSnapshot{
		Version: SchemaVersion,
		Entries: []EntrySnapshot{
			{Token: "abcdef0123456789abcdef0123456789", Seq: 1, State: CommitDefinite},
		},
	}
	restored, _ := RestoreFromSnapshot(snap, testConfig("task-1"))
	if !restored.IsFailClosed() {
		t.Fatal("zero timestamp should produce fail-closed")
	}
}

func TestSnapshot_NegativeWatermark_FailClosed(t *testing.T) {
	snap := LedgerSnapshot{
		Version:            SchemaVersion,
		OutputPersistedSeq: -5,
	}
	restored, _ := RestoreFromSnapshot(snap, testConfig("task-1"))
	if !restored.IsFailClosed() {
		t.Fatal("negative watermark should produce fail-closed")
	}
}

func TestCommitState_String(t *testing.T) {
	cases := []struct {
		state CommitState
		want  string
	}{
		{CommitNone, "none"},
		{CommitDefinite, "definite"},
		{CommitAmbiguous, "ambiguous"},
		{CommitState(99), "invalid"},
	}
	for _, tc := range cases {
		if got := tc.state.String(); got != tc.want {
			t.Errorf("CommitState(%d).String() = %q, want %q", tc.state, got, tc.want)
		}
	}
}

func TestCommitState_HasCommitted(t *testing.T) {
	if CommitNone.HasCommitted() {
		t.Error("None should not HasCommitted")
	}
	if !CommitDefinite.HasCommitted() {
		t.Error("Definite should HasCommitted")
	}
	if !CommitAmbiguous.HasCommitted() {
		t.Error("Ambiguous should HasCommitted")
	}
}

func TestDurableSummary_BlocksReplay(t *testing.T) {
	tests := []struct {
		name    string
		summary DurableSummary
		blocks  bool
	}{
		{"empty", DurableSummary{}, false},
		{"tool_use", DurableSummary{EverHadToolUse: true}, true},
		{"definite", DurableSummary{EverDefinite: true}, true},
		{"ambiguous", DurableSummary{EverAmbiguous: true}, true},
		{"saturated", DurableSummary{EverSaturated: true}, true},
		{"all", DurableSummary{EverHadToolUse: true, EverDefinite: true, EverAmbiguous: true, EverSaturated: true}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.summary.BlocksReplay(); got != tt.blocks {
				t.Errorf("BlocksReplay() = %v, want %v", got, tt.blocks)
			}
		})
	}
}


