package gateway

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

func validModelsDocument() ModelsDocument {
	return ModelsDocument{
		Object:          "list",
		RegistryVersion: "synthetic-registry-v1",
		Models: []ModelDocument{
			{
				ID: "agy/claude-opus-4-6-thinking", Protocol: string(brain.ProtocolAnthropicMessages),
				Streaming: boolPointer(true), Tools: boolPointer(true), Reasoning: boolPointer(true), StructuredOutput: boolPointer(false),
				ContextLimit: 200000, AccountPool: "pool-synthetic", Rotation: string(RotationStrictIndependentRequest),
				Affinity: string(AffinityOriginAccount), Available: boolPointer(true),
			},
		},
	}
}

func TestRegistryCachesAndValidatesCapabilities(t *testing.T) {
	var fetches atomic.Int64
	registry, err := NewRegistry(ModelsFetchFunc(func(context.Context) (ModelsDocument, error) {
		fetches.Add(1)
		return validModelsDocument(), nil
	}), time.Minute)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}

	model := brain.RouteModel("agy/claude-opus-4-6-thinking")
	var wait sync.WaitGroup
	for range 16 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if _, lookupErr := registry.LookupModel(context.Background(), model); lookupErr != nil {
				t.Errorf("LookupModel: %v", lookupErr)
			}
		}()
	}
	wait.Wait()
	if fetches.Load() != 1 {
		t.Fatalf("expected one cached fetch, got %d", fetches.Load())
	}
	if err := registry.ValidateCapability(context.Background(), model, CapabilityRequirement{
		Protocol: brain.ProtocolAnthropicMessages, Streaming: true, Tools: true, Reasoning: true, MinimumContext: 100000,
	}); err != nil {
		t.Fatalf("ValidateCapability accepted row: %v", err)
	}
	if err := registry.ValidateCapability(context.Background(), model, CapabilityRequirement{StructuredOutput: true}); !IsErrorClass(err, ErrorCapability) {
		t.Fatalf("expected structured-output rejection, got %v", err)
	}
	if _, err := registry.LookupModel(context.Background(), brain.RouteModel("synthetic/missing")); !IsErrorClass(err, ErrorUnknownModel) {
		t.Fatalf("expected unknown-model error, got %v", err)
	}
}

func TestRegistryRejectsIncompleteOrUnversionedRows(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ModelsDocument)
	}{
		{name: "missing version", mutate: func(document *ModelsDocument) { document.RegistryVersion = "" }},
		{name: "missing capability", mutate: func(document *ModelsDocument) { document.Models[0].Tools = nil }},
		{name: "missing context", mutate: func(document *ModelsDocument) { document.Models[0].ContextLimit = 0 }},
		{name: "missing pool", mutate: func(document *ModelsDocument) { document.Models[0].AccountPool = "" }},
		{name: "unknown protocol", mutate: func(document *ModelsDocument) { document.Models[0].Protocol = "synthetic-unknown" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document := validModelsDocument()
			test.mutate(&document)
			registry, err := NewRegistry(ModelsFetchFunc(func(context.Context) (ModelsDocument, error) { return document, nil }), time.Minute)
			if err != nil {
				t.Fatalf("NewRegistry: %v", err)
			}
			_, err = registry.Snapshot(context.Background())
			if !IsErrorClass(err, ErrorProtocol) {
				t.Fatalf("expected protocol rejection, got %v", err)
			}
		})
	}
}

func TestRegistryWaiterCanCancel(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	registry, err := NewRegistry(ModelsFetchFunc(func(context.Context) (ModelsDocument, error) {
		close(started)
		<-release
		return validModelsDocument(), nil
	}), time.Minute)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	firstDone := make(chan error, 1)
	go func() {
		_, snapshotErr := registry.Snapshot(context.Background())
		firstDone <- snapshotErr
	}()
	<-started
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = registry.Snapshot(ctx)
	if !IsErrorClass(err, ErrorCancelled) {
		t.Fatalf("expected cancelled waiter, got %v", err)
	}
	close(release)
	if err := <-firstDone; err != nil {
		t.Fatalf("first refresh failed: %v", err)
	}
}

