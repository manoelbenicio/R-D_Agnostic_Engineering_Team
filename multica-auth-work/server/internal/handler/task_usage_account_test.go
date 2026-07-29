package handler

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// ORQ-12. The contract these tests pin down is the PRODUCING account:
//
//	ClaimAgentTask freezes agent_task_queue.credential_account_id, server-side,
//	from the agent's APPROVED assignment, in the same atomic statement that
//	dispatches the task. The usage upsert only COPIES that frozen value, and an
//	already-recorded attribution is immutable.
//
// The distinction matters because an agent can rotate between dispatch and the
// usage report: resolving the assignment at report time would file the spend
// under the account the agent holds NOW, not the one that produced the tokens.
//
// They require the ephemeral Postgres provided by this package's TestMain.
// Without DATABASE_URL the package exits before m.Run and none of them execute,
// which is why the DB gate is the only place their result means anything.

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
// handlerTestRuntimeProvider returns the provider of the runtime the package
// fixtures attach every task to. The claim compares it against accounts.vendor,
// so a fixture that hardcodes a vendor would silently stop matching if the
// package runtime ever changes provider.
func handlerTestRuntimeProvider(t *testing.T) string {
	t.Helper()
	var provider string
	runtimeID := handlerTestRuntimeID(t)
	if err := testPool.QueryRow(context.Background(),
		`SELECT provider FROM agent_runtime WHERE id = $1`, runtimeID).
		Scan(&provider); err != nil {
		t.Fatalf("read runtime provider: %v", err)
	}
	if provider == "handler_test_runtime" {
		provider = "codex"
		if _, err := testPool.Exec(context.Background(),
			`UPDATE agent_runtime SET provider = 'codex' WHERE id = $1`, runtimeID); err != nil {
			t.Fatalf("update runtime provider: %v", err)
		}
	}
	return provider
}

// createTestAccountFull inserts an account with an explicit tenant, vendor and
// status, which is what the authoritative predicates compare against.
func createTestAccountFull(t *testing.T, tenantID, vendor, status string) string {
	t.Helper()
	var accountID string
	if err := testPool.QueryRow(context.Background(), `
		INSERT INTO accounts (vendor, tenant_id, status)
		VALUES ($1, $2, $3)
		RETURNING account_id
	`, vendor, tenantID, status).Scan(&accountID); err != nil {
		t.Fatalf("insert account: %v", err)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM accounts WHERE account_id = $1`, accountID)
	})
	return accountID
}

// createTestAccount inserts an account that SATISFIES every predicate: the
// workspace as tenant, the task runtime's provider as vendor, status available.
// The column set and the status value come from migration 123_rotation, where
// vendor and tenant_id are NOT NULL and the CHECK does not accept 'active'.
func createTestAccount(t *testing.T, _ string) string {
	t.Helper()
	accountID := createTestAccountFull(t, testWorkspaceID, handlerTestRuntimeProvider(t), "available")
	return accountID
}

// approveAccount marks the account allowed for its own tenant, which is what
// ClaimAgentTask requires before it will freeze the account onto a task.
// approveAccountFull writes the approval with an explicit tenant, allowed flag
// and worktype scope. scope is passed as a pointer so a test can store SQL NULL,
// which the authoritative policy REJECTS.
func approveAccountFull(t *testing.T, accountID, tenantID string, allowed bool, scope *string) {
	t.Helper()
	if _, err := testPool.Exec(context.Background(), `
		INSERT INTO approved_accounts (tenant_id, account_id, allowed, worktype_scope)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (tenant_id, account_id)
		DO UPDATE SET allowed = EXCLUDED.allowed, worktype_scope = EXCLUDED.worktype_scope
	`, tenantID, accountID, allowed, scope); err != nil {
		t.Fatalf("approve account: %v", err)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM approved_accounts WHERE account_id = $1`, accountID)
	})
}

// approveAccount approves for the test workspace with the only accepted scope.
func approveAccount(t *testing.T, accountID string) {
	t.Helper()
	general := "GENERAL"
	approveAccountFull(t, accountID, testWorkspaceID, true, &general)
}

// assignAgentToAccount points assignments.agent_id at the given account. The
// staged migration makes assignments(account_id) unique, so a previous holder is
// released first.
func assignAgentToAccount(t *testing.T, agentID, accountID string) {
	t.Helper()
	ctx := context.Background()
	if _, err := testPool.Exec(ctx,
		`DELETE FROM assignments WHERE account_id = $1 AND agent_id <> $2`, accountID, agentID); err != nil {
		t.Fatalf("release previous holder: %v", err)
	}
	if _, err := testPool.Exec(ctx, `
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

func enqueueTaskForAgentWithRuntime(t *testing.T, agentID, runtimeID string) string {
	t.Helper()
	var taskID string
	if err := testPool.QueryRow(context.Background(), `
		INSERT INTO agent_task_queue (agent_id, runtime_id, status, priority)
		VALUES ($1, $2, 'queued', 0)
		RETURNING id
	`, agentID, runtimeID).Scan(&taskID); err != nil {
		t.Fatalf("enqueue task: %v", err)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM agent_task_queue WHERE id = $1`, taskID)
	})
	return taskID
}

