package gateway

import (
	"sort"
	"sync"
	"time"
)

// Selection / continuation error classes. They are deterministic routing
// decisions and never leak account identity or credential material.
const (
	// ErrorNoEligibleAccount is returned when a route has no eligible account
	// for a fresh (independent) selection.
	ErrorNoEligibleAccount ErrorClass = "no_eligible_account"
	// ErrorContinuationUnavailable is returned when a stateful continuation
	// cannot be honored: the referenced state has no known owner, or the owner
	// is ineligible, and the route's affinity contract does not permit a
	// stateless rebind. It fails closed rather than silently rerouting stateful
	// continuation to a different account.
	ErrorContinuationUnavailable ErrorClass = "continuation_owner_unavailable"
	// ErrorContinuationCapacity is returned when the bounded binding table is
	// full of still-live entries, so admitting a new binding would require
	// evicting an active continuation. It fails closed instead.
	ErrorContinuationCapacity ErrorClass = "continuation_capacity"
	// ErrorAccountCapacity is returned when adding a new account would exceed
	// the configured account bound. It fails closed rather than growing the
	// rotation order without limit.
	ErrorAccountCapacity ErrorClass = "account_capacity"
)

const (
	defaultMaxBindings = 4096
	defaultBindingTTL  = 15 * time.Minute
	maxBindingTTL      = time.Hour
	defaultMaxAccounts = 256
)

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

// ContinuationRefs identifies stateful continuation. As a request field it
// references state created by an earlier request; as a Bind argument it names
// the handle a successful request produced. Precedence is fixed and
// deterministic: previous_response_id, then prompt cache, then tool turn. A
// zero ContinuationRefs denotes an independent request that rotates.
type ContinuationRefs struct {
	PreviousResponseID string
	PromptCacheID      string
	ToolTurnID         string
}

// affinity resolves the ordered (reason, key) pair for a continuation
// reference. A blank key means the request is independent and must rotate.
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
	account   string
	reason    SelectionReason
	expiresAt time.Time
}

// SelectorConfig configures a Selector. Accounts, Rotation and Affinity are
// required; BindingTTL and MaxBindings default when zero.
type SelectorConfig struct {
	Rotation    RotationMode
	Affinity    AffinityMode
	Accounts    []string
	BindingTTL  time.Duration
	MaxBindings int
	MaxAccounts int

	now func() time.Time // test clock injection; defaults to time.Now
}

// Selector implements concurrency-safe account selection for a single route.
//
// Independent requests (zero ContinuationRefs) rotate: for
// RotationStrictIndependentRequest each atomically advances one shared cursor to
// the next eligible account; for RotationFailureOnly they stick to the current
// eligible account until it leaves rotation. Independent selection NEVER creates
// an affinity binding — a first, continuation-capable turn is just an
// independent request until it succeeds.
//
// Continuation ownership is established explicitly via Bind after a request
// succeeds, keyed by the handle the response produced (previous_response_id,
// prompt cache, tool turn). A later request that References that handle is
// pinned to the owning account (affinity hit) without advancing the cursor, so
// dependent continuations never perturb independent rotation.
//
// When a referenced handle has no known owner, or the owner is ineligible, the
// behavior is governed by the route's AffinityMode:
//   - AffinityOriginAccount (and AffinityNone for stateful refs): fail closed
//     with ErrorContinuationUnavailable. Stateful continuation is never silently
//     rerouted to a different account.
//   - AffinityStatelessMaterialize: rotate to a fresh eligible account and
//     (re)bind, a documented stateless-continuation fallback.
//
// Binding state is bounded: entries expire after BindingTTL, can be released
// explicitly via Unbind, and the table is capped at MaxBindings. When the cap
// is reached only expired entries are reclaimed; a full table of live entries
// fails closed with ErrorContinuationCapacity rather than evicting an active
// continuation.
type Selector struct {
	rotation     RotationMode
	affinityMode AffinityMode
	bindingTTL   time.Duration
	maxBindings  int
	maxAccounts  int
	now          func() time.Time

	mu       sync.Mutex
	order    []string
	status   map[string]AccountStatus
	cursor   int
	sequence uint64
	bindings map[string]affinityBinding
}

