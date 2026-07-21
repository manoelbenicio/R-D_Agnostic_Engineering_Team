package gateway

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Task 8.5 acceptance — deterministic failure classification for every required
// class using the production ClassifyFailure + a mock HTTP transport (client.do
// / classifyStatus) + the production Coordinator for downstream retry/no-thrash.
// No test-local stateful router is used; this is component evidence only, not
// live OmniRoute proof.

// A) Production ClassifyFailure decision matrix: class / scope / retry / quota.
func TestTask85ClassifyFailureAcceptanceMatrix(t *testing.T) {
	cases := []struct {
		name      string
		signal    FailureSignal
		class     ErrorClass
		scope     CircuitScope
		retryable bool
		quota     QuotaState
	}{
		{"expired_access_token", FailureSignal{StatusCode: http.StatusUnauthorized, AuthOutcome: AuthOutcomeAccessExpired}, ErrorAuthentication, CircuitAccount, true, QuotaUnknown},
		{"revoked_refresh_token", FailureSignal{StatusCode: http.StatusUnauthorized, AuthOutcome: AuthOutcomeRefreshRevoked}, ErrorAuthentication, CircuitAccount, false, QuotaUnknown},
		{"unauthorized_401", FailureSignal{StatusCode: http.StatusUnauthorized}, ErrorAuthentication, CircuitAccount, false, QuotaUnknown},
		{"forbidden_403", FailureSignal{StatusCode: http.StatusForbidden}, ErrorAuthorization, CircuitAccount, false, QuotaUnknown},
		{"account_429", FailureSignal{StatusCode: http.StatusTooManyRequests, RateScope: RateLimitScopeAccount}, ErrorRateLimited, CircuitAccount, true, QuotaLimited},
		{"quota_exhaustion", FailureSignal{StatusCode: http.StatusTooManyRequests, RateScope: RateLimitScopeAccount, QuotaExhausted: true}, ErrorRateLimited, CircuitAccount, true, QuotaExhausted},
		{"provider_global_429", FailureSignal{StatusCode: http.StatusTooManyRequests, RateScope: RateLimitScopeProvider}, ErrorRateLimited, CircuitProvider, true, QuotaLimited},
		{"server_5xx", FailureSignal{StatusCode: http.StatusInternalServerError}, ErrorUpstream, CircuitProvider, true, QuotaUnknown},
		{"timeout", FailureSignal{Timeout: true}, ErrorTimeout, CircuitLocal, true, QuotaUnknown},
		{"malformed_upstream", FailureSignal{Malformed: true}, ErrorProtocol, CircuitProvider, false, QuotaUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := ClassifyFailure(tc.signal)
			if d.Class != tc.class || d.Scope != tc.scope || d.Retryable != tc.retryable || d.Quota != tc.quota {
				t.Fatalf("%s: got %#v", tc.name, d)
			}
		})
	}
}

// B) Mock-transport classification via the production client path
// (client.do → classifyStatus / classifyTransportError) + redaction: bounded
// errors must never echo response bodies or secrets.
func task85Client(t *testing.T, transport roundTripFunc) *Client {
	t.Helper()
	c, err := NewClient(ClientOptions{
		Gateway:        testGatewayConfig(t, "http://synthetic.invalid"),
		Endpoints:      EndpointSet{Liveness: "/health/live", Readiness: "/v1/models"},
		Credential:     &syntheticCredentialSource{},
		HTTPClient:     &http.Client{Transport: transport},
		RequestTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestTask85MockTransportStatusClassificationAndRedaction(t *testing.T) {
	const leak = "SECRET-LEAK-SENTINEL"
	statusCases := []struct {
		status int
		class  ErrorClass
	}{
		{http.StatusUnauthorized, ErrorAuthentication},
		{http.StatusForbidden, ErrorAuthorization},
		{http.StatusTooManyRequests, ErrorRateLimited},
		{http.StatusInternalServerError, ErrorUpstream},
		{http.StatusServiceUnavailable, ErrorOverloaded},
	}
	for _, tc := range statusCases {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			client := task85Client(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
				resp := syntheticResponse(req, tc.status, "application/json", `{"error":"`+leak+`"}`)
				resp.Header.Set(HeaderAccountID, "raw-account-"+leak)
				return resp, nil
			}))
			_, err := client.CheckReadiness(context.Background(), testCorrelation())
			if !IsErrorClass(err, tc.class) {
				t.Fatalf("status %d: class=%v want %v", tc.status, err, tc.class)
			}
			if strings.Contains(err.Error(), leak) {
				t.Fatalf("status %d: error leaked response/secret content: %s", tc.status, err.Error())
			}
		})
	}

	// Transport timeout → ErrorTimeout (retryable), no leak.
	t.Run("transport_timeout", func(t *testing.T) {
		client := task85Client(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, context.DeadlineExceeded
		}))
		_, err := client.CheckReadiness(context.Background(), testCorrelation())
		if !IsErrorClass(err, ErrorTimeout) {
			t.Fatalf("transport timeout: got %v", err)
		}
	})

	// Malformed upstream body on /v1/models → ErrorProtocol, no leak.
	t.Run("malformed_upstream", func(t *testing.T) {
		client := task85Client(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return syntheticResponse(req, http.StatusOK, "application/json", `{malformed `+leak), nil
		}))
		_, err := client.FetchModels(context.Background(), testCorrelation())
		if !IsErrorClass(err, ErrorProtocol) {
			t.Fatalf("malformed: got %v", err)
		}
		if strings.Contains(err.Error(), leak) {
			t.Fatalf("malformed: error leaked body: %s", err.Error())
		}
	})
}