func TestRegistryDoesNotExposeMutableFallbackSlices(t *testing.T) {
	document := validModelsDocument()
	document.Models = append(document.Models, ModelDocument{
		ID: "synthetic/fallback", Protocol: string(brain.ProtocolAnthropicMessages),
		Streaming: boolPointer(true), Tools: boolPointer(true), Reasoning: boolPointer(false), StructuredOutput: boolPointer(false),
		ContextLimit: 1000, AccountPool: "pool-fallback", Rotation: string(RotationFailureOnly), Affinity: string(AffinityNone), Available: boolPointer(true),
	})
	document.Models[0].Fallback = []string{"synthetic/fallback"}
	registry, _ := NewRegistry(ModelsFetchFunc(func(context.Context) (ModelsDocument, error) { return document, nil }), time.Minute)
	first, err := registry.LookupModel(context.Background(), brain.RouteModel(document.Models[0].ID))
	if err != nil {
		t.Fatalf("LookupModel: %v", err)
	}
	first.Fallback[0] = "synthetic/mutated"
	second, err := registry.LookupModel(context.Background(), brain.RouteModel(document.Models[0].ID))
	if err != nil {
		t.Fatalf("LookupModel second: %v", err)
	}
	if second.Fallback[0] != "synthetic/fallback" {
		t.Fatal("cached registry exposed mutable fallback state")
	}
}

func TestRegistryPreservesFetcherError(t *testing.T) {
	expected := &GatewayError{Operation: operationModels, Class: ErrorAuthentication}
	registry, _ := NewRegistry(ModelsFetchFunc(func(context.Context) (ModelsDocument, error) {
		return ModelsDocument{}, expected
	}), time.Minute)
	_, err := registry.Snapshot(context.Background())
	if !errors.Is(err, expected) {
		t.Fatalf("expected deterministic fetch error, got %v", err)
	}
}

func TestRegistryRefreshFailureBackoffIsConcurrentAndClockBounded(t *testing.T) {
	expected := &GatewayError{Operation: operationModels, Class: ErrorUpstream, Retryable: true}
	var fetches atomic.Int64
	var fail atomic.Bool
	fail.Store(true)
	registry, err := NewRegistry(ModelsFetchFunc(func(context.Context) (ModelsDocument, error) {
		fetches.Add(1)
		if fail.Load() {
			return ModelsDocument{}, expected
		}
		return validModelsDocument(), nil
	}), time.Minute)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	now := time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)
	registry.now = func() time.Time { return now }

	const callers = 64
	start := make(chan struct{})
	errorsSeen := make(chan error, callers)
	var wait sync.WaitGroup
	for range callers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			_, snapshotErr := registry.Snapshot(context.Background())
			errorsSeen <- snapshotErr
		}()
	}
	close(start)
	wait.Wait()
	close(errorsSeen)
	for snapshotErr := range errorsSeen {
		if !errors.Is(snapshotErr, expected) {
			t.Fatalf("concurrent caller did not receive cached failure: %v", snapshotErr)
		}
	}
	if fetches.Load() != 1 {
		t.Fatalf("failure single-flight fetched %d times, want 1", fetches.Load())
	}

	for range 8 {
		if _, snapshotErr := registry.Snapshot(context.Background()); !errors.Is(snapshotErr, expected) {
			t.Fatalf("repeated caller did not receive cached failure: %v", snapshotErr)
		}
	}
	if fetches.Load() != 1 {
		t.Fatalf("negative cache allowed %d fetches inside backoff", fetches.Load())
	}

	now = now.Add(RegistryRefreshFailureBackoff - time.Nanosecond)
	if _, snapshotErr := registry.Snapshot(context.Background()); !errors.Is(snapshotErr, expected) {
		t.Fatalf("failure was not cached through backoff boundary: %v", snapshotErr)
	}
	if fetches.Load() != 1 {
		t.Fatalf("negative cache refreshed before boundary: %d fetches", fetches.Load())
	}

	now = now.Add(time.Nanosecond)
	fail.Store(false)
	snapshot, err := registry.Snapshot(context.Background())
	if err != nil || snapshot.Version != validModelsDocument().RegistryVersion {
		t.Fatalf("registry did not recover at backoff boundary: snapshot=%+v err=%v", snapshot, err)
	}
	if fetches.Load() != 2 {
		t.Fatalf("registry recovery fetched %d times, want 2", fetches.Load())
	}
}

