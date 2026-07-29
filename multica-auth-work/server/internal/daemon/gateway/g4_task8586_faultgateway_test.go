package gateway

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Task 8.5/8.6 non-destructive fault-gateway (F1) acceptance — production
// Coordinator driven against a local httptest fault server (no live provider,
// no account/credential state). Deterministic sleep hook (no real backoff).
// Component evidence only — not live OmniRoute proof.

func newFaultCoordinator(t *testing.T) *Coordinator {
	t.Helper()
	c, err := NewCoordinatorFromConfig(CoordinatorConfig{
		Policy: RetryPolicy{MaxAttempts: 3, EndToEndDeadline: 30 * time.Second, PreCommitOnly: true},
		now:    time.Now,
		sleep:  func(ctx context.Context, _ time.Duration) error { return ctx.Err() },
	})
	if err != nil {
		t.Fatalf("NewCoordinatorFromConfig: %v", err)
	}
	return c
}

// faultHTTPClient disables keep-alives so a canceled request deterministically
// closes the TCP connection, letting the server observe the disconnect (its
// request context fires). This avoids the pooled-connection cancellation hang.
var faultHTTPClient = &http.Client{Transport: &http.Transport{DisableKeepAlives: true}}

// faultGet issues a ctx-bound GET, drains and closes the body, returns status.
func faultGet(ctx context.Context, url string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	resp, err := faultHTTPClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, nil
}

// F1: pre-output 503 then success — safe retry before any output.
func TestTask8586FaultPreOutput503ThenRetrySucceeds(t *testing.T) {
	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := newFaultCoordinator(t)
	res, err := c.Execute(context.Background(), "f1", func(ctx context.Context, _ int) (AttemptResult, error) {
		status, doErr := faultGet(ctx, srv.URL)
		if doErr != nil {
			return AttemptResult{}, ClassifyFailure(FailureSignal{Timeout: true}).AsError("f1", FailureSignal{Timeout: true})
		}
		if status != http.StatusOK {
			sig := FailureSignal{StatusCode: status}
			return AttemptResult{}, ClassifyFailure(sig).AsError("f1", sig) // 503 -> retryable, pre-output
		}
		return AttemptResult{OutputCommitted: true}, nil
	})
	if err != nil || res.Attempts != 2 || hits.Load() != 2 {
		t.Fatalf("F1: attempts=%d hits=%d err=%v want 2/2/nil", res.Attempts, hits.Load(), err)
	}
}

// F2: partial stream (output) then broken connection — no replay.
func TestTask8586FaultPartialOutputThenBreakNoReplay(t *testing.T) {
	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			_, _ = w.Write([]byte("data: partial-chunk\n\n"))
			f.Flush()
		}
		if hj, ok := w.(http.Hijacker); ok { // break the stream abruptly after partial output
			if conn, _, err := hj.Hijack(); err == nil {
				_ = conn.Close()
			}
		}
	}))
	defer srv.Close()

	c := newFaultCoordinator(t)
	res, err := c.Execute(context.Background(), "f2", func(ctx context.Context, _ int) (AttemptResult, error) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
		resp, doErr := faultHTTPClient.Do(req)
		if doErr != nil {
			return AttemptResult{}, ClassifyFailure(FailureSignal{Timeout: true}).AsError("f2", FailureSignal{Timeout: true})
		}
		defer resp.Body.Close()
		buf := make([]byte, 64)
		n, _ := resp.Body.Read(buf)
		committed := n > 0 && strings.Contains(string(buf[:n]), "partial")
		_, readErr := io.Copy(io.Discard, resp.Body) // stream breaks here
		if readErr != nil || committed {
			sig := FailureSignal{StatusCode: http.StatusInternalServerError} // retryable class
			return AttemptResult{OutputCommitted: committed}, ClassifyFailure(sig).AsError("f2", sig)
		}
		return AttemptResult{OutputCommitted: true}, nil
	})
	// Retryable class, but output was committed -> Coordinator must NOT replay.
	if err == nil || res.Attempts != 1 || hits.Load() != 1 {
		t.Fatalf("F2: replayed after partial output: attempts=%d hits=%d err=%v", res.Attempts, hits.Load(), err)
	}
}

