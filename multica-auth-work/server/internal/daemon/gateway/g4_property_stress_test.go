package gateway

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"runtime"
	"sort"
	"sync"
	"testing"
	"time"
)

const (
	g4RoundRobinSeed   int64 = 0x440401
	g4AffinitySeed     int64 = 0x440402
	g4CancellationSeed int64 = 0x440603
	g4RetrySeed        int64 = 0x440604
	g4CircuitSeed      int64 = 0x440505
	g4LifecycleSeed    int64 = 0x440706
)

func TestG4PropertyStrictRoundRobinFixedSeed(t *testing.T) {
	random := rand.New(rand.NewSource(g4RoundRobinSeed))
	const iterations = 24
	for iteration := 0; iteration < iterations; iteration++ {
		accountCount := 2 + random.Intn(7)
		requestCount := 96 + random.Intn(65)
		accounts := make([]string, accountCount)
		for index := range accounts {
			accounts[index] = fmt.Sprintf("synthetic-account-%02d", index)
		}
		router := newSyntheticRouter(accounts...)
		start := make(chan struct{})
		selections := make(chan syntheticRouteSelection, requestCount)
		var workers sync.WaitGroup
		for requestIndex := 0; requestIndex < requestCount; requestIndex++ {
			workers.Add(1)
			go func() {
				defer workers.Done()
				<-start
				selection, err := router.selectRoute(syntheticRouteRequest{})
				if err != nil {
					selections <- syntheticRouteSelection{}
					return
				}
				selections <- selection
			}()
		}
		close(start)
		workers.Wait()
		close(selections)
		ordered := make([]syntheticRouteSelection, 0, requestCount)
		for selection := range selections {
			ordered = append(ordered, selection)
		}
		sort.Slice(ordered, func(left, right int) bool { return ordered[left].Sequence < ordered[right].Sequence })
		if len(ordered) != requestCount {
			t.Fatalf("seed=%d iteration=%d selection count mismatch", g4RoundRobinSeed, iteration)
		}
		for index, selection := range ordered {
			if selection.Sequence != uint64(index+1) || selection.Account != accounts[index%accountCount] {
				t.Fatalf("seed=%d iteration=%d strict sequence mismatch at %d", g4RoundRobinSeed, iteration, index)
			}
		}
	}
}

func TestG4PropertyContinuationAffinityFixedSeed(t *testing.T) {
	random := rand.New(rand.NewSource(g4AffinitySeed))
	router := newSyntheticRouter("synthetic-account-a", "synthetic-account-b", "synthetic-account-c", "synthetic-account-d")
	const affinityKeys = 48
	const repeatsPerKey = 32
	type affinityFixture struct {
		request syntheticRouteRequest
		account string
	}
	fixtures := make([]affinityFixture, affinityKeys)
	jobs := make([]int, 0, affinityKeys*repeatsPerKey)
	for index := range fixtures {
		identifier := fmt.Sprintf("synthetic-affinity-%02d", index)
		request := syntheticAffinityRequest(index%3, identifier)
		origin, err := router.selectRoute(request)
		if err != nil {
			t.Fatal(err)
		}
		fixtures[index] = affinityFixture{request: request, account: origin.Account}
		for repeat := 0; repeat < repeatsPerKey; repeat++ {
			jobs = append(jobs, index)
		}
	}
	random.Shuffle(len(jobs), func(left, right int) { jobs[left], jobs[right] = jobs[right], jobs[left] })
	start := make(chan struct{})
	failures := make(chan string, len(jobs))
	var workers sync.WaitGroup
	for _, fixtureIndex := range jobs {
		fixtureIndex := fixtureIndex
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			selection, err := router.selectRoute(fixtures[fixtureIndex].request)
			if err != nil || selection.Account != fixtures[fixtureIndex].account {
				failures <- fmt.Sprintf("fixture-%d", fixtureIndex)
			}
		}()
	}
	close(start)
	workers.Wait()
	close(failures)
	if failure, exists := <-failures; exists {
		t.Fatalf("seed=%d affinity mismatch for %s", g4AffinitySeed, failure)
	}

	for kind := 0; kind < 3; kind++ {
		edgeRouter := newSyntheticRouter("synthetic-account-a", "synthetic-account-b", "synthetic-account-c")
		request := syntheticAffinityRequest(kind, fmt.Sprintf("synthetic-rebind-%d", kind))
		origin, err := edgeRouter.selectRoute(request)
		if err != nil {
			t.Fatal(err)
		}
		edgeRouter.setStatus(origin.Account, syntheticQuarantined)
		replacement, err := edgeRouter.selectRoute(request)
		if err != nil || replacement.Account == origin.Account {
			t.Fatalf("seed=%d affinity did not rebind from an ineligible owner", g4AffinitySeed)
		}
		edgeRouter.setStatus(origin.Account, syntheticEligible)
		continued, err := edgeRouter.selectRoute(request)
		if err != nil || continued.Account != replacement.Account {
			t.Fatalf("seed=%d stale affinity owner reclaimed continuation", g4AffinitySeed)
		}
	}

	firstRouter := newSyntheticRouter("synthetic-account-a", "synthetic-account-b", "synthetic-account-c")
	firstRequest := syntheticAffinityRequest(random.Intn(3), "synthetic-concurrent-first")
	const concurrentFirstRequests = 128
	firstSelections := runSyntheticSelections(t, firstRouter, firstRequest, concurrentFirstRequests)
	for _, selection := range firstSelections[1:] {
		if selection.Account != firstSelections[0].Account {
			t.Fatalf("seed=%d concurrent first affinity split owners", g4AffinitySeed)
		}
	}
	nextIndependent, err := firstRouter.selectRoute(syntheticRouteRequest{})
	if err != nil || nextIndependent.Account != "synthetic-account-b" {
		t.Fatalf("seed=%d affinity continuations advanced independent rotation", g4AffinitySeed)
	}
}

