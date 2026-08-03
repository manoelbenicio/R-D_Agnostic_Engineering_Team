package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

func TestMigration138ImmutableAttestationUpdateDeleteRejected55006(t *testing.T) {
	f, ctx := setupMigration138(t)
	seedMigration138CurrentNativeEpoch(t, ctx, f)
	insertMigration138Request(t, ctx, f, migration138ProbeID)
	acceptMigration138(t, ctx, f, migration138ProbeID, "ready", "ready", time.Now())

	for _, statement := range []string{
		`UPDATE runtime_credential_readiness_attestation
		 SET expires_at = expires_at + interval '1 second'`,
		`DELETE FROM runtime_credential_readiness_attestation`,
		`UPDATE runtime_credential_readiness_probe_request
		 SET request_digest = repeat('d', 64)`,
		`DELETE FROM runtime_credential_readiness_probe_result`,
	} {
		expectMigration138SQLState(t, ctx, f, statement, "55006")
	}
}

func TestMigration138ExactReplayReturnsOriginalWithoutExpiryExtension(t *testing.T) {
	f, ctx := setupMigration138(t)
	seedMigration138CurrentNativeEpoch(t, ctx, f)
	insertMigration138Request(t, ctx, f, migration138ProbeID)
	observed := time.Now().UTC().Truncate(time.Microsecond)
	firstID, firstAccepted, firstExpires := acceptMigration138(
		t, ctx, f, migration138ProbeID, "ready", "ready", observed,
	)
	time.Sleep(10 * time.Millisecond)
	secondID, secondAccepted, secondExpires := acceptMigration138(
		t, ctx, f, migration138ProbeID, "ready", "ready", observed,
	)
	if firstID != secondID || !firstAccepted.Equal(secondAccepted) ||
		!firstExpires.Equal(secondExpires) {
		t.Fatalf("exact replay changed immutable result: first=%s/%s/%s second=%s/%s/%s",
			firstID, firstAccepted, firstExpires, secondID, secondAccepted, secondExpires)
	}
	current, err := getCurrentMigration138(t, ctx, f)
	if err != nil || current.State != "ready" {
		t.Fatalf("generated current-read state=%q err=%v, want ready", current.State, err)
	}
}

func TestMigration138ChangedReplayRejected(t *testing.T) {
	f, ctx := setupMigration138(t)
	seedMigration138CurrentNativeEpoch(t, ctx, f)
	insertMigration138Request(t, ctx, f, migration138ProbeID)
	observed := time.Now().UTC().Truncate(time.Microsecond)
	acceptMigration138(t, ctx, f, migration138ProbeID, "ready", "ready", observed)

	tx := beginMigration138Tx(t, ctx, f)
	defer tx.Rollback(ctx)
	_, err := db.New(tx).AcceptOrReplayCredentialReadinessResult(
		ctx, migration138AcceptParams(
			migration138ProbeID, "unready", "probe_timeout", observed,
		),
	)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("changed replay err=%v, want no rows", err)
	}
}

func TestMigration138ObservationAndServerExpiryBoundaries(t *testing.T) {
	f, ctx := setupMigration138(t)
	seedMigration138CurrentNativeEpoch(t, ctx, f)

	request := insertMigration138Request(t, ctx, f, migration138ProbeID)
	issued := request.IssuedAt.Time
	expectMigration138AcceptRejected(t, ctx, f, issued.Add(-time.Microsecond))
	expectMigration138AcceptRejected(t, ctx, f, time.Now().Add(time.Minute))

	_, accepted, expires := acceptMigration138(
		t, ctx, f, migration138ProbeID, "unready", "probe_timeout", issued,
	)
	if !expires.Equal(accepted.Add(5 * time.Minute)) {
		t.Fatalf("server expiry=%s, want accepted+300s=%s",
			expires, accepted.Add(5*time.Minute))
	}
	if _, err := getCurrentMigration138(t, ctx, f); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("fresh exact unready current-read err=%v, want no rows", err)
	}
}

