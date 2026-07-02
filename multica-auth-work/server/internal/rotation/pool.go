package rotation

import (
	"context"
	"errors"
	"sort"
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
