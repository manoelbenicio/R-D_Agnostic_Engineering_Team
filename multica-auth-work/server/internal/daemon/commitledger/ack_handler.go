package commitledger

import (
	"log/slog"
	"sync"
)

// AckHandler processes persisted_through_seq acknowledgements from the
// server and advances the corresponding ledger output watermarks.
// The flow: server (handler) → HTTP response → client → AckHandler → ledger.
//
// This is distinct from tool commit state:
//   - Tool commit (Definite): set when tool_result is received (immediate)
//   - Output persistence: set when server acks the batch (async watermark)
type AckHandler struct {
	mu       sync.RWMutex
	registry *LedgerRegistry
	logger   *slog.Logger
}

// NewAckHandler creates an acknowledgement handler backed by the registry.
func NewAckHandler(registry *LedgerRegistry, logger *slog.Logger) *AckHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &AckHandler{
		registry: registry,
		logger:   logger,
	}
}

// ProcessOutputAck handles a persisted_through_seq acknowledgement for output.
// This is called when the server responds to ReportTaskMessages with the
// highest seq it has durably persisted.
//
// Gap safety: if newSeq < ledger's current watermark, it's a no-op.
// If the correlation has no active ledger, the ack is silently discarded.
func (h *AckHandler) ProcessOutputAck(correlationID string, persistedThroughSeq int64) {
	if persistedThroughSeq <= 0 {
		return
	}
	ledger := h.registry.Get(correlationID)
	if ledger == nil {
		return
	}
	ledger.AcknowledgeOutputPersisted(persistedThroughSeq)
}

// ReportTaskMessagesResponse is the response from the server's
// ReportTaskMessages endpoint, extended with the persisted_through_seq field.
type ReportTaskMessagesResponse struct {
	PersistedThroughSeq int64 `json:"persisted_through_seq"`
}
