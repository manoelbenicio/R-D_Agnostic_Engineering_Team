package gateway

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

// res_readiness_test.go — deterministic fake-fetch regressions for the
// gateway model-registry readiness poller (RES). All timing is driven by an
// injected clock (Registry.now); no wall-clock sleeps gate correctness. The
// fake fetch is a ModelsFetchFunc so no network/OmniRoute is touched.

const resReadinessModel = brain.RouteModel("agy/claude-opus-4-6-thinking")

// modelsDocumentVersion returns the standard valid document tagged with a
// specific registry version so freshness/staleness can be observed.
func modelsDocumentVersion(version string) ModelsDocument {
	doc := validModelsDocument()
	doc.RegistryVersion = version
	return doc
}

// transientError is a retryable gateway failure of the given class, standing in
// for a 429 / 5xx / timeout upstream response.
func transientError(class ErrorClass) *GatewayError {
	return &GatewayError{Operation: "registry.refresh", Class: class, Retryable: true}
}

// RES-1: transient 429/5xx/timeout then success -> a FRESH ready snapshot is
// admitted (never a partial/failed one), and the negative-cache backoff is
// honored between the failure and the retry.
func TestRegistryTransientThenSuccessAdmitsFreshReady(t *testing.T) {
	for _, class := range []ErrorClass{ErrorRateLimited, ErrorOverloaded, ErrorUpstream, ErrorTimeout} {
		t.Run(string(class), func(t *testing.T) {
			var fetches atomic.Int64
			var failNext atomic.Bool
			failNext.Store(true)
			registry, err := NewRegistry(ModelsFetchFunc(func(context.Context) (ModelsDocument, error) {
				fetches.Add(1)
				if failNext.Load() {
					return ModelsDocument{}, transientError(class)
				}
				return modelsDocumentVersion("fresh-v1"), nil
			}), time.Minute)
			if err != nil {
				t.Fatalf("NewRegistry: %v", err)
			}
			clock := time.Unix(1_000_000, 0).UTC()
			registry.now = func() time.Time { return clock }

			// First poll: transient failure is surfaced (fail closed), not a snapshot.
			if _, err := registry.Snapshot(context.Background()); !IsErrorClass(err, class) {
				t.Fatalf("first poll: got %v, want class %s", err, class)
			}
			if fetches.Load() != 1 {
				t.Fatalf("after first poll fetches=%d, want 1", fetches.Load())
			}

			// Within the negative-cache backoff window the poller must NOT hammer
			// upstream: the cached failure is returned without a new fetch.
			clock = clock.Add(RegistryRefreshFailureBackoff - time.Millisecond)
			if _, err := registry.Snapshot(context.Background()); !IsErrorClass(err, class) {
				t.Fatalf("within backoff: got %v, want cached class %s", err, class)
			}
			if fetches.Load() != 1 {
				t.Fatalf("backoff window re-fetched: fetches=%d, want 1", fetches.Load())
			}

			// After the backoff elapses the next poll retries; upstream is healthy
			// now, so a FRESH ready snapshot is admitted.
			failNext.Store(false)
			clock = clock.Add(2 * RegistryRefreshFailureBackoff)
			snapshot, err := registry.Snapshot(context.Background())
			if err != nil {
				t.Fatalf("post-backoff poll: unexpected error %v", err)
			}
			if snapshot.Version != "fresh-v1" {
				t.Fatalf("expected FRESH snapshot version fresh-v1, got %q", snapshot.Version)
			}
			if fetches.Load() != 2 {
				t.Fatalf("expected exactly one retry fetch (total 2), got %d", fetches.Load())
			}
			spec, err := registry.LookupModel(context.Background(), resReadinessModel)
			if err != nil || !spec.Available {
				t.Fatalf("fresh-ready model not admitted: spec.Available=%v err=%v", spec.Available, err)
			}
		})
	}
}

