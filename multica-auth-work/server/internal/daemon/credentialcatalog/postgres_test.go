package credentialcatalog

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

func TestNewPostgresCatalogGenerationAdapterFailsClosedWithoutPool(t *testing.T) {
	if _, err := NewPostgresCatalogGenerationAdapter(nil); !errors.Is(err, ErrMissingProductionAdapter) {
		t.Fatalf("error = %v, want ErrMissingProductionAdapter", err)
	}
	var adapter *PostgresCatalogGenerationAdapter
	if _, err := adapter.LoadLatestGeneration(t.Context(), CatalogIdentity{}); !errors.Is(err, ErrMissingProductionAdapter) {
		t.Fatalf("nil LoadLatestGeneration error = %v", err)
	}
	if err := adapter.AppendGeneration(t.Context(), CatalogGeneration{}); !errors.Is(err, ErrMissingProductionAdapter) {
		t.Fatalf("nil AppendGeneration error = %v", err)
	}
}

func TestReconstructCatalogGenerationIsLosslessAndDigestValid(t *testing.T) {
	started := time.Unix(1_750_000_000, 123_456_000).UTC()
	published := started.Add(2 * time.Second)
	health := started.Add(time.Second)
	homeRef := "60000000-0000-5000-8000-00000000000c"
	identity := CatalogIdentity{
		WorkspaceID: "60000000-0000-4000-8000-000000000002",
		DaemonID:    "daemon-a",
		CatalogID:   "60000000-0000-4000-8000-000000000009",
	}
	expected := CatalogGeneration{
		CatalogIdentity: identity, PreviousGeneration: 0, Generation: 1,
		ScanKind: ScanStartup, Watermark: WatermarkNormal, StartedAt: started, PublishedAt: published,
		Entries: []CatalogEntryRecord{{
			HomeRef: homeRef, NameRef: "name_" + strings.Repeat("c", 43), Provider: ProviderCodex,
			Approved: true, State: StateHealthy, ActiveRefs: 1, FirstSeenAt: started,
			LastSeenAt: health, LastFullScanAt: published, HealthWatermark: &health,
			TTL: 90*time.Second + 123*time.Nanosecond, RetentionDeadline: published.Add(24 * time.Hour),
		}},
		Lifecycle: []LifecycleRecord{},
	}
	expected.CatalogDigest = ComputeCatalogDigest(expected)

	catalogID := mustPostgresTestUUID(t, identity.CatalogID)
	homeID := mustPostgresTestUUID(t, homeRef)
	actual, err := reconstructCatalogGeneration(identity, db.GetPublishedCredentialHomeCatalogGenerationRow{
		ID: mustPostgresTestUUID(t, "60000000-0000-4000-8000-00000000000a"), CatalogID: catalogID,
		PreviousGeneration: 0, Generation: 1, ScanKind: "startup", CatalogDigest: expected.CatalogDigest,
		StartedAt:   pgtype.Timestamptz{Time: started, Valid: true},
		PublishedAt: pgtype.Timestamptz{Time: published, Valid: true}, Watermark: "normal",
	}, []db.CredentialHomeCatalogEntry{{
		CatalogID: catalogID, Generation: 1, HomeRef: homeID,
		NameRef: pgtype.Text{String: expected.Entries[0].NameRef, Valid: true}, Provider: "codex",
		Approved: true, State: "healthy", ActiveRefs: 1,
		FirstSeenAt:       pgtype.Timestamptz{Time: started, Valid: true},
		LastSeenAt:        pgtype.Timestamptz{Time: health, Valid: true},
		LastFullScanAt:    pgtype.Timestamptz{Time: published, Valid: true},
		HealthWatermark:   pgtype.Timestamptz{Time: health, Valid: true},
		TtlNanoseconds:    int64(expected.Entries[0].TTL),
		RetentionDeadline: pgtype.Timestamptz{Time: expected.Entries[0].RetentionDeadline, Valid: true},
	}}, nil)
	if err != nil {
		t.Fatalf("reconstruct: %v", err)
	}
	if err := validateGeneration(actual); err != nil {
		t.Fatalf("reconstructed generation failed validation: %v", err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("reconstructed generation mismatch\nactual: %#v\nexpected: %#v", actual, expected)
	}
}

func TestPostgresCatalogGenerationAdapterRejectsInvalidIdentityAndMapsCASConflicts(t *testing.T) {
	if _, _, err := parseCatalogIdentity(CatalogIdentity{WorkspaceID: "not-a-uuid", DaemonID: "d", CatalogID: "also-bad"}); !errors.Is(err, ErrInvalidGeneration) {
		t.Fatalf("identity error = %v, want ErrInvalidGeneration", err)
	}
	if err := mapCatalogAppendError("publish", pgx.ErrNoRows); !errors.Is(err, ErrGenerationConflict) {
		t.Fatalf("CAS error = %v, want ErrGenerationConflict", err)
	}
}

func mustPostgresTestUUID(t *testing.T, value string) pgtype.UUID {
	t.Helper()
	var result pgtype.UUID
	if err := result.Scan(value); err != nil || !result.Valid {
		t.Fatalf("parse UUID %q: %v", value, err)
	}
	return result
}
