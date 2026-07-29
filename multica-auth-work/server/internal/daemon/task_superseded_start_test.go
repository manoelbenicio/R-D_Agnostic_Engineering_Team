package daemon

import (
	"errors"
	"fmt"
	"testing"
)

// TestIsTaskSupersededStartError guards the rerun/max_concurrent_tasks=1 race
// hardening: a StartTask failure caused by the server's StartAgentTask matching
// no runnable row (task superseded/cancelled between claim and start) MUST be
// classified as superseded so runTask treats it as a benign cancellation rather
// than an execution failure that would trip the agent-brain gateway circuit.
func TestIsTaskSupersededStartError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"pgx no rows (current backend 400 body)",
			fmt.Errorf("POST /api/daemon/tasks/x/start returned 400: {\"error\":\"start task: no rows in result set\"}"), true},
		{"explicit superseded (future 409 contract)",
			errors.New("task not startable: superseded or cancelled"), true},
		{"not startable phrasing", errors.New("task not startable"), true},
		{"unrelated transient error", errors.New("connection refused"), false},
		{"unrelated 500", errors.New("returned 500: internal error"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isTaskSupersededStartError(c.err); got != c.want {
				t.Fatalf("isTaskSupersededStartError(%v) = %v, want %v", c.err, got, c.want)
			}
		})
	}
}