// RES-2: persistent failure -> the poller always fails closed and never admits
// a usable snapshot, no matter how many times it is polled across the backoff.
func TestRegistryPersistentFailureFailsClosed(t *testing.T) {
	var fetches atomic.Int64
	registry, err := NewRegistry(ModelsFetchFunc(func(context.Context) (ModelsDocument, error) {
		fetches.Add(1)
		return ModelsDocument{}, transientError(ErrorOverloaded)
	}), time.Minute)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	clock := time.Unix(2_000_000, 0).UTC()
	registry.now = func() time.Time { return clock }

	for i := 0; i < 5; i++ {
		if _, err := registry.Snapshot(context.Background()); !IsErrorClass(err, ErrorOverloaded) {
			t.Fatalf("poll %d: got %v, want fail-closed ErrorOverloaded", i, err)
		}
		// LookupModel must also fail closed (never returns an available model).
		if _, err := registry.LookupModel(context.Background(), resReadinessModel); err == nil {
			t.Fatalf("poll %d: LookupModel admitted a model under persistent failure", i)
		}
		clock = clock.Add(RegistryRefreshFailureBackoff + time.Millisecond)
	}
	if fetches.Load() == 0 {
		t.Fatal("expected the poller to keep attempting upstream across backoff windows")
	}
}

// RES-3: single-flight coalesces concurrent polls -> N concurrent readiness
// polls on a cold registry trigger exactly ONE upstream fetch, and all callers
// receive the same fresh snapshot.
func TestRegistrySingleFlightCoalescesConcurrentPolls(t *testing.T) {
	var fetches atomic.Int64
	release := make(chan struct{})
	registry, err := NewRegistry(ModelsFetchFunc(func(context.Context) (ModelsDocument, error) {
		fetches.Add(1)
		<-release // hold the single in-flight refresh open until all pollers pile on
		return modelsDocumentVersion("fresh-coalesced"), nil
	}), time.Minute)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	registry.now = func() time.Time { return time.Unix(3_000_000, 0).UTC() } // fixed clock

	const pollers = 16
	var wait sync.WaitGroup
	versions := make([]string, pollers)
	errs := make([]error, pollers)
	for i := 0; i < pollers; i++ {
		wait.Add(1)
		go func(idx int) {
			defer wait.Done()
			snapshot, err := registry.Snapshot(context.Background())
			errs[idx] = err
			versions[idx] = snapshot.Version
		}(i)
	}
	// Give the followers time to observe the in-flight refresh and coalesce onto
	// it, then release the single fetch.
	time.Sleep(50 * time.Millisecond)
	close(release)
	wait.Wait()

	if fetches.Load() != 1 {
		t.Fatalf("single-flight violated: %d concurrent polls caused %d fetches, want 1", pollers, fetches.Load())
	}
	for i := 0; i < pollers; i++ {
		if errs[i] != nil || versions[i] != "fresh-coalesced" {
			t.Fatalf("poller %d did not receive the coalesced fresh snapshot: version=%q err=%v", i, versions[i], errs[i])
		}
	}
}

// RES-4: Retry-After + backoff honored. The classification layer parses and
// carries the upstream Retry-After hint (the value the pre-commit backoff
// honors), and the registry poller applies a bounded negative-cache backoff.
func TestGatewayRetryAfterAndBackoffHonored(t *testing.T) {
	now := time.Unix(4_000_000, 0).UTC()

	// parseRetryAfter: delta-seconds, HTTP-date, and reject invalid/out-of-range.
	if got := parseRetryAfter("5", now); got != 5*time.Second {
		t.Fatalf("parseRetryAfter(\"5\") = %v, want 5s", got)
	}
	if got := parseRetryAfter(now.Add(3*time.Second).UTC().Format(http.TimeFormat), now); got <= 0 || got > 4*time.Second {
		t.Fatalf("parseRetryAfter(http-date +3s) = %v, want ~3s", got)
	}
	for _, bad := range []string{"", "0", "-1", "not-a-number", "999999999999"} {
		if got := parseRetryAfter(bad, now); got != 0 {
			t.Fatalf("parseRetryAfter(%q) = %v, want 0", bad, got)
		}
	}

	// classifyStatus: 429/503 are retryable and carry the parsed Retry-After;
	// 500/408 retryable; 401 terminal (not retryable).
	classify := func(status int, retryAfter string) *GatewayError {
		header := http.Header{}
		if retryAfter != "" {
			header.Set("Retry-After", retryAfter)
		}
		return classifyStatus("readiness", &http.Response{StatusCode: status, Header: header})
	}
	if e := classify(http.StatusTooManyRequests, "7"); e.Class != ErrorRateLimited || !e.Retryable || e.RetryAfter != 7*time.Second {
		t.Fatalf("429: class=%s retryable=%v retryAfter=%v", e.Class, e.Retryable, e.RetryAfter)
	}
	if e := classify(http.StatusServiceUnavailable, "3"); e.Class != ErrorOverloaded || !e.Retryable || e.RetryAfter != 3*time.Second {
		t.Fatalf("503: class=%s retryable=%v retryAfter=%v", e.Class, e.Retryable, e.RetryAfter)
	}
	if e := classify(http.StatusInternalServerError, ""); e.Class != ErrorUpstream || !e.Retryable {
		t.Fatalf("500: class=%s retryable=%v", e.Class, e.Retryable)
	}
	if e := classify(http.StatusRequestTimeout, ""); e.Class != ErrorTimeout || !e.Retryable {
		t.Fatalf("408: class=%s retryable=%v", e.Class, e.Retryable)
	}
	if e := classify(http.StatusUnauthorized, "5"); e.Class != ErrorAuthentication || e.Retryable {
		t.Fatalf("401: class=%s retryable=%v (must be terminal)", e.Class, e.Retryable)
	}

	// retryAfter(err) extracts exactly the hint the pre-commit backoff honors.
	if got := retryAfter(&GatewayError{Class: ErrorRateLimited, RetryAfter: 9 * time.Second}); got != 9*time.Second {
		t.Fatalf("retryAfter(hint) = %v, want 9s", got)
	}
	if got := retryAfter(errors.New("non-gateway")); got != 0 {
		t.Fatalf("retryAfter(non-gateway) = %v, want 0", got)
	}

	// The registry readiness poller's bounded negative-cache backoff is fixed
	// and non-zero so a failed poll loop cannot hammer upstream.
	if RegistryRefreshFailureBackoff <= 0 || RegistryRefreshFailureBackoff > time.Minute {
		t.Fatalf("RegistryRefreshFailureBackoff = %v, want a small bounded positive backoff", RegistryRefreshFailureBackoff)
	}
}