func syntheticAffinityRequest(kind int, identifier string) syntheticRouteRequest {
	switch kind {
	case 0:
		return syntheticRouteRequest{PreviousResponseID: identifier}
	case 1:
		return syntheticRouteRequest{PromptCacheID: identifier}
	default:
		return syntheticRouteRequest{ToolTurnID: identifier}
	}
}

func runSyntheticSelections(t *testing.T, router *syntheticRouter, request syntheticRouteRequest, count int) []syntheticRouteSelection {
	t.Helper()
	start := make(chan struct{})
	results := make(chan syntheticRouteSelection, count)
	var workers sync.WaitGroup
	for index := 0; index < count; index++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			selection, err := router.selectRoute(request)
			if err != nil {
				results <- syntheticRouteSelection{}
				return
			}
			results <- selection
		}()
	}
	close(start)
	workers.Wait()
	close(results)
	selections := make([]syntheticRouteSelection, 0, count)
	for selection := range results {
		if selection.Sequence == 0 {
			t.Fatal("synthetic selection failed")
		}
		selections = append(selections, selection)
	}
	return selections
}

func TestG4PropertyCancellationReleasesCapacityFixedSeed(t *testing.T) {
	random := rand.New(rand.NewSource(g4CancellationSeed))
	harness := newSyntheticRetryHarness()
	const requests = 384
	cancels := make([]context.CancelFunc, requests)
	done := make(chan error, requests)
	for index := 0; index < requests; index++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancels[index] = cancel
		started := make(chan struct{})
		go func() { done <- harness.waitForCancellation(ctx, started) }()
		<-started
	}
	if harness.active.Load() != requests {
		t.Fatalf("seed=%d expected %d acquired slots, got %d", g4CancellationSeed, requests, harness.active.Load())
	}
	random.Shuffle(len(cancels), func(left, right int) { cancels[left], cancels[right] = cancels[right], cancels[left] })
	for _, cancel := range cancels {
		cancel()
	}
	for index := 0; index < requests; index++ {
		if err := <-done; !IsErrorClass(err, ErrorCancelled) {
			t.Fatalf("seed=%d cancellation classification mismatch", g4CancellationSeed)
		}
	}
	if harness.active.Load() != 0 {
		t.Fatalf("seed=%d cancellation leaked %d slots", g4CancellationSeed, harness.active.Load())
	}
}

