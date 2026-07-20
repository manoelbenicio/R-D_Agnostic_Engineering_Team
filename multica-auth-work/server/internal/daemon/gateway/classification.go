package gateway

import (
	"net/http"
	"time"
)

// AuthOutcome disambiguates authentication failures that share HTTP status 401
// but differ in recoverability. An expired access token is refreshable once; a
// revoked refresh token is terminal and must quarantine the account.
type AuthOutcome uint8

const (
	AuthOutcomeNone AuthOutcome = iota
	AuthOutcomeAccessExpired
	AuthOutcomeRefreshRevoked
)

// RateLimitScope declares the blast radius of a 429 when the upstream (or a
// gateway-side signal) can attribute it. An account-scoped limit lets other
// accounts keep serving; a provider-global limit must not trigger account
// thrashing.
type RateLimitScope uint8

const (
	RateLimitScopeUnknown RateLimitScope = iota
	RateLimitScopeAccount
	RateLimitScopeProvider
)

// FailureSignal is the deterministic, content-free description of an upstream
// attempt outcome. It is assembled from transport results, HTTP status and
// bounded metadata headers — never from response bodies, credentials or prompt
// content.
type FailureSignal struct {
	// Canceled and Timeout reflect client-side/context outcomes that never
	// reached a classified HTTP status.
	Canceled bool
	Timeout  bool
	// Malformed marks a structurally invalid upstream payload that arrived on
	// an otherwise-successful transport.
	Malformed bool
	// LocalOverload marks a gateway-local capacity rejection (bounded queue
	// full, local concurrency ceiling) that is distinct from any provider or
	// account throttle.
	LocalOverload bool
	// StatusCode is the upstream HTTP status when one was received.
	StatusCode int
	// AuthOutcome refines a 401 into refreshable vs terminal.
	AuthOutcome AuthOutcome
	// RateScope refines a 429 into account vs provider-global.
	RateScope RateLimitScope
	// QuotaExhausted marks that the selected account has no remaining quota, as
	// opposed to a transient account throttle. It refines account-scoped 429
	// handling so callers fall back rather than hammering the same account.
	QuotaExhausted bool
	// RetryAfter carries any honored upstream Retry-After hint.
	RetryAfter time.Duration
}

// FailureDecision is the deterministic routing verdict for a FailureSignal. It
// pairs the client-facing error class with the circuit scope that should
// absorb the failure and the recoverability of a pre-commit retry.
type FailureDecision struct {
	Class     ErrorClass
	Scope     CircuitScope
	Retryable bool
	Quota     QuotaState
}

// ClassifyFailure maps a FailureSignal to a deterministic FailureDecision.
//
// The mapping is total and side-effect free so identical signals always yield
// identical decisions. The throttle/overload taxonomy is kept fully distinct:
//
//	cancelled              -> cancelled       / local    / not retryable
//	timeout                -> timeout         / local    / retryable
//	malformed upstream     -> protocol        / provider / not retryable
//	local overload         -> overloaded      / local    / retryable
//	401 access expired     -> authentication  / account  / retryable (refresh once)
//	401 refresh revoked    -> authentication  / account  / not retryable (quarantine)
//	403                    -> authorization   / account  / not retryable
//	429 account throttle   -> rate_limited    / account  / retryable   / quota=limited
//	429 account exhausted  -> rate_limited    / account  / retryable   / quota=exhausted
//	429 provider-global    -> rate_limited    / provider / retryable   / quota=limited
//	503                    -> overloaded      / local    / retryable
//	5xx                    -> upstream        / provider / retryable
//	4xx (other)            -> invalid_request / provider / not retryable
func ClassifyFailure(signal FailureSignal) FailureDecision {
	switch {
	case signal.Canceled:
		return FailureDecision{Class: ErrorCancelled, Scope: CircuitLocal, Retryable: false, Quota: QuotaUnknown}
	case signal.Timeout:
		return FailureDecision{Class: ErrorTimeout, Scope: CircuitLocal, Retryable: true, Quota: QuotaUnknown}
	case signal.Malformed:
		return FailureDecision{Class: ErrorProtocol, Scope: CircuitProvider, Retryable: false, Quota: QuotaUnknown}
	case signal.LocalOverload:
		return FailureDecision{Class: ErrorOverloaded, Scope: CircuitLocal, Retryable: true, Quota: QuotaUnknown}
	}

	switch {
	case signal.StatusCode == http.StatusUnauthorized:
		retryable := signal.AuthOutcome == AuthOutcomeAccessExpired
		return FailureDecision{Class: ErrorAuthentication, Scope: CircuitAccount, Retryable: retryable, Quota: QuotaUnknown}
	case signal.StatusCode == http.StatusForbidden:
		return FailureDecision{Class: ErrorAuthorization, Scope: CircuitAccount, Retryable: false, Quota: QuotaUnknown}
	case signal.StatusCode == http.StatusTooManyRequests:
		return classifyRateLimited(signal)
	case signal.StatusCode == http.StatusRequestTimeout:
		return FailureDecision{Class: ErrorTimeout, Scope: CircuitLocal, Retryable: true, Quota: QuotaUnknown}
	case signal.StatusCode == http.StatusServiceUnavailable:
		return FailureDecision{Class: ErrorOverloaded, Scope: CircuitLocal, Retryable: true, Quota: QuotaUnknown}
	case signal.StatusCode >= 500:
		return FailureDecision{Class: ErrorUpstream, Scope: CircuitProvider, Retryable: true, Quota: QuotaUnknown}
	case signal.StatusCode >= 400:
		return FailureDecision{Class: ErrorInvalidRequest, Scope: CircuitProvider, Retryable: false, Quota: QuotaUnknown}
	default:
		// A non-error status reached the classifier without a malformed flag;
		// treat it as a protocol fault rather than silently succeeding.
		return FailureDecision{Class: ErrorProtocol, Scope: CircuitProvider, Retryable: false, Quota: QuotaUnknown}
	}
}

// classifyRateLimited keeps the three 429 taxonomies distinct: provider-global
// throttle (provider circuit), account quota exhaustion (account circuit, quota
// exhausted), and transient account throttle (account circuit, quota limited).
// All remain retryable because recovery is a scoped fallback/backoff, not a
// terminal failure.
func classifyRateLimited(signal FailureSignal) FailureDecision {
	if signal.RateScope == RateLimitScopeProvider {
		return FailureDecision{Class: ErrorRateLimited, Scope: CircuitProvider, Retryable: true, Quota: QuotaLimited}
	}
	if signal.QuotaExhausted {
		return FailureDecision{Class: ErrorRateLimited, Scope: CircuitAccount, Retryable: true, Quota: QuotaExhausted}
	}
	return FailureDecision{Class: ErrorRateLimited, Scope: CircuitAccount, Retryable: true, Quota: QuotaLimited}
}

// AsError renders a FailureDecision as a bounded GatewayError for the given
// operation. Retryable decisions carry the honored Retry-After hint. The error
// deliberately excludes any account, credential or response-body detail.
func (d FailureDecision) AsError(operation string, signal FailureSignal) *GatewayError {
	return &GatewayError{
		Operation:  operation,
		Class:      d.Class,
		StatusCode: signal.StatusCode,
		Retryable:  d.Retryable,
		RetryAfter: signal.RetryAfter,
	}
}
