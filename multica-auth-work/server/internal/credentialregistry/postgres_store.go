package credentialregistry

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TxBeginner is implemented by *pgxpool.Pool. Keeping the boundary narrow
// permits deterministic transaction failure tests without a live database.
type TxBeginner interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// PostgresStore is the durable production implementation of Store. It relies
// on the frozen migration-132/133/134 relations; construction does not probe or
// mutate the database, so K1 may wire it only after those migrations integrate.
type PostgresStore struct {
	db TxBeginner
}

var (
	_ TxBeginner = (*pgxpool.Pool)(nil)
	_ Store      = (*PostgresStore)(nil)
)

func NewPostgresStore(db TxBeginner) *PostgresStore {
	return &PostgresStore{db: db}
}

// NewPostgresResolver is the production composition point owned by this
// package. Calling it from the shared task-claim and terminal-task paths is a
// K1 integration dependency; this package deliberately does not edit them.
func NewPostgresResolver(pool *pgxpool.Pool) *Resolver {
	return NewResolver(NewPostgresStore(pool))
}

const lockCandidateSQL = `
SELECT
    t.id::text,
    t.status,
    b.id::text,
    b.agent_id::text,
    b.workspace_id::text,
    b.runtime_id::text,
    b.runtime_session_id::text,
    b.runtime_standard_version_id::text,
    b.active_configuration_version_id::text,
    b.effective_configuration_digest,
    b.provider_capability_digest,
    b.provider,
    a.home_ref,
    b.binding_generation,
    a.catalog_generation,
    c.catalog_generation,
    a.approval_state,
    a.status,
    a.worktype_scope,
    (
        SELECT count(*)::integer
        FROM runtime_binding_home_assignments owners
        WHERE owners.home_ref = a.home_ref
          AND owners.released_at IS NULL
    ),
    (
        SELECT count(*)::integer
        FROM runtime_binding_home_assignments binding_assignments
        WHERE binding_assignments.runtime_binding_id = b.id
          AND binding_assignments.released_at IS NULL
    ),
    b.task_concurrency_limit
FROM agent_task_queue t
JOIN runtime_bindings b
  ON b.id = $2::uuid
 AND b.agent_id = t.agent_id
 AND b.runtime_id = t.runtime_id
JOIN runtime_binding_home_assignments a
  ON a.runtime_binding_id = b.id
 AND a.released_at IS NULL
JOIN credential_home_catalog c
  ON c.home_ref = a.home_ref
WHERE t.id = $1::uuid
  AND b.agent_id = $3::uuid
  AND b.workspace_id = $4::uuid
  AND b.transport_binding = 'native_credential_home'
  AND c.lifecycle_state = 'healthy'
FOR UPDATE OF t, b, a, c`

const countActiveReservationsSQL = `
SELECT count(*)::integer
FROM runtime_binding_task_reservations
WHERE runtime_binding_id = $1::uuid
  AND released_at IS NULL`

const insertReservationSQL = `
INSERT INTO runtime_binding_task_reservations (
    task_id,
    runtime_binding_id,
    binding_generation,
    home_ref,
    catalog_generation
) VALUES ($1::uuid, $2::uuid, $3, $4, $5)`

const claimTaskSQL = `
UPDATE agent_task_queue
SET status = 'dispatched', dispatched_at = now()
WHERE id = $1::uuid
  AND status = 'queued'`

const insertTaskSnapshotSQL = `
INSERT INTO runtime_task_snapshots (
    task_id,
    runtime_session_id,
    runtime_id,
    agent_id,
    workspace_id,
    runtime_standard_version_id,
    runtime_configuration_version_id,
    effective_configuration_digest,
    runtime_binding_id,
    binding_generation,
    home_ref,
    catalog_generation,
    provider_capability_digest
) VALUES (
    $1::uuid, $2::uuid, $3::uuid, $4::uuid, $5::uuid, $6::uuid, $7::uuid,
    $8, $9::uuid, $10, $11, $12, $13
)`

const lockReleaseSQL = `
SELECT released_at IS NOT NULL
FROM runtime_binding_task_reservations
WHERE task_id = $1::uuid
  AND runtime_binding_id = $2::uuid
  AND binding_generation = $3
  AND catalog_generation = $4
FOR UPDATE`

const releaseReservationSQL = `
UPDATE runtime_binding_task_reservations
SET released_at = now()
WHERE task_id = $1::uuid
  AND runtime_binding_id = $2::uuid
  AND binding_generation = $3
  AND catalog_generation = $4
  AND released_at IS NULL`

