// Package commitledger implements a durable bounded ledger that tracks the
// monotonic commit state of tool invocations during task execution. It
// provides HMAC-pseudonymous tool tokens, gap-safe persisted_through_seq
// acknowledgement, cap/eviction with durable safety latch, schema versioning,
// and fail-closed restart semantics.
//
// Tool lifecycle:
//   - tool_use = started (not yet committed)
//   - tool_result = committed immediately (side effect occurred)
//   - output persistence = separate watermark (server batch ack)
//
// Replay gate semantics:
//   - Any definite (committed) OR ambiguous entry blocks automatic replay
//   - Only a ledger with zero tool activity (no entries ever) permits replay
//   - Unknown/corrupt/unavailable state fails closed (blocks)
//
// Scope: W1 daemon/server only. No gateway edits.
package commitledger

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

// SchemaVersion is bumped when the on-disk/in-memory format changes. On
// restart, if the loaded version doesn't match, the ledger fails closed.
const SchemaVersion = 1

// Frozen capacity limits.
const (
	MaxEntriesGlobal  = 4096
	MaxEntriesPerTask = 1024
	TerminalTTL       = 24 * time.Hour
)

// CommitState is the monotonic outcome state of a single tool invocation.
// Transitions: None → Definite or None → Ambiguous. Definite → Ambiguous
// is allowed (crash after commit). Ambiguous and Definite are both terminal
// for the purpose of "did something happen". Neither can regress to None.
type CommitState uint8

const (
	CommitNone      CommitState = 0 // tool_use started, not yet committed
	CommitDefinite  CommitState = 1 // tool_result received or output acked
	CommitAmbiguous CommitState = 2 // crash/timeout/gap — unknown outcome
)

func (s CommitState) String() string {
	switch s {
	case CommitNone:
		return "none"
	case CommitDefinite:
		return "definite"
	case CommitAmbiguous:
		return "ambiguous"
	default:
		return "invalid"
	}
}

// IsValid returns whether this is a recognized state value.
func (s CommitState) IsValid() bool {
	return s <= CommitAmbiguous
}

// HasCommitted returns true if tool activity occurred (definite or ambiguous).
// This is what blocks replay — any evidence of side effects.
func (s CommitState) HasCommitted() bool {
	return s == CommitDefinite || s == CommitAmbiguous
}

// Entry is a single tool-call record in the ledger. It stores only the
// pseudonymous token (never the raw call_id or body).
type Entry struct {
	Token      string      // HMAC-SHA256 pseudonymous token (32 hex chars)
	Seq        int64       // monotonic sequence within the task (positive, contiguous)
	State      CommitState // monotonic: none → definite | ambiguous; definite → ambiguous
	CreatedAt  time.Time
	ResolvedAt time.Time // zero until state leaves None
}

// Config controls ledger bounds. HMACSecret is mandatory.
type Config struct {
	// HMACSecret must be a stable >=32-byte key injected from config/env.
	// Empty/short key causes New to fail. Never randomly generated.
	HMACSecret []byte

	// TaskID is used for domain separation in HMAC. Required.
	TaskID string
}

// Validate checks config requirements.
func (c *Config) Validate() error {
	if len(c.HMACSecret) < 32 {
		return errors.New("commitledger: HMACSecret must be at least 32 bytes (stable, injected)")
	}
	if c.TaskID == "" {
		return errors.New("commitledger: TaskID is required")
	}
	return nil
}

// DurableSummary is the safety latch that survives eviction, restart, and
// corruption. Once set, these flags never reset to false. The replay gate
// consults this rather than scanning entries (which may be evicted).
type DurableSummary struct {
	EverHadToolUse    bool // any tool_use ever recorded
	EverDefinite      bool // any entry ever reached Definite
	EverAmbiguous     bool // any entry ever reached Ambiguous
	EverSaturated     bool // cap overflow forced fail-closed
	HighestSeqSeen    int64
	TotalToolUseCount int64 // lifetime count (including evicted)
}

// BlocksReplay returns true if any tool activity ever occurred that would
// make automatic replay unsafe.
func (s *DurableSummary) BlocksReplay() bool {
	return s.EverHadToolUse || s.EverDefinite || s.EverAmbiguous || s.EverSaturated
}

