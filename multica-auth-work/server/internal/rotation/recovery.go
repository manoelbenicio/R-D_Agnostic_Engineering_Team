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

const (
	maxNativeRetirementAttempts = 6
	nativeRetirementLease       = 30 * time.Second
)

var nativeRetirementBackoff = [...]time.Duration{
	0, 5 * time.Second, 30 * time.Second, 2 * time.Minute,
	10 * time.Minute, 30 * time.Minute,
}

// NativeRotationRecoveryScheduler owns durable database scheduling only. It
// cannot resolve a home path or perform a credential action.
type NativeRotationRecoveryScheduler struct {
	store *PGStore
	owner string
}

func NewNativeRotationRecoveryScheduler(store *PGStore, owner string) *NativeRotationRecoveryScheduler {
	return &NativeRotationRecoveryScheduler{store: store, owner: owner}
}

// ScheduleDue creates durable, non-overlapping scheduled attempts. Delivery
// and authenticated acceptance belong to the excluded section-5 transport.
func (s *NativeRotationRecoveryScheduler) ScheduleDue(
	ctx context.Context,
	now time.Time,
	batchSize int32,
) ([]NativeRetirementRequestV1, error) {
	if s == nil || s.store == nil || s.owner == "" || batchSize < 1 {
		return nil, ErrInvalidNativeIdentity
	}
	operations, err := dbgen.New(s.store.pool).ListDueNativeRetirementOperations(
		ctx, dbgen.ListDueNativeRetirementOperationsParams{
			Now: pgtype.Timestamptz{Time: now, Valid: true}, BatchSize: batchSize,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("rotation: list due retirement operations: %w", err)
	}
	requests := make([]NativeRetirementRequestV1, 0, len(operations))
	for _, operation := range operations {
		request, err := s.store.scheduleNativeRetirement(ctx, s.owner, operation, now)
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, ErrNativeCASConflict) {
			continue
		}
		if err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	return requests, nil
}

func (s *PGStore) scheduleNativeRetirement(
	ctx context.Context,
	owner string,
	operation dbgen.NativeRotationOperation,
	now time.Time,
) (NativeRetirementRequestV1, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return NativeRetirementRequestV1{}, err
	}
	defer tx.Rollback(ctx)
	q := dbgen.New(tx)

	homeEpoch := operation.CurrentHomeEpoch
	lifetimeID := operation.CurrentLifetimeID
	homeRef := operation.CurrentHomeRef
	if operation.State == "aborted_candidate_retirement_pending" {
		homeEpoch = operation.TargetHomeEpoch
		lifetimeID = operation.TargetLifetimeID
		homeRef = operation.TargetHomeRef
	}
	if _, err := q.AcquireNativeRotationHomeLocks(ctx, []pgtype.UUID{homeRef}); err != nil {
		return NativeRetirementRequestV1{}, err
	}
	epoch, err := q.GetNativeTaskHomeEpoch(ctx, dbgen.GetNativeTaskHomeEpochParams{
		TaskID: operation.TaskID, HomeEpoch: homeEpoch,
	})
	if err != nil {
		return NativeRetirementRequestV1{}, err
	}
	locked, _, err := lockNativeRetirementChain(ctx, q, epoch, operation)
	if err != nil || locked.State != operation.State || locked.StateVersion != operation.StateVersion {
		return NativeRetirementRequestV1{}, ErrNativeCASConflict
	}
	attemptCount, err := q.CountNativeRetirementAttempts(ctx, operation.ID)
	if err != nil || attemptCount >= maxNativeRetirementAttempts {
		return NativeRetirementRequestV1{}, ErrNativeCASConflict
	}
	attemptID := mustNativeUUID(uuid.NewString())
	requestID := mustNativeUUID(uuid.NewString())
	attempt, err := q.CreateNativeRetirementAttempt(ctx, dbgen.CreateNativeRetirementAttemptParams{
		ID: attemptID, RetirementRequestID: requestID, OperationID: operation.ID,
		TaskID: operation.TaskID, HomeEpoch: homeEpoch, LifetimeID: lifetimeID,
		HomeRef: homeRef, DaemonID: epoch.DaemonID, DaemonBootID: epoch.DaemonBootID,
		RuntimeSessionID: epoch.RuntimeSessionID, RuntimeBindingID: epoch.RuntimeBindingID,
		BindingGeneration: epoch.BindingGeneration, SchedulerOwner: owner,
		AttemptNumber:  attemptCount + 1,
		LeaseExpiresAt: pgtype.Timestamptz{Time: now.Add(nativeRetirementLease), Valid: true},
		NextAttemptAt:  pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		return NativeRetirementRequestV1{}, err
	}
	if err := q.RecordNativeRetirementAttemptEvent(ctx, dbgen.RecordNativeRetirementAttemptEventParams{
		AttemptID: attempt.ID, StateVersion: attempt.StateVersion, State: attempt.State,
		TransitionRequestID: nativeEventID(), ReasonCode: "retirement_scheduled",
	}); err != nil {
		return NativeRetirementRequestV1{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return NativeRetirementRequestV1{}, err
	}
	return NativeRetirementRequestV1{
		AttemptID: uuidString(attempt.ID), RetirementRequestID: uuidString(attempt.RetirementRequestID),
		OperationID: uuidString(operation.ID),
		Retiring:    NativeHomeReceiptV1{Identity: identityFromEpoch(epoch)},
		DaemonID:    epoch.DaemonID, DaemonBootID: uuidString(epoch.DaemonBootID),
		RuntimeSessionID:  uuidString(epoch.RuntimeSessionID),
		RuntimeBindingID:  uuidString(epoch.RuntimeBindingID),
		BindingGeneration: epoch.BindingGeneration, AttemptNumber: attempt.AttemptNumber,
	}, nil
}

// RecordResult accepts only value-free bounded results. An indeterminate or
// exhausted result quarantines; a proven completed failure may schedule the
// same operation later, but never overlaps accepted/executing work.
func (s *NativeRotationRecoveryScheduler) RecordResult(
	ctx context.Context,
	result NativeRetirementResultV1,
	now time.Time,
) error {
	if s == nil || s.store == nil {
		return ErrInvalidNativeIdentity
	}
	return s.store.recordNativeRetirementResult(ctx, s.owner, result, now)
}

func (s *PGStore) recordNativeRetirementResult(
	ctx context.Context,
	owner string,
	result NativeRetirementResultV1,
	now time.Time,
) error {
	if len(result.ChannelBindingDigest) != 64 || len(result.RequestBodyDigest) != 64 {
		return ErrInvalidNativeIdentity
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := dbgen.New(tx)
	attempt, err := q.GetNativeRetirementAttempt(ctx, mustNativeUUID(result.AttemptID))
	if err != nil || uuidString(attempt.RetirementRequestID) != result.RetirementRequestID {
		return ErrNativeCASConflict
	}
	operation, err := q.GetNativeRotationOperation(ctx, attempt.OperationID)
	if err != nil {
		return err
	}
	epoch, err := q.GetNativeTaskHomeEpoch(ctx, dbgen.GetNativeTaskHomeEpochParams{
		TaskID: attempt.TaskID, HomeEpoch: attempt.HomeEpoch,
	})
	if err != nil {
		return err
	}
	if _, err := q.AcquireNativeRotationHomeLocks(ctx, []pgtype.UUID{attempt.HomeRef}); err != nil {
		return err
	}
	operation, lockedLifetime, err := lockNativeRetirementChain(ctx, q, epoch, operation)
	if err != nil {
		return err
	}
	attempt, err = q.LockNativeRetirementAttempt(ctx, attempt.ID)
	if err != nil {
		return err
	}
	nextAttemptState := result.State
	if nextAttemptState != "succeeded" && nextAttemptState != "retryable_failure" {
		nextAttemptState = "quarantined"
	}
	nextAt := pgtype.Timestamptz{}
	if nextAttemptState == "retryable_failure" && attempt.AttemptNumber < maxNativeRetirementAttempts {
		nextAt = pgtype.Timestamptz{
			Time: now.Add(nativeRetirementBackoff[attempt.AttemptNumber]), Valid: true,
		}
	} else if nextAttemptState == "retryable_failure" {
		nextAttemptState = "quarantined"
	}
	advanced, err := q.AdvanceNativeRetirementAttempt(ctx, dbgen.AdvanceNativeRetirementAttemptParams{
		NextState: nextAttemptState, SchedulerOwner: owner,
		ChannelBindingDigest: nativeText(result.ChannelBindingDigest),
		RequestBodyDigest:    nativeText(result.RequestBodyDigest),
		ResultCode:           nativeText(result.ResultCode), NextAttemptAt: nextAt,
		ID: attempt.ID, ExpectedState: attempt.State,
		ExpectedStateVersion: attempt.StateVersion,
	})
	if err != nil {
		return err
	}
	if err := q.RecordNativeRetirementAttemptEvent(ctx, dbgen.RecordNativeRetirementAttemptEventParams{
		AttemptID: advanced.ID, StateVersion: advanced.StateVersion,
		State: advanced.State, TransitionRequestID: nativeEventID(),
		ReasonCode: result.ResultCode,
	}); err != nil {
		return err
	}
	nextOperationState := operation.State
	updateOperationSchedule := nextAttemptState == "retryable_failure"
	switch nextAttemptState {
	case "succeeded":
		expectedLifetimeState := "retiring"
		assignmentID := operation.CurrentAssignmentID
		if operation.State == "committed_retirement_pending" {
			nextOperationState = "committed_retired"
		} else if operation.State == "aborted_candidate_retirement_pending" {
			nextOperationState = "aborted_candidate_retired"
			expectedLifetimeState = "recovery_pending"
			assignmentID = operation.TargetAssignmentID
		} else {
			return ErrNativeCASConflict
		}
		lifetime, lifetimeErr := q.AdvanceNativeHomeLifetime(ctx, dbgen.AdvanceNativeHomeLifetimeParams{
			NextState: "released", ID: attempt.LifetimeID,
			ExpectedState:        expectedLifetimeState,
			ExpectedStateVersion: lockedLifetime.StateVersion,
		})
		if lifetimeErr != nil {
			return lifetimeErr
		}
		if err := recordLifetimeEvent(ctx, q, lifetime, "retirement_proven"); err != nil {
			return err
		}
		if _, err := q.ReleaseNativeRotationAssignment(ctx, dbgen.ReleaseNativeRotationAssignmentParams{
			ReasonCode: nativeText("retirement_proven"), ID: assignmentID,
		}); err != nil {
			return err
		}
	case "quarantined":
		nextOperationState = "quarantined"
	}
	if nextOperationState != operation.State || updateOperationSchedule {
		operation, err = q.AdvanceNativeRotationOperation(ctx, dbgen.AdvanceNativeRotationOperationParams{
			NextState: nextOperationState, ReasonCode: result.ResultCode,
			ID: operation.ID, ExpectedState: operation.State,
			ExpectedStateVersion: operation.StateVersion,
			NextAttemptAt:        nextAt,
		})
		if err != nil {
			return err
		}
		if err := recordOperationEvent(ctx, q, operation); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func lockNativeRetirementChain(
	ctx context.Context,
	q *dbgen.Queries,
	epoch dbgen.RuntimeTaskHomeEpoch,
	operation dbgen.NativeRotationOperation,
) (dbgen.NativeRotationOperation, dbgen.RuntimeHomeLifetime, error) {
	if _, err := q.LockNativeRotationTask(ctx, dbgen.LockNativeRotationTaskParams{
		TaskID: epoch.TaskID, AgentID: epoch.AgentID, RuntimeID: epoch.RuntimeID,
	}); err != nil {
		return dbgen.NativeRotationOperation{}, dbgen.RuntimeHomeLifetime{}, err
	}
	if _, err := q.LockNativeRotationBinding(ctx, dbgen.LockNativeRotationBindingParams{
		RuntimeBindingID: epoch.RuntimeBindingID, WorkspaceID: epoch.WorkspaceID,
		RuntimeSessionID: epoch.RuntimeSessionID, RuntimeID: epoch.RuntimeID,
		AgentID: epoch.AgentID, BindingGeneration: epoch.BindingGeneration,
	}); err != nil {
		return dbgen.NativeRotationOperation{}, dbgen.RuntimeHomeLifetime{}, err
	}
	if _, err := q.LockNativeRotationAssignments(ctx, []pgtype.UUID{epoch.HomeAssignmentID}); err != nil {
		return dbgen.NativeRotationOperation{}, dbgen.RuntimeHomeLifetime{}, err
	}
	if _, err := q.LockNativeRotationCatalogs(ctx, []pgtype.UUID{epoch.CatalogID}); err != nil {
		return dbgen.NativeRotationOperation{}, dbgen.RuntimeHomeLifetime{}, err
	}
	if _, err := q.LockNativeRotationEntries(ctx, []pgtype.UUID{epoch.CatalogEntryID}); err != nil {
		return dbgen.NativeRotationOperation{}, dbgen.RuntimeHomeLifetime{}, err
	}
	if _, err := q.LockNativeRotationLifecycle(ctx, dbgen.LockNativeRotationLifecycleParams{
		CatalogIds: []pgtype.UUID{epoch.CatalogID},
		HomeRefs:   []pgtype.UUID{epoch.HomeRef},
	}); err != nil {
		return dbgen.NativeRotationOperation{}, dbgen.RuntimeHomeLifetime{}, err
	}
	if _, err := q.LockNativeRotationEpochs(ctx, dbgen.LockNativeRotationEpochsParams{
		TaskID: epoch.TaskID, HomeEpochs: []int64{epoch.HomeEpoch},
	}); err != nil {
		return dbgen.NativeRotationOperation{}, dbgen.RuntimeHomeLifetime{}, err
	}
	lifetimes, err := q.LockNativeRotationLifetimes(ctx, []pgtype.UUID{epoch.LifetimeID})
	if err != nil || len(lifetimes) != 1 {
		return dbgen.NativeRotationOperation{}, dbgen.RuntimeHomeLifetime{}, ErrNativeCASConflict
	}
	lockedOperation, err := q.LockNativeRotationOperation(ctx, operation.ID)
	if err != nil {
		return dbgen.NativeRotationOperation{}, dbgen.RuntimeHomeLifetime{}, err
	}
	return lockedOperation, lifetimes[0], nil
}

func identityFromEpoch(e dbgen.RuntimeTaskHomeEpoch) NativeHomeIdentityV1 {
	return NativeHomeIdentityV1{
		TaskID: uuidString(e.TaskID), WorkspaceID: uuidString(e.WorkspaceID),
		AgentID: uuidString(e.AgentID), RuntimeID: uuidString(e.RuntimeID),
		RuntimeSessionID: uuidString(e.RuntimeSessionID), DaemonID: e.DaemonID,
		DaemonBootID: uuidString(e.DaemonBootID), Provider: e.Provider,
		RuntimeBindingID: uuidString(e.RuntimeBindingID), BindingGeneration: e.BindingGeneration,
		HomeAssignmentID: uuidString(e.HomeAssignmentID), CatalogID: uuidString(e.CatalogID),
		CatalogEntryID: uuidString(e.CatalogEntryID), CatalogGeneration: e.CatalogGeneration,
		HomeRef: uuidString(e.HomeRef), HomeEpoch: e.HomeEpoch,
		LifetimeID:           uuidString(e.LifetimeID),
		AcquisitionRequestID: uuidString(e.AcquisitionRequestID),
	}
}

func uuidString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	value, _ := uuid.FromBytes(id.Bytes[:])
	return value.String()
}