func TestRegistryInvalidateFencesInFlightSuccess(t *testing.T) {
	firstStarted := make(chan struct{})
	firstRelease := make(chan struct{})
	var fetches atomic.Int64
	registry, err := NewRegistry(ModelsFetchFunc(func(context.Context) (ModelsDocument, error) {
		call := fetches.Add(1)
		if call == 1 {
			close(firstStarted)
			<-firstRelease
			document := validModelsDocument()
			document.RegistryVersion = "synthetic-stale-success"
			return document, nil
		}
		document := validModelsDocument()
		document.RegistryVersion = "synthetic-current-success"
		return document, nil
	}), time.Minute)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}

	results := make(chan RegistrySnapshot, 2)
	errorsSeen := make(chan error, 2)
	for range 2 {
		go func() {
			snapshot, snapshotErr := registry.Snapshot(context.Background())
			results <- snapshot
			errorsSeen <- snapshotErr
		}()
	}
	<-firstStarted
	registry.Invalidate()
	close(firstRelease)

	for range 2 {
		if snapshotErr := <-errorsSeen; snapshotErr != nil {
			t.Fatalf("invalidated caller received an error: %v", snapshotErr)
		}
		if snapshot := <-results; snapshot.Version != "synthetic-current-success" {
			t.Fatalf("stale in-flight success committed: version=%q", snapshot.Version)
		}
	}
	if fetches.Load() != 2 {
		t.Fatalf("invalidation performed %d fetches, want exactly 2", fetches.Load())
	}
	cached, err := registry.Snapshot(context.Background())
	if err != nil || cached.Version != "synthetic-current-success" || fetches.Load() != 2 {
		t.Fatalf("current generation was not cached: snapshot=%+v fetches=%d err=%v", cached, fetches.Load(), err)
	}
}

func TestRegistryInvalidateFencesInFlightFailureWithoutPoisoningNegativeCache(t *testing.T) {
	firstStarted := make(chan struct{})
	firstRelease := make(chan struct{})
	staleFailure := &GatewayError{Operation: operationModels, Class: ErrorAuthentication}
	currentFailure := &GatewayError{Operation: operationModels, Class: ErrorUpstream, Retryable: true}
	var fetches atomic.Int64
	registry, err := NewRegistry(ModelsFetchFunc(func(context.Context) (ModelsDocument, error) {
		if fetches.Add(1) == 1 {
			close(firstStarted)
			<-firstRelease
			return ModelsDocument{}, staleFailure
		}
		return ModelsDocument{}, currentFailure
	}), time.Minute)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		_, snapshotErr := registry.Snapshot(context.Background())
		done <- snapshotErr
	}()
	<-firstStarted
	registry.Invalidate()
	close(firstRelease)

	if snapshotErr := <-done; !errors.Is(snapshotErr, currentFailure) || errors.Is(snapshotErr, staleFailure) {
		t.Fatalf("stale failure escaped generation fence: %v", snapshotErr)
	}
	if fetches.Load() != 2 {
		t.Fatalf("invalidated failure performed %d fetches, want exactly 2", fetches.Load())
	}
	if _, snapshotErr := registry.Snapshot(context.Background()); !errors.Is(snapshotErr, currentFailure) {
		t.Fatalf("current-generation failure was not negative-cached: %v", snapshotErr)
	}
	if fetches.Load() != 2 {
		t.Fatalf("negative cache refetched after current failure: %d", fetches.Load())
	}
}