// enqueueTaskForAgent seeds a QUEUED task, which is the only status
// ClaimAgentTask will pick up. The package helper seeds 'running' instead, so it
// cannot exercise the freeze.
func enqueueTaskForAgent(t *testing.T, agentID string) string {
	t.Helper()
	return enqueueTaskForAgentWithRuntime(t, agentID, handlerTestRuntimeID(t))
}

func createHandlerTestAgentWithRuntime(t *testing.T, name, runtimeID string) string {
	t.Helper()
	var agentID string
	if err := testPool.QueryRow(context.Background(), `
		INSERT INTO agent (
			workspace_id, name, description, runtime_mode, runtime_config,
			runtime_id, visibility, max_concurrent_tasks, owner_id,
			instructions, custom_env, custom_args, mcp_config
		)
		VALUES ($1, $2, '', 'cloud', '{}'::jsonb, $3, 'private', 1, $4, '', '{}'::jsonb, '[]'::jsonb, '{}'::jsonb)
		RETURNING id
	`, testWorkspaceID, name, runtimeID, testUserID).Scan(&agentID); err != nil {
		t.Fatalf("create handler test agent with runtime: %v", err)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM agent WHERE id = $1`, agentID)
	})
	return agentID
}

// claimTask runs the real atomic claim and returns the frozen account, if any.
func claimTask(t *testing.T, agentID, wantTaskID string) (string, bool) {
	t.Helper()
	task, err := db.New(testPool).ClaimAgentTask(context.Background(), uuidParam(t, agentID))
	if err != nil {
		t.Fatalf("claim task: %v", err)
	}
	if got := uuidText(task.ID); got != wantTaskID {
		t.Fatalf("claimed %s, want %s", got, wantTaskID)
	}
	if !task.CredentialAccountID.Valid {
		return "", false
	}
	return uuidText(task.CredentialAccountID), true
}

func uuidText(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}

func upsertUsage(t *testing.T, taskID string, in, out int64) {
	t.Helper()
	if err := db.New(testPool).UpsertTaskUsage(context.Background(), db.UpsertTaskUsageParams{
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

func readUsageAccount(t *testing.T, taskID string) (string, bool) {
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

func readUsageTokens(t *testing.T, taskID string) (int64, int64) {
	t.Helper()
	var in, out int64
	if err := testPool.QueryRow(context.Background(),
		`SELECT input_tokens, output_tokens FROM task_usage WHERE task_id = $1`, taskID).
		Scan(&in, &out); err != nil {
		t.Fatalf("read tokens: %v", err)
	}
	return in, out
}

// Happy path: the claim freezes the approved account and the usage row copies it.
func TestTaskUsageAccountID_FrozenAtClaimAndCopiedToUsage(t *testing.T) {
	accountID := createTestAccount(t, "orq12-frozen")
	approveAccount(t, accountID)
	agentID := createHandlerTestAgent(t, "ORQ12 Frozen", nil)
	assignAgentToAccount(t, agentID, accountID)
	taskID := enqueueTaskForAgent(t, agentID)

	frozen, ok := claimTask(t, agentID, taskID)
	if !ok {
		t.Fatal("claim must freeze credential_account_id when the assignment is approved")
	}
	if frozen != accountID {
		t.Fatalf("frozen account = %s, want %s", frozen, accountID)
	}

	upsertUsage(t, taskID, 10, 20)
	got, ok := readUsageAccount(t, taskID)
	if !ok || got != accountID {
		t.Fatalf("usage account = %q ok=%v, want %s", got, ok, accountID)
	}
}

// THE case the review blocked on: the agent is reassigned A -> B AFTER the claim
// but BEFORE the first usage report. The usage must still name A, because A
// produced the tokens.
func TestTaskUsageAccountID_ReassignmentBeforeFirstReportKeepsProducingAccount(t *testing.T) {
	accountA := createTestAccount(t, "orq12-producer-A")
	accountB := createTestAccount(t, "orq12-successor-B")
	approveAccount(t, accountA)
	approveAccount(t, accountB)
	agentID := createHandlerTestAgent(t, "ORQ12 ReassignBeforeReport", nil)
	assignAgentToAccount(t, agentID, accountA)
	taskID := enqueueTaskForAgent(t, agentID)

	frozen, ok := claimTask(t, agentID, taskID)
	if !ok || frozen != accountA {
		t.Fatalf("claim should freeze A: got %q ok=%v want %s", frozen, ok, accountA)
	}

	// The agent rotates to B before any usage is reported.
	assignAgentToAccount(t, agentID, accountB)

	upsertUsage(t, taskID, 11, 22)

	got, ok := readUsageAccount(t, taskID)
	if !ok {
		t.Fatal("usage must carry the producing account")
	}
	if got == accountB {
		t.Fatalf("usage was misattributed to the successor account %s; a live assignment lookup is the defect this test exists for", accountB)
	}
	if got != accountA {
		t.Fatalf("usage account = %s, want the producing account %s", got, accountA)
	}
}

// A recorded attribution is immutable: a later re-report cannot rewrite it, and
// the token correction must still land.
func TestTaskUsageAccountID_ReReportCannotRewriteRecordedAccount(t *testing.T) {
	accountA := createTestAccount(t, "orq12-immutable-A")
	accountB := createTestAccount(t, "orq12-immutable-B")
	approveAccount(t, accountA)
	approveAccount(t, accountB)
	agentID := createHandlerTestAgent(t, "ORQ12 Immutable", nil)
	assignAgentToAccount(t, agentID, accountA)
	taskID := enqueueTaskForAgent(t, agentID)
	claimTask(t, agentID, taskID)

	upsertUsage(t, taskID, 5, 5)

	// Force the queue snapshot to B to prove the usage row does not follow it.
	if _, err := testPool.Exec(context.Background(),
		`UPDATE agent_task_queue SET credential_account_id = $1 WHERE id = $2`,
		accountB, taskID); err != nil {
		t.Fatalf("force queue snapshot: %v", err)
	}
	upsertUsage(t, taskID, 7, 9)

	got, ok := readUsageAccount(t, taskID)
	if !ok || got != accountA {
		t.Fatalf("recorded attribution must be immutable: got %q ok=%v want %s", got, ok, accountA)
	}
	if in, out := readUsageTokens(t, taskID); in != 7 || out != 9 {
		t.Fatalf("tokens = (%d,%d), want (7,9)", in, out)
	}
}

// An agent with no APPROVED assignment leaves NULL at claim, and the usage row
// inherits NULL. Inventing an account here would misattribute spend.
func TestTaskUsageAccountID_NullWhenNoApprovedAssignment(t *testing.T) {
	t.Run("no_assignment_at_all", func(t *testing.T) {
		agentID := createHandlerTestAgent(t, "ORQ12 NoAssignment", nil)
		taskID := enqueueTaskForAgent(t, agentID)
		if frozen, ok := claimTask(t, agentID, taskID); ok {
			t.Fatalf("expected NULL, got %s", frozen)
		}
		upsertUsage(t, taskID, 1, 2)
		if got, ok := readUsageAccount(t, taskID); ok {
			t.Fatalf("usage must stay NULL, got %s", got)
		}
	})

	t.Run("assigned_but_not_approved", func(t *testing.T) {
		accountID := createTestAccount(t, "orq12-unapproved")
		agentID := createHandlerTestAgent(t, "ORQ12 Unapproved", nil)
		assignAgentToAccount(t, agentID, accountID) // deliberately NOT approved
		taskID := enqueueTaskForAgent(t, agentID)
		if frozen, ok := claimTask(t, agentID, taskID); ok {
			t.Fatalf("an unapproved assignment must not be frozen, got %s", frozen)
		}
	})

	t.Run("approval_revoked", func(t *testing.T) {
		accountID := createTestAccount(t, "orq12-revoked")
		approveAccount(t, accountID)
		if _, err := testPool.Exec(context.Background(),
			`UPDATE approved_accounts SET allowed = false WHERE account_id = $1`, accountID); err != nil {
			t.Fatalf("revoke approval: %v", err)
		}
		agentID := createHandlerTestAgent(t, "ORQ12 Revoked", nil)
		assignAgentToAccount(t, agentID, accountID)
		taskID := enqueueTaskForAgent(t, agentID)
		if frozen, ok := claimTask(t, agentID, taskID); ok {
			t.Fatalf("a revoked approval must not be frozen, got %s", frozen)
		}
	})
}

// A row written before the column existed keeps NULL and is not retro-attributed
// by a later report. The pre-migration row is simulated by writing usage for a
// task whose queue snapshot is NULL, which is exactly the legacy shape.
func TestTaskUsageAccountID_LegacyRowStaysNullAndIsNotRetroAttributed(t *testing.T) {
	agentID := createHandlerTestAgent(t, "ORQ12 Legacy", nil)
	taskID := enqueueTaskForAgent(t, agentID)
	claimTask(t, agentID, taskID) // no assignment: snapshot stays NULL

	upsertUsage(t, taskID, 3, 4)
	if got, ok := readUsageAccount(t, taskID); ok {
		t.Fatalf("legacy row must be NULL, got %s", got)
	}

	// The account only appears later. A legacy row must NOT be back-filled from
	// an assignment that did not produce it.
	accountID := createTestAccount(t, "orq12-legacy-late")
	approveAccount(t, accountID)
	assignAgentToAccount(t, agentID, accountID)
	upsertUsage(t, taskID, 5, 6)

	if got, ok := readUsageAccount(t, taskID); ok {
		t.Fatalf("legacy row was retro-attributed to %s; the queue snapshot is the only source", got)
	}
	// A NULL queue snapshot that is later frozen DOES fill the usage row: that is
	// the same account, just recorded late.
	if _, err := testPool.Exec(context.Background(),
		`UPDATE agent_task_queue SET credential_account_id = $1 WHERE id = $2`,
		accountID, taskID); err != nil {
		t.Fatalf("freeze late: %v", err)
	}
	upsertUsage(t, taskID, 7, 8)
	if got, ok := readUsageAccount(t, taskID); !ok || got != accountID {
		t.Fatalf("a NULL attribution must be fillable: got %q ok=%v want %s", got, ok, accountID)
	}
}

// Deleting the account must lose only the attribution, never the tokens.
func TestTaskUsageAccountID_AccountDeletePreservesTokens(t *testing.T) {
	accountID := createTestAccount(t, "orq12-deleted")
	approveAccount(t, accountID)
	agentID := createHandlerTestAgent(t, "ORQ12 Deleted", nil)
	assignAgentToAccount(t, agentID, accountID)
	taskID := enqueueTaskForAgent(t, agentID)
	claimTask(t, agentID, taskID)
	upsertUsage(t, taskID, 42, 84)

	if got, ok := readUsageAccount(t, taskID); !ok || got != accountID {
		t.Fatalf("precondition: usage should carry %s, got %q ok=%v", accountID, got, ok)
	}

	if _, err := testPool.Exec(context.Background(),
		`DELETE FROM accounts WHERE account_id = $1`, accountID); err != nil {
		t.Fatalf("delete account: %v", err)
	}

	// The usage row must still exist, with its tokens, and a NULL attribution.
	in, out := readUsageTokens(t, taskID)
	if in != 42 || out != 84 {
		t.Fatalf("tokens must survive the account deletion: got (%d,%d), want (42,84)", in, out)
	}
	if got, ok := readUsageAccount(t, taskID); ok {
		t.Fatalf("attribution should be NULL after ON DELETE SET NULL, got %s", got)
	}
	// And the queue snapshot must be nulled the same way, not cascade-deleted.
	var queueRows int
	if err := testPool.QueryRow(context.Background(),
		`SELECT count(*) FROM agent_task_queue WHERE id = $1 AND credential_account_id IS NULL`,
		taskID).Scan(&queueRows); err != nil {
		t.Fatalf("read queue row: %v", err)
	}
	if queueRows != 1 {
		t.Fatalf("queue row must survive with a NULL snapshot, found %d", queueRows)
	}
}

// The conflict target is still (task_id, provider, model): a re-report updates
// the one row instead of adding a second one that would double-count tokens.
func TestTaskUsageAccountID_ConflictKeepsExactlyOneRow(t *testing.T) {
	accountID := createTestAccount(t, "orq12-onerow")
	approveAccount(t, accountID)
	agentID := createHandlerTestAgent(t, "ORQ12 OneRow", nil)
	assignAgentToAccount(t, agentID, accountID)
	taskID := enqueueTaskForAgent(t, agentID)
	claimTask(t, agentID, taskID)

	upsertUsage(t, taskID, 1, 1)
	upsertUsage(t, taskID, 2, 2)
	upsertUsage(t, taskID, 3, 3)

	var rows int
	if err := testPool.QueryRow(context.Background(),
		`SELECT count(*) FROM task_usage WHERE task_id = $1`, taskID).Scan(&rows); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if rows != 1 {
		t.Fatalf("three reports must collapse into one row, found %d", rows)
	}
	if in, out := readUsageTokens(t, taskID); in != 3 || out != 3 {
		t.Fatalf("last report must win on tokens: got (%d,%d), want (3,3)", in, out)
	}
}

// The report is workspace-scoped and keeps the unattributable bucket visible.
func TestListTaskUsageByAccount_ScopedToWorkspaceAndKeepsNullBucket(t *testing.T) {
	accountID := createTestAccount(t, "orq12-report")
	approveAccount(t, accountID)
	assignedAgent := createHandlerTestAgent(t, "ORQ12 ReportAssigned", nil)
	assignAgentToAccount(t, assignedAgent, accountID)
	assignedTask := enqueueTaskForAgent(t, assignedAgent)
	claimTask(t, assignedAgent, assignedTask)

	unassignedAgent := createHandlerTestAgent(t, "ORQ12 ReportUnassigned", nil)
	unassignedTask := enqueueTaskForAgent(t, unassignedAgent)
	claimTask(t, unassignedAgent, unassignedTask)

	from := pgtype.Timestamptz{Time: time.Now().Add(-time.Minute), Valid: true}
	upsertUsage(t, assignedTask, 100, 200)
	upsertUsage(t, unassignedTask, 3, 4)
	to := pgtype.Timestamptz{Time: time.Now().Add(time.Minute), Valid: true}

	queries := db.New(testPool)
	ws := parseUUID(testWorkspaceID)
	rows, err := queries.ListTaskUsageByAccount(context.Background(), db.ListTaskUsageByAccountParams{
		WorkspaceID: ws, FromTs: from, ToTs: to,
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
		db.GetTaskUsageAccountAttributionParams{WorkspaceID: ws, FromTs: from, ToTs: to})
	if err != nil {
		t.Fatalf("attribution: %v", err)
	}
	if attr.TotalRows != attr.AttributedRows+attr.UnattributedRows {
		t.Fatalf("counters must add up: %+v", attr)
	}
	if attr.AttributedRows < 1 || attr.UnattributedRows < 1 {
		t.Fatalf("window should contain both kinds of row: %+v", attr)
	}

	// Another workspace must see none of it: the scope is enforced in SQL, not
	// only in the handler.
	var otherWorkspace string
	// migration 001 defines workspace(id, name, slug, description, settings,
	// created_at, updated_at). There is no owner column: ownership lives in
	// `member`, and this check does not need a member row because it calls the
	// query directly rather than going through the handler.
	if err := testPool.QueryRow(context.Background(), `
		INSERT INTO workspace (name, slug)
		VALUES ('ORQ12 Other', 'orq12-other-' || substr(gen_random_uuid()::text, 1, 8))
		RETURNING id
	`).Scan(&otherWorkspace); err != nil {
		t.Skipf("could not create a second workspace for the scope check: %v", err)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM workspace WHERE id = $1`, otherWorkspace)
	})
	otherAttr, err := queries.GetTaskUsageAccountAttribution(context.Background(),
		db.GetTaskUsageAccountAttributionParams{
			WorkspaceID: parseUUID(otherWorkspace), FromTs: from, ToTs: to,
		})
	if err != nil {
		t.Fatalf("attribution for the other workspace: %v", err)
	}
	if otherAttr.TotalRows != 0 {
		t.Fatalf("another workspace must not see this usage: %+v", otherAttr)
	}
}

