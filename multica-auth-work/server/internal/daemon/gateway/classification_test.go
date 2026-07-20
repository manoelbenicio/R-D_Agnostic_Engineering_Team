package gateway

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestClassifyFailureCoversTheFullAuthQuotaRateAndUpstreamMatrix(t *testing.T) {
	cases := []struct {
		name      string
		signal    FailureSignal
		class     ErrorClass
		scope     CircuitScope
		retryable bool
		quota     QuotaState
	}{
		{
			name:   "expired-access-token",
			signal: FailureSignal{StatusCode: http.StatusUnauthorized, AuthOutcome: AuthOutcomeAccessExpired},
			class:  ErrorAuthentication, scope: CircuitAccount, retryable: true, quota: QuotaUnknown,
		},
		{
			name:   "revoked-refresh-token",
			signal: FailureSignal{StatusCode: http.StatusUnauthorized, AuthOutcome: AuthOutcomeRefreshRevoked},
			class:  ErrorAuthentication, scope: CircuitAccount, retryable: false, quota: QuotaUnknown,
		},
		{
			name:   "unauthorized-unknown-auth",
			signal: FailureSignal{StatusCode: http.StatusUnauthorized},
			class:  ErrorAuthentication, scope: CircuitAccount, retryable: false, quota: QuotaUnknown,
		},
		{
			name:   "forbidden",
			signal: FailureSignal{StatusCode: http.StatusForbidden},
			class:  ErrorAuthorization, scope: CircuitAccount, retryable: false, quota: QuotaUnknown,
		},
		{
			// Taxonomy 1: transient account throttle.
			name:   "account-throttle-429",
			signal: FailureSignal{StatusCode: http.StatusTooManyRequests, RateScope: RateLimitScopeAccount},
			class:  ErrorRateLimited, scope: CircuitAccount, retryable: true, quota: QuotaLimited,
		},
		{
			// Unscoped 429 defaults to a transient account throttle.
			name:   "account-throttle-unscoped-429",
			signal: FailureSignal{StatusCode: http.StatusTooManyRequests},
			class:  ErrorRateLimited, scope: CircuitAccount, retryable: true, quota: QuotaLimited,
		},
		{
			// Taxonomy 2: account quota exhausted (distinct quota state, fall back).
			name:   "account-quota-exhausted-429",
			signal: FailureSignal{StatusCode: http.StatusTooManyRequests, RateScope: RateLimitScopeAccount, QuotaExhausted: true},
			class:  ErrorRateLimited, scope: CircuitAccount, retryable: true, quota: QuotaExhausted,
		},
		{
			// Taxonomy 3: provider-global throttle (provider circuit, no account thrash).
			name:   "provider-global-throttle-429",
			signal: FailureSignal{StatusCode: http.StatusTooManyRequests, RateScope: RateLimitScopeProvider},
			class:  ErrorRateLimited, scope: CircuitProvider, retryable: true, quota: QuotaLimited,
		},
		{
			// Taxonomy 3b: provider-global quota-exhausted signal still stays
			// provider-scoped and does not thrash accounts.
			name:   "provider-global-exhausted-429",
			signal: FailureSignal{StatusCode: http.StatusTooManyRequests, RateScope: RateLimitScopeProvider, QuotaExhausted: true},
			class:  ErrorRateLimited, scope: CircuitProvider, retryable: true, quota: QuotaLimited,
		},
		{
			// Taxonomy 4: gateway-local overload, distinct from provider/account throttle.
			name:   "local-overload",
			signal: FailureSignal{LocalOverload: true},
			class:  ErrorOverloaded, scope: CircuitLocal, retryable: true, quota: QuotaUnknown,
		},
		{
			name:   "server-5xx",
			signal: FailureSignal{StatusCode: http.StatusInternalServerError},
			class:  ErrorUpstream, scope: CircuitProvider, retryable: true, quota: QuotaUnknown,
		},
		{
			name:   "service-unavailable",
			signal: FailureSignal{StatusCode: http.StatusServiceUnavailable},
			class:  ErrorOverloaded, scope: CircuitLocal, retryable: true, quota: QuotaUnknown,
		},
		{
			name:   "timeout",
			signal: FailureSignal{Timeout: true},
			class:  ErrorTimeout, scope: CircuitLocal, retryable: true, quota: QuotaUnknown,
		},
		{
			name:   "http-408-timeout",
			signal: FailureSignal{StatusCode: http.StatusRequestTimeout},
			class:  ErrorTimeout, scope: CircuitLocal, retryable: true, quota: QuotaUnknown,
		},
		{
			name:   "malformed-upstream",
			signal: FailureSignal{Malformed: true},
			class:  ErrorProtocol, scope: CircuitProvider, retryable: false, quota: QuotaUnknown,
		},
		{
			name:   "cancelled",
			signal: FailureSignal{Canceled: true},
			class:  ErrorCancelled, scope: CircuitLocal, retryable: false, quota: QuotaUnknown,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decision := ClassifyFailure(tc.signal)
			if decision.Class != tc.class || decision.Scope != tc.scope || decision.Retryable != tc.retryable || decision.Quota != tc.quota {
				t.Fatalf("classification mismatch: got %#v", decision)
			}
		})
	}
}

