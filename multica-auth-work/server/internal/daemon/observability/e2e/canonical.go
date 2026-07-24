package e2e

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// Canonical correlation derivation shared by BOTH processes (server + daemon).
// Because request_id and session_id are DETERMINISTICALLY derived from the raw
// task id (and chat session id), every hop that knows the task id computes the
// IDENTICAL join keys independently — no cross-process propagation required, so
// ingress↔route join on request_id and admission↔delivery join on session_id.

// CanonicalTaskID is the raw task UUID, used identically by queue, admission,
// persist, and every other hop.
func CanonicalTaskID(taskUUID string) string {
	return strings.TrimSpace(taskUUID)
}

// CanonicalRequestID is derived deterministically from the task id so the
// server ingress hop and the daemon-injected OTLP trusted request id are the
// SAME value. Empty task id yields empty (caller fails closed).
func CanonicalRequestID(taskID string) string {
	t := strings.TrimSpace(taskID)
	if t == "" {
		return ""
	}
	return "req-" + canonicalDigest(t)
}

// CanonicalSessionID is derived deterministically from chat_session_id when
// present, else the task id, so daemon admission and realtime terminal delivery
// derive the IDENTICAL session id. Empty inputs yield empty.
func CanonicalSessionID(chatSessionID, taskID string) string {
	seed := strings.TrimSpace(chatSessionID)
	if seed == "" {
		seed = strings.TrimSpace(taskID)
	}
	if seed == "" {
		return ""
	}
	return "sess-" + canonicalDigest(seed)
}

func canonicalDigest(v string) string {
	d := sha256.Sum256([]byte(v))
	return hex.EncodeToString(d[:16])
}