func TestG4PropertyRetryPreCommitBoundaryFixedSeed(t *testing.T) {
	random := rand.New(rand.NewSource(g4RetrySeed))
	harness := newSyntheticRetryHarness()
	const cases = 512
	type retryCase struct {
		id               string
		attempts         []syntheticAttempt
		expectedAttempts int
		expectError      bool
	}
	fixtures := make([]retryCase, cases)
	retryableFailures := []syntheticFailureKind{syntheticScoped429, syntheticServer5xx, syntheticTimeout}
	for index := range fixtures {
		failureCount := 1 + random.Intn(5)
		attempts := make([]syntheticAttempt, failureCount, failureCount+1)
		for attemptIndex := range attempts {
			attempts[attemptIndex].Failure = retryableFailures[random.Intn(len(retryableFailures))]
		}
		commitIndex := -1
		if random.Intn(2) == 0 {
			commitIndex = random.Intn(failureCount)
			if random.Intn(2) == 0 {
				attempts[commitIndex].OutputCommitted = true
			} else {
				attempts[commitIndex].ToolActionCommitted = true
			}
		}
		fixture := retryCase{id: fmt.Sprintf("synthetic-property-retry-%03d", index), attempts: attempts}
		if commitIndex >= 0 {
			fixture.expectedAttempts = commitIndex + 1
			fixture.expectError = true
		} else {
			fixture.attempts = append(fixture.attempts, syntheticAttempt{})
			fixture.expectedAttempts = failureCount + 1
		}
		fixtures[index] = fixture
	}
	start := make(chan struct{})
	failures := make(chan int, cases)
	var workers sync.WaitGroup
	for index := range fixtures {
		index := index
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			fixture := fixtures[index]
			result, err := harness.execute(context.Background(), fixture.id, fixture.attempts)
			if result.Attempts != fixture.expectedAttempts || (err != nil) != fixture.expectError {
				failures <- index
			}
		}()
	}
	close(start)
	workers.Wait()
	close(failures)
	if index, exists := <-failures; exists {
		t.Fatalf("seed=%d retry boundary mismatch at case %d", g4RetrySeed, index)
	}
	if harness.active.Load() != 0 {
		t.Fatalf("seed=%d retry harness leaked capacity", g4RetrySeed)
	}

	started := make(chan struct{}, 1)
	release := make(chan struct{})
	leaderDone := make(chan error, 1)
	go func() {
		_, err := harness.execute(context.Background(), "synthetic-concurrent-dedup", []syntheticAttempt{{Started: started, Release: release}})
		leaderDone <- err
	}()
	<-started
	const duplicates = 128
	duplicateDone := make(chan syntheticExecution, duplicates)
	for index := 0; index < duplicates; index++ {
		go func() {
			result, _ := harness.execute(context.Background(), "synthetic-concurrent-dedup", []syntheticAttempt{{}})
			duplicateDone <- result
		}()
	}
	for spins := 0; harness.followers.Load() != duplicates && spins < 100000; spins++ {
		runtime.Gosched()
	}
	if harness.followers.Load() != duplicates {
		t.Fatalf("seed=%d concurrent duplicates did not join the in-flight request", g4RetrySeed)
	}
	close(release)
	if err := <-leaderDone; err != nil {
		t.Fatalf("seed=%d dedup leader failed: %v", g4RetrySeed, err)
	}
	for index := 0; index < duplicates; index++ {
		if result := <-duplicateDone; !result.Deduplicated || result.Attempts != 0 {
			t.Fatalf("seed=%d duplicate request executed independently", g4RetrySeed)
		}
	}
	if harness.active.Load() != 0 || harness.followers.Load() != 0 {
		t.Fatalf("seed=%d dedup accounting did not drain", g4RetrySeed)
	}
}

type syntheticCircuit struct {
	mu                sync.Mutex
	state             CircuitState
	failures          int
	threshold         int
	openDuration      time.Duration
	halfOpenMaxProbes int
	openedAt          time.Time
	halfOpenInFlight  int
}

func newSyntheticCircuit(threshold int, openDuration time.Duration, halfOpenMaxProbes int) *syntheticCircuit {
	return &syntheticCircuit{
		state: CircuitClosed, threshold: threshold, openDuration: openDuration,
		halfOpenMaxProbes: halfOpenMaxProbes,
	}
}