// ---------------------------------------------------------------------------
// Refusal matrix: one case per authoritative predicate.
//
// Source of truth: adr-orq21-orq12-producing-account-snapshot.md, sha256
// f34e6dcc69112292c782294b857df419790f1a2ff1ef87f53150529bcc49139b, plus the
// ORQ-21 R3 rulings. Every case below asserts that a predicate MISS leaves the
// snapshot NULL rather than borrowing an account. The policy is deliberately
// unforgiving: an unimported or mislabelled approval is non-executable, not
// silently promoted.
// ---------------------------------------------------------------------------

func freezeAttemptWithRuntime(t *testing.T, name, runtimeID, accountID string) (agentID, taskID, frozen string, ok bool) {
	t.Helper()
	agentID = createHandlerTestAgentWithRuntime(t, name, runtimeID)
	assignAgentToAccount(t, agentID, accountID)
	taskID = enqueueTaskForAgentWithRuntime(t, agentID, runtimeID)
	frozen, ok = claimTask(t, agentID, taskID)
	return agentID, taskID, frozen, ok
}

// freezeAttempt wires one agent + assignment + task and returns the frozen
// account after a real claim.
func freezeAttempt(t *testing.T, name, accountID string) (agentID, taskID, frozen string, ok bool) {
	t.Helper()
	return freezeAttemptWithRuntime(t, name, handlerTestRuntimeID(t), accountID)
}

