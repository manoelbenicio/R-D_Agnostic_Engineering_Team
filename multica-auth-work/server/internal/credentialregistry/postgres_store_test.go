package credentialregistry

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeRow struct {
	values []any
	err    error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return fmt.Errorf("scan destinations=%d values=%d", len(dest), len(r.values))
	}
	for i := range dest {
		target := reflect.ValueOf(dest[i])
		if target.Kind() != reflect.Pointer || target.IsNil() {
			return fmt.Errorf("destination %d is not a pointer", i)
		}
		value := reflect.ValueOf(r.values[i])
		if !value.Type().AssignableTo(target.Elem().Type()) {
			if !value.Type().ConvertibleTo(target.Elem().Type()) {
				return fmt.Errorf("value %d type %s cannot fill %s", i, value.Type(), target.Elem().Type())
			}
			value = value.Convert(target.Elem().Type())
		}
		target.Elem().Set(value)
	}
	return nil
}

type execResult struct {
	tag pgconn.CommandTag
	err error
}

type fakeTx struct {
	pgx.Tx
	mu         sync.Mutex
	rows       []fakeRow
	execs      []execResult
	statements []string
	committed  bool
	rolledBack bool
}

func (tx *fakeTx) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	tx.statements = append(tx.statements, normalizedSQL(sql))
	if len(tx.rows) == 0 {
		return fakeRow{err: errors.New("unexpected QueryRow")}
	}
	row := tx.rows[0]
	tx.rows = tx.rows[1:]
	return row
}

func (tx *fakeTx) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	tx.statements = append(tx.statements, normalizedSQL(sql))
	if len(tx.execs) == 0 {
		return pgconn.CommandTag{}, errors.New("unexpected Exec")
	}
	result := tx.execs[0]
	tx.execs = tx.execs[1:]
	return result.tag, result.err
}

func (tx *fakeTx) Commit(context.Context) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	tx.statements = append(tx.statements, "COMMIT")
	tx.committed = true
	return nil
}

func (tx *fakeTx) Rollback(context.Context) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	tx.statements = append(tx.statements, "ROLLBACK")
	tx.rolledBack = true
	return nil
}

type fakeBeginner struct {
	tx       pgx.Tx
	err      error
	options  pgx.TxOptions
	beginCnt int
}

func (b *fakeBeginner) BeginTx(_ context.Context, options pgx.TxOptions) (pgx.Tx, error) {
	b.beginCnt++
	b.options = options
	return b.tx, b.err
}

func candidateRow(candidate Candidate) fakeRow {
	return fakeRow{values: []any{
		candidate.TaskID,
		candidate.TaskStatus,
		candidate.BindingID,
		candidate.AgentID,
		candidate.WorkspaceID,
		candidate.RuntimeID,
		candidate.RuntimeSessionID,
		candidate.StandardVersionID,
		candidate.ConfigurationVersionID,
		candidate.ConfigurationDigest,
		candidate.CapabilityDigest,
		candidate.Provider,
		candidate.HomeRef,
		candidate.BindingGeneration,
		candidate.AssignmentCatalogGeneration,
		candidate.CatalogGeneration,
		candidate.Approval,
		candidate.Status,
		candidate.WorktypeScope,
		candidate.AssignmentOwners,
		candidate.BindingAssignments,
		candidate.TaskConcurrencyLimit,
	}}
}

func normalizedSQL(sql string) string {
	return strings.Join(strings.Fields(sql), " ")
}

