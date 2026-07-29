//go:build orq21db

package credentialregistry

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type seedFixture struct {
	workspaceID string
	agentID     string
	accountID   string
	vendor      string
	scope       string
}

func runSeedScript(ctx context.Context, dbURL string, fixture seedFixture) ([]byte, error) {
	script, err := filepath.Abs("../../../scripts/staging/seed_approved_assignment.sql")
	if err != nil {
		return nil, err
	}
	args := []string{
		dbURL, "-X",
		"-v", "workspace_id=" + fixture.workspaceID,
		"-v", "agent_id=" + fixture.agentID,
		"-v", "account_id=" + fixture.accountID,
		"-v", "vendor=" + fixture.vendor,
		"-v", "priority=10",
		"-v", "home_dir=/private/orq21/" + fixture.accountID + "/home",
		"-v", "config_dir=/private/orq21/" + fixture.accountID + "/config",
		"-v", "worktype_scope=" + fixture.scope,
		"-f", script,
	}
	return exec.CommandContext(ctx, "psql", args...).CombinedOutput()
}

func seedAgentFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool, workspaceID, agentID, suffix string) {
	t.Helper()
	runtimeID := uuid.NewString()
	if _, err := pool.Exec(ctx, `INSERT INTO agent_runtime
		(id, workspace_id, name, runtime_mode, provider, status, device_info, metadata, last_seen_at)
		VALUES ($1, $2, $3, 'local', 'kiro', 'online', 'orq21 seed gate', '{}'::jsonb, now())`,
		runtimeID, workspaceID, "ORQ21 seed runtime "+suffix); err != nil {
		t.Fatalf("insert seed runtime: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO agent
		(id, workspace_id, name, runtime_mode, runtime_id)
		VALUES ($1, $2, $3, 'local', $4)`,
		agentID, workspaceID, "ORQ21 seed agent "+suffix, runtimeID); err != nil {
		t.Fatalf("insert seed agent: %v", err)
	}
}

func TestSeedApprovedAssignmentIdempotencyRevocationAndExclusivity(t *testing.T) {
	dbURL := os.Getenv("ORQ21_TEST_DATABASE_URL")
	if dbURL == "" {
		t.Fatal("ORQ21_TEST_DATABASE_URL is required for the orq21db gate")
	}
	if _, err := exec.LookPath("psql"); err != nil {
		t.Fatalf("psql is required for the seed-script gate: %v", err)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	t.Cleanup(pool.Close)
	workspaceID := uuid.NewString()
	if _, err := pool.Exec(ctx, `INSERT INTO workspace (id, name, slug)
		VALUES ($1, 'ORQ21 seed gate', $2)`, workspaceID, "orq21-seed-"+uuid.NewString()); err != nil {
		t.Fatalf("insert seed workspace: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM workspace WHERE id=$1`, workspaceID)
	})

	first := seedFixture{
		workspaceID: workspaceID,
		agentID:     uuid.NewString(),
		accountID:   uuid.NewString(),
		vendor:      "kiro",
		scope:       "GENERAL",
	}
	seedAgentFixture(t, ctx, pool, workspaceID, first.agentID, "first")
	for run := 1; run <= 2; run++ {
		output, err := runSeedScript(ctx, dbURL, first)
		if err != nil {
			t.Fatalf("idempotent seed run %d: %v output=%s", run, err, output)
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE approved_accounts SET allowed=false
		WHERE tenant_id=$1 AND account_id=$2`, workspaceID, first.accountID); err != nil {
		t.Fatalf("revoke approval: %v", err)
	}
	output, err := runSeedScript(ctx, dbURL, first)
	if err == nil || !strings.Contains(string(output), "E_ACCOUNT_REVOKED") {
		t.Fatalf("revoked rerun error=%v output=%s", err, output)
	}
	var allowed bool
	if err := pool.QueryRow(ctx, `SELECT allowed FROM approved_accounts
		WHERE tenant_id=$1 AND account_id=$2`, workspaceID, first.accountID).Scan(&allowed); err != nil {
		t.Fatalf("read preserved revocation: %v", err)
	}
	if allowed {
		t.Fatal("seed rerun reactivated a revoked approval")
	}

	sharedAccountID := uuid.NewString()
	competitors := []seedFixture{
		{workspaceID: workspaceID, agentID: uuid.NewString(), accountID: sharedAccountID, vendor: "kiro", scope: "GENERAL"},
		{workspaceID: workspaceID, agentID: uuid.NewString(), accountID: sharedAccountID, vendor: "kiro", scope: "GENERAL"},
	}
	seedAgentFixture(t, ctx, pool, workspaceID, competitors[0].agentID, "competitor-a")
	seedAgentFixture(t, ctx, pool, workspaceID, competitors[1].agentID, "competitor-b")
	type result struct {
		output []byte
		err    error
	}
	results := make(chan result, len(competitors))
	for _, competitor := range competitors {
		competitor := competitor
		go func() {
			output, err := runSeedScript(ctx, dbURL, competitor)
			results <- result{output: output, err: err}
		}()
	}
	successes := 0
	exclusiveFailures := 0
	for range competitors {
		got := <-results
		switch {
		case got.err == nil:
			successes++
		case strings.Contains(string(got.output), "E_ACCOUNT_ALREADY_ASSIGNED"):
			exclusiveFailures++
		default:
			t.Fatalf("unexpected concurrent seed result: %v output=%s", got.err, got.output)
		}
	}
	if successes != 1 || exclusiveFailures != 1 {
		t.Fatalf("concurrent results successes=%d exclusive_failures=%d", successes, exclusiveFailures)
	}
	var assignmentCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM assignments WHERE account_id=$1`, sharedAccountID).Scan(&assignmentCount); err != nil {
		t.Fatalf("count exclusive assignments: %v", err)
	}
	if assignmentCount != 1 {
		t.Fatalf("shared account assignments=%d, want 1", assignmentCount)
	}

	canonical := seedFixture{
		workspaceID: workspaceID,
		agentID:     uuid.NewString(),
		accountID:   uuid.NewString(),
		vendor:      " AGY ",
		scope:       "GENERAL",
	}
	seedAgentFixture(t, ctx, pool, workspaceID, canonical.agentID, "canonical-vendor")
	if output, err = runSeedScript(ctx, dbURL, canonical); err != nil {
		t.Fatalf("canonical vendor seed: %v output=%s", err, output)
	}
	var storedVendor string
	if err := pool.QueryRow(ctx, `SELECT vendor FROM accounts WHERE account_id=$1`,
		canonical.accountID).Scan(&storedVendor); err != nil {
		t.Fatalf("read canonical vendor: %v", err)
	}
	if storedVendor != "antigravity" {
		t.Fatalf("stored vendor=%q, want exact canonical antigravity", storedVendor)
	}

	unsupported := seedFixture{
		workspaceID: workspaceID,
		agentID:     uuid.NewString(),
		accountID:   uuid.NewString(),
		vendor:      "kiro",
		scope:       "CHEAP",
	}
	seedAgentFixture(t, ctx, pool, workspaceID, unsupported.agentID, "unsupported-scope")
	output, err = runSeedScript(ctx, dbURL, unsupported)
	if err == nil || !strings.Contains(string(output), "E_WORKTYPE_SCOPE_UNSUPPORTED") {
		t.Fatalf("unsupported scope error=%v output=%s", err, output)
	}
	var rows int
	if err := pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM accounts WHERE account_id=$1) +
		(SELECT count(*) FROM assignments WHERE account_id=$1)`, unsupported.accountID).Scan(&rows); err != nil {
		t.Fatalf("count unsupported-scope rows: %v", err)
	}
	if rows != 0 {
		t.Fatalf("unsupported scope mutated %d rows", rows)
	}

	var credentialRows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM credentials
		WHERE account_id IN ($1, $2, $3, $4)`,
		first.accountID, sharedAccountID, canonical.accountID, unsupported.accountID).Scan(&credentialRows); err != nil {
		t.Fatalf("count credential rows: %v", err)
	}
	if credentialRows != 0 {
		t.Fatalf("seed gate created credential rows=%d", credentialRows)
	}
}
