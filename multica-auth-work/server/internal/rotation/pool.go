package rotation

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"
)

var errNilStore = errors.New("rotation: nil store")

type Pool struct {
	store Store
}

func NewPool(store Store) *Pool {
	return &Pool{store: store}
}

func (p *Pool) SelectNext(ctx context.Context, vendor, tenantID string, now time.Time) (Account, error) {
	return p.selectNext(ctx, vendor, tenantID, now, nil)
}

func (p *Pool) selectNext(ctx context.Context, vendor, tenantID string, now time.Time, skip map[string]struct{}) (Account, error) {
	if policy, opts, ok, err := selectionPolicy(ctx); ok || err != nil {
		if err != nil {
			return Account{}, err
		}
		return p.selectNextPolicy(ctx, vendor, tenantID, now, skip, policy, opts)
	}
	return p.selectNextPriority(ctx, vendor, tenantID, now, skip)
}

func (p *Pool) selectNextPriority(ctx context.Context, vendor, tenantID string, now time.Time, skip map[string]struct{}) (Account, error) {
	if p == nil || p.store == nil {
		return Account{}, errNilStore
	}
	accounts, err := p.store.ListAccounts(ctx, vendor, tenantID)
	if err != nil {
		return Account{}, err
	}
	sort.SliceStable(accounts, func(i, j int) bool {
		if accounts[i].Priority == accounts[j].Priority {
			return accounts[i].AccountID < accounts[j].AccountID
		}
		return accounts[i].Priority < accounts[j].Priority
	})
	for _, account := range accounts {
		if _, found := skip[account.AccountID]; found {
			continue
		}
		if accountSelectable(account, now) {
			return account, nil
		}
	}
	return Account{}, ErrNoAccountAvailable
}

func (p *Pool) selectNextPolicy(ctx context.Context, vendor, tenantID string, now time.Time, skip map[string]struct{}, policy RotationPolicy, opts selectionOptions) (Account, error) {
	switch policy.Type {
	case PolicyTypeFallback:
		return p.selectNextFallbackPolicy(ctx, vendor, tenantID, now, skip, policy)
	case PolicyTypeLoadBalancing:
		return p.selectNextLoadBalancedPolicy(ctx, vendor, tenantID, now, skip, policy, opts)
	default:
		return p.selectNextPriority(ctx, vendor, tenantID, now, skip)
	}
}

func (p *Pool) selectNextFallbackPolicy(ctx context.Context, vendor, tenantID string, now time.Time, skip map[string]struct{}, policy RotationPolicy) (Account, error) {
	items := policy.Ordered()
	if len(items) == 0 {
		return Account{}, ErrNoAccountAvailable
	}

	var lastErr error
	for _, item := range items {
		plan := NewRetryPlan(item.Retries)
		for attempt := 0; ; attempt++ {
			account, err := p.selectPolicyItem(ctx, vendor, tenantID, now, skip, item)
			if err == nil {
				return account, nil
			}
			if errors.Is(err, ErrNoAccountAvailable) {
				break
			}
			lastErr = err
			if ClassifyError(err, 0) == FAILOVER_NOW || !plan.ShouldRetry(attempt) {
				break
			}
			if err := sleepWithContext(ctx, Jitter(NextBackoff(attempt))); err != nil {
				return Account{}, err
			}
		}
	}
	if lastErr != nil {
		return Account{}, lastErr
	}
	return Account{}, ErrNoAccountAvailable
}

func (p *Pool) selectNextLoadBalancedPolicy(ctx context.Context, vendor, tenantID string, now time.Time, skip map[string]struct{}, policy RotationPolicy, opts selectionOptions) (Account, error) {
	item := PickConsistent(policy, opts.affinitySeed(vendor, tenantID))
	if item == (PolicyItem{}) {
		return Account{}, ErrNoAccountAvailable
	}
	return p.selectPolicyItem(ctx, vendor, tenantID, now, skip, item)
}

func (p *Pool) selectPolicyItem(ctx context.Context, requestedVendor, tenantID string, now time.Time, skip map[string]struct{}, item PolicyItem) (Account, error) {
	itemVendor := item.Vendor
	if itemVendor == "" {
		itemVendor = requestedVendor
	}
	if itemVendor == "" {
		return Account{}, ErrNoAccountAvailable
	}

	accounts, err := p.listSortedAccounts(ctx, itemVendor, tenantID)
	if err != nil {
		return Account{}, err
	}
	for _, account := range accounts {
		if _, found := skip[account.AccountID]; found {
			continue
		}
		if item.AccountRef != "" && item.AccountRef != "any-of-vendor" && item.AccountRef != account.AccountID {
			continue
		}
		if accountSelectable(account, now) {
			return account, nil
		}
	}
	return Account{}, ErrNoAccountAvailable
}

func (p *Pool) listSortedAccounts(ctx context.Context, vendor, tenantID string) ([]Account, error) {
	if p == nil || p.store == nil {
		return nil, errNilStore
	}
	accounts, err := p.store.ListAccounts(ctx, vendor, tenantID)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(accounts, func(i, j int) bool {
		if accounts[i].Priority == accounts[j].Priority {
			return accounts[i].AccountID < accounts[j].AccountID
		}
		return accounts[i].Priority < accounts[j].Priority
	})
	return accounts, nil
}

func accountSelectable(account Account, now time.Time) bool {
	switch account.Status {
	case StatusAvailable, StatusLeased:
		return true
	case StatusCooldown:
		return account.CooldownUntil != nil && !now.Before(*account.CooldownUntil)
	default:
		return false
	}
}

type selectionOptions struct {
	WorkType WorkType
	TraceID  string
	AgentID  string
}

type selectionOptionsKey struct{}

// WithSelectionPolicy enables policy-driven selection for a call path without
// changing the RotationService contract. An empty workType defaults to GENERAL.
func WithSelectionPolicy(ctx context.Context, workType WorkType, traceID string) context.Context {
	return withSelectionOptions(ctx, selectionOptions{WorkType: workType, TraceID: traceID})
}

func withSelectionOptions(ctx context.Context, opts selectionOptions) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, selectionOptionsKey{}, opts)
}

func withSelectionAgentID(ctx context.Context, agentID string) context.Context {
	opts, ok := getSelectionOptions(ctx)
	if !ok {
		return ctx
	}
	opts.AgentID = agentID
	return withSelectionOptions(ctx, opts)
}

func getSelectionOptions(ctx context.Context) (selectionOptions, bool) {
	if ctx == nil {
		return selectionOptions{}, false
	}
	opts, ok := ctx.Value(selectionOptionsKey{}).(selectionOptions)
	return opts, ok
}

func selectionPolicy(ctx context.Context) (RotationPolicy, selectionOptions, bool, error) {
	opts, ok := getSelectionOptions(ctx)
	if !ok {
		return RotationPolicy{}, selectionOptions{}, false, nil
	}
	workType := opts.WorkType
	if workType == "" {
		workType = WorkTypeGeneral
	}
	policy, err := ResolvePolicy(strings.ToLower(string(workType)))
	if err != nil {
		return RotationPolicy{}, opts, true, err
	}
	return policy, opts, true, nil
}

func (o selectionOptions) affinitySeed(vendor, tenantID string) string {
	if o.TraceID != "" {
		return o.TraceID
	}
	if o.AgentID != "" {
		return o.AgentID
	}
	return vendor + "/" + tenantID
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
