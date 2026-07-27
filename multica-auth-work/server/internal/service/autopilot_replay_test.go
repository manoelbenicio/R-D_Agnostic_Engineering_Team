package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/daemon/commitledger"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

func TestAutopilotReplayGate_NilHookFailsClosed(t *testing.T) {
	svc := &AutopilotService{
		ReplayGateHook: nil,
	}

	apID := uuid.New()
	ap := db.Autopilot{
		ID:           pgtype.UUID{Bytes: apID, Valid: true},
		AssigneeType: "agent",
		AssigneeID:   pgtype.UUID{Bytes: apID, Valid: true},
	}
	agent := db.Agent{
		ID: pgtype.UUID{Bytes: apID, Valid: true},
	}

	err := svc.checkAutopilotReplayGate(context.Background(), ap, agent)
	if err == nil {
		t.Fatal("expected nil ReplayGateHook to return error (fail-closed), got nil")
	}

	if !errors.Is(err, commitledger.ErrReplayBlocked) {
		t.Errorf("expected error to wrap ErrReplayBlocked, got: %v", err)
	}
}

func TestAutopilotReplayGate_AllowedWhenNoToolUse(t *testing.T) {
	registry := commitledger.NewLedgerRegistry()
	hook := commitledger.NewReplayGateHook(registry, slog.Default())

	svc := &AutopilotService{
		ReplayGateHook: hook,
	}

	apID := uuid.New()
	apIDStr := apID.String()
	ledger, err := commitledger.New(commitledger.Config{
		TaskID:     apIDStr,
		HMACSecret: []byte("01234567890123456789012345678901"),
	})
	if err != nil {
		t.Fatalf("failed to create test ledger: %v", err)
	}
	registry.Register(apIDStr, ledger)

	ap := db.Autopilot{
		ID: pgtype.UUID{Bytes: apID, Valid: true},
	}
	agent := db.Agent{
		ID: pgtype.UUID{Bytes: apID, Valid: true},
	}

	err = svc.checkAutopilotReplayGate(context.Background(), ap, agent)
	if err != nil {
		t.Fatalf("expected allowed replay gate check for clean state, got: %v", err)
	}
}

func TestAutopilotReplayGate_BlockedWhenToolUseRecorded(t *testing.T) {
	registry := commitledger.NewLedgerRegistry()
	hook := commitledger.NewReplayGateHook(registry, slog.Default())

	apID := uuid.New()
	apIDStr := apID.String()

	// Register a fail-closed or tool-active ledger for this correlation ID
	ledger := commitledger.NewFailClosed(apIDStr)
	registry.Register(apIDStr, ledger)

	svc := &AutopilotService{
		ReplayGateHook: hook,
	}

	ap := db.Autopilot{
		ID: pgtype.UUID{Bytes: apID, Valid: true},
	}
	agent := db.Agent{
		ID: pgtype.UUID{Bytes: apID, Valid: true},
	}

	err := svc.checkAutopilotReplayGate(context.Background(), ap, agent)
	if err == nil {
		t.Fatal("expected replay gate error for fail-closed ledger, got nil")
	}

	if !errors.Is(err, commitledger.ErrReplayBlocked) {
		t.Errorf("expected error to wrap ErrReplayBlocked, got: %v", err)
	}
}