func TestClassifyFailureKeepsThrottleAndOverloadTaxonomiesDistinct(t *testing.T) {
	accountThrottle := ClassifyFailure(FailureSignal{StatusCode: http.StatusTooManyRequests, RateScope: RateLimitScopeAccount})
	accountExhausted := ClassifyFailure(FailureSignal{StatusCode: http.StatusTooManyRequests, RateScope: RateLimitScopeAccount, QuotaExhausted: true})
	providerThrottle := ClassifyFailure(FailureSignal{StatusCode: http.StatusTooManyRequests, RateScope: RateLimitScopeProvider})
	localOverload := ClassifyFailure(FailureSignal{LocalOverload: true})

	// Account throttle vs account quota exhaustion: same class/scope but the
	// quota state must differ so callers can fall back rather than hammer.
	if accountThrottle.Quota != QuotaLimited || accountExhausted.Quota != QuotaExhausted {
		t.Fatalf("account throttle and exhaustion not distinguished: %v vs %v", accountThrottle.Quota, accountExhausted.Quota)
	}
	if accountThrottle.Scope != CircuitAccount || accountExhausted.Scope != CircuitAccount {
		t.Fatalf("account taxonomies must stay account-scoped")
	}
	// Provider-global throttle must be provider-scoped, never account-scoped.
	if providerThrottle.Scope != CircuitProvider {
		t.Fatalf("provider-global throttle leaked scope %v", providerThrottle.Scope)
	}
	// Local overload must be its own class and local scope.
	if localOverload.Class != ErrorOverloaded || localOverload.Scope != CircuitLocal {
		t.Fatalf("local overload not distinct: %#v", localOverload)
	}
	// All four remain retryable (recovery is scoped backoff/fallback), but no
	// two of them collapse to the same (class, scope, quota) tuple except the
	// two account-scoped ones which are separated by quota state.
	type tuple struct {
		class ErrorClass
		scope CircuitScope
		quota QuotaState
	}
	seen := map[tuple]int{}
	for _, d := range []FailureDecision{accountThrottle, accountExhausted, providerThrottle, localOverload} {
		if !d.Retryable {
			t.Fatalf("throttle/overload taxonomy unexpectedly non-retryable: %#v", d)
		}
		seen[tuple{d.Class, d.Scope, d.Quota}]++
	}
	if len(seen) != 4 {
		t.Fatalf("throttle/overload taxonomies collapsed: %d distinct tuples, want 4", len(seen))
	}
}

func TestClassifyFailureIsDeterministicAndTotal(t *testing.T) {
	signal := FailureSignal{StatusCode: http.StatusTooManyRequests, RateScope: RateLimitScopeProvider}
	first := ClassifyFailure(signal)
	for i := 0; i < 100; i++ {
		if ClassifyFailure(signal) != first {
			t.Fatal("classification is not deterministic")
		}
	}
	// Cancellation and timeout precedence over any accompanying status code.
	if got := ClassifyFailure(FailureSignal{Canceled: true, StatusCode: http.StatusInternalServerError}); got.Class != ErrorCancelled {
		t.Fatalf("cancellation must take precedence, got %s", got.Class)
	}
	if got := ClassifyFailure(FailureSignal{Timeout: true, StatusCode: http.StatusOK}); got.Class != ErrorTimeout {
		t.Fatalf("timeout must take precedence, got %s", got.Class)
	}
}

func TestFailureDecisionAsErrorIsBoundedAndRetryAware(t *testing.T) {
	signal := FailureSignal{StatusCode: http.StatusTooManyRequests, RateScope: RateLimitScopeAccount, RetryAfter: 3 * time.Second}
	decision := ClassifyFailure(signal)
	err := decision.AsError("provider.dispatch", signal)
	if !IsErrorClass(err, ErrorRateLimited) {
		t.Fatalf("class mismatch: %v", err)
	}
	if !err.Retryable || err.RetryAfter != 3*time.Second || err.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("retry metadata not preserved: %#v", err)
	}
	// The rendered error must not carry account, credential or body detail.
	rendered := err.Error()
	for _, leak := range []string{"account", "token", "bearer", "cookie", "secret"} {
		if strings.Contains(strings.ToLower(rendered), leak) {
			t.Fatalf("error rendering leaked %q: %s", leak, rendered)
		}
	}
}