func TestClaimFreeze_RefusesTenantMismatchOnAccount(t *testing.T) {
	// The account is approved and usable, but it belongs to another workspace.
	other := uuid.NewString()
	accountID := createTestAccountFull(t, other, handlerTestRuntimeProvider(t), "available")
	general := "GENERAL"
	approveAccountFull(t, accountID, testWorkspaceID, true, &general)

	if _, _, frozen, ok := freezeAttempt(t, "ORQ12 TenantMismatchAccount", accountID); ok {
		t.Fatalf("an account from another workspace must not be frozen, got %s", frozen)
	}
}

func TestClaimFreeze_RefusesTenantMismatchOnApproval(t *testing.T) {
	// The account is in this workspace, but the approval was granted elsewhere.
	accountID := createTestAccountFull(t, testWorkspaceID, handlerTestRuntimeProvider(t), "available")
	general := "GENERAL"
	approveAccountFull(t, accountID, uuid.NewString(), true, &general)

	if _, _, frozen, ok := freezeAttempt(t, "ORQ12 TenantMismatchApproval", accountID); ok {
		t.Fatalf("an approval from another tenant must not be frozen, got %s", frozen)
	}
}

func TestClaimFreeze_RefusesVendorMismatch(t *testing.T) {
	accountID := createTestAccountFull(t, testWorkspaceID, "orq12-not-the-runtime-vendor", "available")
	approveAccount(t, accountID)

	if _, _, frozen, ok := freezeAttempt(t, "ORQ12 VendorMismatch", accountID); ok {
		t.Fatalf("a vendor that does not match the task runtime must not be frozen, got %s", frozen)
	}
}

