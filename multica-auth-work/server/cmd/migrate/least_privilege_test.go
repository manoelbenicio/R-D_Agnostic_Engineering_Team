package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ORQ-60 — Least Privilege Security Audit & Role Boundary Test Suite.
//
// Mandatory Security Gate Rules:
// 1. Never print connection strings, passwords, or raw DATABASE_URL in logs or test output (content-free).
// 2. No plaintext fallback defaults. Requires explicit TEST_DATABASE_URL or DATABASE_URL environment variable.
// 3. Fail-closed: Never skip security gate assertions when invoked in test environments.
// 4. Assert exact demotion: rolsuper=false, rolcreaterole=false, rolcreatedb=false, rolreplication=false, rolbypassrls=false.
// 5. Prove real DML (INSERT, SELECT, UPDATE, DELETE) on dedicated app-owned fixture table AND assert DDL denial (CREATE ROLE, schema CREATE).
// 6. Sanitize all dynamic identifiers using pgx.Identifier.Sanitize().

func connectTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}

	if dbURL == "" {
		t.Fatalf("FAIL: TEST_DATABASE_URL or DATABASE_URL environment variable must be explicitly provided for security gate execution")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("FAIL: unable to connect to test database (credentials scrubbed)")
		return nil
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("FAIL: unable to ping test database connection (credentials scrubbed)")
		return nil
	}

	return pool
}

func TestLeastPrivilege_RolePermissions(t *testing.T) {
	pool := connectTestDB(t)
	defer pool.Close()

	ctx := context.Background()

	// Query role security flags for application roles
	rows, err := pool.Query(ctx, `
		SELECT rolname, rolsuper, rolcreaterole, rolcreatedb, rolreplication, rolbypassrls
		FROM pg_roles
		WHERE rolname IN ('multica_transition', 'multica_app')
	`)
	if err != nil {
		t.Fatalf("FAIL: failed to query pg_roles: %v", err)
	}
	defer rows.Close()

	roleCount := 0
	for rows.Next() {
		roleCount++
		var roleName string
		var super, createRole, createDB, replication, bypassRLS bool

		if err := rows.Scan(&roleName, &super, &createRole, &createDB, &replication, &bypassRLS); err != nil {
			t.Fatalf("FAIL: failed to scan role row: %v", err)
		}

		if super {
			t.Fatalf("FAIL: application role %s has SUPERUSER=true; must be demoted", roleName)
		}
		if createRole {
			t.Fatalf("FAIL: application role %s has CREATEROLE=true", roleName)
		}
		if createDB {
			t.Fatalf("FAIL: application role %s has CREATEDB=true", roleName)
		}
		if replication {
			t.Fatalf("FAIL: application role %s has REPLICATION=true", roleName)
		}
		if bypassRLS {
			t.Fatalf("FAIL: application role %s has BYPASSRLS=true", roleName)
		}

		t.Logf("PASS: Role %s security attributes verified (SUPERUSER=false, CREATEROLE=false, CREATEDB=false, REPLICATION=false, BYPASSRLS=false)", roleName)
	}

	if roleCount == 0 {
		t.Fatalf("FAIL: zero target application roles ('multica_transition', 'multica_app') found in pg_roles")
	}
}

