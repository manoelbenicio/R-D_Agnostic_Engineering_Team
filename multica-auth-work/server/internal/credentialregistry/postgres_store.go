package credentialregistry

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// TxBeginner is implemented by *pgxpool.Pool. Keeping this boundary narrow
// lets the package verify transaction failures without owning shared wiring.
type TxBeginner interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// PostgresStore is the durable Store implementation over migrations 132-134.
// It stores only opaque UUIDs and policy metadata; it never resolves a home to
// a host path or reads credential material.
type PostgresStore struct {
	db TxBeginner
}

var (
	_ TxBeginner = (*pgxpool.Pool)(nil)
	_ Store      = (*PostgresStore)(nil)
)

func NewPostgresStore(database TxBeginner) *PostgresStore {
	return &PostgresStore{db: database}
}

// NewPostgresResolver is the C1-owned composition point. K1 still owns wiring
// it into shared claim and task-completion paths.
func NewPostgresResolver(pool *pgxpool.Pool) *Resolver {
	return NewResolver(NewPostgresStore(pool))
}

const selectTaskRuntimeSQL = `
SELECT runtime_id
FROM agent_task_queue
WHERE id = $1 AND agent_id = $2`

// This query reads only immutable/locked metadata after the generated C2 lock
// primitives have acquired task -> binding -> assignment -> catalog locks.
// The capability digest is sourced from the activation that produced the
// binding's current generation and configuration, never from caller input.
const selectCandidateMetadataSQL = `
SELECT
    rs.provider,
    e.provider,
    e.approved,
    e.state,
    e.health_watermark IS NOT NULL
        AND e.health_watermark + e.ttl >= now()
        AND e.retention_deadline > now(),
    standard.active_version_id,
    activation.capability_digest,
    (
        SELECT count(*)::integer
        FROM runtime_home_assignment owners
        WHERE owners.home_ref = assignment.home_ref
          AND owners.state IN ('active', 'draining')
    ),
    (
        SELECT count(*)::integer
        FROM runtime_home_assignment binding_assignments
        WHERE binding_assignments.binding_id = binding.id
          AND binding_assignments.state IN ('active', 'draining')
    )
FROM runtime_binding binding
JOIN runtime_session rs ON rs.id = binding.session_id
JOIN runtime_standard standard ON standard.id = rs.standard_id
JOIN runtime_home_assignment assignment
  ON assignment.id = $2 AND assignment.binding_id = binding.id
JOIN credential_home_catalog_entry e
  ON e.id = assignment.catalog_entry_id
 AND e.catalog_id = assignment.catalog_id
 AND e.generation = assignment.catalog_generation
 AND e.home_ref = assignment.home_ref
JOIN LATERAL (
    SELECT a.capability_digest
    FROM runtime_configuration_activation a
    WHERE a.binding_id = binding.id
      AND a.new_version_id = binding.active_configuration_version_id
      AND a.binding_generation = binding.generation
      AND a.effective_configuration_digest = binding.effective_configuration_digest
    ORDER BY a.activated_at DESC, a.id DESC
    LIMIT 1
) activation ON true
WHERE binding.id = $1`

const claimTaskSQL = `
UPDATE agent_task_queue
SET status = 'dispatched', dispatched_at = now()
WHERE id = $1 AND agent_id = $2 AND runtime_id = $3 AND status = 'queued'`