func TestPostgresStoreClaimAndSnapshotCommitInOneTransaction(t *testing.T) {
	tx := &fakeTx{
		rows: []fakeRow{candidateRow(approvedCandidate()), {values: []any{2}}},
		execs: []execResult{
			{tag: pgconn.NewCommandTag("INSERT 0 1")},
			{tag: pgconn.NewCommandTag("UPDATE 1")},
			{tag: pgconn.NewCommandTag("INSERT 0 1")},
		},
	}
	beginner := &fakeBeginner{tx: tx}
	assignment, err := NewResolver(NewPostgresStore(beginner)).Resolve(context.Background(), approvedRequest())
	if err != nil {
		t.Fatalf("Resolve() error=%v", err)
	}
	if assignment.BindingGeneration != 11 || assignment.CatalogGeneration != 17 {
		t.Fatalf("assignment=%+v", assignment)
	}
	if beginner.options.IsoLevel != pgx.ReadCommitted || !tx.committed || tx.rolledBack {
		t.Fatalf("options=%+v committed=%v rolled_back=%v", beginner.options, tx.committed, tx.rolledBack)
	}
	if len(tx.statements) != 6 ||
		!strings.Contains(tx.statements[0], "FOR UPDATE OF t, b, a, c") ||
		!strings.Contains(tx.statements[1], "FROM runtime_binding_task_reservations") ||
		!strings.HasPrefix(tx.statements[2], "INSERT INTO runtime_binding_task_reservations") ||
		!strings.HasPrefix(tx.statements[3], "UPDATE agent_task_queue") ||
		!strings.HasPrefix(tx.statements[4], "INSERT INTO runtime_task_snapshots") ||
		tx.statements[5] != "COMMIT" {
		t.Fatalf("transaction statements=%v", tx.statements)
	}
}

func TestPostgresStoreValidationFailureRollsBackBeforeMutation(t *testing.T) {
	candidate := approvedCandidate()
	candidate.Approval = ApprovalRevoked
	tx := &fakeTx{rows: []fakeRow{candidateRow(candidate), {values: []any{0}}}}
	_, err := NewResolver(NewPostgresStore(&fakeBeginner{tx: tx})).Resolve(context.Background(), approvedRequest())
	if !errors.Is(err, ErrNoApprovedAssignment) {
		t.Fatalf("Resolve() error=%v", err)
	}
	if tx.committed || !tx.rolledBack {
		t.Fatalf("committed=%v rolled_back=%v", tx.committed, tx.rolledBack)
	}
	for _, statement := range tx.statements {
		if strings.HasPrefix(statement, "UPDATE ") || strings.HasPrefix(statement, "INSERT ") {
			t.Fatalf("validation failure executed mutation: %s", statement)
		}
	}
}

func TestPostgresStoreSnapshotFailureRollsBackTaskClaim(t *testing.T) {
	tx := &fakeTx{
		rows: []fakeRow{candidateRow(approvedCandidate()), {values: []any{0}}},
		execs: []execResult{
			{tag: pgconn.NewCommandTag("INSERT 0 1")},
			{tag: pgconn.NewCommandTag("UPDATE 1")},
			{err: errors.New("snapshot storage unavailable")},
		},
	}
	_, err := NewResolver(NewPostgresStore(&fakeBeginner{tx: tx})).Resolve(context.Background(), approvedRequest())
	if !errors.Is(err, ErrRegistryUnavailable) {
		t.Fatalf("Resolve() error=%v", err)
	}
	if tx.committed || !tx.rolledBack {
		t.Fatalf("claim escaped failed snapshot transaction: committed=%v rolled_back=%v", tx.committed, tx.rolledBack)
	}
}

func TestPostgresStoreTaskCASFailureRollsBackReservation(t *testing.T) {
	tx := &fakeTx{
		rows: []fakeRow{candidateRow(approvedCandidate()), {values: []any{0}}},
		execs: []execResult{
			{tag: pgconn.NewCommandTag("INSERT 0 1")},
			{tag: pgconn.NewCommandTag("UPDATE 0")},
		},
	}
	_, err := NewResolver(NewPostgresStore(&fakeBeginner{tx: tx})).Resolve(context.Background(), approvedRequest())
	if !errors.Is(err, ErrTaskConflict) {
		t.Fatalf("Resolve() error=%v, want task conflict", err)
	}
	if tx.committed || !tx.rolledBack {
		t.Fatalf("reservation escaped failed task CAS: committed=%v rolled_back=%v", tx.committed, tx.rolledBack)
	}
	for _, statement := range tx.statements {
		if strings.HasPrefix(statement, "INSERT INTO runtime_task_snapshots") {
			t.Fatalf("task CAS failure wrote snapshot: %s", statement)
		}
	}
}