func (c *syntheticCircuit) allow(now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state == CircuitOpen && !now.Before(c.openedAt.Add(c.openDuration)) {
		c.state = CircuitHalfOpen
		c.halfOpenInFlight = 0
	}
	switch c.state {
	case CircuitClosed:
		return true
	case CircuitHalfOpen:
		if c.halfOpenInFlight >= c.halfOpenMaxProbes {
			return false
		}
		c.halfOpenInFlight++
		return true
	default:
		return false
	}
}

func (c *syntheticCircuit) record(now time.Time, success bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch c.state {
	case CircuitClosed:
		if success {
			c.failures = 0
			return
		}
		c.failures++
		if c.failures >= c.threshold {
			c.state = CircuitOpen
			c.openedAt = now
		}
	case CircuitHalfOpen:
		if c.halfOpenInFlight > 0 {
			c.halfOpenInFlight--
		}
		if success {
			c.state = CircuitClosed
			c.failures = 0
			c.halfOpenInFlight = 0
			return
		}
		c.state = CircuitOpen
		c.failures = c.threshold
		c.openedAt = now
		c.halfOpenInFlight = 0
	}
}

func (c *syntheticCircuit) snapshot() (CircuitState, int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state, c.halfOpenInFlight
}

func TestG4PropertyCircuitTransitionsFixedSeed(t *testing.T) {
	random := rand.New(rand.NewSource(g4CircuitSeed))
	base := time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)
	const iterations = 64
	for iteration := 0; iteration < iterations; iteration++ {
		threshold := 2 + random.Intn(5)
		openDuration := time.Duration(10+random.Intn(91)) * time.Millisecond
		maxProbes := 1 + random.Intn(8)
		clock := base.Add(time.Duration(iteration) * time.Hour)
		circuit := newSyntheticCircuit(threshold, openDuration, maxProbes)
		for failure := 0; failure < threshold; failure++ {
			if !circuit.allow(clock) {
				t.Fatalf("seed=%d iteration=%d circuit opened early", g4CircuitSeed, iteration)
			}
			circuit.record(clock, false)
		}
		if state, _ := circuit.snapshot(); state != CircuitOpen || circuit.allow(clock.Add(openDuration-time.Nanosecond)) {
			t.Fatalf("seed=%d iteration=%d open circuit transition mismatch", g4CircuitSeed, iteration)
		}

		start := make(chan struct{})
		allowed := make(chan bool, maxProbes+32)
		var workers sync.WaitGroup
		for probe := 0; probe < maxProbes+32; probe++ {
			workers.Add(1)
			go func() {
				defer workers.Done()
				<-start
				allowed <- circuit.allow(clock.Add(openDuration))
			}()
		}
		close(start)
		workers.Wait()
		close(allowed)
		allowedCount := 0
		for accepted := range allowed {
			if accepted {
				allowedCount++
			}
		}
		if allowedCount != maxProbes {
			t.Fatalf("seed=%d iteration=%d half-open probes=%d want=%d", g4CircuitSeed, iteration, allowedCount, maxProbes)
		}
		circuit.record(clock.Add(openDuration), true)
		if state, inFlight := circuit.snapshot(); state != CircuitClosed || inFlight != 0 {
			t.Fatalf("seed=%d iteration=%d successful probe did not close circuit", g4CircuitSeed, iteration)
		}

		for failure := 0; failure < threshold; failure++ {
			if !circuit.allow(clock.Add(2 * openDuration)) {
				t.Fatalf("seed=%d iteration=%d circuit failed to reopen setup", g4CircuitSeed, iteration)
			}
			circuit.record(clock.Add(2*openDuration), false)
		}
		if !circuit.allow(clock.Add(3 * openDuration)) {
			t.Fatalf("seed=%d iteration=%d half-open probe was not admitted", g4CircuitSeed, iteration)
		}
		circuit.record(clock.Add(3*openDuration), false)
		if state, inFlight := circuit.snapshot(); state != CircuitOpen || inFlight != 0 {
			t.Fatalf("seed=%d iteration=%d failed probe did not reopen circuit", g4CircuitSeed, iteration)
		}
	}
}

