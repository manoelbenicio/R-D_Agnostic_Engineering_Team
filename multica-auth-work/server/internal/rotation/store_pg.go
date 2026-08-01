package rotation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAccountNotFound = errors.New("rotation: account not found")
	ErrNoAssignment    = errors.New("rotation: no current assignment")
)

type PGStore struct {
	pool *pgxpool.Pool
}

var _ Store = (*PGStore)(nil)

func NewPGStore(pool *pgxpool.Pool) *PGStore {
	return &PGStore{pool: pool}
}

const accountColumns = `account_id, vendor, tenant_id, priority, home_dir, config_dir, status,
	tokens_per_window, tokens_used, window_start, cooldown_until, last_error`

func scanAccount(row pgx.Row) (Account, error) {
	var account Account
	if err := row.Scan(
		&account.AccountID,
		&account.Vendor,
		&account.TenantID,
		&account.Priority,
		&account.HomeDir,
		&account.ConfigDir,
		&account.Status,
		&account.TokensPerWin,
		&account.TokensUsed,
		&account.WindowStart,
		&account.CooldownUntil,
		&account.LastError,
	); err != nil {
		return Account{}, err
	}
	return account, nil
}

func (s *PGStore) ListAccounts(ctx context.Context, vendor, tenantID string) ([]Account, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+accountColumns+`
		  FROM accounts
		 WHERE vendor = $1 AND tenant_id = $2
		 ORDER BY priority ASC, created_at ASC, account_id ASC
	`, vendor, tenantID)
	if err != nil {
		return nil, fmt.Errorf("rotation: list accounts: %w", err)
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		account, err := scanAccount(rows)
		if err != nil {
			return nil, fmt.Errorf("rotation: scan account: %w", err)
		}
		accounts = append(accounts, account)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rotation: iterate accounts: %w", err)
	}
	return accounts, nil
}

func (s *PGStore) GetAccount(ctx context.Context, accountID string) (Account, error) {
	account, err := scanAccount(s.pool.QueryRow(ctx, `
		SELECT `+accountColumns+`
		  FROM accounts
		 WHERE account_id = $1
	`, accountID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Account{}, ErrAccountNotFound
		}
		return Account{}, fmt.Errorf("rotation: get account: %w", err)
	}
	return account, nil
}

func (s *PGStore) UpdateAccountStatus(ctx context.Context, accountID string, status AccountStatus, cooldownUntil *time.Time) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE accounts
		   SET status = $2,
		       cooldown_until = $3,
		       updated_at = now()
		 WHERE account_id = $1
	`, accountID, string(status), cooldownUntil)
	if err != nil {
		return fmt.Errorf("rotation: update account status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAccountNotFound
	}
	return nil
}

func (s *PGStore) RecordUsage(ctx context.Context, accountID string, tokensUsed int64, windowStart time.Time) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE accounts
		   SET tokens_used = $2,
		       window_start = $3,
		       updated_at = now()
		 WHERE account_id = $1
	`, accountID, tokensUsed, windowStart)
	if err != nil {
		return fmt.Errorf("rotation: record usage: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAccountNotFound
	}
	return nil
}

func (s *PGStore) Assign(ctx context.Context, agentID, accountID string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO assignments (agent_id, account_id, assigned_at)
		VALUES ($1, $2, now())
		ON CONFLICT (agent_id) DO UPDATE
		   SET account_id = EXCLUDED.account_id,
		       assigned_at = EXCLUDED.assigned_at
	`, agentID, accountID)
	if err != nil {
		return fmt.Errorf("rotation: assign: %w", err)
	}
	return nil
}

func (s *PGStore) CurrentAssignment(ctx context.Context, agentID string) (string, error) {
	var accountID string
	err := s.pool.QueryRow(ctx, `
		SELECT account_id
		  FROM assignments
		 WHERE agent_id = $1
	`, agentID).Scan(&accountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNoAssignment
		}
		return "", fmt.Errorf("rotation: current assignment: %w", err)
	}
	return accountID, nil
}

func (s *PGStore) RecordRotation(ctx context.Context, agentID, fromAccountID, toAccountID string, reason RotationReason, at time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO rotation_events (agent_id, from_account_id, to_account_id, reason, at)
		VALUES ($1, $2, $3, $4, $5)
	`, agentID, nullableUUID(fromAccountID), nullableUUID(toAccountID), string(reason), at)
	if err != nil {
		return fmt.Errorf("rotation: record rotation: %w", err)
	}
	return nil
}

func nullableUUID(value string) any {
	if value == "" {
		return nil
	}
	return value
}
