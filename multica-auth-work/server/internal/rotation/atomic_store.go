package rotation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	dbgen "github.com/multica-ai/multica/server/pkg/db/generated"
)

var (
	ErrAtomicRotationRequired = errors.New("rotation: atomic assignment store required")
	ErrStaleAssignment        = errors.New("rotation: stale assignment")
)

// AtomicRotationStore serializes one agent's session mutation across processes.
// prepare runs only after the current assignment has been locked and verified.
// A prepare or persistence failure rolls the transaction back, leaving the
// previous assignment and rotation audit unchanged.
type AtomicRotationStore interface {
	RotateAssignmentAtomic(
		ctx context.Context,
		agentID, expectedAccountID, nextAccountID string,
		reason RotationReason,
		at time.Time,
		prepare func(context.Context) error,
	) error
}

var _ AtomicRotationStore = (*PGStore)(nil)

func (s *PGStore) RotateAssignmentAtomic(
	ctx context.Context,
	agentID, expectedAccountID, nextAccountID string,
	reason RotationReason,
	at time.Time,
	prepare func(context.Context) error,
) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("rotation: begin atomic assignment: %w", err)
	}
	defer tx.Rollback(ctx)

	var currentAccountID string
	if err := tx.QueryRow(ctx, `
		SELECT account_id
		  FROM assignments
		 WHERE agent_id = $1
		 FOR UPDATE
	`, agentID).Scan(&currentAccountID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrStaleAssignment
		}
		return fmt.Errorf("rotation: lock current assignment: %w", err)
	}
	if currentAccountID != expectedAccountID {
		return ErrStaleAssignment
	}

	if prepare != nil {
		if err := prepare(ctx); err != nil {
			return err
		}
	}

	tag, err := tx.Exec(ctx, `
		UPDATE assignments
		   SET account_id = $3,
		       assigned_at = now()
		 WHERE agent_id = $1
		   AND account_id = $2
	`, agentID, expectedAccountID, nextAccountID)
	if err != nil {
		return fmt.Errorf("rotation: update atomic assignment: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrStaleAssignment
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO rotation_events (agent_id, from_account_id, to_account_id, reason, at)
		VALUES ($1, $2, $3, $4, $5)
	`, agentID, nullableUUID(expectedAccountID), nullableUUID(nextAccountID), string(reason), at); err != nil {
		return fmt.Errorf("rotation: record atomic rotation: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("rotation: commit atomic assignment: %w", err)
	}
	return nil
}

var _ NativeRotationStore = (*PGStore)(nil)

func nativeUUID(value string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(value); err != nil {
		return pgtype.UUID{}, fmt.Errorf("%w: uuid", ErrInvalidNativeIdentity)
	}
	return id, nil
}

func mustNativeUUID(value string) pgtype.UUID {
	var id pgtype.UUID
	_ = id.Scan(value)
	return id
}

func nativeText(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: value != ""}
}

func nativeEventID() pgtype.UUID {
	return mustNativeUUID(uuid.NewString())
}

func receiptFor(identity NativeHomeIdentityV1, state string, version int16, digest string) NativeHomeReceiptV1 {
	return NativeHomeReceiptV1{
		Identity: identity,
		Fence: NativeHomeFenceV1{
			LifetimeState:         state,
			LifetimeStateVersion:  version,
			ProcessIdentityDigest: digest,
		},
	}
}

func (s *PGStore) ReserveNativeCandidate(
	ctx context.Context,
	request NativeRotationRequestV1,
) (NativeHomeReceiptV1, error) {
	if err := request.validate(); err != nil {
		return NativeHomeReceiptV1{}, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return NativeHomeReceiptV1{}, fmt.Errorf("rotation: begin A1: %w", err)
	}
	defer tx.Rollback(ctx)
	q := dbgen.New(tx)
	if err := lockNativePhase(ctx, q, request, false, false); err != nil {
		return NativeHomeReceiptV1{}, err
	}

	target := request.Target
	if _, err := q.CreateNativeTargetAssignment(ctx, dbgen.CreateNativeTargetAssignmentParams{
		ID:                mustNativeUUID(target.HomeAssignmentID),
		BindingID:         mustNativeUUID(target.RuntimeBindingID),
		WorkspaceID:       mustNativeUUID(target.WorkspaceID),
		CatalogID:         mustNativeUUID(target.CatalogID),
		CatalogEntryID:    mustNativeUUID(target.CatalogEntryID),
		CatalogGeneration: target.CatalogGeneration,
		HomeRef:           mustNativeUUID(target.HomeRef),
		BindingGeneration: target.BindingGeneration,
		AssignedBy:        mustNativeUUID(request.AssignedBy),
		ReasonCode:        nativeText(request.ReasonCode),
	}); err != nil {
		return NativeHomeReceiptV1{}, fmt.Errorf("rotation: create target assignment: %w", err)
	}
	if _, err := q.CreateNativeTaskHomeEpoch(ctx, epochParams(target)); err != nil {
		return NativeHomeReceiptV1{}, fmt.Errorf("rotation: create target epoch: %w", err)
	}
	if _, err := q.CreateNativeHomeLifetime(ctx, lifetimeParams(target)); err != nil {
		return NativeHomeReceiptV1{}, fmt.Errorf("rotation: create target lifetime: %w", err)
	}
	if err := q.RecordNativeHomeLifetimeEvent(ctx, dbgen.RecordNativeHomeLifetimeEventParams{
		LifetimeID: mustNativeUUID(target.LifetimeID), StateVersion: 1,
		State: "pending_local", TransitionRequestID: nativeEventID(),
		ReasonCode: nativeText("candidate_reserved"),
	}); err != nil {
		return NativeHomeReceiptV1{}, fmt.Errorf("rotation: record target lifetime: %w", err)
	}
	operation, err := q.CreateNativeRotationOperation(ctx, operationParams(request))
	if err != nil {
		return NativeHomeReceiptV1{}, fmt.Errorf("rotation: create operation: %w", err)
	}
	if err := q.RecordNativeRotationOperationEvent(ctx, dbgen.RecordNativeRotationOperationEventParams{
		OperationID: operation.ID, StateVersion: 1, State: "candidate_reserved",
		TransitionRequestID: mustNativeUUID(request.TransitionRequestID),
		ReasonCode:          request.ReasonCode,
	}); err != nil {
		return NativeHomeReceiptV1{}, fmt.Errorf("rotation: record operation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return NativeHomeReceiptV1{}, fmt.Errorf("rotation: commit A1: %w", err)
	}
	return receiptFor(target, "pending_local", 1, ""), nil
}

func (s *PGStore) RecordNativePreparation(
	ctx context.Context,
	request NativeRotationRequestV1,
	succeeded bool,
	processDigest string,
) (NativeHomeReceiptV1, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return NativeHomeReceiptV1{}, fmt.Errorf("rotation: begin A2: %w", err)
	}
	defer tx.Rollback(ctx)
	q := dbgen.New(tx)
	if err := lockNativePhase(ctx, q, request, true, true); err != nil {
		return NativeHomeReceiptV1{}, err
	}
	nextState := "aborted_candidate_retirement_pending"
	target := receiptFor(request.Target, "pending_local", 1, "")
	if succeeded {
		lifetime, err := q.AdvanceNativeHomeLifetime(ctx, dbgen.AdvanceNativeHomeLifetimeParams{
			NextState: "acquired", ID: mustNativeUUID(request.Target.LifetimeID),
			ExpectedState: "pending_local", ExpectedStateVersion: 1,
		})
		if err != nil {
			return NativeHomeReceiptV1{}, fmt.Errorf("rotation: acquire target lifetime: %w", err)
		}
		if err := q.RecordNativeHomeLifetimeEvent(ctx, dbgen.RecordNativeHomeLifetimeEventParams{
			LifetimeID: lifetime.ID, StateVersion: lifetime.StateVersion,
			State: "acquired", TransitionRequestID: nativeEventID(),
			ReasonCode: nativeText("local_gate_acquired"),
		}); err != nil {
			return NativeHomeReceiptV1{}, fmt.Errorf("rotation: record acquired target: %w", err)
		}
		nextState = "candidate_prepared"
		target = receiptFor(request.Target, "acquired", 2, processDigest)
	} else {
		lifetime, err := q.AdvanceNativeHomeLifetime(ctx, dbgen.AdvanceNativeHomeLifetimeParams{
			NextState: "recovery_pending", ID: mustNativeUUID(request.Target.LifetimeID),
			ExpectedState: "pending_local", ExpectedStateVersion: 1,
		})
		if err != nil {
			return NativeHomeReceiptV1{}, fmt.Errorf("rotation: fence failed target: %w", err)
		}
		if err := recordLifetimeEvent(ctx, q, lifetime, "candidate_preparation_failed"); err != nil {
			return NativeHomeReceiptV1{}, err
		}
		target = receiptFor(request.Target, "recovery_pending", 2, "")
	}
	operation, err := q.AdvanceNativeRotationOperation(ctx, dbgen.AdvanceNativeRotationOperationParams{
		NextState: nextState, ReasonCode: request.ReasonCode,
		ID: mustNativeUUID(request.OperationID), ExpectedState: "candidate_reserved",
		ExpectedStateVersion: 1,
	})
	if err != nil {
		return NativeHomeReceiptV1{}, fmt.Errorf("rotation: advance A2 operation: %w", err)
	}
	if err := q.RecordNativeRotationOperationEvent(ctx, dbgen.RecordNativeRotationOperationEventParams{
		OperationID: operation.ID, StateVersion: operation.StateVersion,
		State: operation.State, TransitionRequestID: nativeEventID(),
		ReasonCode: operation.ReasonCode,
	}); err != nil {
		return NativeHomeReceiptV1{}, fmt.Errorf("rotation: record A2 operation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return NativeHomeReceiptV1{}, fmt.Errorf("rotation: commit A2: %w", err)
	}
	return target, nil
}

func (s *PGStore) CommitNativeSwap(
	ctx context.Context,
	request NativeRotationRequestV1,
	processDigest string,
) (NativeRotationResultV1, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return NativeRotationResultV1{}, fmt.Errorf("rotation: begin B: %w", err)
	}
	defer tx.Rollback(ctx)
	q := dbgen.New(tx)
	if err := lockNativePhase(ctx, q, request, true, true); err != nil {
		return NativeRotationResultV1{}, err
	}
	rows, err := q.SwapNativeRotationAssignments(ctx, dbgen.SwapNativeRotationAssignmentsParams{
		CurrentAssignmentID: mustNativeUUID(request.Current.Identity.HomeAssignmentID),
		TargetAssignmentID:  mustNativeUUID(request.Target.HomeAssignmentID),
		ReasonCode:          nativeText(request.ReasonCode),
	})
	if err != nil || rows != 2 {
		if err == nil {
			err = ErrNativeCASConflict
		}
		return NativeRotationResultV1{}, fmt.Errorf("rotation: swap assignments: %w", err)
	}
	oldLifetime, err := q.AdvanceNativeHomeLifetime(ctx, dbgen.AdvanceNativeHomeLifetimeParams{
		NextState: "retiring", ID: mustNativeUUID(request.Current.Identity.LifetimeID),
		ExpectedState:        "process_started",
		ExpectedStateVersion: request.Current.Fence.LifetimeStateVersion,
	})
	if err != nil {
		return NativeRotationResultV1{}, fmt.Errorf("rotation: retire current lifetime: %w", err)
	}
	if err := recordLifetimeEvent(ctx, q, oldLifetime, "swap_committed"); err != nil {
		return NativeRotationResultV1{}, err
	}
	targetLifetime, err := q.AdvanceNativeHomeLifetime(ctx, dbgen.AdvanceNativeHomeLifetimeParams{
		NextState: "process_started", ProcessIdentityDigest: nativeText(processDigest),
		ID: mustNativeUUID(request.Target.LifetimeID), ExpectedState: "acquired",
		ExpectedStateVersion: 2,
	})
	if err != nil {
		return NativeRotationResultV1{}, fmt.Errorf("rotation: start target lifetime: %w", err)
	}
	if err := recordLifetimeEvent(ctx, q, targetLifetime, "swap_committed"); err != nil {
		return NativeRotationResultV1{}, err
	}
	operation, err := q.AdvanceNativeRotationOperation(ctx, dbgen.AdvanceNativeRotationOperationParams{
		NextState: "committed_retirement_pending", ReasonCode: request.ReasonCode,
		ID: mustNativeUUID(request.OperationID), ExpectedState: "candidate_prepared",
		ExpectedStateVersion: 2,
		NextAttemptAt:        pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	})
	if err != nil {
		return NativeRotationResultV1{}, fmt.Errorf("rotation: commit swap operation: %w", err)
	}
	if err := recordOperationEvent(ctx, q, operation); err != nil {
		return NativeRotationResultV1{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return s.ResolveNativeCommit(ctx, request)
	}
	return NativeRotationResultV1{
		OperationRequestID: request.OperationRequestID,
		Outcome:            NativeCommittedRetirementPending,
		Current: receiptFor(request.Current.Identity, "retiring", oldLifetime.StateVersion,
			request.Current.Fence.ProcessIdentityDigest),
		Target:     receiptFor(request.Target, "process_started", targetLifetime.StateVersion, processDigest),
		ReasonCode: request.ReasonCode,
	}, nil
}

func (s *PGStore) ResolveNativeCommit(
	ctx context.Context,
	request NativeRotationRequestV1,
) (NativeRotationResultV1, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return unknownNativeResult(request), errors.Join(ErrNativeCommitUnknown, err)
	}
	defer tx.Rollback(ctx)
	q := dbgen.New(tx)
	if err := lockNativePhase(ctx, q, request, true, true); err != nil {
		return unknownNativeResult(request), errors.Join(ErrNativeCommitUnknown, err)
	}
	operation, err := q.LockNativeRotationOperation(ctx, mustNativeUUID(request.OperationID))
	if err != nil {
		return unknownNativeResult(request), errors.Join(ErrNativeCommitUnknown, err)
	}
	switch operation.State {
	case "committed_retirement_pending", "committed_retired":
		outcome := NativeCommittedRetirementPending
		if operation.State == "committed_retired" {
			outcome = NativeCommittedRetired
		}
		return NativeRotationResultV1{
			OperationRequestID: request.OperationRequestID, Outcome: outcome,
			Current: request.Current, Target: receiptFor(request.Target, "process_started", 3, ""),
			ReasonCode: operation.ReasonCode,
		}, nil
	case "candidate_prepared":
		operation, err = q.AdvanceNativeRotationOperation(ctx, dbgen.AdvanceNativeRotationOperationParams{
			NextState:  "aborted_candidate_retirement_pending",
			ReasonCode: "swap_definitely_not_committed",
			ID:         operation.ID, ExpectedState: operation.State,
			ExpectedStateVersion: operation.StateVersion,
		})
		if err != nil {
			return unknownNativeResult(request), errors.Join(ErrNativeCommitUnknown, err)
		}
		if err := recordOperationEvent(ctx, q, operation); err != nil {
			return unknownNativeResult(request), errors.Join(ErrNativeCommitUnknown, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return unknownNativeResult(request), errors.Join(ErrNativeCommitUnknown, err)
		}
		return NativeRotationResultV1{
			OperationRequestID: request.OperationRequestID,
			Outcome:            NativeDefinitelyNotCommitted,
			Current:            request.Current, Target: receiptFor(request.Target, "acquired", 2, ""),
			ReasonCode: operation.ReasonCode,
		}, nil
	default:
		if operation.State != "swap_commit_unknown_fenced" {
			fenced, casErr := q.AdvanceNativeRotationOperation(ctx, dbgen.AdvanceNativeRotationOperationParams{
				NextState:  "swap_commit_unknown_fenced",
				ReasonCode: "mixed_state_commit_unknown",
				ID:         operation.ID, ExpectedState: operation.State,
				ExpectedStateVersion: operation.StateVersion,
			})
			if casErr == nil {
				_ = recordOperationEvent(ctx, q, fenced)
				_ = tx.Commit(ctx)
			}
		}
		return unknownNativeResult(request), ErrNativeCommitUnknown
	}
}

func lockNativePhase(
	ctx context.Context,
	q *dbgen.Queries,
	request NativeRotationRequestV1,
	includeTarget, includeOperation bool,
) error {
	current, target := request.Current.Identity, request.Target
	if _, err := q.AcquireNativeRotationHomeLocks(ctx, []pgtype.UUID{
		mustNativeUUID(current.HomeRef), mustNativeUUID(target.HomeRef),
	}); err != nil {
		return fmt.Errorf("rotation: acquire home locks: %w", err)
	}
	if _, err := q.LockNativeRotationTask(ctx, dbgen.LockNativeRotationTaskParams{
		TaskID: mustNativeUUID(current.TaskID), AgentID: mustNativeUUID(current.AgentID),
		RuntimeID: mustNativeUUID(current.RuntimeID),
	}); err != nil {
		return fmt.Errorf("rotation: lock task: %w", err)
	}
	if _, err := q.LockNativeRotationBinding(ctx, dbgen.LockNativeRotationBindingParams{
		RuntimeBindingID: mustNativeUUID(current.RuntimeBindingID),
		WorkspaceID:      mustNativeUUID(current.WorkspaceID),
		RuntimeSessionID: mustNativeUUID(current.RuntimeSessionID),
		RuntimeID:        mustNativeUUID(current.RuntimeID), AgentID: mustNativeUUID(current.AgentID),
		BindingGeneration: current.BindingGeneration,
	}); err != nil {
		return fmt.Errorf("rotation: lock binding: %w", err)
	}
	assignments := []pgtype.UUID{mustNativeUUID(current.HomeAssignmentID)}
	epochs := []int64{current.HomeEpoch}
	lifetimes := []pgtype.UUID{mustNativeUUID(current.LifetimeID)}
	if includeTarget {
		assignments = append(assignments, mustNativeUUID(target.HomeAssignmentID))
		epochs = append(epochs, target.HomeEpoch)
		lifetimes = append(lifetimes, mustNativeUUID(target.LifetimeID))
	}
	if rows, err := q.LockNativeRotationAssignments(ctx, assignments); err != nil || len(rows) != len(assignments) {
		return ErrNativeCASConflict
	}
	if rows, err := q.LockNativeRotationCatalogs(ctx, []pgtype.UUID{
		mustNativeUUID(current.CatalogID), mustNativeUUID(target.CatalogID),
	}); err != nil || len(rows) < 1 {
		return ErrNativeCASConflict
	}
	if rows, err := q.LockNativeRotationEntries(ctx, []pgtype.UUID{
		mustNativeUUID(current.CatalogEntryID), mustNativeUUID(target.CatalogEntryID),
	}); err != nil || len(rows) < 1 {
		return ErrNativeCASConflict
	}
	if _, err := q.LockNativeRotationLifecycle(ctx, dbgen.LockNativeRotationLifecycleParams{
		CatalogIds: []pgtype.UUID{mustNativeUUID(current.CatalogID), mustNativeUUID(target.CatalogID)},
		HomeRefs:   []pgtype.UUID{mustNativeUUID(current.HomeRef), mustNativeUUID(target.HomeRef)},
	}); err != nil {
		return fmt.Errorf("rotation: lock lifecycle: %w", err)
	}
	if rows, err := q.LockNativeRotationEpochs(ctx, dbgen.LockNativeRotationEpochsParams{
		TaskID: mustNativeUUID(current.TaskID), HomeEpochs: epochs,
	}); err != nil || len(rows) != len(epochs) {
		return ErrNativeCASConflict
	}
	if rows, err := q.LockNativeRotationLifetimes(ctx, lifetimes); err != nil || len(rows) != len(lifetimes) {
		return ErrNativeCASConflict
	}
	if includeOperation {
		if _, err := q.LockNativeRotationOperation(ctx, mustNativeUUID(request.OperationID)); err != nil {
			return fmt.Errorf("rotation: lock operation: %w", err)
		}
	}
	return nil
}

func epochParams(i NativeHomeIdentityV1) dbgen.CreateNativeTaskHomeEpochParams {
	return dbgen.CreateNativeTaskHomeEpochParams{
		TaskID: mustNativeUUID(i.TaskID), HomeEpoch: i.HomeEpoch,
		WorkspaceID: mustNativeUUID(i.WorkspaceID), AgentID: mustNativeUUID(i.AgentID),
		RuntimeID: mustNativeUUID(i.RuntimeID), RuntimeSessionID: mustNativeUUID(i.RuntimeSessionID),
		DaemonID: i.DaemonID, DaemonBootID: mustNativeUUID(i.DaemonBootID),
		Provider: i.Provider, RuntimeBindingID: mustNativeUUID(i.RuntimeBindingID),
		BindingGeneration: i.BindingGeneration, HomeAssignmentID: mustNativeUUID(i.HomeAssignmentID),
		CatalogID: mustNativeUUID(i.CatalogID), CatalogEntryID: mustNativeUUID(i.CatalogEntryID),
		CatalogGeneration: i.CatalogGeneration, HomeRef: mustNativeUUID(i.HomeRef),
		LifetimeID:           mustNativeUUID(i.LifetimeID),
		AcquisitionRequestID: mustNativeUUID(i.AcquisitionRequestID),
	}
}

func lifetimeParams(i NativeHomeIdentityV1) dbgen.CreateNativeHomeLifetimeParams {
	return dbgen.CreateNativeHomeLifetimeParams{
		ID: mustNativeUUID(i.LifetimeID), AcquisitionRequestID: mustNativeUUID(i.AcquisitionRequestID),
		TaskID: mustNativeUUID(i.TaskID), HomeEpoch: i.HomeEpoch,
		AgentID: mustNativeUUID(i.AgentID), RuntimeID: mustNativeUUID(i.RuntimeID),
		RuntimeSessionID: mustNativeUUID(i.RuntimeSessionID),
		RuntimeBindingID: mustNativeUUID(i.RuntimeBindingID), BindingGeneration: i.BindingGeneration,
		HomeAssignmentID: mustNativeUUID(i.HomeAssignmentID), WorkspaceID: mustNativeUUID(i.WorkspaceID),
		DaemonID: i.DaemonID, CatalogID: mustNativeUUID(i.CatalogID),
		CatalogGeneration: i.CatalogGeneration, HomeRef: mustNativeUUID(i.HomeRef),
		DaemonBootID: mustNativeUUID(i.DaemonBootID), Provider: i.Provider,
	}
}

func operationParams(r NativeRotationRequestV1) dbgen.CreateNativeRotationOperationParams {
	return dbgen.CreateNativeRotationOperationParams{
		ID: mustNativeUUID(r.OperationID), OperationRequestID: mustNativeUUID(r.OperationRequestID),
		TaskID:              mustNativeUUID(r.Current.Identity.TaskID),
		RuntimeBindingID:    mustNativeUUID(r.Current.Identity.RuntimeBindingID),
		CurrentHomeEpoch:    r.Current.Identity.HomeEpoch,
		CurrentHomeRef:      mustNativeUUID(r.Current.Identity.HomeRef),
		CurrentAssignmentID: mustNativeUUID(r.Current.Identity.HomeAssignmentID),
		CurrentLifetimeID:   mustNativeUUID(r.Current.Identity.LifetimeID),
		TargetHomeEpoch:     r.Target.HomeEpoch, TargetHomeRef: mustNativeUUID(r.Target.HomeRef),
		TargetAssignmentID: mustNativeUUID(r.Target.HomeAssignmentID),
		TargetLifetimeID:   mustNativeUUID(r.Target.LifetimeID), ReasonCode: r.ReasonCode,
	}
}

func recordLifetimeEvent(ctx context.Context, q *dbgen.Queries, lifetime dbgen.RuntimeHomeLifetime, reason string) error {
	return q.RecordNativeHomeLifetimeEvent(ctx, dbgen.RecordNativeHomeLifetimeEventParams{
		LifetimeID: lifetime.ID, StateVersion: lifetime.StateVersion,
		State: lifetime.State, TransitionRequestID: nativeEventID(),
		ReasonCode: nativeText(reason),
	})
}

func recordOperationEvent(ctx context.Context, q *dbgen.Queries, operation dbgen.NativeRotationOperation) error {
	return q.RecordNativeRotationOperationEvent(ctx, dbgen.RecordNativeRotationOperationEventParams{
		OperationID: operation.ID, StateVersion: operation.StateVersion,
		State: operation.State, TransitionRequestID: nativeEventID(),
		ReasonCode: operation.ReasonCode,
	})
}

func unknownNativeResult(request NativeRotationRequestV1) NativeRotationResultV1 {
	return NativeRotationResultV1{
		OperationRequestID: request.OperationRequestID,
		Outcome:            NativeCommitUnknownFenced,
		Current:            request.Current, Target: receiptFor(request.Target, "acquired", 2, ""),
		ReasonCode: "commit_unknown_fenced",
	}
}