func TestG4PropertyAccountLifecycleFixedSeed(t *testing.T) {
	random := rand.New(rand.NewSource(g4LifecycleSeed))
	router := newSyntheticRouter("synthetic-account-00", "synthetic-account-01", "synthetic-account-02")
	baseline := router.snapshot()
	accountCount := 3
	const iterations = 128
	for iteration := 0; iteration < iterations; iteration++ {
		action := random.Intn(6)
		snapshot := router.snapshot()
		account := snapshot.Order[random.Intn(len(snapshot.Order))]
		switch action {
		case 0:
			router.setStatus(account, syntheticQuarantined)
		case 1:
			router.setStatus(account, syntheticRemoved)
		case 2:
			router.setStatus(account, syntheticEligible)
		case 3:
			if accountCount < 10 {
				router.add(fmt.Sprintf("synthetic-account-%02d", accountCount))
				accountCount++
			} else {
				router.add(account)
			}
		case 4:
			router.stop()
			assertSyntheticUnavailableBatch(t, router, 32)
			router.restart(fmt.Sprintf("synthetic-version-%03d", iteration))
		case 5:
			router.rollback(baseline)
			accountCount = len(baseline.Order)
		}
		assertSyntheticLifecycleBatch(t, router, 32)
	}
	router.rollback(baseline)
	final := router.snapshot()
	if final.ConfigVersion != baseline.ConfigVersion || fmt.Sprint(final.Order) != fmt.Sprint(baseline.Order) {
		t.Fatalf("seed=%d final rollback mismatch", g4LifecycleSeed)
	}
}

func assertSyntheticUnavailableBatch(t *testing.T, router *syntheticRouter, count int) {
	t.Helper()
	start := make(chan struct{})
	errorsSeen := make(chan error, count)
	var workers sync.WaitGroup
	for index := 0; index < count; index++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			_, err := router.selectRoute(syntheticRouteRequest{})
			errorsSeen <- err
		}()
	}
	close(start)
	workers.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		if !errors.Is(err, errSyntheticRouterUnavailable) {
			t.Fatal("stopped synthetic router admitted a request")
		}
	}
}

func assertSyntheticLifecycleBatch(t *testing.T, router *syntheticRouter, count int) {
	t.Helper()
	expected := router.snapshot()
	eligible := make(map[string]struct{})
	for account, state := range expected.Status {
		if state == syntheticEligible {
			eligible[account] = struct{}{}
		}
	}
	start := make(chan struct{})
	results := make(chan syntheticRouteSelection, count)
	errorsSeen := make(chan error, count)
	var workers sync.WaitGroup
	for index := 0; index < count; index++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			selection, err := router.selectRoute(syntheticRouteRequest{})
			results <- selection
			errorsSeen <- err
		}()
	}
	close(start)
	workers.Wait()
	close(results)
	close(errorsSeen)
	for err := range errorsSeen {
		if len(eligible) == 0 {
			if !errors.Is(err, errSyntheticRouterUnavailable) {
				t.Fatal("empty synthetic pool did not fail closed")
			}
		} else if err != nil {
			t.Fatalf("eligible synthetic pool rejected a request: %v", err)
		}
	}
	selections := make([]syntheticRouteSelection, 0, count)
	for selection := range results {
		if len(eligible) == 0 {
			if selection.Sequence != 0 || selection.Account != "" {
				t.Fatal("empty synthetic pool returned a route")
			}
			continue
		}
		if _, exists := eligible[selection.Account]; !exists {
			t.Fatalf("ineligible synthetic account selected: %s", selection.Account)
		}
		selections = append(selections, selection)
	}
	if len(selections) == 0 {
		return
	}
	sort.Slice(selections, func(left, right int) bool { return selections[left].Sequence < selections[right].Sequence })
	cursor := expected.Cursor
	for index, selection := range selections {
		if index > 0 && selection.Sequence != selections[index-1].Sequence+1 {
			t.Fatal("synthetic lifecycle batch lost an atomic selection sequence")
		}
		matched := false
		for offset := 0; offset < len(expected.Order); offset++ {
			candidateIndex := (cursor + offset) % len(expected.Order)
			candidate := expected.Order[candidateIndex]
			if expected.Status[candidate] != syntheticEligible {
				continue
			}
			if selection.Account != candidate {
				t.Fatalf("synthetic lifecycle round-robin mismatch: got %s want %s", selection.Account, candidate)
			}
			cursor = (candidateIndex + 1) % len(expected.Order)
			matched = true
			break
		}
		if !matched {
			t.Fatal("synthetic lifecycle reference model found no eligible account")
		}
	}
}