// Ledger is the bounded, durable commit ledger for a single task's tool
// invocations. It is concurrency-safe.
type Ledger struct {
	mu      sync.RWMutex
	config  Config
	version int

	// entries indexed by pseudonymous token
	entries map[string]*Entry
	// ordered slice for iteration/eviction (oldest first)
	order []*Entry

	// outputPersistedSeq is the highest seq the server has acknowledged
	// as durably persisted for output (batch ack). Distinct from tool
	// commit state — this tracks whether the server has the output.
	outputPersistedSeq int64

	// summary is the durable safety latch. Survives eviction.
	summary DurableSummary

	// closed marks the ledger as terminated.
	closed bool

	// failClosed is set on schema mismatch, corrupt load, or saturation.
	failClosed bool
}

// New creates a fresh ledger for the given task. Returns error if config
// is invalid (e.g., missing HMAC secret).
func New(cfg Config) (*Ledger, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &Ledger{
		config:  cfg,
		version: SchemaVersion,
		entries: make(map[string]*Entry, 64),
		order:   make([]*Entry, 0, 64),
	}, nil
}

// NewFailClosed creates a ledger that always blocks replay.
// Used on restart when schema version doesn't match or data is corrupt.
func NewFailClosed(taskID string) *Ledger {
	return &Ledger{
		config:     Config{TaskID: taskID, HMACSecret: make([]byte, 32)},
		version:    SchemaVersion,
		entries:    make(map[string]*Entry),
		failClosed: true,
		summary:    DurableSummary{EverSaturated: true},
	}
}

// TokenizeCallID produces a pseudonymous HMAC-SHA256 token from a raw call_id.
// Domain-separated with taskID and length-delimited to prevent extension attacks.
// This token is safe to log/emit in telemetry.
func (l *Ledger) TokenizeCallID(callID string) string {
	l.mu.RLock()
	secret := l.config.HMACSecret
	taskID := l.config.TaskID
	l.mu.RUnlock()

	mac := hmac.New(sha256.New, secret)
	// Domain separation: length-delimited taskID + callID
	taskBytes := []byte(taskID)
	callBytes := []byte(callID)
	// Write length prefix (4 bytes big-endian) + data for each field
	mac.Write([]byte{byte(len(taskBytes) >> 24), byte(len(taskBytes) >> 16), byte(len(taskBytes) >> 8), byte(len(taskBytes))})
	mac.Write(taskBytes)
	mac.Write([]byte{byte(len(callBytes) >> 24), byte(len(callBytes) >> 16), byte(len(callBytes) >> 8), byte(len(callBytes))})
	mac.Write(callBytes)
	return hex.EncodeToString(mac.Sum(nil))[:32] // 128-bit prefix
}

// RecordToolUse registers a new tool_use at the given sequence number.
// tool_use means "started" — the tool has been invoked but we don't yet
// know if it completed. Returns the pseudonymous token.
//
// Seq must be positive and strictly greater than the highest seq previously
// recorded. Violations fail closed (mark saturated).
func (l *Ledger) RecordToolUse(callID string, seq int64) (string, error) {
	if callID == "" {
		return "", errors.New("commitledger: empty callID")
	}
	if seq <= 0 {
		return "", errors.New("commitledger: seq must be positive")
	}

	token := l.TokenizeCallID(callID)

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return "", errors.New("commitledger: ledger is closed")
	}
	if l.failClosed {
		// Accept the record for tracking but ledger stays fail-closed
		l.summary.EverHadToolUse = true
		l.summary.TotalToolUseCount++
		return token, nil
	}

	// Enforce strict increasing seq
	if seq <= l.summary.HighestSeqSeen {
		// Duplicate or out-of-order: check if it's an idempotent re-record
		if _, exists := l.entries[token]; exists {
			return token, nil // idempotent
		}
		// Non-idempotent out-of-order: saturate
		l.saturate()
		return token, fmt.Errorf("commitledger: seq %d not strictly increasing (highest: %d); saturated", seq, l.summary.HighestSeqSeen)
	}

	// Cap check: if at capacity and no evictable entries, saturate
	if len(l.order) >= MaxEntriesPerTask {
		evicted := l.evictTerminalOldest()
		if !evicted {
			l.saturate()
			return token, fmt.Errorf("commitledger: at capacity (%d) with no evictable entries; saturated", MaxEntriesPerTask)
		}
	}

	now := time.Now()
	entry := &Entry{
		Token:     token,
		Seq:       seq,
		State:     CommitNone,
		CreatedAt: now,
	}

	l.entries[token] = entry
	l.order = append(l.order, entry)
	l.summary.HighestSeqSeen = seq
	l.summary.EverHadToolUse = true
	l.summary.TotalToolUseCount++

	return token, nil
}