// F3: concurrent requests sharing a request-id — exactly one upstream hit.
func TestTask8586FaultConcurrentDedupOneUpstreamHit(t *testing.T) {
	var hits atomic.Int64
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		select {
		case started <- struct{}{}:
		default:
		}
		<-release
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := newFaultCoordinator(t)
	call := func() (ExecutionResult, error) {
		return c.Execute(context.Background(), "f3-shared", func(ctx context.Context, _ int) (AttemptResult, error) {
			status, doErr := faultGet(ctx, srv.URL)
			if doErr != nil || status != http.StatusOK {
				return AttemptResult{}, ClassifyFailure(FailureSignal{StatusCode: http.StatusInternalServerError}).AsError("f3", FailureSignal{StatusCode: http.StatusInternalServerError})
			}
			return AttemptResult{OutputCommitted: true}, nil
		})
	}
	leaderDone := make(chan ExecutionResult, 1)
	go func() { r, _ := call(); leaderDone <- r }()
	<-started // leader has hit upstream and is blocked

	const followers = 24
	followerDone := make(chan ExecutionResult, followers)
	for i := 0; i < followers; i++ {
		go func() { r, _ := call(); followerDone <- r }()
	}
	waitFor(t, func() bool { return c.FollowerSlots() == followers })
	close(release)

	if leader := <-leaderDone; leader.Deduplicated || leader.Attempts != 1 {
		t.Fatalf("F3 leader: %+v", leader)
	}
	for i := 0; i < followers; i++ {
		if r := <-followerDone; !r.Deduplicated {
			t.Fatalf("F3 follower not deduplicated: %+v", r)
		}
	}
	if hits.Load() != 1 {
		t.Fatalf("F3: upstream hit %d times, want exactly 1", hits.Load())
	}
	waitFor(t, func() bool { return c.ActiveSlots() == 0 && c.FollowerSlots() == 0 })
}

// F4: cancel a held stream — upstream request is canceled, slot returns to zero,
// and a fresh resubmission (id not poisoned) succeeds.
func TestTask8586FaultCancelHeldStreamThenResubmit(t *testing.T) {
	var hits atomic.Int64
	arrived := make(chan struct{}, 1)
	firstCanceled := make(chan struct{})
	stop := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) == 1 {
			select {
			case arrived <- struct{}{}:
			default:
			}
			select {
			case <-r.Context().Done(): // client canceled -> upstream request aborted
				close(firstCanceled)
			case <-stop: // cleanup fallback: never outlive the test
			}
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()
	defer close(stop) // LIFO: closes before srv.Close so no handler blocks Close

	c := newFaultCoordinator(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := c.Execute(ctx, "f4", func(runCtx context.Context, _ int) (AttemptResult, error) {
			status, doErr := faultGet(runCtx, srv.URL)
			if doErr != nil {
				return AttemptResult{}, runCtx.Err() // canceled upstream call
			}
			if status != http.StatusOK {
				return AttemptResult{}, ClassifyFailure(FailureSignal{StatusCode: status}).AsError("f4", FailureSignal{StatusCode: status})
			}
			return AttemptResult{OutputCommitted: true}, nil
		})
		done <- err
	}()

	<-arrived // deterministic: upstream received the held request
	waitFor(t, func() bool { return c.ActiveSlots() == 1 })
	cancel()
	<-firstCanceled // deterministic: the upstream request's context was canceled
	if err := <-done; !IsErrorClass(err, ErrorCancelled) {
		t.Fatalf("F4: expected cancelled, got %v", err)
	}
	waitFor(t, func() bool { return c.ActiveSlots() == 0 })

	// Fresh resubmission of the same id must run (not poisoned) and succeed.
	res, err := c.Execute(context.Background(), "f4", func(ctx context.Context, _ int) (AttemptResult, error) {
		status, doErr := faultGet(ctx, srv.URL)
		if doErr != nil || status != http.StatusOK {
			return AttemptResult{}, ClassifyFailure(FailureSignal{StatusCode: http.StatusInternalServerError}).AsError("f4", FailureSignal{StatusCode: http.StatusInternalServerError})
		}
		return AttemptResult{OutputCommitted: true}, nil
	})
	if err != nil || res.Deduplicated || res.Attempts != 1 {
		t.Fatalf("F4 resubmit: %+v err=%v", res, err)
	}
	if hits.Load() < 2 {
		t.Fatalf("F4: upstream hits=%d want >=2 (canceled + resubmit)", hits.Load())
	}
}