func TestMigration138CurrentGenerationAndDaemonBootRaceFailsClosed(t *testing.T) {
	f, ctx := setupMigration138(t)
	seedMigration138CurrentNativeEpoch(t, ctx, f)

	for name, mutate := range map[string]func(*db.CreateCredentialReadinessProbeRequestParams){
		"daemon boot": func(arg *db.CreateCredentialReadinessProbeRequestParams) {
			arg.DaemonBootID = mustMigration138UUID(
				t, "00000000-0000-0000-0000-000000000099")
		},
		"binding generation": func(arg *db.CreateCredentialReadinessProbeRequestParams) {
			arg.BindingGeneration = 2
		},
		"catalog generation": func(arg *db.CreateCredentialReadinessProbeRequestParams) {
			arg.CatalogGeneration = 2
		},
	} {
		t.Run(name, func(t *testing.T) {
			arg := migration138CreateParams(t, migration138ProbeID)
			mutate(&arg)
			tx := beginMigration138Tx(t, ctx, f)
			defer tx.Rollback(ctx)
			_, err := db.New(tx).CreateCredentialReadinessProbeRequest(ctx, arg)
			if !errors.Is(err, pgx.ErrNoRows) {
				t.Fatalf("stale authority create err=%v, want no rows", err)
			}
		})
	}

	insertMigration138Request(t, ctx, f, migration138ProbeID)
	execMigration138(t, ctx, f, `
		UPDATE runtime_binding SET state = 'draining'
		WHERE id = '00000000-0000-0000-0000-000000000013'`)
	expectMigration138AcceptRejected(t, ctx, f, time.Now())
}

