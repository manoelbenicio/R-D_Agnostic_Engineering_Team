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
	nativeStore NativeRotationStore
	detector    ExhaustionDetector
	auth        AccountAuthenticator
	nativePrep  NativeHomePreparer
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

// NewNativeService constructs only the pathless native rotation core. It does
// not compose handlers, transport, credential execution, or production
// scheduling.
func NewNativeService(store NativeRotationStore, preparer NativeHomePreparer) *Service {
	return &Service{nativeStore: store, nativePrep: preparer}
}

// RotateNative executes A1, performs local preparation after A1 commits, then
// records A2 and performs B. A store commit ambiguity is resolved by the store
// on a fresh transaction; this method never cleans either receipt on an
// unknown outcome.
func (s *Service) RotateNative(
	ctx context.Context,
	request NativeRotationRequestV1,
) (NativeRotationResultV1, error) {
	if s == nil || s.nativeStore == nil {
		return NativeRotationResultV1{}, ErrAtomicRotationRequired
	}
	if s.nativePrep == nil {
		return NativeRotationResultV1{}, errNilAuthenticator
	}
	if err := request.validate(); err != nil {
		return NativeRotationResultV1{}, err
	}

	target, err := s.nativeStore.ReserveNativeCandidate(ctx, request)
	if err != nil {
		return NativeRotationResultV1{}, err
	}

	processDigest, prepareErr := s.nativePrep.PrepareNativeHome(ctx, target)
	if prepareErr != nil {
		if _, err := s.nativeStore.RecordNativePreparation(ctx, request, false, ""); err != nil {
			return NativeRotationResultV1{}, errors.Join(prepareErr, err)
		}
		return NativeRotationResultV1{
			OperationRequestID: request.OperationRequestID,
			Outcome:            NativeDefinitelyNotCommitted,
			Current:            request.Current,
			Target:             target,
			ReasonCode:         "candidate_preparation_failed",
		}, prepareErr
	}
	if len(processDigest) != 64 {
		return NativeRotationResultV1{}, ErrInvalidNativeIdentity
	}
	if _, err := s.nativeStore.RecordNativePreparation(
		ctx, request, true, processDigest,
	); err != nil {
		return NativeRotationResultV1{}, err
	}
	return s.nativeStore.CommitNativeSwap(ctx, request, processDigest)
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
	atomicStore, ok := s.store.(AtomicRotationStore)
	if !ok {
		return Account{}, ErrAtomicRotationRequired
	}

	lock := s.agentLock(agentID)
	lock.Lock()
	defer lock.Unlock()

	fromAccountID, err := s.store.CurrentAssignment(ctx, agentID)
	if err != nil {
		return Account{}, err
	}
	var current Account
	if fromAccountID != "" {
		current, err = s.store.GetAccount(ctx, fromAccountID)
		if err != nil {
			return Account{}, err
		}
	}

	skip := map[string]struct{}{}
	if fromAccountID != "" {
		skip[fromAccountID] = struct{}{}
	}

	var lastLoginErr error
	for attempts := 0; attempts < s.maxAttempts; attempts++ {
		next, err := s.pool.selectNext(withSelectionAgentID(ctx, agentID), vendor, tenantID, now, skip)
		if err != nil {
			return Account{}, err
		}
		skip[next.AccountID] = struct{}{}

		prepared := false
		var prepareErr error
		err = atomicStore.RotateAssignmentAtomic(
			ctx,
			agentID,
			fromAccountID,
			next.AccountID,
			reason,
			now,
			func(prepareCtx context.Context) error {
				sessionID, loginErr := s.auth.Login(prepareCtx, next)
				if loginErr != nil {
					if cleanupErr := s.auth.Logout(prepareCtx, next); cleanupErr != nil {
						loginErr = errors.Join(loginErr, cleanupErr)
					}
					prepareErr = loginErr
					return loginErr
				}
				ok, waitErr := s.auth.WaitAuthenticated(prepareCtx, sessionID, s.authTimeout)
				if waitErr == nil && !ok {
					waitErr = errAuthenticationRejected
				}
				if waitErr != nil {
					if cleanupErr := s.auth.Logout(prepareCtx, next); cleanupErr != nil {
						prepareErr = errors.Join(waitErr, cleanupErr)
						return prepareErr
					}
					prepareErr = waitErr
					return waitErr
				}
				prepared = true
				return nil
			},
		)
		if err != nil {
			if prepared {
				if cleanupErr := s.auth.Logout(ctx, next); cleanupErr != nil {
					err = errors.Join(err, cleanupErr)
				}
			}
			if errors.Is(err, ErrStaleAssignment) {
				return Account{}, err
			}
			if prepareErr == nil {
				return Account{}, err
			}
			lastLoginErr = err
			if updateErr := s.store.UpdateAccountStatus(ctx, next.AccountID, StatusDegraded, nil); updateErr != nil {
				return Account{}, updateErr
			}
			continue
		}

		if fromAccountID != "" {
			if err := s.auth.Logout(ctx, current); err != nil {
				return next, err
			}
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