// NewSelector builds a strict/failure-only Selector with fail-closed
// origin-account affinity and default binding bounds. It is a convenience over
// NewSelectorFromConfig for the common case.
func NewSelector(rotation RotationMode, accounts ...string) (*Selector, error) {
	return NewSelectorFromConfig(SelectorConfig{
		Rotation: rotation,
		Affinity: AffinityOriginAccount,
		Accounts: accounts,
	})
}

// NewSelectorFromConfig builds a Selector from an explicit configuration.
func NewSelectorFromConfig(cfg SelectorConfig) (*Selector, error) {
	if cfg.Rotation != RotationStrictIndependentRequest && cfg.Rotation != RotationFailureOnly {
		return nil, &GatewayError{Operation: "selector", Class: ErrorInvalidConfiguration}
	}
	if cfg.Affinity != AffinityNone && cfg.Affinity != AffinityOriginAccount && cfg.Affinity != AffinityStatelessMaterialize {
		return nil, &GatewayError{Operation: "selector", Class: ErrorInvalidConfiguration}
	}
	ttl := cfg.BindingTTL
	if ttl == 0 {
		ttl = defaultBindingTTL
	}
	if ttl < 0 || ttl > maxBindingTTL {
		return nil, &GatewayError{Operation: "selector", Class: ErrorInvalidConfiguration}
	}
	maxBindings := cfg.MaxBindings
	if maxBindings == 0 {
		maxBindings = defaultMaxBindings
	}
	if maxBindings < 1 {
		return nil, &GatewayError{Operation: "selector", Class: ErrorInvalidConfiguration}
	}
	maxAccounts := cfg.MaxAccounts
	if maxAccounts == 0 {
		maxAccounts = defaultMaxAccounts
	}
	if maxAccounts < 1 || maxAccounts < len(cfg.Accounts) {
		return nil, &GatewayError{Operation: "selector", Class: ErrorInvalidConfiguration}
	}
	now := cfg.now
	if now == nil {
		now = time.Now
	}
	status := make(map[string]AccountStatus, len(cfg.Accounts))
	order := make([]string, 0, len(cfg.Accounts))
	for _, account := range cfg.Accounts {
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
		rotation:     cfg.Rotation,
		affinityMode: cfg.Affinity,
		bindingTTL:   ttl,
		maxBindings:  maxBindings,
		maxAccounts:  maxAccounts,
		now:          now,
		order:        order,
		status:       status,
		bindings:     make(map[string]affinityBinding),
	}, nil
}

// Select returns the account for a request. Independent requests (zero refs)
// rotate and never bind. Continuation requests are pinned to the owning account
// established by a prior Bind; an unknown or ineligible owner fails closed under
// origin-account affinity, or is materialized under stateless affinity.
func (s *Selector) Select(refs ContinuationRefs) (Selection, error) {
	reason, key := refs.affinity()

	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	s.sweepExpiredLocked(now)

	// A truly independent request rotates and never binds.
	if key == "" {
		return s.rotateLocked(SelectionIndependentRotation)
	}
	// A route without continuation affinity cannot honor a stateful reference;
	// fail closed rather than route the continuation to an arbitrary account
	// (consistent with the origin-account fail-closed path below).
	if s.affinityMode == AffinityNone {
		return Selection{}, &GatewayError{Operation: "selector.select", Class: ErrorContinuationUnavailable}
	}

	binding, exists := s.bindings[key]
	if exists && s.eligibleLocked(binding.account) {
		// Affinity hit: refresh the binding's lifetime so an actively-used
		// continuation is never reclaimed by the expiry sweep, and pin to the
		// owning account without advancing the shared cursor so independent
		// rotation is untouched.
		binding.expiresAt = now.Add(s.bindingTTL)
		s.bindings[key] = binding
		s.sequence++
		return Selection{Sequence: s.sequence, Account: binding.account, Reason: binding.reason}, nil
	}

	// Unknown owner or ineligible owner. Only a route that explicitly declares
	// stateless materialization may rebind; otherwise fail closed so stateful
	// continuation is never silently rerouted.
	if s.affinityMode != AffinityStatelessMaterialize {
		return Selection{}, &GatewayError{Operation: "selector.select", Class: ErrorContinuationUnavailable}
	}
	selection, err := s.rotateLocked(reason)
	if err != nil {
		return Selection{}, err
	}
	if err := s.putBindingLocked(key, selection.Account, reason, now); err != nil {
		return Selection{}, err
	}
	return selection, nil
}

