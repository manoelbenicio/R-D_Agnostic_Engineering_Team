package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// ORQ-35 / ORQ-60 — Ephemeral PostgreSQL 17 HBA & OS Peer Map Confinement Test Suite.
//
// Proves on real PostgreSQL 17 binary (/usr/bin/postgres):
// 1. OS Peer Map Confinement: OS user 'postgres' (or mapped OS user) reaches recovery role multica_recovery;
//    unmapped OS identities (ec2-user/other) CANNOT reach multica_recovery via peer auth.
// 2. TCP SCRAM Enforcement: Connection without password or with wrong password fails;
//    app DSN with correct runtime secret succeeds.
// 3. Reversible Rollback: pg_hba.conf reload and rollback are atomically reversible via pg_ctl reload.

func getFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func TestPeerHBA_PG17_ConfinementAndRollback(t *testing.T) {
	if _, err := os.Stat("/usr/bin/postgres"); os.IsNotExist(err) {
		t.Skip("PostgreSQL 17 binary /usr/bin/postgres not found, skipping ephemeral PG17 test")
	}

	currentOSUserObj, err := user.Current()
	if err != nil {
		t.Fatalf("failed to get current OS user: %v", err)
	}
	currentOSUser := currentOSUserObj.Username

	// Create ephemeral directories
	tmpDir, err := os.MkdirTemp("", "pg17_orq35_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dataDir := filepath.Join(tmpDir, "data")
	sockDir := filepath.Join(tmpDir, "sockets")
	if err := os.MkdirAll(sockDir, 0755); err != nil {
		t.Fatalf("failed to create socket dir: %v", err)
	}

	port := getFreePort(t)

	// Step 1: Run initdb
	initCmd := exec.Command("/usr/bin/initdb", "-D", dataDir, "-U", "postgres", "--auth=trust")
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("initdb failed: %v\nOutput: %s", err, string(out))
	}

	// Step 2: Start postgres process
	pgCmd := exec.Command("/usr/bin/postgres", "-D", dataDir, "-k", sockDir, "-p", fmt.Sprintf("%d", port))
	if err := pgCmd.Start(); err != nil {
		t.Fatalf("failed to start postgres: %v", err)
	}
	defer func() {
		if pgCmd.Process != nil {
			_ = pgCmd.Process.Kill()
		}
	}()

	// Wait for PG to start accepting connections on trust
	sockDSN := fmt.Sprintf("host=%s port=%d dbname=postgres user=postgres", sockDir, port)
	var pool *pgx.Conn
	var connErr error
	for i := 0; i < 30; i++ {
		time.Sleep(200 * time.Millisecond)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		pool, connErr = pgx.Connect(ctx, sockDSN)
		cancel()
		if connErr == nil {
			break
		}
	}
	if connErr != nil {
		t.Fatalf("ephemeral postgres failed to start: %v", connErr)
	}

	ctx := context.Background()

	// Step 3: Bootstrap roles and database
	appSecret := "secure_scram_secret_998877"
	if _, err := pool.Exec(ctx, "CREATE ROLE multica_recovery WITH LOGIN SUPERUSER CREATEROLE CREATEDB BYPASSRLS REPLICATION;"); err != nil {
		pool.Close(context.Background())
		t.Fatalf("failed to create multica_recovery: %v", err)
	}
	if _, err := pool.Exec(ctx, fmt.Sprintf("CREATE ROLE multica_transition WITH LOGIN SUPERUSER CREATEROLE CREATEDB BYPASSRLS REPLICATION PASSWORD '%s';", appSecret)); err != nil {
		pool.Close(context.Background())
		t.Fatalf("failed to create multica_transition: %v", err)
	}
	if _, err := pool.Exec(ctx, "CREATE DATABASE multica_transition OWNER multica_transition;"); err != nil {
		pool.Close(context.Background())
		t.Fatalf("failed to create db multica_transition: %v", err)
	}
	pool.Close(context.Background())

	// Step 4: Configure pg_ident.conf and pg_hba.conf for ORQ-35 peer map confinement
	// recovery_map maps system user 'postgres' to DB user 'multica_recovery' ONLY.
	pgIdentContent := `# ORQ-35 User Map
recovery_map    postgres                multica_recovery
`
	if err := os.WriteFile(filepath.Join(dataDir, "pg_ident.conf"), []byte(pgIdentContent), 0600); err != nil {
		t.Fatalf("failed to write pg_ident.conf: %v", err)
	}

	pgHBAContent := `# ORQ-35 Client Auth Rules
local   all             multica_recovery                          peer    map=recovery_map
local   all             all                                       scram-sha-256
host    all             all               127.0.0.1/32            scram-sha-256
host    all             all               ::1/128                 scram-sha-256
host    all             all               all                     scram-sha-256
`
	if err := os.WriteFile(filepath.Join(dataDir, "pg_hba.conf"), []byte(pgHBAContent), 0600); err != nil {
		t.Fatalf("failed to write pg_hba.conf: %v", err)
	}

	// Reload PG configuration via pg_ctl
	reloadCmd := exec.Command("/usr/bin/pg_ctl", "reload", "-D", dataDir)
	if out, err := reloadCmd.CombinedOutput(); err != nil {
		t.Fatalf("pg_ctl reload failed: %v\nOutput: %s", err, string(out))
	}
	time.Sleep(300 * time.Millisecond)

	// =========================================================================
	// ASSERTION 1: Peer Authentication Confinement (Negative Proof)
	// =========================================================================
	// Current OS user is 'ec2-user' (or unmapped OS user).
	// Attempting unix socket peer auth as 'multica_recovery' MUST FAIL
	// because 'ec2-user' is not mapped to 'multica_recovery' in recovery_map.
	unmappedSockDSN := fmt.Sprintf("host=%s port=%d dbname=multica_transition user=multica_recovery", sockDir, port)
	ctx1, cancel1 := context.WithTimeout(context.Background(), 3*time.Second)
	_, err = pgx.Connect(ctx1, unmappedSockDSN)
	cancel1()

	if err == nil {
		t.Fatalf("FAIL: unmapped OS user %q successfully connected as multica_recovery via peer auth; expected rejection", currentOSUser)
	} else if !strings.Contains(err.Error(), "Peer authentication failed") {
		t.Fatalf("FAIL: expected 'Peer authentication failed' for unmapped OS user %q, got: %v", currentOSUser, err)
	}
	t.Logf("PASS: Confirmed OS peer confinement — unmapped OS user %q denied peer access to multica_recovery: %v", currentOSUser, err)

	// =========================================================================
	// ASSERTION 1b: Peer Authentication Confinement (Positive Proof)
	// =========================================================================
	// Add current OS user to pg_ident.conf recovery_map, reload, verify peer connection succeeds.
	pgIdentContentAllowed := fmt.Sprintf(`# ORQ-35 User Map
recovery_map    postgres                multica_recovery
recovery_map    %s                      multica_recovery
`, currentOSUser)
	if err := os.WriteFile(filepath.Join(dataDir, "pg_ident.conf"), []byte(pgIdentContentAllowed), 0600); err != nil {
		t.Fatalf("failed to update pg_ident.conf: %v", err)
	}
	if out, err := exec.Command("/usr/bin/pg_ctl", "reload", "-D", dataDir).CombinedOutput(); err != nil {
		t.Fatalf("pg_ctl reload failed: %v\nOutput: %s", err, string(out))
	}
	time.Sleep(300 * time.Millisecond)

	ctx1b, cancel1b := context.WithTimeout(context.Background(), 3*time.Second)
	mappedConn, err := pgx.Connect(ctx1b, unmappedSockDSN)
	cancel1b()
	if err != nil {
		t.Fatalf("FAIL: mapped OS user %q failed peer auth connection to multica_recovery: %v", currentOSUser, err)
	}
	var currentDBUser string
	if err := mappedConn.QueryRow(context.Background(), "SELECT current_user").Scan(&currentDBUser); err != nil || currentDBUser != "multica_recovery" {
		mappedConn.Close(context.Background())
		t.Fatalf("FAIL: expected current_user=multica_recovery, got %q (err: %v)", currentDBUser, err)
	}
	mappedConn.Close(context.Background())
	t.Logf("PASS: Confirmed OS peer confinement — mapped OS user %q successfully authenticated as %q via peer map", currentOSUser, currentDBUser)

	// Revert pg_ident.conf back to postgres-only mapping
	if err := os.WriteFile(filepath.Join(dataDir, "pg_ident.conf"), []byte(pgIdentContent), 0600); err != nil {
		t.Fatalf("failed to restore pg_ident.conf: %v", err)
	}
	_ = exec.Command("/usr/bin/pg_ctl", "reload", "-D", dataDir).Run()
	time.Sleep(300 * time.Millisecond)

	// =========================================================================
	// ASSERTION 2: TCP SCRAM Enforcement
	// =========================================================================
	// 2a. TCP without password -> FAILS
	noPwdDSN := fmt.Sprintf("host=127.0.0.1 port=%d dbname=multica_transition user=multica_transition", port)
	ctx2a, cancel2a := context.WithTimeout(context.Background(), 3*time.Second)
	_, err = pgx.Connect(ctx2a, noPwdDSN)
	cancel2a()
	if err == nil {
		t.Fatalf("FAIL: TCP connection without password succeeded; expected SCRAM password prompt rejection")
	}
	t.Logf("PASS: Confirmed TCP access without password rejected as expected: %v", err)

	// 2b. TCP with wrong password -> FAILS
	wrongPwdDSN := fmt.Sprintf("host=127.0.0.1 port=%d dbname=multica_transition user=multica_transition password=wrong_password", port)
	ctx2b, cancel2b := context.WithTimeout(context.Background(), 3*time.Second)
	_, err = pgx.Connect(ctx2b, wrongPwdDSN)
	cancel2b()
	if err == nil {
		t.Fatalf("FAIL: TCP connection with wrong password succeeded; expected rejection")
	} else if !strings.Contains(err.Error(), "password authentication failed") {
		t.Fatalf("FAIL: expected password authentication failed, got: %v", err)
	}
	t.Logf("PASS: Confirmed TCP access with wrong password rejected: %v", err)

	// 2c. TCP with correct password -> SUCCEEDS
	correctPwdDSN := fmt.Sprintf("host=127.0.0.1 port=%d dbname=multica_transition user=multica_transition password=%s", port, appSecret)
	ctx2c, cancel2c := context.WithTimeout(context.Background(), 3*time.Second)
	appConn, err := pgx.Connect(ctx2c, correctPwdDSN)
	cancel2c()
	if err != nil {
		t.Fatalf("FAIL: TCP connection with correct password failed: %v", err)
	}
	var testOne int
	if err := appConn.QueryRow(context.Background(), "SELECT 1").Scan(&testOne); err != nil || testOne != 1 {
		appConn.Close(context.Background())
		t.Fatalf("FAIL: query SELECT 1 failed on valid app DSN: %v", err)
	}
	appConn.Close(context.Background())
	t.Logf("PASS: Confirmed TCP application DSN connection with correct SCRAM secret succeeded (SELECT 1 = %d)", testOne)

	// =========================================================================
	// ASSERTION 2d: Execute Least Privilege Cutover & Go Test Suite Assertions
	// =========================================================================
	cutoverSQLPath := "../../../../scripts/ops/least_privilege_cutover.sql"
	cutoverSQL, err := os.ReadFile(cutoverSQLPath)
	if err != nil {
		t.Fatalf("failed to read cutover SQL file: %v", err)
	}

	// Connect as multica_transition (currently superuser) to run cutover SQL
	superuserDSN := fmt.Sprintf("host=127.0.0.1 port=%d dbname=multica_transition user=multica_transition password=%s", port, appSecret)
	adminConn, err := pgx.Connect(context.Background(), superuserDSN)
	if err != nil {
		t.Fatalf("failed to connect as superuser for cutover execution: %v", err)
	}
	if _, err := adminConn.Exec(context.Background(), string(cutoverSQL)); err != nil {
		adminConn.Close(context.Background())
		t.Fatalf("failed to execute least_privilege_cutover.sql: %v", err)
	}
	adminConn.Close(context.Background())
	t.Logf("PASS: Successfully executed least_privilege_cutover.sql on ephemeral PostgreSQL 17")

	// Set TEST_DATABASE_URL environment variable for TestLeastPrivilege test suite
	t.Setenv("TEST_DATABASE_URL", correctPwdDSN)

	// Run role security flag assertions
	t.Run("SubTest_LeastPrivilege_RolePermissions", TestLeastPrivilege_RolePermissions)

	// Run DML and DDL boundary assertions
	t.Run("SubTest_LeastPrivilege_DMLandDDLBoundary", TestLeastPrivilege_DMLandDDLBoundary)

	// =========================================================================
	// ASSERTION 3: Reversible Rollback via HBA Reload
	// =========================================================================
	rollbackHBAContent := `# Rollback Trust Posture
local   all             all                                       trust
host    all             all               127.0.0.1/32            trust
`
	if err := os.WriteFile(filepath.Join(dataDir, "pg_hba.conf"), []byte(rollbackHBAContent), 0600); err != nil {
		t.Fatalf("failed to write rollback pg_hba.conf: %v", err)
	}

	if out, err := exec.Command("/usr/bin/pg_ctl", "reload", "-D", dataDir).CombinedOutput(); err != nil {
		t.Fatalf("pg_ctl reload failed during rollback test: %v\nOutput: %s", err, string(out))
	}
	time.Sleep(300 * time.Millisecond)

	// Under rollback trust posture, noPwdDSN should now succeed
	ctx3, cancel3 := context.WithTimeout(context.Background(), 3*time.Second)
	rbConn, err := pgx.Connect(ctx3, noPwdDSN)
	cancel3()
	if err != nil {
		t.Fatalf("FAIL: rollback trust connection failed: %v", err)
	}
	rbConn.Close(context.Background())
	t.Logf("PASS: Verified rollback HBA reload — trust posture restored successfully")

	// Restore hardened HBA posture and verify re-application
	if err := os.WriteFile(filepath.Join(dataDir, "pg_hba.conf"), []byte(pgHBAContent), 0600); err != nil {
		t.Fatalf("failed to restore hardened pg_hba.conf: %v", err)
	}
	_ = exec.Command("/usr/bin/pg_ctl", "reload", "-D", dataDir).Run()
	time.Sleep(300 * time.Millisecond)

	ctx3b, cancel3b := context.WithTimeout(context.Background(), 3*time.Second)
	_, err = pgx.Connect(ctx3b, noPwdDSN)
	cancel3b()
	if err == nil {
		t.Fatalf("FAIL: re-applied hardened HBA failed to enforce password requirement")
	}
	t.Logf("PASS: Verified atomic reversibility — re-applied hardened HBA re-enforced SCRAM authentication")
}