func (s *PostgresStore) ReserveAssignment(
	ctx context.Context,
	request Request,
	validate CandidateValidator,
) (Candidate, error) {
	if s == nil || s.db == nil || validate == nil {
		return Candidate{}, ErrRegistryUnavailable
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return Candidate{}, ErrRegistryUnavailable
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	candidate, err := loadLockedCandidate(ctx, tx, request)
	if err != nil {
		return Candidate{}, mapPostgresError(err)
	}
	if err := tx.QueryRow(ctx, countActiveReservationsSQL, request.BindingID).Scan(&candidate.ActiveTasks); err != nil {
		return Candidate{}, mapPostgresError(err)
	}

	// This callback is the only validation authority. It runs while all source
	// rows remain locked and strictly before the task or snapshot is mutated.
	if err := validate(candidate); err != nil {
		return Candidate{}, err
	}

	if _, err := tx.Exec(ctx, insertReservationSQL,
		candidate.TaskID,
		candidate.BindingID,
		candidate.BindingGeneration,
		candidate.HomeRef,
		candidate.CatalogGeneration,
	); err != nil {
		return Candidate{}, mapPostgresError(err)
	}
	tag, err := tx.Exec(ctx, claimTaskSQL, request.TaskID)
	if err != nil {
		return Candidate{}, mapPostgresError(err)
	}
	if tag.RowsAffected() != 1 {
		return Candidate{}, ErrTaskConflict
	}
	if _, err := tx.Exec(ctx, insertTaskSnapshotSQL,
		candidate.TaskID,
		candidate.RuntimeSessionID,
		candidate.RuntimeID,
		candidate.AgentID,
		candidate.WorkspaceID,
		candidate.StandardVersionID,
		candidate.ConfigurationVersionID,
		candidate.ConfigurationDigest,
		candidate.BindingID,
		candidate.BindingGeneration,
		candidate.HomeRef,
		candidate.CatalogGeneration,
		candidate.CapabilityDigest,
	); err != nil {
		return Candidate{}, mapPostgresError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Candidate{}, mapPostgresError(err)
	}
	committed = true
	candidate.ActiveTasks++
	return candidate, nil
}

func loadLockedCandidate(ctx context.Context, tx pgx.Tx, request Request) (Candidate, error) {
	var candidate Candidate
	err := tx.QueryRow(ctx, lockCandidateSQL,
		request.TaskID,
		request.BindingID,
		request.AgentID,
		request.WorkspaceID,
	).Scan(
		&candidate.TaskID,
		&candidate.TaskStatus,
		&candidate.BindingID,
		&candidate.AgentID,
		&candidate.WorkspaceID,
		&candidate.RuntimeID,
		&candidate.RuntimeSessionID,
		&candidate.StandardVersionID,
		&candidate.ConfigurationVersionID,
		&candidate.ConfigurationDigest,
		&candidate.CapabilityDigest,
		&candidate.Provider,
		&candidate.HomeRef,
		&candidate.BindingGeneration,
		&candidate.AssignmentCatalogGeneration,
		&candidate.CatalogGeneration,
		&candidate.Approval,
		&candidate.Status,
		&candidate.WorktypeScope,
		&candidate.AssignmentOwners,
		&candidate.BindingAssignments,
		&candidate.TaskConcurrencyLimit,
	)
	return candidate, err
}

func (s *PostgresStore) ReleaseAssignment(ctx context.Context, request ReleaseRequest) error {
	if s == nil || s.db == nil {
		return ErrRegistryUnavailable
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return ErrRegistryUnavailable
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	var alreadyReleased bool
	if err := tx.QueryRow(ctx, lockReleaseSQL,
		request.TaskID,
		request.BindingID,
		request.BindingGeneration,
		request.CatalogGeneration,
	).Scan(&alreadyReleased); err != nil {
		return mapPostgresError(err)
	}
	if !alreadyReleased {
		tag, err := tx.Exec(ctx, releaseReservationSQL,
			request.TaskID,
			request.BindingID,
			request.BindingGeneration,
			request.CatalogGeneration,
		)
		if err != nil {
			return mapPostgresError(err)
		}
		if tag.RowsAffected() != 1 {
			return ErrTaskConflict
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return mapPostgresError(err)
	}
	committed = true
	return nil
}

func mapPostgresError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNoApprovedAssignment
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505", "40001", "40P01":
			return ErrTaskConflict
		}
	}
	return ErrRegistryUnavailable
}
