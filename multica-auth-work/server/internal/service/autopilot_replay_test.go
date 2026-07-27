package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// --- doubles -----------------------------------------------------------------

// recordingGate captures every correlation ID it is asked about so a test can
// assert *which* identity the service correlates by.
type recordingGate struct {
	seen []string
	err  error
}

func (g *recordingGate) AllowReplay(_ context.Context, correlationID string) error {
	g.seen = append(g.seen, correlationID)
	return g.err
}

type stubCorrelation struct {
	id     string
	ok     bool
	err    error
	called int
}

func (c *stubCorrelation) PriorTaskCorrelationID(_ context.Context, _ db.Autopilot, _ db.Agent) (string, bool, error) {
	c.called++
	return c.id, c.ok, c.err
}

func testAutopilotAndAgent(t *testing.T) (db.Autopilot, db.Agent, string) {
	t.Helper()
	apID := uuid.New()
	agentID := uuid.New()
	ap := db.Autopilot{
		ID:           pgtype.UUID{Bytes: apID, Valid: true},
		AssigneeType: "agent",
		AssigneeID:   pgtype.UUID{Bytes: agentID, Valid: true},
	}
	agent := db.Agent{ID: pgtype.UUID{Bytes: agentID, Valid: true}}
	return ap, agent, apID.String()
}

// --- unwired production behaviour -------------------------------------------

// The production constructor must leave the gate disarmed. If this ever flips,
// every deployment starts exercising an unwired safety gate.
func TestNewAutopilotService_LeavesReplayGateDisarmed(t *testing.T) {
	svc := NewAutopilotService(nil, nil, nil, nil)

	if svc.ReplayGateArmed() {
		t.Fatal("production constructor must leave the replay gate disarmed")
	}
	if svc.replayGate != nil || svc.replayCorrelation != nil {
		t.Fatalf("expected both collaborators nil, got gate=%v correlation=%v", svc.replayGate, svc.replayCorrelation)
	}
}

// This is the regression that the previous implementation would have failed:
// with the gate unwired, admission must not refuse a single dispatch. Built
// through the *production* constructor on purpose — a struct literal would hide
// exactly the wiring defect this guards against.
func TestReplayGateSkipReason_ProductionServiceDoesNotBlockWhenUnwired(t *testing.T) {
	svc := NewAutopilotService(nil, nil, nil, nil)
	ap, agent, _ := testAutopilotAndAgent(t)

	reason, skip := svc.replayGateSkipReason(context.Background(), ap, agent)

	if skip {
		t.Fatalf("unwired replay gate must not skip dispatch; got skip=true reason=%q", reason)
	}
	if reason != "" {
		t.Fatalf("unwired replay gate must not produce an admission reason, got %q", reason)
	}
}

// --- arming contract ---------------------------------------------------------

func TestNewAutopilotServiceWithReplayGate_RejectsPartialWiring(t *testing.T) {
	if _, err := NewAutopilotServiceWithReplayGate(nil, nil, nil, nil, &recordingGate{}, nil); !errors.Is(err, ErrAutopilotReplayGateIncomplete) {
		t.Fatalf("gate without correlation resolver must be rejected, got %v", err)
	}
	if _, err := NewAutopilotServiceWithReplayGate(nil, nil, nil, nil, nil, &stubCorrelation{}); !errors.Is(err, ErrAutopilotReplayGateIncomplete) {
		t.Fatalf("correlation resolver without gate must be rejected, got %v", err)
	}
}

// --- wired-on behaviour ------------------------------------------------------

