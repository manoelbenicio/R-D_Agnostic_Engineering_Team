package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/multica-ai/multica/server/internal/rotation"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

const (
	defaultCredentialDiscoveryDedupLimit = 1024
	defaultCredentialDiscoveryDedupTTL   = 5 * time.Minute
	maxCredentialDiscoveryIDLength       = 128
	maxCredentialDiscoveryProviderLength = 64
	maxCredentialDiscoveryStatusLength   = 32
	maxCredentialDiscoveryExpiryLength   = 128
)

var (
	errInvalidCredentialDiscoveryObservation = errors.New("daemon: invalid credential session discovery observation")
	errCredentialDiscoveryProducerCapacity   = errors.New("daemon: credential session discovery producer capacity exhausted")
)

// CredentialSessionDiscoveryObservation is the non-secret active-assignment
// metadata accepted from session discovery. It intentionally has no credential,
// token, config-home, or arbitrary metadata fields.
type CredentialSessionDiscoveryObservation struct {
	AgentID     string
	AccountID   string
	Provider    string
	WorkspaceID string
	Status      string
	ExpiresAt   string
}

// CredentialSessionDiscoveryEmitter is the transport boundary used by the
// producer. Implementations may publish to the daemon event channel or dispatch
// locally; tests inject an in-memory loopback.
type CredentialSessionDiscoveryEmitter interface {
	EmitCredentialSessionDiscovery(context.Context, protocol.Message) error
}

type credentialDiscoveryDedupKey struct {
	agentID     string
	accountID   string
	provider    string
	workspaceID string
}

type credentialDiscoveryDedupEntry struct {
	sequence  uint64
	expiresAt time.Time
	inFlight  bool
}

// CredentialSessionDiscoveryProducer classifies active-session observations
// and emits only expired/exhausted events. Its dedup state is process-local,
// concurrency-safe, time-bounded, and size-bounded.
type CredentialSessionDiscoveryProducer struct {
	emitter CredentialSessionDiscoveryEmitter
	limit   int
	ttl     time.Duration

	mu       sync.Mutex
	sequence uint64
	seen     map[credentialDiscoveryDedupKey]credentialDiscoveryDedupEntry
}

func NewCredentialSessionDiscoveryProducer(emitter CredentialSessionDiscoveryEmitter) *CredentialSessionDiscoveryProducer {
	return newCredentialSessionDiscoveryProducer(emitter, defaultCredentialDiscoveryDedupLimit, defaultCredentialDiscoveryDedupTTL)
}

func newCredentialSessionDiscoveryProducer(emitter CredentialSessionDiscoveryEmitter, limit int, ttl time.Duration) *CredentialSessionDiscoveryProducer {
	if limit <= 0 {
		limit = defaultCredentialDiscoveryDedupLimit
	}
	if ttl <= 0 {
		ttl = defaultCredentialDiscoveryDedupTTL
	}
	return &CredentialSessionDiscoveryProducer{
		emitter: emitter,
		limit:   limit,
		ttl:     ttl,
		seen:    make(map[credentialDiscoveryDedupKey]credentialDiscoveryDedupEntry),
	}
}

// Produce classifies one active-session observation and emits the exact
// provider/workspace/account identity only when 4.1 reports exhaustion. It
// returns emitted=false for usable sessions and duplicate exhausted events.
func (p *CredentialSessionDiscoveryProducer) Produce(ctx context.Context, observation CredentialSessionDiscoveryObservation, now time.Time) (bool, error) {
	if p == nil || p.emitter == nil {
		return false, errInvalidCredentialDiscoveryObservation
	}
	observation, err := validateCredentialDiscoveryObservation(observation)
	if err != nil {
		return false, err
	}

	key := credentialDiscoveryDedupKey{
		agentID:     observation.AgentID,
		accountID:   observation.AccountID,
		provider:    observation.Provider,
		workspaceID: observation.WorkspaceID,
	}
	session := rotation.DiscoverySession{
		Provider:  observation.Provider,
		Status:    observation.Status,
		ExpiresAt: observation.ExpiresAt,
	}
	if !rotation.DetectDiscoverySession(observation.Provider, session, now).Exhausted {
		p.clearDedup(key)
		return false, nil
	}

	sequence, reserved, err := p.reserveDedup(key, now)
	if err != nil || !reserved {
		return false, err
	}
	payload, err := json.Marshal(credentialSessionDiscoveryPayload{
		AgentID:     observation.AgentID,
		AccountID:   observation.AccountID,
		Provider:    observation.Provider,
		WorkspaceID: observation.WorkspaceID,
		Status:      observation.Status,
		ExpiresAt:   observation.ExpiresAt,
	})
	if err != nil {
		p.releaseDedup(key, sequence)
		return false, err
	}
	err = p.emitter.EmitCredentialSessionDiscovery(ctx, protocol.Message{
		Type:    eventDaemonCredentialSessionDiscovery,
		Payload: payload,
	})
	if err != nil {
		p.releaseDedup(key, sequence)
		return false, err
	}
	p.completeDedup(key, sequence, now)
	return true, nil
}