// Direct un-normalized drift injected into DB must fail closed under exact claim matching.
func TestClaimFreeze_RefusesDirectNonCanonicalDriftInDB(t *testing.T) {
	ctx := context.Background()
	runtimeID := handlerTestRuntimeID(t)
	var originalProvider string
	if err := testPool.QueryRow(ctx, `SELECT provider FROM agent_runtime WHERE id = $1`, runtimeID).Scan(&originalProvider); err != nil {
		t.Fatalf("read runtime provider: %v", err)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `UPDATE agent_runtime SET provider = $1 WHERE id = $2`, originalProvider, runtimeID)
	})

	t.Run("unnormalized_runtime_provider_in_db", func(t *testing.T) {
		if _, err := testPool.Exec(ctx, `UPDATE agent_runtime SET provider = 'agy' WHERE id = $1`, runtimeID); err != nil {
			t.Fatalf("set provider to agy: %v", err)
		}
		accountID := createTestAccountFull(t, testWorkspaceID, "antigravity", "available")
		approveAccount(t, accountID)

		if _, _, frozen, ok := freezeAttemptWithRuntime(t, "ORQ12 UnnormalizedRuntime", runtimeID, accountID); ok {
			t.Fatalf("un-normalized runtime provider 'agy' in DB must fail closed on claim, got %s", frozen)
		}
	})

	t.Run("unnormalized_account_vendor_in_db", func(t *testing.T) {
		if _, err := testPool.Exec(ctx, `UPDATE agent_runtime SET provider = 'antigravity' WHERE id = $1`, runtimeID); err != nil {
			t.Fatalf("set provider: %v", err)
		}
		accountID := createTestAccountFull(t, testWorkspaceID, " AGY ", "available")
		approveAccount(t, accountID)

		if _, _, frozen, ok := freezeAttemptWithRuntime(t, "ORQ12 UnnormalizedVendor", runtimeID, accountID); ok {
			t.Fatalf("un-normalized vendor ' AGY ' in DB must fail closed on claim, got %s", frozen)
		}
	})
}