// The defect this exposes: correlating by autopilot_id. Commit ledgers are
// registered per task, so an autopilot ID is a key that is never present and
// the gate would block forever. The assertion is explicit about both sides —
// the task ID must be used and the autopilot ID must not.
func TestReplayGateSkipReason_CorrelatesByPriorTaskNotAutopilotID(t *testing.T) {
	priorTaskID := uuid.New().String()
	gate := &recordingGate{}
	correlation := &stubCorrelation{id: priorTaskID, ok: true}

	svc, err := NewAutopilotServiceWithReplayGate(nil, nil, nil, nil, gate, correlation)
	if err != nil {
		t.Fatalf("arming the gate failed: %v", err)
	}
	ap, agent, apIDStr := testAutopilotAndAgent(t)

	reason, skip := svc.replayGateSkipReason(context.Background(), ap, agent)

	if skip {
		t.Fatalf("gate allowed replay, dispatch must proceed; got reason=%q", reason)
	}
	if len(gate.seen) != 1 {
		t.Fatalf("expected exactly one gate consultation, got %d", len(gate.seen))
	}
	if gate.seen[0] != priorTaskID {
		t.Errorf("gate must be consulted with the prior task ID %q, got %q", priorTaskID, gate.seen[0])
	}
	if gate.seen[0] == apIDStr {
		t.Errorf("gate must never be consulted with the autopilot ID %q", apIDStr)
	}
}

// First run: there is no prior task, so there is nothing to replay. The gate
// must not even be consulted, otherwise a fail-closed checker would block every
// autopilot's very first dispatch.
func TestReplayGateSkipReason_AllowsFirstRunWithoutConsultingGate(t *testing.T) {
	gate := &recordingGate{err: errors.New("must not be called")}
	correlation := &stubCorrelation{ok: false}

	svc, err := NewAutopilotServiceWithReplayGate(nil, nil, nil, nil, gate, correlation)
	if err != nil {
		t.Fatalf("arming the gate failed: %v", err)
	}
	ap, agent, _ := testAutopilotAndAgent(t)

	reason, skip := svc.replayGateSkipReason(context.Background(), ap, agent)

	if skip {
		t.Fatalf("first run must be allowed, got skip=true reason=%q", reason)
	}
	if len(gate.seen) != 0 {
		t.Fatalf("gate must not be consulted when there is no prior task, saw %v", gate.seen)
	}
	if correlation.called != 1 {
		t.Fatalf("expected one correlation lookup, got %d", correlation.called)
	}
}

func TestReplayGateSkipReason_BlocksWhenGateRefuses(t *testing.T) {
	gate := &recordingGate{err: errors.New("replay blocked")}
	correlation := &stubCorrelation{id: uuid.New().String(), ok: true}

	svc, err := NewAutopilotServiceWithReplayGate(nil, nil, nil, nil, gate, correlation)
	if err != nil {
		t.Fatalf("arming the gate failed: %v", err)
	}
	ap, agent, _ := testAutopilotAndAgent(t)

	reason, skip := svc.replayGateSkipReason(context.Background(), ap, agent)

	if !skip {
		t.Fatal("a refusing gate must skip the dispatch")
	}
	if !strings.Contains(reason, "replay gate blocked") {
		t.Errorf("admission reason must name the replay gate, got %q", reason)
	}
	if strings.Contains(reason, correlation.id) {
		t.Errorf("admission reason must not leak the correlation ID, got %q", reason)
	}
}

// Without an authoritative correlation the service cannot distinguish a first
// run from a replay, so an armed deployment fails closed. This only affects
// deployments that opted in.
func TestReplayGateSkipReason_FailsClosedWhenCorrelationErrors(t *testing.T) {
	gate := &recordingGate{}
	correlation := &stubCorrelation{err: errors.New("lookup failed")}

	svc, err := NewAutopilotServiceWithReplayGate(nil, nil, nil, nil, gate, correlation)
	if err != nil {
		t.Fatalf("arming the gate failed: %v", err)
	}
	ap, agent, _ := testAutopilotAndAgent(t)

	reason, skip := svc.replayGateSkipReason(context.Background(), ap, agent)

	if !skip {
		t.Fatal("an unavailable correlation lookup must fail closed on an armed service")
	}
	if !strings.Contains(reason, "prior task correlation unavailable") {
		t.Errorf("reason must state the correlation was unavailable, got %q", reason)
	}
	if len(gate.seen) != 0 {
		t.Fatalf("gate must not be consulted when correlation failed, saw %v", gate.seen)
	}
}