func TestRegistryInvalidatePreservesCancellationWithoutPoisoningCache(t *testing.T) {
	firstStarted := make(chan struct{})
	firstRelease := make(chan struct{})
	var fetches atomic.Int64
	registry, err := NewRegistry(ModelsFetchFunc(func(context.Context) (ModelsDocument, error) {
		if fetches.Add(1) == 1 {
			close(firstStarted)
			<-firstRelease
		}
		document := validModelsDocument()
		document.RegistryVersion = "synthetic-current-after-cancel"
		return document, nil
	}), time.Minute)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, snapshotErr := registry.Snapshot(ctx)
		done <- snapshotErr
	}()
	<-firstStarted
	registry.Invalidate()
	cancel()
	close(firstRelease)

	if snapshotErr := <-done; !IsErrorClass(snapshotErr, ErrorCancelled) {
		t.Fatalf("invalidated cancellation classification changed: %v", snapshotErr)
	}
	if fetches.Load() != 1 {
		t.Fatalf("cancelled stale caller retried unexpectedly: %d", fetches.Load())
	}
	snapshot, err := registry.Snapshot(context.Background())
	if err != nil || snapshot.Version != "synthetic-current-after-cancel" || fetches.Load() != 2 {
		t.Fatalf("cancelled stale completion poisoned cache: snapshot=%+v fetches=%d err=%v", snapshot, fetches.Load(), err)
	}
}

func TestRegistryRejectsEveryFallbackCycleIndependentOfDocumentOrder(t *testing.T) {
	tests := []struct {
		name  string
		order []string
		edges map[string][]string
	}{
		{
			name: "self cycle", order: []string{"synthetic/a"},
			edges: map[string][]string{"synthetic/a": {"synthetic/a"}},
		},
		{
			name: "two node cycle", order: []string{"synthetic/a", "synthetic/b"},
			edges: map[string][]string{"synthetic/a": {"synthetic/b"}, "synthetic/b": {"synthetic/a"}},
		},
		{
			name: "deep cycle", order: []string{"synthetic/a", "synthetic/b", "synthetic/c", "synthetic/d"},
			edges: map[string][]string{
				"synthetic/a": {"synthetic/b"},
				"synthetic/b": {"synthetic/c"},
				"synthetic/c": {"synthetic/d"},
				"synthetic/d": {"synthetic/b"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			orders := [][]string{test.order, reversedStrings(test.order)}
			for _, order := range orders {
				_, err := buildSnapshot(fallbackGraphDocument(order, test.edges), time.Unix(0, 0))
				if !IsErrorClass(err, ErrorProtocol) {
					t.Fatalf("cycle accepted for order %v: %v", order, err)
				}
			}
		})
	}
}

func TestRegistryAcceptsFallbackDAGIndependentOfDocumentOrder(t *testing.T) {
	order := []string{"synthetic/a", "synthetic/b", "synthetic/c", "synthetic/d"}
	edges := map[string][]string{
		"synthetic/a": {"synthetic/b", "synthetic/c"},
		"synthetic/b": {"synthetic/d"},
		"synthetic/c": {"synthetic/d"},
		"synthetic/d": nil,
	}
	for _, candidateOrder := range [][]string{order, reversedStrings(order)} {
		snapshot, err := buildSnapshot(fallbackGraphDocument(candidateOrder, edges), time.Unix(0, 0))
		if err != nil || len(snapshot.Models) != len(order) {
			t.Fatalf("valid DAG rejected for order %v: models=%d err=%v", candidateOrder, len(snapshot.Models), err)
		}
	}
}

func TestRegistryBoundsDocumentSizeAndFallbackDepth(t *testing.T) {
	t.Run("model count above bound", func(t *testing.T) {
		order := syntheticModelOrder(MaxRegistryModels + 1)
		_, err := buildSnapshot(fallbackGraphDocument(order, nil), time.Unix(0, 0))
		if !IsErrorClass(err, ErrorProtocol) {
			t.Fatalf("oversized model document accepted: %v", err)
		}
	})

	t.Run("linear chain above depth bound", func(t *testing.T) {
		order, edges := syntheticLinearFallbackChain(MaxRegistryFallbackDepth + 1)
		_, err := buildSnapshot(fallbackGraphDocument(order, edges), time.Unix(0, 0))
		if !IsErrorClass(err, ErrorProtocol) {
			t.Fatalf("oversized fallback chain accepted: %v", err)
		}
	})

	t.Run("exact bounds remain compatible", func(t *testing.T) {
		modelOrder := syntheticModelOrder(MaxRegistryModels)
		modelSnapshot, err := buildSnapshot(fallbackGraphDocument(modelOrder, nil), time.Unix(0, 0))
		if err != nil || len(modelSnapshot.Models) != MaxRegistryModels {
			t.Fatalf("model-count boundary rejected: models=%d err=%v", len(modelSnapshot.Models), err)
		}

		chainOrder, edges := syntheticLinearFallbackChain(MaxRegistryFallbackDepth)
		chainSnapshot, err := buildSnapshot(fallbackGraphDocument(chainOrder, edges), time.Unix(0, 0))
		if err != nil || len(chainSnapshot.Models) != MaxRegistryFallbackDepth+1 {
			t.Fatalf("fallback-depth boundary rejected: models=%d err=%v", len(chainSnapshot.Models), err)
		}
	})
}

func syntheticModelOrder(count int) []string {
	order := make([]string, count)
	for index := range order {
		order[index] = fmt.Sprintf("synthetic/model-%04d", index)
	}
	return order
}

func syntheticLinearFallbackChain(depth int) ([]string, map[string][]string) {
	order := syntheticModelOrder(depth + 1)
	edges := make(map[string][]string, depth)
	// Point from lexicographically later nodes to earlier nodes so validation
	// encounters and memoizes the tail before the root. This guards against a
	// traversal-order-dependent depth check.
	for index := 1; index <= depth; index++ {
		edges[order[index]] = []string{order[index-1]}
	}
	return order, edges
}

func fallbackGraphDocument(order []string, edges map[string][]string) ModelsDocument {
	template := validModelsDocument().Models[0]
	models := make([]ModelDocument, 0, len(order))
	for _, model := range order {
		row := template
		row.ID = model
		row.Fallback = append([]string(nil), edges[model]...)
		models = append(models, row)
	}
	return ModelsDocument{Object: "list", RegistryVersion: "synthetic-fallback-graph-v1", Models: models}
}

func reversedStrings(source []string) []string {
	reversed := append([]string(nil), source...)
	for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
		reversed[left], reversed[right] = reversed[right], reversed[left]
	}
	return reversed
}

