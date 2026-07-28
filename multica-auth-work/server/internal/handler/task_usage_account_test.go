package handler

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// ORQ-12. These tests require the ephemeral Postgres provided by the package
// TestMain: without DATABASE_URL the package exits before m.Run and none of
// them execute, which is why the ORQ-26 DB gate is the only place their result
// is meaningful.
//
// What they pin down is the property the column exists for: the account is
// resolved SERVER-SIDE from agent_task_queue.agent_id -> assignments.account_id,
// and a row that cannot be attributed stays NULL instead of being guessed.

func uuidParam(t *testing.T, s string) pgtype.UUID {
	t.Helper()
	parsed, err := uuid.Parse(s)
	if err != nil {
		t.Fatalf("parse uuid %q: %v", s, err)
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}
}

// createTestAccount inserts a rotation account and returns its id. The column
// set and the status value come from migration 123_rotation: `vendor` and
// `tenant_id` are NOT NULL, and status is constrained to
// available|leased|exhausted|cooldown|degraded, so 'active' would violate the
// CHECK.
func createTestAccount(t *testing.T, vendorLabel string) string {
	t.Helper()
	var accountID string
	if err := testPool.QueryRow(context.Background(), `
		INSERT INTO accounts (vendor, tenant_id, status)
		VALUES ($1, gen_random_uuid(), 'available')
		RETURNING account_id
	`, vendorLabel).Scan(&accountID); err != nil {
		t.Fatalf("insert account: %v", err)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM accounts WHERE account_id = $1`, accountID)
	})
	return accountID
}

// assignAgentToAccount points assignments.agent_id at the given account.
func assignAgentToAccount(t *testing.T, agentID, accountID string) {
	t.Helper()
	if _, err := testPool.Exec(context.Background(), `
		INSERT INTO assignments (agent_id, account_id)
		VALUES ($1, $2)
		ON CONFLICT (agent_id) DO UPDATE SET account_id = EXCLUDED.account_id
	`, agentID, accountID); err != nil {
		t.Fatalf("insert assignment: %v", err)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM assignments WHERE agent_id = $1`, agentID)
	})
}