func TestPostgresStoreReleaseIsFencedAndIdempotent(t *testing.T) {
	release := ReleaseRequest{
		TaskID: "00000000-0000-0000-0000-000000000101", BindingID: "00000000-0000-0000-0000-000000000102",
		BindingGeneration: 11, CatalogGeneration: 17,
	}
	first := &fakeTx{
		rows:  []fakeRow{{values: []any{false}}},
		execs: []execResult{{tag: pgconn.NewCommandTag("UPDATE 1")}},
	}
	if err := NewResolver(NewPostgresStore(&fakeBeginner{tx: first})).Release(context.Background(), release); err != nil {
		t.Fatalf("first Release() error=%v", err)
	}
	if !first.committed || first.rolledBack || len(first.execs) != 0 {
		t.Fatalf("first release committed=%v rolled_back=%v pending_exec=%d", first.committed, first.rolledBack, len(first.execs))
	}

	replay := &fakeTx{rows: []fakeRow{{values: []any{true}}}}
	if err := NewResolver(NewPostgresStore(&fakeBeginner{tx: replay})).Release(context.Background(), release); err != nil {
		t.Fatalf("replay Release() error=%v", err)
	}
	if !replay.committed || replay.rolledBack || len(replay.statements) != 2 {
		t.Fatalf("replay statements=%v", replay.statements)
	}
}

type racePool struct {
	mu     sync.Mutex
	active int
	limit  int
}

func (p *racePool) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	p.mu.Lock()
	return &raceTx{pool: p}, nil
}

type raceTx struct {
	pgx.Tx
	pool      *racePool
	closed    bool
	candidate Candidate
	staged    bool
}

func (tx *raceTx) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	if strings.Contains(sql, "FROM agent_task_queue") {
		candidate := approvedCandidate()
		candidate.TaskID = args[0].(string)
		candidate.TaskConcurrencyLimit = tx.pool.limit
		tx.candidate = candidate
		return candidateRow(candidate)
	}
	return fakeRow{values: []any{tx.pool.active}}
}

func (tx *raceTx) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	if strings.Contains(sql, "INSERT INTO runtime_task_snapshots") {
		tx.staged = true
		return pgconn.NewCommandTag("INSERT 0 1"), nil
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (tx *raceTx) Commit(context.Context) error {
	if tx.staged {
		tx.pool.active++
	}
	tx.close()
	return nil
}

func (tx *raceTx) Rollback(context.Context) error {
	tx.close()
	return nil
}

func (tx *raceTx) close() {
	if !tx.closed {
		tx.closed = true
		tx.pool.mu.Unlock()
	}
}

func TestPostgresStoreConcurrentTransactionsCannotOversubscribe(t *testing.T) {
	const workers, limit = 96, 5
	pool := &racePool{limit: limit}
	resolver := NewResolver(NewPostgresStore(pool))
	start := make(chan struct{})
	results := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			request := approvedRequest()
			request.TaskID = fmt.Sprintf("00000000-0000-0000-0002-%012d", i)
			_, err := resolver.Resolve(context.Background(), request)
			results <- err
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)

	successes := 0
	for err := range results {
		if err == nil {
			successes++
		} else if !errors.Is(err, ErrCapacityExhausted) {
			t.Errorf("unexpected result: %v", err)
		}
	}
	if successes != limit || pool.active != limit {
		t.Fatalf("successes=%d active=%d, want %d", successes, pool.active, limit)
	}
}