// Alias acceptance belongs to write-path normalization (normalizeProvider), which
// canonicalises 'agy' / trim / case -> 'antigravity' for storage, so exact claim
// (acc.vendor = rt.provider) succeeds.
func TestWritePath_NormalizeProviderAndClaim(t *testing.T) {
	if got := normalizeProvider("agy"); got != "antigravity" {
		t.Fatalf("normalizeProvider('agy') = %q, want 'antigravity'", got)
	}
	if got := normalizeProvider(" AGY "); got != "antigravity" {
		t.Fatalf("normalizeProvider(' AGY ') = %q, want 'antigravity'", got)
	}
	if got := normalizeProvider(" Codex "); got != "codex" {
		t.Fatalf("normalizeProvider(' Codex ') = %q, want 'codex'", got)
	}
	if got := normalizeProvider(" KIRO "); got != "kiro" {
		t.Fatalf("normalizeProvider(' KIRO ') = %q, want 'kiro'", got)
	}

	ctx := context.Background()
	runtimeID := handlerTestRuntimeID(t)
	var original string
	if err := testPool.QueryRow(ctx,
		`SELECT provider FROM agent_runtime WHERE id = $1`, runtimeID).Scan(&original); err != nil {
		t.Fatalf("read provider: %v", err)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(),
			`UPDATE agent_runtime SET provider = $1 WHERE id = $2`, original, runtimeID)
	})

	// Simulate write-path normalization on runtime registration
	canonicalProvider := normalizeProvider("agy")
	if _, err := testPool.Exec(ctx,
		`UPDATE agent_runtime SET provider = $1 WHERE id = $2`, canonicalProvider, runtimeID); err != nil {
		t.Fatalf("set canonical provider: %v", err)
	}

	accountID := createTestAccountFull(t, testWorkspaceID, "antigravity", "available")
	approveAccount(t, accountID)

	_, _, frozen, ok := freezeAttemptWithRuntime(t, "ORQ12 CanonicalWritePathClaim", runtimeID, accountID)
	if !ok || frozen != accountID {
		t.Fatalf("canonical write path claim failed: got %q ok=%v want %s", frozen, ok, accountID)
	}
}

func TestClaimFreeze_AccountStatusPolicy(t *testing.T) {
	// Only available and leased can produce work.
	for _, status := range []string{"exhausted", "cooldown", "degraded"} {
		t.Run("refuses_"+status, func(t *testing.T) {
			accountID := createTestAccountFull(t, testWorkspaceID, handlerTestRuntimeProvider(t), status)
			approveAccount(t, accountID)
			if _, _, frozen, ok := freezeAttempt(t, "ORQ12 Status "+status, accountID); ok {
				t.Fatalf("status %s must not be frozen, got %s", status, frozen)
			}
		})
	}
	t.Run("accepts_leased", func(t *testing.T) {
		accountID := createTestAccountFull(t, testWorkspaceID, handlerTestRuntimeProvider(t), "leased")
		approveAccount(t, accountID)
		_, _, frozen, ok := freezeAttempt(t, "ORQ12 Status leased", accountID)
		if !ok || frozen != accountID {
			t.Fatalf("leased must be usable: got %q ok=%v want %s", frozen, ok, accountID)
		}
	})
}

func TestClaimFreeze_WorktypeScopePolicy(t *testing.T) {
	provider := handlerTestRuntimeProvider(t)
	for _, tc := range []struct {
		name  string
		scope *string
	}{
		{"refuses_heavy", strPtr("HEAVY")},
		{"refuses_cheap", strPtr("CHEAP")},
		{"refuses_review", strPtr("REVIEW")},
		// NULL is rejected on purpose: no implicit NULL -> GENERAL, no backfill.
		{"refuses_null", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			accountID := createTestAccountFull(t, testWorkspaceID, provider, "available")
			approveAccountFull(t, accountID, testWorkspaceID, true, tc.scope)
			if _, _, frozen, ok := freezeAttempt(t, "ORQ12 Scope "+tc.name, accountID); ok {
				t.Fatalf("scope %v must not be frozen, got %s", tc.scope, frozen)
			}
		})
	}
}

