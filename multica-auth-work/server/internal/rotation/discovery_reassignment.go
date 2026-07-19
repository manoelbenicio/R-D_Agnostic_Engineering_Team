package rotation

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	errInvalidDiscoveryReassignment = errors.New("rotation: invalid discovery reassignment request")
	errMissingCurrentAssignment     = errors.New("rotation: discovery reassignment requires a current assignment")
	errDiscoveryAssignmentBoundary  = errors.New("rotation: discovery assignment is outside the requested provider or tenant")
)

// ReassignDiscoverySession connects the source-only discovery classifier to
// account selection and reassignment. expectedAccountID binds the observation
// to the account that was active when discovery ran; duplicate or delayed
// observations become no-ops after another call has already reassigned it.
//
// The entire check-and-rotate sequence is serialized by the same per-agent
// lock used by OnExhaustion. No credential material is accepted or inspected.
func (s *Service) ReassignDiscoverySession(
	ctx context.Context,
	agentID string,
	expectedAccountID string,
	provider string,
	tenantID string,
	session DiscoverySession,
	now time.Time,
) (Account, bool, error) {
	if !DetectDiscoverySession(provider, session, now).Exhausted {
		return Account{}, false, nil
	}
	if s == nil || s.store == nil || s.pool == nil {
		return Account{}, false, errNilStore
	}
	if s.auth == nil {
		return Account{}, false, errNilAuthenticator
	}

	agentID = strings.TrimSpace(agentID)
	expectedAccountID = strings.TrimSpace(expectedAccountID)
	provider = canonicalDiscoveryProvider(provider)
	tenantID = strings.TrimSpace(tenantID)
	if agentID == "" || expectedAccountID == "" || provider == "" || tenantID == "" {
		return Account{}, false, errInvalidDiscoveryReassignment
	}

	lock := s.agentLock(agentID)
	lock.Lock()
	defer lock.Unlock()

	currentAccountID, err := s.store.CurrentAssignment(ctx, agentID)
	if err != nil {
		return Account{}, false, err
	}
	if currentAccountID == "" {
		return Account{}, false, errMissingCurrentAssignment
	}
	if currentAccountID != expectedAccountID {
		return Account{}, false, nil
	}

	current, err := s.store.GetAccount(ctx, currentAccountID)
	if err != nil {
		return Account{}, false, err
	}
	if !sameDiscoveryProvider(provider, current.Vendor) || strings.TrimSpace(current.TenantID) != tenantID {
		return Account{}, false, errDiscoveryAssignmentBoundary
	}
	if err := s.store.UpdateAccountStatus(ctx, currentAccountID, StatusExhausted, nil); err != nil {
		return Account{}, false, err
	}

	next, err := s.onExhaustionLocked(ctx, agentID, provider, tenantID, ReasonQuotaReactive, now)
	if err != nil {
		return Account{}, false, err
	}
	return next, true, nil
}