func validateCredentialDiscoveryObservation(observation CredentialSessionDiscoveryObservation) (CredentialSessionDiscoveryObservation, error) {
	fields := []struct {
		name     string
		value    string
		required bool
		max      int
	}{
		{name: "agent_id", value: observation.AgentID, required: true, max: maxCredentialDiscoveryIDLength},
		{name: "account_id", value: observation.AccountID, required: true, max: maxCredentialDiscoveryIDLength},
		{name: "provider", value: observation.Provider, required: true, max: maxCredentialDiscoveryProviderLength},
		{name: "workspace_id", value: observation.WorkspaceID, required: true, max: maxCredentialDiscoveryIDLength},
		{name: "status", value: observation.Status, max: maxCredentialDiscoveryStatusLength},
		{name: "expires_at", value: observation.ExpiresAt, max: maxCredentialDiscoveryExpiryLength},
	}
	for _, field := range fields {
		trimmed := strings.TrimSpace(field.value)
		if field.value != trimmed || (field.required && trimmed == "") || len(trimmed) > field.max {
			return CredentialSessionDiscoveryObservation{}, fmt.Errorf("%w: %s", errInvalidCredentialDiscoveryObservation, field.name)
		}
	}
	if observation.Status == "" && observation.ExpiresAt == "" {
		return CredentialSessionDiscoveryObservation{}, fmt.Errorf("%w: status/expires_at", errInvalidCredentialDiscoveryObservation)
	}
	return observation, nil
}

func (p *CredentialSessionDiscoveryProducer) reserveDedup(key credentialDiscoveryDedupKey, now time.Time) (uint64, bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pruneExpiredLocked(now)
	if _, exists := p.seen[key]; exists {
		return 0, false, nil
	}
	if len(p.seen) >= p.limit && !p.evictOldestCompletedLocked() {
		return 0, false, errCredentialDiscoveryProducerCapacity
	}
	p.sequence++
	p.seen[key] = credentialDiscoveryDedupEntry{sequence: p.sequence, inFlight: true}
	return p.sequence, true, nil
}

func (p *CredentialSessionDiscoveryProducer) completeDedup(key credentialDiscoveryDedupKey, sequence uint64, now time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	entry, exists := p.seen[key]
	if !exists || entry.sequence != sequence {
		return
	}
	entry.inFlight = false
	entry.expiresAt = now.Add(p.ttl)
	p.seen[key] = entry
}

func (p *CredentialSessionDiscoveryProducer) releaseDedup(key credentialDiscoveryDedupKey, sequence uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if entry, exists := p.seen[key]; exists && entry.sequence == sequence {
		delete(p.seen, key)
	}
}

func (p *CredentialSessionDiscoveryProducer) clearDedup(key credentialDiscoveryDedupKey) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.seen, key)
}

func (p *CredentialSessionDiscoveryProducer) pruneExpiredLocked(now time.Time) {
	for key, entry := range p.seen {
		if !entry.inFlight && !now.Before(entry.expiresAt) {
			delete(p.seen, key)
		}
	}
}

func (p *CredentialSessionDiscoveryProducer) evictOldestCompletedLocked() bool {
	var oldestKey credentialDiscoveryDedupKey
	var oldest credentialDiscoveryDedupEntry
	found := false
	for key, entry := range p.seen {
		if entry.inFlight || (found && entry.sequence >= oldest.sequence) {
			continue
		}
		oldestKey = key
		oldest = entry
		found = true
	}
	if found {
		delete(p.seen, oldestKey)
	}
	return found
}