func TestLeastPrivilege_DMLandDDLBoundary(t *testing.T) {
	pool := connectTestDB(t)
	defer pool.Close()

	ctx := context.Background()

	// 1. Assert current connection user identity and demoted status
	var currentUser string
	var currentSuper bool
	if err := pool.QueryRow(ctx, "SELECT current_user, rolsuper FROM pg_roles WHERE rolname = current_user").Scan(&currentUser, &currentSuper); err != nil {
		t.Fatalf("FAIL: unable to query current session user identity: %v", err)
	}

	if currentUser != "multica_transition" && currentUser != "multica_app" {
		t.Fatalf("FAIL: test connection must execute as demoted application role ('multica_transition' or 'multica_app'), got current_user=%q", currentUser)
	}

	if currentSuper {
		t.Fatalf("FAIL: connected session user %q has SUPERUSER=true; must be demoted", currentUser)
	}

	t.Logf("PASS: Verified session connection user identity %q (SUPERUSER=false)", currentUser)

	// 2. Require dedicated pre-created app-owned fixture table "orq60_app_fixture" for DML proof (no arbitrary table fallbacks)
	fixtureTable := "orq60_app_fixture"
	var exists bool
	if err := pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_tables WHERE schemaname='public' AND tablename=$1)", fixtureTable).Scan(&exists); err != nil {
		t.Fatalf("FAIL: unable to query pg_tables for fixture table %s: %v", fixtureTable, err)
	}
	if !exists {
		t.Fatalf("FAIL: dedicated pre-created fixture table %q must exist in public schema prior to security gate execution", fixtureTable)
	}

	sanitizedFixture := pgx.Identifier{fixtureTable}.Sanitize()

	// 3. Execute and assert ALL FOUR DML operations (INSERT, SELECT, UPDATE, DELETE)
	testID := int(time.Now().UnixNano() % 2147483647)
	testVal := fmt.Sprintf("val_%d", testID)

	// DML 1: INSERT
	if _, err := pool.Exec(ctx, fmt.Sprintf("INSERT INTO %s (id, content) VALUES ($1, $2)", sanitizedFixture), testID, testVal); err != nil {
		t.Fatalf("FAIL: DML INSERT failed on fixture table %s under demoted role %s: %v", sanitizedFixture, currentUser, err)
	}

	// DML 2: SELECT
	var readVal string
	if err := pool.QueryRow(ctx, fmt.Sprintf("SELECT content FROM %s WHERE id = $1", sanitizedFixture), testID).Scan(&readVal); err != nil || readVal != testVal {
		t.Fatalf("FAIL: DML SELECT failed on fixture table %s under demoted role %s: %v", sanitizedFixture, currentUser, err)
	}

	// DML 3: UPDATE
	updatedVal := fmt.Sprintf("updated_%d", testID)
	if _, err := pool.Exec(ctx, fmt.Sprintf("UPDATE %s SET content = $1 WHERE id = $2", sanitizedFixture), updatedVal, testID); err != nil {
		t.Fatalf("FAIL: DML UPDATE failed on fixture table %s under demoted role %s: %v", sanitizedFixture, currentUser, err)
	}

	// DML 4: DELETE
	if _, err := pool.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE id = $1", sanitizedFixture), testID); err != nil {
		t.Fatalf("FAIL: DML DELETE failed on fixture table %s under demoted role %s: %v", sanitizedFixture, currentUser, err)
	}

	t.Logf("PASS: All four DML operations (INSERT, SELECT, UPDATE, DELETE) verified successfully on fixture table %s under demoted role %q", sanitizedFixture, currentUser)

	// 4. Assert DDL Rejections Fail-Closed
	// DDL Test A: CREATE ROLE
	unauthorizedRole := fmt.Sprintf("orq60_unauthorized_role_%d", testID)
	sanitizedRole := pgx.Identifier{unauthorizedRole}.Sanitize()
	if _, err := pool.Exec(ctx, fmt.Sprintf("CREATE ROLE %s", sanitizedRole)); err == nil {
		_, _ = pool.Exec(ctx, fmt.Sprintf("DROP ROLE IF EXISTS %s", sanitizedRole))
		t.Fatalf("FAIL: demoted application role %q was able to execute administrative DDL CREATE ROLE", currentUser)
	} else if !strings.Contains(err.Error(), "permission denied") && !strings.Contains(err.Error(), "must be superuser") {
		t.Fatalf("FAIL: unexpected error for CREATE ROLE denial: %v", err)
	}

	// DDL Test B: Schema CREATE (CREATE TABLE)
	unauthorizedTable := fmt.Sprintf("orq60_unauthorized_table_%d", testID)
	sanitizedTable := pgx.Identifier{unauthorizedTable}.Sanitize()
	if _, err := pool.Exec(ctx, fmt.Sprintf("CREATE TABLE %s (id INT)", sanitizedTable)); err == nil {
		_, _ = pool.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", sanitizedTable))
		t.Fatalf("FAIL: demoted application role %q was able to execute schema DDL CREATE TABLE", currentUser)
	} else if !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("FAIL: unexpected error for CREATE TABLE schema denial: %v", err)
	}

	t.Logf("PASS: Administrative DDL (CREATE ROLE) and Schema DDL (CREATE TABLE) rejections verified fail-closed for role %q", currentUser)
}