func upsertUsage(t *testing.T, taskID string, in, out int64) {
	t.Helper()
	queries := db.New(testPool)
	if err := queries.UpsertTaskUsage(context.Background(), db.UpsertTaskUsageParams{
		TaskID:       uuidParam(t, taskID),
		Provider:     "orq12-test-provider",
		Model:        "orq12-test-model",
		InputTokens:  in,
		OutputTokens: out,
	}); err != nil {
		t.Fatalf("upsert task usage: %v", err)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM task_usage WHERE task_id = $1`, taskID)
	})
}

func readAccountID(t *testing.T, taskID string) (string, bool) {
	t.Helper()
	var accountID *string
	if err := testPool.QueryRow(context.Background(),
		`SELECT account_id::text FROM task_usage WHERE task_id = $1`, taskID).Scan(&accountID); err != nil {
		t.Fatalf("read account_id: %v", err)
	}
	if accountID == nil {
		return "", false
	}
	return *accountID, true
}

// The happy path: an assigned agent's usage carries that account.
func TestTaskUsageAccountID_ResolvedFromAssignment(t *testing.T) {
	accountID := createTestAccount(t, "orq12-resolved")
	agentID := createHandlerTestAgent(t, "ORQ12 Resolved", nil)
	assignAgentToAccount(t, agentID, accountID)
	taskID := createHandlerTestTaskForAgent(t, agentID)

	upsertUsage(t, taskID, 10, 20)

	got, ok := readAccountID(t, taskID)
	if !ok {
		t.Fatal("account_id must be set when the agent has an assignment")
	}
	if got != accountID {
		t.Fatalf("account_id = %s, want %s", got, accountID)
	}
}

// An agent with no assignment must leave NULL. Inventing an account here would
// misattribute spend, which is the failure this column exists to prevent.
func TestTaskUsageAccountID_NullWhenAgentHasNoAssignment(t *testing.T) {
	agentID := createHandlerTestAgent(t, "ORQ12 Unassigned", nil)
	taskID := createHandlerTestTaskForAgent(t, agentID)

	upsertUsage(t, taskID, 1, 2)

	if got, ok := readAccountID(t, taskID); ok {
		t.Fatalf("unassigned agent must leave account_id NULL, got %s", got)
	}
}

// The caller cannot choose the account: UpsertTaskUsageParams has no account
// field at all, so a compromised or buggy reporter cannot bill another account.
// Re-reporting after the assignment moves must not rewrite the snapshot either.
func TestTaskUsageAccountID_SnapshotSurvivesReassignment(t *testing.T) {
	first := createTestAccount(t, "orq12-first")
	second := createTestAccount(t, "orq12-second")
	agentID := createHandlerTestAgent(t, "ORQ12 Reassigned", nil)
	assignAgentToAccount(t, agentID, first)
	taskID := createHandlerTestTaskForAgent(t, agentID)

	upsertUsage(t, taskID, 5, 5)
	if got, _ := readAccountID(t, taskID); got != first {
		t.Fatalf("initial snapshot = %s, want %s", got, first)
	}

	// The agent rotates to another account, then the same row is re-reported.
	assignAgentToAccount(t, agentID, second)
	upsertUsage(t, taskID, 7, 9)

	got, ok := readAccountID(t, taskID)
	if !ok {
		t.Fatal("re-report must not clear the snapshot")
	}
	if got != first {
		t.Fatalf("snapshot was rewritten to %s; a recorded attribution must be immutable (want %s)",
			got, first)
	}

	// The token correction itself must have landed.
	var in, out int64
	if err := testPool.QueryRow(context.Background(),
		`SELECT input_tokens, output_tokens FROM task_usage WHERE task_id = $1`, taskID).
		Scan(&in, &out); err != nil {
		t.Fatalf("read tokens: %v", err)
	}
	if in != 7 || out != 9 {
		t.Fatalf("tokens = (%d,%d), want (7,9)", in, out)
	}
}

// A row that starts unattributable must be fillable once the assignment exists:
// COALESCE fills a NULL, and only a NULL.
func TestTaskUsageAccountID_NullIsFilledOnLaterReport(t *testing.T) {
	agentID := createHandlerTestAgent(t, "ORQ12 LateAssign", nil)
	taskID := createHandlerTestTaskForAgent(t, agentID)

	upsertUsage(t, taskID, 1, 1)
	if got, ok := readAccountID(t, taskID); ok {
		t.Fatalf("expected NULL before assignment, got %s", got)
	}

	accountID := createTestAccount(t, "orq12-late")
	assignAgentToAccount(t, agentID, accountID)
	upsertUsage(t, taskID, 2, 2)

	got, ok := readAccountID(t, taskID)
	if !ok || got != accountID {
		t.Fatalf("late assignment should fill the NULL snapshot: got %q ok=%v want %s", got, ok, accountID)
	}
}

// The report must expose the unattributable tail instead of hiding it, and the
// attribution counters must add up.
func TestListTaskUsageByAccount_ReportsNullBucketExplicitly(t *testing.T) {
	accountID := createTestAccount(t, "orq12-report")
	assignedAgent := createHandlerTestAgent(t, "ORQ12 ReportAssigned", nil)
	assignAgentToAccount(t, assignedAgent, accountID)
	assignedTask := createHandlerTestTaskForAgent(t, assignedAgent)

	unassignedAgent := createHandlerTestAgent(t, "ORQ12 ReportUnassigned", nil)
	unassignedTask := createHandlerTestTaskForAgent(t, unassignedAgent)

	from := time.Now().Add(-time.Minute)
	upsertUsage(t, assignedTask, 100, 200)
	upsertUsage(t, unassignedTask, 3, 4)
	to := time.Now().Add(time.Minute)

	queries := db.New(testPool)
	rows, err := queries.ListTaskUsageByAccount(context.Background(), db.ListTaskUsageByAccountParams{
		FromTs: pgtype.Timestamptz{Time: from, Valid: true},
		ToTs:   pgtype.Timestamptz{Time: to, Valid: true},
	})
	if err != nil {
		t.Fatalf("list by account: %v", err)
	}

	var sawAttributed, sawNull bool
	for _, row := range rows {
		if row.Attributable {
			sawAttributed = true
			continue
		}
		sawNull = true
		if row.AccountID.Valid {
			t.Fatal("the non-attributable bucket must carry a NULL account_id")
		}
		if row.RowCount == 0 {
			t.Fatal("the NULL bucket must report its row count, not be collapsed")
		}
	}
	if !sawAttributed || !sawNull {
		t.Fatalf("both buckets must appear: attributed=%v null=%v", sawAttributed, sawNull)
	}

	attr, err := queries.GetTaskUsageAccountAttribution(context.Background(),
		db.GetTaskUsageAccountAttributionParams{
			FromTs: pgtype.Timestamptz{Time: from, Valid: true},
			ToTs:   pgtype.Timestamptz{Time: to, Valid: true},
		})
	if err != nil {
		t.Fatalf("attribution: %v", err)
	}
	if attr.TotalRows != attr.AttributedRows+attr.UnattributedRows {
		t.Fatalf("counters must add up: total=%d attributed=%d unattributed=%d",
			attr.TotalRows, attr.AttributedRows, attr.UnattributedRows)
	}
	if attr.AttributedRows < 1 || attr.UnattributedRows < 1 {
		t.Fatalf("window should contain both kinds of row: %+v", attr)
	}
}