// RecordToolResult marks a tool call as committed (tool_result received).
// The tool executed and produced a result — side effect occurred.
// Transitions: None → Definite.
func (l *Ledger) RecordToolResult(token string) error {
	if len(token) < 32 {
		return errors.New("commitledger: invalid token format")
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.failClosed {
		return nil // already fail-closed, no further tracking needed
	}

	entry, ok := l.entries[token]
	if !ok {
		// Unknown token: could be evicted or never recorded.
		// Mark summary as having had definite activity (fail-closed safe).
		l.summary.EverDefinite = true
		return nil
	}

	if entry.State == CommitAmbiguous {
		// Already ambiguous — definite→ambiguous is allowed but not
		// ambiguous→definite. Already in a terminal blocking state.
		return nil
	}

	entry.State = CommitDefinite
	entry.ResolvedAt = time.Now()
	l.summary.EverDefinite = true
	return nil
}

// MarkAmbiguous transitions an entry to CommitAmbiguous.
// Valid from None or Definite (definite→ambiguous is allowed: crash after
// commit means we don't know if downstream saw it).
// Never regresses: Ambiguous is the highest severity.
func (l *Ledger) MarkAmbiguous(token string) error {
	if len(token) < 8 {
		return errors.New("commitledger: token too short")
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.failClosed {
		l.summary.EverAmbiguous = true
		return nil
	}

	entry, ok := l.entries[token]
	if !ok {
		// Unknown/evicted: record ambiguous in summary
		l.summary.EverAmbiguous = true
		return nil
	}

	if entry.State == CommitAmbiguous {
		return nil // already ambiguous, idempotent
	}

	// None → Ambiguous or Definite → Ambiguous are both valid
	entry.State = CommitAmbiguous
	entry.ResolvedAt = time.Now()
	l.summary.EverAmbiguous = true
	return nil
}

// MarkAllUnresolvedAmbiguous transitions all CommitNone entries to
// CommitAmbiguous. Used on drain timeout or crash recovery.
func (l *Ledger) MarkAllUnresolvedAmbiguous() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	count := 0
	for _, e := range l.order {
		if e.State == CommitNone {
			e.State = CommitAmbiguous
			e.ResolvedAt = now
			count++
		}
	}
	if count > 0 {
		l.summary.EverAmbiguous = true
	}
	return count
}

// AcknowledgeOutputPersisted advances the output-persisted watermark.
// This is the server's batch ack: "I have durably stored output through
// this seq." Only advances forward (gap-safe).
func (l *Ledger) AcknowledgeOutputPersisted(newSeq int64) {
	if newSeq <= 0 {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if newSeq <= l.outputPersistedSeq {
		return // already advanced past this
	}
	l.outputPersistedSeq = newSeq
}

// OutputPersistedSeq returns the current output watermark.
func (l *Ledger) OutputPersistedSeq() int64 {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.outputPersistedSeq
}

// Lookup returns the commit state for a pseudonymous token.
// Returns CommitAmbiguous for:
//   - fail-closed ledgers
//   - unknown/evicted tokens (safe default: treat missing as ambiguous)
func (l *Ledger) Lookup(token string) CommitState {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.failClosed {
		return CommitAmbiguous
	}

	entry, ok := l.entries[token]
	if !ok {
		// Unknown/evicted: ambiguous (fail-closed safe)
		return CommitAmbiguous
	}
	return entry.State
}

// LookupByCallID tokenizes and then looks up.
func (l *Ledger) LookupByCallID(callID string) CommitState {
	return l.Lookup(l.TokenizeCallID(callID))
}

// Close terminates the ledger. No new entries can be recorded.
func (l *Ledger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.closed = true
}

// IsClosed reports whether the ledger has been terminated.
func (l *Ledger) IsClosed() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.closed
}

// IsFailClosed reports whether the ledger is in fail-closed mode.
func (l *Ledger) IsFailClosed() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.failClosed
}

// Summary returns the durable safety latch (copy).
func (l *Ledger) Summary() DurableSummary {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.summary
}

// Stats returns current ledger statistics for diagnostics.
// No raw task IDs — use pseudonymous correlation only.
type LedgerStats struct {
	Version            int
	ActiveEntries      int
	OutputPersistedSeq int64
	Closed             bool
	FailClosed         bool
	Unresolved         int
	Definite           int
	Ambiguous          int
	Summary            DurableSummary
}

