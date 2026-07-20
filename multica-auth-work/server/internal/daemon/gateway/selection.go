package gateway

import (
	"sort"
	"sync"
)

// ErrorNoEligibleAccount is the deterministic fail-closed class returned when a
// route has no eligible account for a new selection. It is a routing decision,
// never a leak of account identity or credential material.
const ErrorNoEligibleAccount ErrorClass = "no_eligible_account"

// AccountStatus is the eligibility state of a pooled account for selection.
// Only AccountEligible accounts are selectable; the remaining states model the
// quarantine / cooldown / removal lifecycle without exposing why an account
// left rotation.
type AccountStatus uint8

const (
	AccountEligible AccountStatus = iota
	AccountQuarantined
	AccountCooldown
	AccountRemoved
)

// ContinuationRefs carries the stateful references that force continuation
// affinity. Precedence is fixed and deterministic: a provider conversation
// (previous_response_id) binds before a prompt cache, which binds before a
// tool-turn. An empty ContinuationRefs describes an independent request that
// participates in strict round-robin rotation.
type ContinuationRefs struct {
	PreviousResponseID string
	PromptCacheID      string
	ToolTurnID         string
}

// affinity resolves the ordered (reason, key) pair for a request. A blank key
// means the request is independent and must rotate.
func (r ContinuationRefs) affinity() (SelectionReason, string) {
	switch {
	case r.PreviousResponseID != "":
		return SelectionContinuation, "previous_response_id:" + r.PreviousResponseID
	case r.PromptCacheID != "":
		return SelectionPromptCache, "prompt_cache:" + r.PromptCacheID
	case r.ToolTurnID != "":
		return SelectionToolTurn, "tool_turn:" + r.ToolTurnID
	default:
		return SelectionIndependentRotation, ""
	}
}

// Selection is the atomic outcome of a Selector.Select call. Sequence is a
// per-Selector monotonic counter that lets callers prove the global rotation
// order is independent of goroutine scheduling.
type Selection struct {
	Sequence uint64
	Account  string
	Reason   SelectionReason
}

type affinityBinding struct {
	account string
	reason  SelectionReason
}

// Selector implements concurrency-safe account selection for a single route.
//
// For RotationStrictIndependentRequest every independent logical request
// atomically advances one shared cursor to the next eligible account, so the
// rotation order is a single sequence regardless of how many requests race.
// Continuation references (previous_response_id, prompt cache, tool turn) pin a
// request to the account that first served the reference and do NOT advance the
// cursor, so dependent continuations never perturb independent rotation. If a
// bound account has left rotation the continuation is re-bound to the next
// eligible account (a documented stateless-continuation fallback) and the stale
// owner does not reclaim it.
//
// For RotationFailureOnly independent requests stick to the current cursor
// account until it becomes ineligible, at which point the cursor advances.
type Selector struct {
	rotation RotationMode

	mu       sync.Mutex
	order    []string
	status   map[string]AccountStatus
	cursor   int
	sequence uint64
	affinity map[string]affinityBinding
}

// NewSelector builds a Selector for the given rotation mode and initial pool.
// Duplicate or blank account identifiers are rejected so the rotation order is
// unambiguous.
func NewSelector(rotation RotationMode, accounts ...string) (*Selector, error) {
	if rotation != RotationStrictIndependentRequest && rotation != RotationFailureOnly {
		return nil, &GatewayError{Operation: "selector", Class: ErrorInvalidConfiguration}
	}
	status := make(map[string]AccountStatus, len(accounts))
	order := make([]string, 0, len(accounts))
	for _, account := range accounts {
		if account == "" {
			return nil, &GatewayError{Operation: "selector", Class: ErrorInvalidConfiguration}
		}
		if _, exists := status[account]; exists {
			return nil, &GatewayError{Operation: "selector", Class: ErrorInvalidConfiguration}
		}
		status[account] = AccountEligible
		order = append(order, account)
	}
	return &Selector{
		rotation: rotation,
		order:    order,
		status:   status,
		affinity: make(map[string]affinityBinding),
	}, nil
}

// Select returns the next account for a request atomically. Independent
// requests rotate; continuation requests honor affinity. It fails closed with
// ErrorNoEligibleAccount when the pool has no eligible account.
func (s *Selector) Select(refs ContinuationRefs) (Selection, error) {
	reason, key := refs.affinity()

	s.mu.Lock()
	defer s.mu.Unlock()

	if key != "" {
		if binding, exists := s.affinity[key]; exists && s.eligibleLocked(binding.account) {
			// Affinity hit: pin to the owning account without advancing the
			// shared cursor so independent rotation is untouched.
			s.sequence++
			return Selection{Sequence: s.sequence, Account: binding.account, Reason: binding.reason}, nil
		}
	}

	account, ok := s.nextEligibleLocked()
	if !ok {
		return Selection{}, &GatewayError{Operation: "selector.select", Class: ErrorNoEligibleAccount}
	}
	s.sequence++
	if key != "" {
		s.affinity[key] = affinityBinding{account: account, reason: reason}
	}
	return Selection{Sequence: s.sequence, Account: account, Reason: reason}, nil
}

// nextEligibleLocked advances the cursor to the next eligible account. For
// failure-only rotation it stays on the current cursor account while it is
// eligible; for strict round-robin it always advances past the returned
// account. Callers must hold s.mu.
func (s *Selector) nextEligibleLocked() (string, bool) {
	if len(s.order) == 0 {
		return "", false
	}
	if s.rotation == RotationFailureOnly {
		current := s.order[s.cursor%len(s.order)]
		if s.eligibleLocked(current) {
			return current, true
		}
	}
	for offset := 0; offset < len(s.order); offset++ {
		index := (s.cursor + offset) % len(s.order)
		account := s.order[index]
		if s.status[account] != AccountEligible {
			continue
		}
		s.cursor = (index + 1) % len(s.order)
		return account, true
	}
	return "", false
}

func (s *Selector) eligibleLocked(account string) bool {
	return s.status[account] == AccountEligible
}

// SetStatus updates the eligibility of an existing account. Unknown accounts
// are ignored so lifecycle callbacks are idempotent.
func (s *Selector) SetStatus(account string, status AccountStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.status[account]; exists {
		s.status[account] = status
	}
}

// Add registers a new account (or re-enables a previously removed one) as
// eligible, appending it to the rotation order the first time it is seen.
func (s *Selector) Add(account string) {
	if account == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.status[account]; !exists {
		s.order = append(s.order, account)
	}
	s.status[account] = AccountEligible
}

// EligibleAccounts returns a sorted snapshot of currently eligible accounts for
// diagnostics and tests. It never mutates selector state.
func (s *Selector) EligibleAccounts() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	eligible := make([]string, 0, len(s.order))
	for _, account := range s.order {
		if s.status[account] == AccountEligible {
			eligible = append(eligible, account)
		}
	}
	sort.Strings(eligible)
	return eligible
}