func TestClaimFreeze_RefusesRevokedApproval(t *testing.T) {
	accountID := createTestAccountFull(t, testWorkspaceID, handlerTestRuntimeProvider(t), "available")
	general := "GENERAL"
	approveAccountFull(t, accountID, testWorkspaceID, false, &general)

	if _, _, frozen, ok := freezeAttempt(t, "ORQ12 RevokedApproval", accountID); ok {
		t.Fatalf("allowed=false must not be frozen, got %s", frozen)
	}
}

// ADR item 6: an agent_task_queue row is one ATTEMPT. A retry inserts a fresh row
// that starts unfrozen and freezes the account valid at ITS OWN claim, so the two
// attempts can legitimately name different accounts and each usage row stays
// correct.
func TestClaimFreeze_RetryAttemptFreezesIndependently(t *testing.T) {
	provider := handlerTestRuntimeProvider(t)
	accountA := createTestAccountFull(t, testWorkspaceID, provider, "available")
	accountB := createTestAccountFull(t, testWorkspaceID, provider, "available")
	approveAccount(t, accountA)
	approveAccount(t, accountB)

	agentID, firstTask, frozen, ok := freezeAttempt(t, "ORQ12 RetryAttempt", accountA)
	if !ok || frozen != accountA {
		t.Fatalf("first attempt should freeze A: got %q ok=%v", frozen, ok)
	}
	upsertUsage(t, firstTask, 10, 10)

	// The agent rotates to B, then a retry attempt is enqueued and claimed.
	assignAgentToAccount(t, agentID, accountB)
	secondTask := enqueueTaskForAgent(t, agentID)

	// The first attempt must be out of the way for the serialisation guard.
	if _, err := testPool.Exec(context.Background(),
		`UPDATE agent_task_queue SET status = 'completed' WHERE id = $1`, firstTask); err != nil {
		t.Fatalf("complete first attempt: %v", err)
	}
	secondFrozen, ok := claimTask(t, agentID, secondTask)
	if !ok || secondFrozen != accountB {
		t.Fatalf("the retry attempt must freeze B: got %q ok=%v want %s", secondFrozen, ok, accountB)
	}
	upsertUsage(t, secondTask, 20, 20)

	// Each attempt keeps its own producing account.
	if got, _ := readUsageAccount(t, firstTask); got != accountA {
		t.Fatalf("first attempt usage = %s, want %s", got, accountA)
	}
	if got, _ := readUsageAccount(t, secondTask); got != accountB {
		t.Fatalf("retry attempt usage = %s, want %s", got, accountB)
	}
}

func TestClaimFreeze_CoveredProvidersOnly(t *testing.T) {
	ctx := context.Background()
	runtimeID := handlerTestRuntimeID(t)
	var originalProvider string
	if err := testPool.QueryRow(ctx, `SELECT provider FROM agent_runtime WHERE id = $1`, runtimeID).Scan(&originalProvider); err != nil {
		t.Fatalf("read runtime provider: %v", err)
	}

	t.Cleanup(func() {
		testPool.Exec(context.Background(), `UPDATE agent_runtime SET provider = $1 WHERE id = $2`, originalProvider, runtimeID)
	})

	for _, covered := range []string{"codex", "kiro", "antigravity"} {
		t.Run("covered_"+covered, func(t *testing.T) {
			if _, err := testPool.Exec(ctx, `UPDATE agent_runtime SET provider = $1 WHERE id = $2`, covered, runtimeID); err != nil {
				t.Fatalf("set provider to %s: %v", covered, err)
			}
			accountID := createTestAccountFull(t, testWorkspaceID, covered, "available")
			approveAccount(t, accountID)

			_, _, frozen, ok := freezeAttempt(t, "ORQ12 Covered "+covered, accountID)
			if !ok || frozen != accountID {
				t.Fatalf("covered provider %s must freeze account: got %q ok=%v want %s", covered, frozen, ok, accountID)
			}
		})
	}

	for _, uncovered := range []string{"openai", "anthropic", "ollama"} {
		t.Run("uncovered_"+uncovered, func(t *testing.T) {
			if _, err := testPool.Exec(ctx, `UPDATE agent_runtime SET provider = $1 WHERE id = $2`, uncovered, runtimeID); err != nil {
				t.Fatalf("set provider to %s: %v", uncovered, err)
			}
			accountID := createTestAccountFull(t, testWorkspaceID, uncovered, "available")
			approveAccount(t, accountID)

			if _, _, frozen, ok := freezeAttempt(t, "ORQ12 Uncovered "+uncovered, accountID); ok {
				t.Fatalf("uncovered provider %s must NOT freeze account, got %s", uncovered, frozen)
			}
		})
	}
}