func (l *Ledger) Stats() LedgerStats {
	l.mu.RLock()
	defer l.mu.RUnlock()

	stats := LedgerStats{
		Version:            l.version,
		ActiveEntries:      len(l.entries),
		OutputPersistedSeq: l.outputPersistedSeq,
		Closed:             l.closed,
		FailClosed:         l.failClosed,
		Summary:            l.summary,
	}
	for _, e := range l.order {
		switch e.State {
		case CommitNone:
			stats.Unresolved++
		case CommitDefinite:
			stats.Definite++
		case CommitAmbiguous:
			stats.Ambiguous++
		}
	}
	return stats
}

// --- Eviction ---

// evictTerminalOldest removes the oldest terminal entry whose TTL has
// expired. Only Definite entries that have been terminal for > TerminalTTL
// are eligible. Active (None) and Ambiguous entries are NEVER evicted.
// Returns true if an entry was evicted.
func (l *Ledger) evictTerminalOldest() bool {
	cutoff := time.Now().Add(-TerminalTTL)
	for i, e := range l.order {
		// Only evict Definite entries past TTL. Never evict None or Ambiguous.
		if e.State == CommitDefinite && !e.ResolvedAt.IsZero() && e.ResolvedAt.Before(cutoff) {
			delete(l.entries, e.Token)
			l.order = append(l.order[:i], l.order[i+1:]...)
			return true
		}
	}
	return false
}

// saturate puts the ledger into fail-closed mode due to capacity overflow.
func (l *Ledger) saturate() {
	l.failClosed = true
	l.summary.EverSaturated = true
}

// --- Snapshot/Restore for durable persistence ---

// Snapshot returns a serializable representation for durable storage.
// The HMAC secret is NEVER included in the snapshot.
type LedgerSnapshot struct {
	Version            int             `json:"version"`
	OutputPersistedSeq int64           `json:"output_persisted_seq"`
	Entries            []EntrySnapshot `json:"entries"`
	Summary            DurableSummary  `json:"summary"`
	Closed             bool            `json:"closed"`
	FailClosed         bool            `json:"fail_closed"`
}

type EntrySnapshot struct {
	Token      string      `json:"token"`
	Seq        int64       `json:"seq"`
	State      CommitState `json:"state"`
	CreatedAt  time.Time   `json:"created_at"`
	ResolvedAt time.Time   `json:"resolved_at,omitempty"`
}

func (l *Ledger) Snapshot() LedgerSnapshot {
	l.mu.RLock()
	defer l.mu.RUnlock()

	entries := make([]EntrySnapshot, len(l.order))
	for i, e := range l.order {
		entries[i] = EntrySnapshot{
			Token:      e.Token,
			Seq:        e.Seq,
			State:      e.State,
			CreatedAt:  e.CreatedAt,
			ResolvedAt: e.ResolvedAt,
		}
	}
	return LedgerSnapshot{
		Version:            SchemaVersion,
		OutputPersistedSeq: l.outputPersistedSeq,
		Entries:            entries,
		Summary:            l.summary,
		Closed:             l.closed,
		FailClosed:         l.failClosed,
	}
}

