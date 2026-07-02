package rotation

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	errNilAuthenticator       = errors.New("rotation: nil account authenticator")
	errAuthenticationRejected = errors.New("rotation: authentication rejected")
)

const (
	defaultAuthenticationTimeout = 30 * time.Second
	defaultMaxLoginAttempts      = 3
)

type ServiceOption func(*Service)

type Service struct {
	store       Store
	detector    ExhaustionDetector
	auth        AccountAuthenticator
	pool        *Pool
	authTimeout time.Duration
	maxAttempts int

	agentLocksMu sync.Mutex
	agentLocks   map[string]*sync.Mutex
}

var _ RotationService = (*Service)(nil)

func NewService(store Store, detector ExhaustionDetector, auth AccountAuthenticator, opts ...ServiceOption) *Service {
	s := &Service{
		store:       store,
		detector:    detector,
		auth:        auth,
		pool:        NewPool(store),
		authTimeout: defaultAuthenticationTimeout,
		maxAttempts: defaultMaxLoginAttempts,
		agentLocks:  map[string]*sync.Mutex{},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
	if s.authTimeout <= 0 {
		s.authTimeout = defaultAuthenticationTimeout
	}
	if s.maxAttempts <= 0 {
		s.maxAttempts = defaultMaxLoginAttempts
	}
	return s
}

func WithAuthenticationTimeout(timeout time.Duration) ServiceOption {
	return func(s *Service) {
		s.authTimeout = timeout
	}
}

func WithMaxLoginAttempts(maxAttempts int) ServiceOption {
	return func(s *Service) {
		s.maxAttempts = maxAttempts
	}
}

func (s *Service) SelectNext(ctx context.Context, vendor, tenantID string, now time.Time) (Account, error) {
	if s == nil || s.pool == nil {
		return Account{}, errNilStore
	}
	return s.pool.SelectNext(ctx, vendor, tenantID, now)
}

func (s *Service) OnExhaustion(ctx context.Context, agentID, vendor, tenantID string, reason RotationReason, now time.Time) (Account, error) {
	if s == nil || s.store == nil {
		return Account{}, errNilStore
	}
	if s.auth == nil {
		return Account{}, errNilAuthenticator
	}

	lock := s.agentLock(agentID)
	lock.Lock()
	defer lock.Unlock()

	fromAccountID, err := s.store.CurrentAssignment(ctx, agentID)
	if err != nil {
		return Account{}, err
	}
	if fromAccountID != "" {
		current, err := s.store.GetAccount(ctx, fromAccountID)
		if err != nil {
			return Account{}, err
		}
		if err := s.auth.Logout(ctx, current); err != nil {
			return Account{}, err
		}
	}

	skip := map[string]struct{}{}
	if fromAccountID != "" {
		skip[fromAccountID] = struct{}{}
	}

	var lastLoginErr error
	for attempts := 0; attempts < s.maxAttempts; attempts++ {
		next, err := s.pool.selectNext(ctx, vendor, tenantID, now, skip)
		if err != nil {
			return Account{}, err
		}
		skip[next.AccountID] = struct{}{}

		sessionID, err := s.auth.Login(ctx, next)
		if err == nil {
			var ok bool
			ok, err = s.auth.WaitAuthenticated(ctx, sessionID, s.authTimeout)
			if err == nil && !ok {
				err = errAuthenticationRejected
			}
		}
		if err != nil {
			lastLoginErr = err
			if updateErr := s.store.UpdateAccountStatus(ctx, next.AccountID, StatusDegraded, nil); updateErr != nil {
				return Account{}, updateErr
			}
			continue
		}

		if err := s.store.Assign(ctx, agentID, next.AccountID); err != nil {
			return Account{}, err
		}
		if err := s.store.RecordRotation(ctx, agentID, fromAccountID, next.AccountID, reason, now); err != nil {
			return Account{}, err
		}
		return next, nil
	}

	if lastLoginErr != nil {
		return Account{}, lastLoginErr
	}
	return Account{}, ErrNoAccountAvailable
}

func (s *Service) agentLock(agentID string) *sync.Mutex {
	s.agentLocksMu.Lock()
	defer s.agentLocksMu.Unlock()
	if lock, ok := s.agentLocks[agentID]; ok {
		return lock
	}
	lock := &sync.Mutex{}
	s.agentLocks[agentID] = lock
	return lock
}