func (s *PostgresStore) ReserveAssignment(
	ctx context.Context,
	request Request,
	validate CandidateValidator,
) (Candidate, error) {
	if s == nil || s.db == nil || validate == nil {
		return Candidate{}, ErrRegistryUnavailable
	}
	ids, err := parseReservationIDs(request)
	if err != nil {
		return Candidate{}, err
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
	queries := db.New(tx)

	// Runtime ID is discovered without a lock only to address the generated
	// task-lock primitive. The locked row below remains the sole authority.
	if err := tx.QueryRow(ctx, selectTaskRuntimeSQL, ids.task, ids.agent).Scan(&ids.runtime); err != nil {
		return Candidate{}, mapPostgresError(err)
	}
	task, err := queries.LockAgentTaskForRuntimeSnapshot(ctx, db.LockAgentTaskForRuntimeSnapshotParams{
		TaskID: ids.task, AgentID: ids.agent, RuntimeID: ids.runtime,
	})
	if err != nil {
		return Candidate{}, mapPostgresError(err)
	}
	binding, err := queries.LockRuntimeBinding(ctx, db.LockRuntimeBindingParams{
		ID: ids.binding, WorkspaceID: ids.workspace,
	})
	if err != nil {
		return Candidate{}, mapPostgresError(err)
	}
	assignment, err := queries.GetActiveRuntimeHomeAssignmentForUpdate(ctx, ids.binding)
	if err != nil {
		return Candidate{}, mapPostgresError(err)
	}
	catalog, err := queries.LockCredentialHomeCatalog(ctx, db.LockCredentialHomeCatalogParams{
		ID: assignment.CatalogID, WorkspaceID: ids.workspace,
	})
	if err != nil {
		return Candidate{}, mapPostgresError(err)
	}

	candidate, err := loadCandidate(ctx, tx, task, binding, assignment, catalog)
	if err != nil {
		return Candidate{}, mapPostgresError(err)
	}
	snapshot, snapshotErr := queries.GetRuntimeTaskSnapshot(ctx, ids.task)
	switch {
	case snapshotErr == nil:
		candidate.ExistingSnapshot = true
		candidate.SnapshotMatches = snapshotMatchesCandidate(snapshot, candidate, assignment)
	case errors.Is(snapshotErr, pgx.ErrNoRows):
		// A new claim continues below.
	default:
		return Candidate{}, mapPostgresError(snapshotErr)
	}

	// This callback is the one admission authority. All durable source rows are
	// locked, and no capacity/task/snapshot mutation has happened yet.
	if err := validate(candidate); err != nil {
		return Candidate{}, err
	}
	standardVersionID, err := util.ParseUUID(candidate.StandardVersionID)
	if err != nil {
		return Candidate{}, ErrInvalidMetadata
	}

	if candidate.ExistingSnapshot {
		if err := tx.Commit(ctx); err != nil {
			return Candidate{}, mapPostgresError(err)
		}
		committed = true
		return candidate, nil
	}

	reserved, err := queries.ReserveNativeRuntimeBindingCapacity(ctx, db.ReserveNativeRuntimeBindingCapacityParams{
		BindingID:                 ids.binding,
		WorkspaceID:               ids.workspace,
		ExpectedBindingGeneration: int64(request.ExpectedBindingGeneration),
		HomeAssignmentID:          assignment.ID,
		ExpectedCatalogGeneration: int64(request.ExpectedCatalogGeneration),
		// Exact freshness (entry watermark + its own TTL) was validated above.
		HealthFreshAfter: pgtype.Timestamptz{Time: time.Unix(0, 0).UTC(), Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Candidate{}, ErrCapacityExhausted
		}
		return Candidate{}, mapPostgresError(err)
	}

	tag, err := tx.Exec(ctx, claimTaskSQL, ids.task, ids.agent, ids.runtime)
	if err != nil {
		return Candidate{}, mapPostgresError(err)
	}
	if tag.RowsAffected() != 1 {
		return Candidate{}, ErrTaskConflict
	}

	_, err = queries.CreateRuntimeTaskSnapshot(ctx, db.CreateRuntimeTaskSnapshotParams{
		TaskID:                        ids.task,
		RuntimeSessionID:              binding.SessionID,
		RuntimeID:                     binding.RuntimeID,
		AgentID:                       binding.AgentID,
		WorkspaceID:                   binding.WorkspaceID,
		RuntimeStandardVersionID:      standardVersionID,
		RuntimeConfigurationVersionID: binding.ActiveConfigurationVersionID,
		EffectiveConfigurationDigest:  binding.EffectiveConfigurationDigest.String,
		RuntimeBindingID:              binding.ID,
		BindingGeneration:             binding.Generation,
		TransportBinding:              binding.TransportBinding,
		HomeAssignmentID:              assignment.ID,
		HomeRef:                       assignment.HomeRef,
		CatalogGeneration:             pgtype.Int8{Int64: assignment.CatalogGeneration, Valid: true},
		CapabilityDigest:              candidate.CapabilityDigest,
	})
	if err != nil {
		return Candidate{}, mapPostgresError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Candidate{}, mapPostgresError(err)
	}
	committed = true
	candidate.ActiveTasks = int(reserved.ActiveTaskCount)
	return candidate, nil
}

type reservationIDs struct {
	task      pgtype.UUID
	binding   pgtype.UUID
	agent     pgtype.UUID
	workspace pgtype.UUID
	runtime   pgtype.UUID
}

func parseReservationIDs(request Request) (reservationIDs, error) {
	if request.ExpectedBindingGeneration > math.MaxInt64 || request.ExpectedCatalogGeneration > math.MaxInt64 {
		return reservationIDs{}, ErrGenerationConflict
	}
	var ids reservationIDs
	var err error
	if ids.task, err = util.ParseUUID(request.TaskID); err != nil {
		return reservationIDs{}, ErrNoApprovedAssignment
	}
	if ids.binding, err = util.ParseUUID(request.BindingID); err != nil {
		return reservationIDs{}, ErrNoApprovedAssignment
	}
	if ids.agent, err = util.ParseUUID(request.AgentID); err != nil {
		return reservationIDs{}, ErrNoApprovedAssignment
	}
	if ids.workspace, err = util.ParseUUID(request.WorkspaceID); err != nil {
		return reservationIDs{}, ErrNoApprovedAssignment
	}
	return ids, nil
}

func loadCandidate(
	ctx context.Context,
	tx pgx.Tx,
	task db.LockAgentTaskForRuntimeSnapshotRow,
	binding db.RuntimeBinding,
	assignment db.RuntimeHomeAssignment,
	catalog db.CredentialHomeCatalog,
) (Candidate, error) {
	var (
		sessionProvider string
		entryProvider   string
		approved        bool
		entryState      string
		healthFresh     bool
		standardVersion pgtype.UUID
		capability      string
		owners          int
		assignments     int
	)
	if err := tx.QueryRow(ctx, selectCandidateMetadataSQL, binding.ID, assignment.ID).Scan(
		&sessionProvider,
		&entryProvider,
		&approved,
		&entryState,
		&healthFresh,
		&standardVersion,
		&capability,
		&owners,
		&assignments,
	); err != nil {
		return Candidate{}, err
	}

	approval := ApprovalPending
	if approved {
		approval = ApprovalApproved
	}
	status := AccountStatus(entryState)
	if entryState == "healthy" {
		status = StatusAvailable
	}
	candidate := Candidate{
		TaskID:                      util.UUIDToString(task.ID),
		TaskStatus:                  task.Status,
		BindingID:                   util.UUIDToString(binding.ID),
		AgentID:                     util.UUIDToString(binding.AgentID),
		WorkspaceID:                 util.UUIDToString(binding.WorkspaceID),
		RuntimeID:                   util.UUIDToString(binding.RuntimeID),
		RuntimeSessionID:            util.UUIDToString(binding.SessionID),
		StandardVersionID:           util.UUIDToString(standardVersion),
		ConfigurationVersionID:      util.UUIDToString(binding.ActiveConfigurationVersionID),
		ConfigurationDigest:         binding.EffectiveConfigurationDigest.String,
		CapabilityDigest:            capability,
		Provider:                    entryProvider,
		SessionProvider:             sessionProvider,
		HomeRef:                     HomeRef(util.UUIDToString(assignment.HomeRef)),
		BindingGeneration:           uint64(binding.Generation),
		AssignmentCatalogGeneration: uint64(assignment.CatalogGeneration),
		CatalogGeneration:           uint64(catalog.Generation),
		Approval:                    approval,
		Status:                      status,
		BindingState:                binding.State,
		TransportBinding:            binding.TransportBinding,
		AssignmentState:             assignment.State,
		CatalogState:                catalog.State,
		CatalogEntryState:           entryState,
		HealthFresh:                 healthFresh,
		AssignmentOwners:            owners,
		BindingAssignments:          assignments,
		ActiveTasks:                 int(binding.ActiveTaskCount),
		TaskConcurrencyLimit:        int(binding.MaxConcurrentTasks),
	}
	return candidate, nil
}

func snapshotMatchesCandidate(snapshot db.RuntimeTaskSnapshot, candidate Candidate, assignment db.RuntimeHomeAssignment) bool {
	return util.UUIDToString(snapshot.TaskID) == candidate.TaskID &&
		util.UUIDToString(snapshot.RuntimeSessionID) == candidate.RuntimeSessionID &&
		util.UUIDToString(snapshot.RuntimeID) == candidate.RuntimeID &&
		util.UUIDToString(snapshot.AgentID) == candidate.AgentID &&
		util.UUIDToString(snapshot.WorkspaceID) == candidate.WorkspaceID &&
		util.UUIDToString(snapshot.RuntimeStandardVersionID) == candidate.StandardVersionID &&
		util.UUIDToString(snapshot.RuntimeConfigurationVersionID) == candidate.ConfigurationVersionID &&
		snapshot.EffectiveConfigurationDigest == candidate.ConfigurationDigest &&
		util.UUIDToString(snapshot.RuntimeBindingID) == candidate.BindingID &&
		snapshot.BindingGeneration == int64(candidate.BindingGeneration) &&
		snapshot.TransportBinding == candidate.TransportBinding &&
		snapshot.HomeAssignmentID == assignment.ID &&
		snapshot.HomeRef == assignment.HomeRef &&
		snapshot.CatalogGeneration.Valid && snapshot.CatalogGeneration.Int64 == int64(candidate.CatalogGeneration) &&
		snapshot.CapabilityDigest == candidate.CapabilityDigest
}

func (s *PostgresStore) ReleaseAssignment(ctx context.Context, request ReleaseRequest) error {
	if s == nil || s.db == nil {
		return ErrRegistryUnavailable
	}
	if request.BindingGeneration > math.MaxInt64 || request.CatalogGeneration > math.MaxInt64 {
		return ErrGenerationConflict
	}
	taskID, err := util.ParseUUID(request.TaskID)
	if err != nil {
		return ErrNoApprovedAssignment
	}
	bindingID, err := util.ParseUUID(request.BindingID)
	if err != nil {
		return ErrNoApprovedAssignment
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
	queries := db.New(tx)

	snapshot, err := queries.GetRuntimeTaskSnapshot(ctx, taskID)
	if err != nil {
		return mapPostgresError(err)
	}
	if _, err := queries.LockAgentTaskForRuntimeSnapshot(ctx, db.LockAgentTaskForRuntimeSnapshotParams{
		TaskID: taskID, AgentID: snapshot.AgentID, RuntimeID: snapshot.RuntimeID,
	}); err != nil {
		return mapPostgresError(err)
	}
	if _, err := queries.LockRuntimeBinding(ctx, db.LockRuntimeBindingParams{
		ID: bindingID, WorkspaceID: snapshot.WorkspaceID,
	}); err != nil {
		return mapPostgresError(err)
	}
	if snapshot.RuntimeBindingID != bindingID ||
		snapshot.BindingGeneration != int64(request.BindingGeneration) ||
		!snapshot.CatalogGeneration.Valid ||
		snapshot.CatalogGeneration.Int64 != int64(request.CatalogGeneration) {
		return ErrGenerationConflict
	}

	_, releaseErr := queries.ReleaseRuntimeTaskSnapshotCapacity(ctx, db.ReleaseRuntimeTaskSnapshotCapacityParams{
		ReleasedBy: pgtype.UUID{}, ReasonCode: "task_terminal",
		TaskID: taskID, RuntimeBindingID: bindingID,
	})
	if errors.Is(releaseErr, pgx.ErrNoRows) {
		release, replayErr := queries.GetRuntimeTaskSnapshotRelease(ctx, taskID)
		if replayErr != nil || release.RuntimeBindingID != bindingID {
			return ErrTaskConflict
		}
	} else if releaseErr != nil {
		return mapPostgresError(releaseErr)
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
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) {
		switch postgresError.Code {
		case "40001":
			return ErrGenerationConflict
		case "23505", "23514", "40P01":
			return ErrTaskConflict
		}
	}
	return ErrRegistryUnavailable
}