// Bind records that a continuation handle produced by a successful request is
// owned by account, so later requests referencing that handle route back to it.
// It is the only way an affinity binding is created; admission never binds.
func (s *Selector) Bind(handle ContinuationRefs, account string) error {
	reason, key := handle.affinity()
	if key == "" {
		return &GatewayError{Operation: "selector.bind", Class: ErrorInvalidRequest}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, known := s.status[account]; !known {
		return &GatewayError{Operation: "selector.bind", Class: ErrorInvalidRequest}
	}
	if s.affinityMode == AffinityNone {
		// A route without continuation affinity cannot own continuation state;
		// fail closed rather than silently accepting a binding it will not honor.
		return &GatewayError{Operation: "selector.bind", Class: ErrorContinuationUnavailable}
	}
	now := s.now()
	s.sweepExpiredLocked(now)
	return s.putBindingLocked(key, account, reason, now)
}

// Unbind releases the binding for a continuation handle, e.g. when the
// continuation/session completes or is cancelled. Unknown handles are ignored.
func (s *Selector) Unbind(handle ContinuationRefs) {
	_, key := handle.affinity()
	if key == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.bindings, key)
}

// putBindingLocked stores or refreshes a binding under the capacity bound.
// Callers must hold s.mu and must have swept expired entries.
func (s *Selector) putBindingLocked(key, account string, reason SelectionReason, now time.Time) error {
	if _, present := s.bindings[key]; !present && len(s.bindings) >= s.maxBindings {
		// Only live entries remain after the sweep; refuse rather than evict an
		// active continuation.
		return &GatewayError{Operation: "selector.bind", Class: ErrorContinuationCapacity}
	}
	s.bindings[key] = affinityBinding{account: account, reason: reason, expiresAt: now.Add(s.bindingTTL)}
	return nil
}

func (s *Selector) sweepExpiredLocked(now time.Time) {
	for key, binding := range s.bindings {
		if !binding.expiresAt.IsZero() && !now.Before(binding.expiresAt) {
			delete(s.bindings, key)
		}
	}
}

// rotateLocked performs a fresh rotation selection with the given reason.
// Callers must hold s.mu.
func (s *Selector) rotateLocked(reason SelectionReason) (Selection, error) {
	account, ok := s.nextEligibleLocked()
	if !ok {
		return Selection{}, &GatewayError{Operation: "selector.select", Class: ErrorNoEligibleAccount}
	}
	s.sequence++
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

// Add registers a new account, or re-enables a known one, as eligible. A new
// account is appended to the rotation order the first time it is seen and is
// rejected with ErrorAccountCapacity once the configured account bound is
// reached, so the rotation order never grows without limit. Re-enabling a known
// account never grows state.
func (s *Selector) Add(account string) error {
	if account == "" {
		return &GatewayError{Operation: "selector.add", Class: ErrorInvalidRequest}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.status[account]; exists {
		s.status[account] = AccountEligible
		return nil
	}
	if len(s.order) >= s.maxAccounts {
		return &GatewayError{Operation: "selector.add", Class: ErrorAccountCapacity}
	}
	s.order = append(s.order, account)
	s.status[account] = AccountEligible
	return nil
}

// Remove hard-prunes an account from the rotation order and status, adjusts the
// cursor, and drops any continuation bindings it owned, returning account and
// binding state toward bounded/zero after churn. Unknown accounts are ignored.
func (s *Selector) Remove(account string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.status[account]; !exists {
		return
	}
	delete(s.status, account)
	removedIndex := -1
	newOrder := make([]string, 0, len(s.order))
	for i, a := range s.order {
		if a == account {
			removedIndex = i
			continue
		}
		newOrder = append(newOrder, a)
	}
	s.order = newOrder
	switch {
	case len(s.order) == 0:
		s.cursor = 0
	default:
		if removedIndex >= 0 && removedIndex < s.cursor {
			s.cursor--
		}
		s.cursor = ((s.cursor % len(s.order)) + len(s.order)) % len(s.order)
	}
	for key, binding := range s.bindings {
		if binding.account == account {
			delete(s.bindings, key)
		}
	}
}

// AccountCount returns the number of accounts currently in the rotation order
// (eligible or not), for diagnostics and bound proofs.
func (s *Selector) AccountCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.order)
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

// BindingCount returns the number of live continuation bindings after reclaiming
// any that have expired. It lets callers and tests prove bound/zero state.
func (s *Selector) BindingCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepExpiredLocked(s.now())
	return len(s.bindings)
}