func TestReadinessCheckerImplementsFrozenFailClosedContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"synthetic-ok"}`))
	}))
	defer server.Close()
	client := newTestClient(t, server.URL, &syntheticCredentialSource{}, time.Second)
	registry, err := NewRegistry(ModelsFetchFunc(func(context.Context) (ModelsDocument, error) {
		return validModelsDocument(), nil
	}), time.Minute)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	checker, err := NewReadinessChecker(client, registry, brain.StrictReadinessPolicy(), func() (brain.Correlation, error) {
		return testCorrelation(), nil
	})
	if err != nil {
		t.Fatalf("NewReadinessChecker: %v", err)
	}
	snapshot, err := checker.CheckGatewayReadiness(context.Background(), brain.ReadinessRequest{
		RouteModel: "agy/claude-opus-4-6-thinking", Protocol: brain.ProtocolAnthropicMessages,
	})
	if err != nil {
		t.Fatalf("CheckGatewayReadiness: %v", err)
	}
	if !snapshot.Live || !snapshot.Authenticated || !snapshot.ModelRegistryReady || !snapshot.SelectedModelReady || !snapshot.SelectedProtocolReady {
		t.Fatalf("incomplete readiness snapshot: %+v", snapshot)
	}
	_, err = checker.CheckGatewayReadiness(context.Background(), brain.ReadinessRequest{
		RouteModel: "agy/claude-opus-4-6-thinking", Protocol: brain.ProtocolOpenAIResponses,
	})
	if !IsErrorClass(err, ErrorCapability) {
		t.Fatalf("protocol mismatch did not fail closed: %v", err)
	}
}