// C) Downstream retry / no-thrash using the production Coordinator + the
// classified errors: retryable classes retry (bounded), non-retryable stop
// immediately, and account-scoped vs provider-global throttles are kept in
// distinct circuit scopes (provider-global does not thrash accounts).
func TestTask85DownstreamRetryAndNoThrash(t *testing.T) {
	coord, err := NewCoordinator(RetryPolicy{MaxAttempts: 3, EndToEndDeadline: 30 * time.Second, PreCommitOnly: true})
	if err != nil {
		t.Fatal(err)
	}

	// Retryable classes retry up to the bound then surface.
	for _, tc := range []struct {
		name   string
		signal FailureSignal
	}{
		{"account_429", FailureSignal{StatusCode: http.StatusTooManyRequests, RateScope: RateLimitScopeAccount}},
		{"provider_global_429", FailureSignal{StatusCode: http.StatusTooManyRequests, RateScope: RateLimitScopeProvider}},
		{"server_5xx", FailureSignal{StatusCode: http.StatusInternalServerError}},
	} {
		t.Run("retryable_"+tc.name, func(t *testing.T) {
			var calls int
			decision := ClassifyFailure(tc.signal)
			res, execErr := coord.Execute(context.Background(), "retry-"+tc.name, func(context.Context, int) (AttemptResult, error) {
				calls++
				return AttemptResult{}, decision.AsError("task85", tc.signal)
			})
			if execErr == nil || res.Attempts != 3 || calls != 3 {
				t.Fatalf("%s: retryable not bounded-retried: attempts=%d calls=%d err=%v", tc.name, res.Attempts, calls, execErr)
			}
		})
	}

	// Non-retryable classes stop immediately (single attempt).
	for _, tc := range []struct {
		name   string
		signal FailureSignal
	}{
		{"revoked_refresh", FailureSignal{StatusCode: http.StatusUnauthorized, AuthOutcome: AuthOutcomeRefreshRevoked}},
		{"forbidden", FailureSignal{StatusCode: http.StatusForbidden}},
		{"malformed", FailureSignal{Malformed: true}},
	} {
		t.Run("nonretryable_"+tc.name, func(t *testing.T) {
			var calls int
			decision := ClassifyFailure(tc.signal)
			res, execErr := coord.Execute(context.Background(), "noretry-"+tc.name, func(context.Context, int) (AttemptResult, error) {
				calls++
				return AttemptResult{}, decision.AsError("task85", tc.signal)
			})
			if execErr == nil || res.Attempts != 1 || calls != 1 {
				t.Fatalf("%s: non-retryable retried: attempts=%d calls=%d", tc.name, res.Attempts, calls)
			}
		})
	}

	// No-thrash: account throttle stays account-scoped; provider-global throttle
	// stays provider-scoped (so provider-global does not rotate/thrash accounts).
	acct := ClassifyFailure(FailureSignal{StatusCode: http.StatusTooManyRequests, RateScope: RateLimitScopeAccount})
	global := ClassifyFailure(FailureSignal{StatusCode: http.StatusTooManyRequests, RateScope: RateLimitScopeProvider})
	if acct.Scope != CircuitAccount {
		t.Fatalf("account throttle scope=%v want account", acct.Scope)
	}
	if global.Scope != CircuitProvider {
		t.Fatalf("provider-global throttle scope=%v want provider (no account thrash)", global.Scope)
	}
	if acct.Scope == global.Scope {
		t.Fatal("account and provider-global throttle collapsed to the same circuit scope")
	}
}