func TestMigration138RollbackRestoresMigration137State(t *testing.T) {
	f, ctx := setupMigration138(t)
	down := f.opts()
	down.Direction = "down"
	down.Files = scopedRealMigrations(t, f.schema,
		"138_runtime_credential_readiness_probes.down.sql")
	if err := runMigrations(ctx, f.pool, down); err != nil {
		t.Fatalf("empty rollback migration 138: %v", err)
	}
	for _, relation := range []string{
		"runtime_credential_readiness_probe_request",
		"runtime_credential_readiness_probe_result",
	} {
		var exists bool
		if err := f.pool.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`,
			f.schema+"."+relation).Scan(&exists); err != nil || exists {
			t.Fatalf("post-down relation %s exists=%v err=%v", relation, exists, err)
		}
	}
	seedMigration136Authority(t, ctx, f)
	seedMigration137RetirementState(t, ctx, f)
}

func TestMigration138SchemaRejectsPathSecretTokenColumns(t *testing.T) {
	f, ctx := setupMigration138(t)
	rows, err := f.pool.Query(ctx, `
		SELECT table_name, column_name
		FROM information_schema.columns
		WHERE table_schema = $1
		  AND table_name IN (
		    'runtime_credential_readiness_probe_request',
		    'runtime_credential_readiness_probe_result'
		  )`, f.schema)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var table, column string
		if err := rows.Scan(&table, &column); err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{
			"path", "account", "credential", "token", "secret", "slot", "body",
		} {
			if strings.Contains(column, forbidden) {
				t.Fatalf("%s.%s contains forbidden material marker %q",
					table, column, forbidden)
			}
		}
	}
}

const (
	migration138TaskID  = "00000000-0000-0000-0000-000000000008"
	migration138ProbeID = "00000000-0000-0000-0000-000000000090"
)

func setupMigration138(t *testing.T) (*fixture, context.Context) {
	t.Helper()
	f := newFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	createMigration136Prerequisites(t, ctx, f)
	opts := f.opts()
	opts.Files = scopedRealMigrations(t, f.schema,
		"132_credential_home_catalog.up.sql",
		"133_runtime_bindings.up.sql",
		"134_runtime_configuration_snapshots.up.sql",
		"136_runtime_home_lifetime_readiness.up.sql",
		"137_native_rotation_epoch_recovery.up.sql",
		"138_runtime_credential_readiness_probes.up.sql",
	)
	if err := runMigrations(ctx, f.pool, opts); err != nil {
		t.Fatalf("apply migrations 132-138: %v", err)
	}
	return f, ctx
}

func seedMigration138CurrentNativeEpoch(
	t *testing.T, ctx context.Context, f *fixture,
) {
	t.Helper()
	seedMigration136Authority(t, ctx, f)
	execMigration136Tx(t, ctx, f, func(tx pgx.Tx) error {
		return execMigration137Batch(ctx, tx, `
			INSERT INTO runtime_task_home_epoch (
				task_id, home_epoch, workspace_id, agent_id, runtime_id,
				runtime_session_id, daemon_id, daemon_boot_id, provider,
				transport_binding, runtime_binding_id, binding_generation,
				home_assignment_id, catalog_id, catalog_entry_id,
				catalog_generation, home_ref, lifetime_id, acquisition_request_id
			) VALUES (
				'00000000-0000-0000-0000-000000000008', 1,
				'00000000-0000-0000-0000-000000000001',
				'00000000-0000-0000-0000-000000000006',
				'00000000-0000-0000-0000-000000000005',
				'00000000-0000-0000-0000-000000000004',
				'daemon-a', '00000000-0000-0000-0000-000000000018',
				'codex', 'native_credential_home',
				'00000000-0000-0000-0000-000000000013', 1,
				'00000000-0000-0000-0000-000000000014',
				'00000000-0000-0000-0000-000000000009',
				'00000000-0000-0000-0000-000000000011', 1,
				'00000000-0000-0000-0000-000000000012',
				'00000000-0000-0000-0000-000000000054',
				'00000000-0000-0000-0000-000000000055'
			);
			INSERT INTO runtime_home_lifetime (
				id, acquisition_request_id, task_id, home_epoch, agent_id,
				runtime_id, runtime_session_id, runtime_binding_id,
				binding_generation, home_assignment_id, workspace_id, daemon_id,
				catalog_id, catalog_generation, home_ref, daemon_boot_id,
				provider, transport_binding, state
			) VALUES (
				'00000000-0000-0000-0000-000000000054',
				'00000000-0000-0000-0000-000000000055',
				'00000000-0000-0000-0000-000000000008', 1,
				'00000000-0000-0000-0000-000000000006',
				'00000000-0000-0000-0000-000000000005',
				'00000000-0000-0000-0000-000000000004',
				'00000000-0000-0000-0000-000000000013', 1,
				'00000000-0000-0000-0000-000000000014',
				'00000000-0000-0000-0000-000000000001', 'daemon-a',
				'00000000-0000-0000-0000-000000000009', 1,
				'00000000-0000-0000-0000-000000000012',
				'00000000-0000-0000-0000-000000000018',
				'codex', 'native_credential_home', 'pending_local'
			);
			INSERT INTO runtime_home_lifetime_event VALUES (
				'00000000-0000-0000-0000-000000000054', 1, 'pending_local',
				'00000000-0000-0000-0000-000000000077', 'reserved', now()
			);
			UPDATE runtime_home_lifetime
			SET state = 'acquired', state_version = 2
			WHERE id = '00000000-0000-0000-0000-000000000054';
			INSERT INTO runtime_home_lifetime_event VALUES (
				'00000000-0000-0000-0000-000000000054', 2, 'acquired',
				'00000000-0000-0000-0000-000000000078', 'acquired', now()
			);
			UPDATE runtime_home_lifetime
			SET state = 'process_started', state_version = 3,
				process_started_at = now(), process_identity_digest = repeat('a', 64)
			WHERE id = '00000000-0000-0000-0000-000000000054';
			INSERT INTO runtime_home_lifetime_event VALUES (
				'00000000-0000-0000-0000-000000000054', 3, 'process_started',
				'00000000-0000-0000-0000-000000000079', 'started', now()
			)
		`)
	})
}

func insertMigration138Request(
	t *testing.T, ctx context.Context, f *fixture, probeID string,
) db.RuntimeCredentialReadinessProbeRequest {
	t.Helper()
	tx := beginMigration138Tx(t, ctx, f)
	defer tx.Rollback(ctx)
	request, err := db.New(tx).CreateCredentialReadinessProbeRequest(
		ctx, migration138CreateParams(t, probeID),
	)
	if err != nil {
		t.Fatalf("generated create probe request: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit generated probe request: %v", err)
	}
	return request
}

func acceptMigration138(
	t *testing.T,
	ctx context.Context,
	f *fixture,
	probeID, state, reason string,
	observed time.Time,
) (string, time.Time, time.Time) {
	t.Helper()
	tx := beginMigration138Tx(t, ctx, f)
	defer tx.Rollback(ctx)
	result, err := db.New(tx).AcceptOrReplayCredentialReadinessResult(
		ctx, migration138AcceptParams(probeID, state, reason, observed),
	)
	if err != nil {
		t.Fatalf("accept/replay probe result: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit probe result: %v", err)
	}
	return migration138UUIDString(result.ID), result.AcceptedAt.Time, result.ExpiresAt.Time
}

func expectMigration138AcceptRejected(
	t *testing.T, ctx context.Context, f *fixture, observed time.Time,
) {
	t.Helper()
	tx := beginMigration138Tx(t, ctx, f)
	defer tx.Rollback(ctx)
	_, err := db.New(tx).AcceptOrReplayCredentialReadinessResult(
		ctx, migration138AcceptParams(
			migration138ProbeID, "ready", "ready", observed,
		),
	)
	if err == nil {
		t.Fatal("invalid probe acceptance succeeded")
	}
}

func beginMigration138Tx(
	t *testing.T, ctx context.Context, f *fixture,
) pgx.Tx {
	t.Helper()
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %s",
		pgx.Identifier{f.schema}.Sanitize())); err != nil {
		tx.Rollback(ctx)
		t.Fatal(err)
	}
	return tx
}

func execMigration138(
	t *testing.T, ctx context.Context, f *fixture, statement string,
) {
	t.Helper()
	tx := beginMigration138Tx(t, ctx, f)
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, statement); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func getCurrentMigration138(
	t *testing.T, ctx context.Context, f *fixture,
) (db.RuntimeCredentialReadinessAttestation, error) {
	t.Helper()
	tx := beginMigration138Tx(t, ctx, f)
	defer tx.Rollback(ctx)
	return db.New(tx).GetCurrentCredentialReadinessAttestation(
		ctx, db.GetCurrentCredentialReadinessAttestationParams{
			TaskID:            mustMigration138UUID(t, migration138TaskID),
			HomeEpoch:         1,
			RuntimeSessionID:  mustMigration138UUID(t, "00000000-0000-0000-0000-000000000004"),
			DaemonBootID:      mustMigration138UUID(t, "00000000-0000-0000-0000-000000000018"),
			RuntimeBindingID:  mustMigration138UUID(t, "00000000-0000-0000-0000-000000000013"),
			BindingGeneration: 1,
			HomeAssignmentID:  mustMigration138UUID(t, "00000000-0000-0000-0000-000000000014"),
			CatalogGeneration: 1,
		},
	)
}

func migration138CreateParams(
	t *testing.T, probeID string,
) db.CreateCredentialReadinessProbeRequestParams {
	t.Helper()
	return db.CreateCredentialReadinessProbeRequestParams{
		ProbeRequestID:    mustMigration138UUID(t, probeID),
		RequestDigest:     strings.Repeat("a", 64),
		TaskID:            mustMigration138UUID(t, migration138TaskID),
		HomeEpoch:         1,
		DaemonBootID:      mustMigration138UUID(t, "00000000-0000-0000-0000-000000000018"),
		RuntimeSessionID:  mustMigration138UUID(t, "00000000-0000-0000-0000-000000000004"),
		RuntimeBindingID:  mustMigration138UUID(t, "00000000-0000-0000-0000-000000000013"),
		BindingGeneration: 1,
		HomeAssignmentID:  mustMigration138UUID(t, "00000000-0000-0000-0000-000000000014"),
		CatalogGeneration: 1,
	}
}

func migration138AcceptParams(
	probeID, state, reason string, observed time.Time,
) db.AcceptOrReplayCredentialReadinessResultParams {
	return db.AcceptOrReplayCredentialReadinessResultParams{
		ProbeRequestID:  mustMigration138UUIDValue(probeID),
		State:           state,
		ReasonCode:      reason,
		ProbeObservedAt: pgtype.Timestamptz{Time: observed, Valid: true},
		RequestDigest:   strings.Repeat("a", 64),
		ResultDigest:    strings.Repeat("b", 64),
	}
}

func mustMigration138UUID(t *testing.T, value string) pgtype.UUID {
	t.Helper()
	result, err := migration138UUID(value)
	if err != nil {
		t.Fatalf("parse test UUID %q: %v", value, err)
	}
	return result
}

func mustMigration138UUIDValue(value string) pgtype.UUID {
	result, err := migration138UUID(value)
	if err != nil {
		panic(err)
	}
	return result
}

func migration138UUID(value string) (pgtype.UUID, error) {
	decoded, err := hex.DecodeString(strings.ReplaceAll(value, "-", ""))
	if err != nil || len(decoded) != 16 {
		return pgtype.UUID{}, fmt.Errorf("invalid UUID")
	}
	var bytes [16]byte
	copy(bytes[:], decoded)
	return pgtype.UUID{Bytes: bytes, Valid: true}, nil
}

func migration138UUIDString(value pgtype.UUID) string {
	raw := hex.EncodeToString(value.Bytes[:])
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		raw[0:8], raw[8:12], raw[12:16], raw[16:20], raw[20:32])
}

func expectMigration138SQLState(
	t *testing.T,
	ctx context.Context,
	f *fixture,
	statement, code string,
) {
	t.Helper()
	tx := beginMigration138Tx(t, ctx, f)
	defer tx.Rollback(ctx)
	_, err := tx.Exec(ctx, statement)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != code {
		t.Fatalf("statement error=%v, want SQLSTATE %s", err, code)
	}
}