// RES-5: stale-ready never reused. After Invalidate (generation fence) or TTL
// expiry, the poller must re-fetch and never serve the previous snapshot.
func TestRegistryStaleReadyNeverReused(t *testing.T) {
	var version atomic.Value // string
	version.Store("stale-v1")
	var fetches atomic.Int64
	ttl := 30 * time.Second
	registry, err := NewRegistry(ModelsFetchFunc(func(context.Context) (ModelsDocument, error) {
		fetches.Add(1)
		return modelsDocumentVersion(version.Load().(string)), nil
	}), ttl)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	clock := time.Unix(5_000_000, 0).UTC()
	registry.now = func() time.Time { return clock }

	// Prime a fresh snapshot v1.
	first, err := registry.Snapshot(context.Background())
	if err != nil || first.Version != "stale-v1" {
		t.Fatalf("prime: version=%q err=%v", first.Version, err)
	}
	if fetches.Load() != 1 {
		t.Fatalf("prime fetches=%d, want 1", fetches.Load())
	}

	// Invalidate: the next poll MUST re-fetch and MUST NOT reuse v1.
	version.Store("fresh-v2")
	registry.Invalidate()
	afterInvalidate, err := registry.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("post-invalidate: %v", err)
	}
	if afterInvalidate.Version != "fresh-v2" {
		t.Fatalf("stale snapshot reused after Invalidate: got %q, want fresh-v2", afterInvalidate.Version)
	}
	if fetches.Load() != 2 {
		t.Fatalf("Invalidate did not force a refresh: fetches=%d, want 2", fetches.Load())
	}

	// TTL expiry: advancing past expiresAt must re-fetch, never serve the expired
	// (now stale) snapshot without a refresh.
	version.Store("fresh-v3")
	clock = clock.Add(ttl + time.Second)
	afterExpiry, err := registry.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("post-expiry: %v", err)
	}
	if afterExpiry.Version != "fresh-v3" {
		t.Fatalf("expired stale snapshot reused: got %q, want fresh-v3", afterExpiry.Version)
	}
	if fetches.Load() != 3 {
		t.Fatalf("TTL expiry did not force a refresh: fetches=%d, want 3", fetches.Load())
	}

	// Within TTL, the fresh snapshot is served from cache (no extra fetch) — the
	// positive cache is the only reuse permitted, and only while still fresh.
	if _, err := registry.Snapshot(context.Background()); err != nil {
		t.Fatalf("within-ttl cached read: %v", err)
	}
	if fetches.Load() != 3 {
		t.Fatalf("within-ttl read re-fetched: fetches=%d, want 3", fetches.Load())
	}
}
