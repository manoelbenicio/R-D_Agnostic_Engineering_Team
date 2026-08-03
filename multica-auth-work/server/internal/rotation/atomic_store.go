package rotation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
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
