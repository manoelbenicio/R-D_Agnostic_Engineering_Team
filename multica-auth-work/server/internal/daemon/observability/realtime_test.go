package observability

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeMonotonicClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeMonotonicClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeMonotonicClock) Advance(duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(duration)
}

func TestRealtimeRecorderDeterministicOrderingAndMonotonicTimings(t *testing.T) {
	clock := &fakeMonotonicClock{now: time.Unix(100, 0)}
	recorder := newRealtimeRecorder(clock)

	request, err := recorder.BeginLatency(LatencyRequest)
	if err != nil {
		t.Fatalf("begin request latency: %v", err)
	}
	clock.Advance(15 * time.Millisecond)
	if _, err := request.End(); err != nil {
		t.Fatalf("end request latency: %v", err)
	}
	if _, err := request.End(); err == nil {
		t.Fatal("duplicate latency terminal was accepted")
	}

	selection, err := recorder.BeginLatency(LatencySelection)
	if err != nil {
		t.Fatalf("begin selection latency: %v", err)
	}
	clock.Advance(3 * time.Millisecond)
	if _, err := selection.End(); err != nil {
		t.Fatalf("end selection latency: %v", err)
	}

	if err := recorder.ObserveQueueDepth(3); err != nil {
		t.Fatalf("observe queue: %v", err)
	}
	clock.Advance(2 * time.Millisecond)
	if err := recorder.ObserveQueueDepth(1); err != nil {
		t.Fatalf("observe queue recovery: %v", err)
	}

	cancellation := recorder.BeginCancellationRelease()
	clock.Advance(7 * time.Millisecond)
	if _, err := cancellation.End(); err != nil {
		t.Fatalf("end cancellation: %v", err)
	}
	recovery := recorder.BeginRecovery()
	clock.Advance(11 * time.Millisecond)
	if _, err := recovery.End(); err != nil {
		t.Fatalf("end recovery: %v", err)
	}

	for _, input := range []FairnessInput{
		{Slot: "slot-2", EligibleDuration: 5 * time.Second, IndependentSelections: 2, AffinityExclusions: 1},
		{Slot: "slot-1", EligibleDuration: 6 * time.Second, IndependentSelections: 3},
		{Slot: "slot-2", EligibleDuration: time.Second, IndependentSelections: 1},
	} {
		if err := recorder.ObserveFairness(input); err != nil {
			t.Fatalf("observe fairness %+v: %v", input, err)
		}
	}

	result, err := recorder.Snapshot()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if len(result.Latencies) != 2 || result.Latencies[0].Kind != LatencyRequest || result.Latencies[1].Kind != LatencySelection {
		t.Fatalf("latencies are not in canonical lexical order: %+v", result.Latencies)
	}
	if result.PeakQueueDepth != 3 || len(result.CancellationRelease) != 1 || result.CancellationRelease[0] != 7*time.Millisecond ||
		len(result.Recovery) != 1 || result.Recovery[0] != 11*time.Millisecond {
		t.Fatalf("unexpected queue/lifecycle result: %+v", result)
	}
	if len(result.FairnessInputs) != 2 || result.FairnessInputs[0].Slot != "slot-1" ||
		result.FairnessInputs[1].Slot != "slot-2" || result.FairnessInputs[1].EligibleDuration != 6*time.Second ||
		result.FairnessInputs[1].IndependentSelections != 3 {
		t.Fatalf("unexpected canonical fairness inputs: %+v", result.FairnessInputs)
	}
}

func TestRealtimeRecorderRejectsIdentityAndContentFieldsAreAbsent(t *testing.T) {
	recorder := NewRealtimeRecorder()
	if err := recorder.ObserveFairness(FairnessInput{Slot: "personal account", IndependentSelections: 1}); err == nil {
		t.Fatal("personal fairness identity was accepted")
	}
	if err := recorder.ObserveQueueDepth(-1); err == nil {
		t.Fatal("negative queue depth was accepted")
	}
	if _, err := recorder.BeginLatency("provider-call"); err == nil {
		t.Fatal("free-form latency kind was accepted")
	}
	result, err := recorder.Snapshot()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	lower := strings.ToLower(string(data))
	for _, forbidden := range []string{
		"authorization", "credential", "secret", "cookie", "account_identity",
		"prompt", "completion", "message_content", "tool_payload", "repository_content", "reasoning_content",
	} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("content-off measurement contains forbidden field %q", forbidden)
		}
	}
}

func TestRealtimeRecorderFreezesAndBoundsWriters(t *testing.T) {
	recorder := NewRealtimeRecorder()
	for index := 0; index < MaxQueueObservations; index++ {
		if err := recorder.ObserveQueueDepth(index); err != nil {
			t.Fatalf("queue observation %d: %v", index, err)
		}
	}
	if err := recorder.ObserveQueueDepth(1); err == nil {
		t.Fatal("queue overflow was silently accepted")
	}
	if _, err := recorder.Snapshot(); err != nil {
		t.Fatalf("freeze snapshot: %v", err)
	}
	if err := recorder.ObserveQueueDepth(0); err == nil {
		t.Fatal("writer appended after snapshot freeze")
	}
	if _, err := recorder.Snapshot(); err == nil {
		t.Fatal("second snapshot was accepted after close")
	}
}

func TestRealtimeRecorderFairnessKeyOverflowStops(t *testing.T) {
	recorder := NewRealtimeRecorder()
	for index := 0; index < MaxFairnessKeys; index++ {
		if err := recorder.ObserveFairness(FairnessInput{Slot: "slot-" + strconv.Itoa(index), IndependentSelections: 1}); err != nil {
			t.Fatalf("fairness observation %d: %v", index, err)
		}
	}
	if err := recorder.ObserveFairness(FairnessInput{Slot: "slot-overflow", IndependentSelections: 1}); err == nil {
		t.Fatal("fairness key overflow was silently accepted")
	}
}