// RestoreFromSnapshot rebuilds the ledger from a persisted snapshot.
// Validates version, entry integrity, seq ordering, token format,
// and state consistency. Any corruption → fail-closed.
// Unresolved (None) entries in a restored snapshot are marked Ambiguous
// (we cannot know if they committed during the gap).
func RestoreFromSnapshot(snap LedgerSnapshot, cfg Config) (*Ledger, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Version check
	if snap.Version != SchemaVersion {
		l := NewFailClosed(cfg.TaskID)
		l.summary = snap.Summary
		l.summary.EverSaturated = true
		return l, nil
	}

	// If snapshot says fail-closed, honour it
	if snap.FailClosed {
		l := NewFailClosed(cfg.TaskID)
		l.summary = snap.Summary
		return l, nil
	}

	l := &Ledger{
		config:             cfg,
		version:            snap.Version,
		outputPersistedSeq: snap.OutputPersistedSeq,
		entries:            make(map[string]*Entry, len(snap.Entries)),
		order:              make([]*Entry, 0, len(snap.Entries)),
		summary:            snap.Summary,
		closed:             snap.Closed,
	}

	// Validate entries
	seenTokens := make(map[string]bool, len(snap.Entries))
	var lastSeq int64
	for _, es := range snap.Entries {
		// Token format: must be exactly 32 hex chars
		if len(es.Token) != 32 {
			return NewFailClosed(cfg.TaskID), nil
		}
		// State validity
		if !es.State.IsValid() {
			return NewFailClosed(cfg.TaskID), nil
		}
		// Duplicate token check
		if seenTokens[es.Token] {
			return NewFailClosed(cfg.TaskID), nil
		}
		seenTokens[es.Token] = true
		// Seq ordering: must be positive and non-decreasing
		if es.Seq <= 0 {
			return NewFailClosed(cfg.TaskID), nil
		}
		if es.Seq < lastSeq {
			return NewFailClosed(cfg.TaskID), nil
		}
		lastSeq = es.Seq
		// Timestamp sanity
		if es.CreatedAt.IsZero() {
			return NewFailClosed(cfg.TaskID), nil
		}

		entry := &Entry{
			Token:      es.Token,
			Seq:        es.Seq,
			State:      es.State,
			CreatedAt:  es.CreatedAt,
			ResolvedAt: es.ResolvedAt,
		}

		// Key invariant: any unresolved (None) entry in a restored snapshot
		// must be marked Ambiguous. We cannot know if it committed during
		// the gap between snapshot and restart.
		if entry.State == CommitNone {
			entry.State = CommitAmbiguous
			entry.ResolvedAt = time.Now()
			l.summary.EverAmbiguous = true
		}

		l.entries[entry.Token] = entry
		l.order = append(l.order, entry)
	}

	// Over-cap check
	if len(l.order) > MaxEntriesPerTask {
		return NewFailClosed(cfg.TaskID), nil
	}

	// Watermark sanity: must be >= 0 and <= highest seq
	if snap.OutputPersistedSeq < 0 {
		return NewFailClosed(cfg.TaskID), nil
	}

	return l, nil
}

// --- DrainOwner with proper ownership guarantees ---

// DrainOwner manages the lifecycle of the drain goroutine for a single task
// execution. Guarantees:
// 1. Exactly one goroutine owns the drain (Start returns false for others)
// 2. Bounded final join with context-based cancellation
// 3. No post-return flush: cancel fires before JoinOrTimeout returns
type DrainOwner struct {
	mu         sync.Mutex
	started    bool
	done       chan struct{}
	finishOnce sync.Once
	ledger     *Ledger
	joinBudget time.Duration
	cancelFn   context.CancelFunc // cancels the drain goroutine's context
}

// NewDrainOwner creates a new drain owner. The cancelFn MUST cancel the
// context that the drain goroutine uses for all I/O (flush, ack, mutation).
// This ensures that when JoinOrTimeout fires, the drain cannot perform
// any further network calls or state mutations.
func NewDrainOwner(ledger *Ledger, joinBudget time.Duration, cancelFn context.CancelFunc) *DrainOwner {
	if joinBudget <= 0 {
		joinBudget = 5 * time.Second
	}
	if cancelFn == nil {
		cancelFn = func() {} // no-op for tests
	}
	return &DrainOwner{
		done:       make(chan struct{}),
		ledger:     ledger,
		joinBudget: joinBudget,
		cancelFn:   cancelFn,
	}
}

// Start marks the drain as started. Returns false if already started
// (ensuring single-owner semantics).
func (d *DrainOwner) Start() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.started {
		return false
	}
	d.started = true
	return true
}

// Finish signals that the drain goroutine has completed. Safe to call
// multiple times (sync.Once protected).
func (d *DrainOwner) Finish() {
	d.finishOnce.Do(func() {
		close(d.done)
	})
}

// JoinOrTimeout waits for the drain to finish, up to the join budget.
// If timeout fires:
// 1. Cancels the drain's context (prevents further flush/ack/mutation)
// 2. Marks all unresolved ledger entries ambiguous
// 3. Returns error
//
// The cancel fires BEFORE return, ensuring no post-return flush is possible.
func (d *DrainOwner) JoinOrTimeout(parentCtx context.Context) error {
	select {
	case <-d.done:
		return nil
	case <-time.After(d.joinBudget):
		// Cancel the drain's context first — this prevents any further
		// network I/O, ack processing, or state mutation in the drain.
		d.cancelFn()
		if d.ledger != nil {
			d.ledger.MarkAllUnresolvedAmbiguous()
		}
		return fmt.Errorf("commitledger: drain did not finish within %s; cancelled and marked ambiguous", d.joinBudget)
	case <-parentCtx.Done():
		d.cancelFn()
		if d.ledger != nil {
			d.ledger.MarkAllUnresolvedAmbiguous()
		}
		return parentCtx.Err()
	}
}

// Done returns a channel that closes when the drain goroutine finishes.
func (d *DrainOwner) Done() <-chan struct{} {
	return d.done
}
