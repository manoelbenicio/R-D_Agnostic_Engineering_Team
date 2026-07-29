package e2e

import "testing"

func TestCanonicalDerivationsAreDeterministicAndIdentical(t *testing.T) {
	task := "11111111-2222-3333-4444-555555555555"
	// request_id: server and daemon independently derive the SAME value.
	if CanonicalRequestID(task) != CanonicalRequestID(task) {
		t.Fatal("request_id derivation must be deterministic")
	}
	if CanonicalRequestID(task) == "" {
		t.Fatal("request_id must be non-empty for a real task")
	}
	if CanonicalRequestID("t-a") == CanonicalRequestID("t-b") {
		t.Fatal("distinct tasks must yield distinct request_ids")
	}
	// task_id is the raw UUID, identical everywhere.
	if CanonicalTaskID(task) != task {
		t.Fatal("task_id must be the raw UUID")
	}
	// session_id: chat present -> from chat; absent -> from task; both deterministic.
	chat := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	if CanonicalSessionID(chat, task) != CanonicalSessionID(chat, task) {
		t.Fatal("session_id derivation must be deterministic")
	}
	if CanonicalSessionID("", task) != CanonicalSessionID("", task) {
		t.Fatal("session_id from task must be deterministic")
	}
	if CanonicalSessionID(chat, task) == CanonicalSessionID("", task) {
		t.Fatal("chat-derived and task-derived session_ids must differ")
	}
	// Charset: derived ids must be safe correlation values (no comma/equals/etc).
	for _, id := range []string{CanonicalRequestID(task), CanonicalSessionID(chat, task), CanonicalSessionID("", task)} {
		for _, r := range id {
			ok := r == '-' || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
			if !ok {
				t.Fatalf("derived id %q contains unsafe char %q", id, r)
			}
		}
	}
}
