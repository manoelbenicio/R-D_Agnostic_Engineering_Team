package handler

import (
	"context"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/service/runtimeconfig"
)

func TestPostgresRuntimeManagerStoreRejectsNilPool(t *testing.T) {
	if _, err := NewPostgresRuntimeManagerStore(nil); err == nil {
		t.Fatal("nil pool accepted")
	}
}

func TestStaticRuntimeManagerAuthoritiesAreImmutableAndFailClosed(t *testing.T) {
	capabilities := rmCapabilities()
	catalog, err := NewStaticCapabilityCatalog(map[string]runtimeconfig.ProviderCapabilities{"provider-a": capabilities})
	if err != nil {
		t.Fatal(err)
	}
	first, err := catalog.Capabilities(context.Background(), "provider-a")
	if err != nil {
		t.Fatal(err)
	}
	delete(first.Models, runtimeconfig.ModelID("model-a"))
	second, err := catalog.Capabilities(context.Background(), "provider-a")
	if err != nil || second.Models[runtimeconfig.ModelID("model-a")].MaxInputTokens == 0 {
		t.Fatalf("catalog mutation escaped clone: %+v, %v", second, err)
	}
	if _, err := catalog.Capabilities(context.Background(), "unknown"); err != ErrRuntimeManagerNotFound {
		t.Fatalf("unknown provider error = %v", err)
	}

	config, issues := documentToConfig(rmValidDocument("model-a"))
	if len(issues) != 0 {
		t.Fatalf("platform document issues: %+v", issues)
	}
	platform, err := NewStaticPlatformLayerSource(config)
	if err != nil {
		t.Fatal(err)
	}
	layer, err := platform.PlatformLayer(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	layer.Delegability[runtimeconfig.FieldModel] = false
	again, err := platform.PlatformLayer(context.Background())
	if err != nil || !again.Delegability[runtimeconfig.FieldModel] {
		t.Fatalf("platform mutation escaped clone: %+v, %v", again, err)
	}
}

func TestPostgresRuntimeManagerStandardLifecycle(t *testing.T) {
	if testPool == nil {
		t.Skip("handler database fixture unavailable")
	}
	store, err := NewPostgresRuntimeManagerStore(testPool)
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithRuntimeManagerActor(context.Background(), testUserID)
	standard, err := store.CreateStandard(ctx, CreateStandardParams{
		Name: "runtime-manager-" + time.Now().UTC().Format("150405.000000000"), RequestID: "create-standard-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM runtime_standard_activation WHERE standard_id = $1`, standard.ID)
		testPool.Exec(context.Background(), `DELETE FROM runtime_standard_version WHERE standard_id = $1`, standard.ID)
		testPool.Exec(context.Background(), `DELETE FROM runtime_standard WHERE id = $1`, standard.ID)
	})

	stored := rmStoredRecord(t, "", 1, "model-a")
	version, err := store.CreateStandardVersion(ctx, CreateStandardVersionParams{
		StandardID: standard.ID, Document: stored.Document, Digest: stored.Digest,
		ApplyClass: "restart", Reason: "create_test_version", RequestID: "create-version-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if version.VersionNumber != 1 || version.ApplyClass != "restart" || version.State != "inactive" {
		t.Fatalf("created version = %+v", version)
	}

	activation, err := store.ActivateStandardVersion(ctx, ActivateStandardVersionParams{
		StandardID: standard.ID, VersionID: version.ID, ExpectedActiveVersionID: nil,
		Reason: "activate_test_version", RequestID: "activate-version-test",
		CapabilityDigest: rmCapabilityDigest(t), ApplyClass: "restart", EffectiveDigest: version.Digest,
	})
	if err != nil {
		t.Fatal(err)
	}
	if activation.ActiveVersionID != version.ID || activation.PreviousVersionID != nil {
		t.Fatalf("activation = %+v", activation)
	}
	loaded, err := store.GetStandard(ctx, standard.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ActiveVersionID == nil || *loaded.ActiveVersionID != version.ID || loaded.ActiveVersionNumber == nil || *loaded.ActiveVersionNumber != 1 {
		t.Fatalf("loaded standard = %+v", loaded)
	}
	versions, next, err := store.ListStandardVersions(ctx, standard.ID, 1, "")
	if err != nil || next != "" || len(versions) != 1 || versions[0].State != "active" {
		t.Fatalf("versions = %+v next=%q err=%v", versions, next, err)
	}
}

func rmCapabilityDigest(t *testing.T) string {
	t.Helper()
	digest, err := runtimeconfig.CapabilityDigest(rmCapabilities())
	if err != nil {
		t.Fatal(err)
	}
	return digest
}

func TestOptionalUUIDRejectsInvalidNonNilExpectation(t *testing.T) {
	invalid := "not-a-uuid"
	if _, err := optionalUUID(&invalid); err == nil {
		t.Fatal("invalid non-nil expected UUID was silently converted to SQL NULL")
	}
	if value, err := optionalUUID(nil); err != nil || value.Valid {
		t.Fatalf("nil expectation = (%+v, %v), want SQL NULL without error", value, err)
	}
}