// reclaimFixture puts one task of this runtime into the exact shape reclaim
// recovers from - `dispatched`, never started, older than the recovery window -
// with the given frozen account snapshot (nil = the legacy NULL snapshot every
// in-flight task carries before the account column ships).
func reclaimFixture(t *testing.T, name, provider string, frozenAccountID *string) (string, string) {
	t.Helper()
	ctx := context.Background()
	runtimeID := handlerTestRuntimeID(t)
	var originalProvider string
	if err := testPool.QueryRow(ctx, `SELECT provider FROM agent_runtime WHERE id = $1`, runtimeID).Scan(&originalProvider); err != nil {
		t.Fatalf("read runtime provider: %v", err)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `UPDATE agent_runtime SET provider = $1 WHERE id = $2`, originalProvider, runtimeID)
	})
	if _, err := testPool.Exec(ctx, `UPDATE agent_runtime SET provider = $1 WHERE id = $2`, provider, runtimeID); err != nil {
		t.Fatalf("set provider to %s: %v", provider, err)
	}

	agentID := createHandlerTestAgent(t, name, nil)
	taskID := enqueueTaskForAgent(t, agentID)
	if _, err := testPool.Exec(ctx, `
		UPDATE agent_task_queue
		SET status = 'dispatched', dispatched_at = now() - INTERVAL '10 minutes',
		    started_at = NULL, credential_account_id = $2
		WHERE id = $1
	`, taskID, frozenAccountID); err != nil {
		t.Fatalf("setup dispatched task: %v", err)
	}
	return runtimeID, taskID
}

// A covered-provider task that reached `dispatched` with no account snapshot is
// exactly the shape of every task in flight when this change ships. Reclaim used
// to drop it from the candidate set, which made it invisible rather than
// fail-closed: it could then be neither recovered nor cancelled. It must be
// observable, and reclaim must still refuse to invent an account for it, so the
// claim path can see it and cancel it fail-closed.
func TestReclaim_SurfacesCoveredProviderLegacyNullSnapshot(t *testing.T) {
	ctx := context.Background()
	queries := db.New(testPool)

	for _, covered := range []string{"codex", "kiro", "antigravity"} {
		t.Run(covered, func(t *testing.T) {
			runtimeID, taskID := reclaimFixture(t, "ORQ12 ReclaimNullSnapshot "+covered, covered, nil)

			reclaimed, err := queries.ReclaimStaleDispatchedTaskForRuntime(ctx, db.ReclaimStaleDispatchedTaskForRuntimeParams{
				RuntimeID:         uuidParam(t, runtimeID),
				ClaimRecoverySecs: 60.0,
			})
			if err != nil {
				t.Fatalf("legacy NULL-snapshot task must stay observable to the claim path, got: %v", err)
			}
			if got := uuidText(reclaimed.ID); got != taskID {
				t.Fatalf("reclaimed task=%s, want %s", got, taskID)
			}
			if reclaimed.CredentialAccountID.Valid {
				t.Fatalf("reclaim must never resolve an account, got %v", reclaimed.CredentialAccountID)
			}

			// The stored row must also still be NULL: no backfill, no borrowed
			// account. The claim path is what turns this into a cancellation.
			var stored *string
			if err := testPool.QueryRow(ctx,
				`SELECT credential_account_id::text FROM agent_task_queue WHERE id = $1`, taskID).
				Scan(&stored); err != nil {
				t.Fatalf("read stored snapshot: %v", err)
			}
			if stored != nil {
				t.Fatalf("stored credential_account_id=%s, want NULL after reclaim", *stored)
			}
		})
	}
}

// The counterpart contract: a task whose account was frozen at claim reclaims
// normally and keeps that exact snapshot, so a lost claim response cannot re-file
// the spend under a different account.
func TestReclaim_PreservesFrozenAccountSnapshot(t *testing.T) {
	ctx := context.Background()
	queries := db.New(testPool)

	accountID := createTestAccountFull(t, testWorkspaceID, "codex", "available")
	approveAccount(t, accountID)
	runtimeID, taskID := reclaimFixture(t, "ORQ12 ReclaimFrozenSnapshot", "codex", &accountID)

	reclaimed, err := queries.ReclaimStaleDispatchedTaskForRuntime(ctx, db.ReclaimStaleDispatchedTaskForRuntimeParams{
		RuntimeID:         uuidParam(t, runtimeID),
		ClaimRecoverySecs: 60.0,
	})
	if err != nil {
		t.Fatalf("reclaim of a frozen-account task must succeed: %v", err)
	}
	if got := uuidText(reclaimed.ID); got != taskID {
		t.Fatalf("reclaimed task=%s, want %s", got, taskID)
	}
	if !reclaimed.CredentialAccountID.Valid || uuidText(reclaimed.CredentialAccountID) != accountID {
		t.Fatalf("reclaim changed the frozen snapshot: got %v, want %s", reclaimed.CredentialAccountID, accountID)
	}
	if !reclaimed.DispatchedAt.Valid || time.Since(reclaimed.DispatchedAt.Time) > time.Minute {
		t.Fatalf("reclaim must refresh dispatched_at, got %v", reclaimed.DispatchedAt)
	}
}
