package credentialcatalog

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// PostgresCatalogGenerationAdapter persists complete immutable catalog
// generations and their current lifecycle projection in one transaction.
// It deliberately accepts only pre-existing, workspace-scoped catalog IDs.
type PostgresCatalogGenerationAdapter struct {
	pool *pgxpool.Pool
}

var _ CatalogGenerationAdapter = (*PostgresCatalogGenerationAdapter)(nil)

func NewPostgresCatalogGenerationAdapter(pool *pgxpool.Pool) (*PostgresCatalogGenerationAdapter, error) {
	if pool == nil {
		return nil, ErrMissingProductionAdapter
	}
	return &PostgresCatalogGenerationAdapter{pool: pool}, nil
}

func (a *PostgresCatalogGenerationAdapter) LoadLatestGeneration(ctx context.Context, identity CatalogIdentity) (*CatalogGeneration, error) {
	if a == nil || a.pool == nil {
		return nil, ErrMissingProductionAdapter
	}
	catalogID, workspaceID, err := parseCatalogIdentity(identity)
	if err != nil {
		return nil, err
	}
	q := db.New(a.pool)
	catalog, err := q.GetCredentialHomeCatalogForWorkspace(ctx, db.GetCredentialHomeCatalogForWorkspaceParams{
		ID: catalogID, WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("credential catalog: load catalog identity: %w", err)
	}
	if catalog.DaemonID != identity.DaemonID {
		return nil, fmt.Errorf("%w: catalog daemon mismatch", ErrInvalidGeneration)
	}
	if catalog.Generation == 0 {
		if catalog.LifecycleGeneration != 0 {
			return nil, fmt.Errorf("%w: unpublished lifecycle generation", ErrInvalidGeneration)
		}
		return nil, nil
	}
	if catalog.Generation < 0 || catalog.LifecycleGeneration != catalog.Generation {
		return nil, fmt.Errorf("%w: inconsistent published generation", ErrInvalidGeneration)
	}

	generationRow, err := q.GetPublishedCredentialHomeCatalogGeneration(ctx, db.GetPublishedCredentialHomeCatalogGenerationParams{
		CatalogID: catalogID, WorkspaceID: workspaceID, Generation: catalog.Generation,
	})
	if err != nil {
		return nil, fmt.Errorf("credential catalog: load published generation: %w", err)
	}
	entryRows, err := q.ListCredentialHomeCatalogGenerationEntries(ctx, db.ListCredentialHomeCatalogGenerationEntriesParams{
		CatalogID: catalogID, WorkspaceID: workspaceID, Generation: catalog.Generation,
	})
	if err != nil {
		return nil, fmt.Errorf("credential catalog: load generation entries: %w", err)
	}
	lifecycleRows, err := q.ListCredentialHomeCatalogGenerationLifecycle(ctx, db.ListCredentialHomeCatalogGenerationLifecycleParams{
		CatalogID: catalogID, WorkspaceID: workspaceID, Generation: catalog.Generation,
	})
	if err != nil {
		return nil, fmt.Errorf("credential catalog: load generation lifecycle: %w", err)
	}

	generation, err := reconstructCatalogGeneration(identity, generationRow, entryRows, lifecycleRows)
	if err != nil {
		return nil, err
	}
	if err := validateGeneration(generation); err != nil {
		return nil, fmt.Errorf("credential catalog: stored generation integrity failure: %w", err)
	}
	return &generation, nil
}

func (a *PostgresCatalogGenerationAdapter) AppendGeneration(ctx context.Context, generation CatalogGeneration) error {
	if a == nil || a.pool == nil {
		return ErrMissingProductionAdapter
	}
	if err := validateGeneration(generation); err != nil {
		return err
	}
	catalogID, workspaceID, err := parseCatalogIdentity(generation.CatalogIdentity)
	if err != nil {
		return err
	}
	previous, current, err := generationNumbers(generation)
	if err != nil {
		return err
	}

	tx, err := a.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return fmt.Errorf("credential catalog: begin generation transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := db.New(tx)

	catalog, err := q.LockCredentialHomeCatalog(ctx, db.LockCredentialHomeCatalogParams{ID: catalogID, WorkspaceID: workspaceID})
	if err != nil {
		return mapCatalogAppendError("lock catalog", err)
	}
	if catalog.DaemonID != generation.DaemonID || catalog.Generation != previous || catalog.LifecycleGeneration != previous {
		return ErrGenerationConflict
	}
	if _, err = q.BeginCredentialHomeCatalogReconciliation(ctx, db.BeginCredentialHomeCatalogReconciliationParams{
		ID: catalogID, WorkspaceID: workspaceID, ExpectedGeneration: previous,
	}); err != nil {
		return mapCatalogAppendError("begin reconciliation", err)
	}
	if _, err = q.AdvanceCredentialHomeCatalogLifecycleGeneration(ctx, db.AdvanceCredentialHomeCatalogLifecycleGenerationParams{
		LifecycleGeneration: current, CatalogID: catalogID, WorkspaceID: workspaceID,
		ExpectedPublishedGeneration: previous, ExpectedLifecycleGeneration: previous,
	}); err != nil {
		return mapCatalogAppendError("advance lifecycle generation", err)
	}

	for _, record := range generation.Lifecycle {
		homeRef, parseErr := parseUUID(record.HomeRef, "lifecycle home_ref")
		if parseErr != nil {
			return parseErr
		}
		if record.ActiveRefs > math.MaxInt32 {
			return fmt.Errorf("%w: lifecycle active refs overflow", ErrInvalidGeneration)
		}
		if _, err = q.SaveCredentialHomeCatalogLifecycleRecord(ctx, db.SaveCredentialHomeCatalogLifecycleRecordParams{
			HomeRef: homeRef, NameRef: record.NameRef, Provider: string(record.Provider),
			LifecycleState: string(record.State), ReasonCode: string(record.ReasonCode),
			ActiveRefs: int32(record.ActiveRefs), LifecycleGeneration: current,
			UpdatedAt: timestamp(record.UpdatedAt), RetentionDeadline: timestamp(record.RetentionDeadline),
			CatalogID: catalogID, WorkspaceID: workspaceID,
		}); err != nil {
			return mapCatalogAppendError("save lifecycle record", err)
		}
	}

	generationRow, err := q.CreateCredentialHomeCatalogGeneration(ctx, db.CreateCredentialHomeCatalogGenerationParams{
		CatalogID: catalogID, PreviousGeneration: previous, Generation: current,
		ScanKind: string(generation.ScanKind), Counters: []byte("{}"),
		CatalogDigest: generation.CatalogDigest, StartedAt: timestamp(generation.StartedAt),
		PublishedAt: timestamp(generation.PublishedAt),
	})
	if err != nil {
		return mapCatalogAppendError("create generation", err)
	}
	for _, entry := range generation.Entries {
		homeRef, parseErr := parseUUID(entry.HomeRef, "entry home_ref")
		if parseErr != nil {
			return parseErr
		}
		if entry.ActiveRefs > math.MaxInt32 {
			return fmt.Errorf("%w: entry active refs overflow", ErrInvalidGeneration)
		}
		if _, err = q.CreateCredentialHomeCatalogEntry(ctx, db.CreateCredentialHomeCatalogEntryParams{
			GenerationID: generationRow.ID, CatalogID: catalogID, Generation: current,
			HomeRef: homeRef, NameRef: entry.NameRef, Provider: string(entry.Provider),
			Approved: entry.Approved, State: string(entry.State), ReasonCode: string(entry.ReasonCode),
			ActiveRefs: int32(entry.ActiveRefs), FirstSeenAt: timestamp(entry.FirstSeenAt),
			LastSeenAt: timestamp(entry.LastSeenAt), LastFullScanAt: timestamp(entry.LastFullScanAt),
			HealthWatermark: optionalTimestamp(entry.HealthWatermark), MissingWatermark: optionalTimestamp(entry.MissingWatermark),
			TtlNanoseconds: int64(entry.TTL), RetentionDeadline: timestamp(entry.RetentionDeadline),
		}); err != nil {
			return mapCatalogAppendError("create generation entry", err)
		}
	}
	if _, err = q.PublishCredentialHomeCatalogGeneration(ctx, db.PublishCredentialHomeCatalogGenerationParams{
		Generation: current, Watermark: string(generation.Watermark), CatalogID: catalogID, PreviousGeneration: previous,
	}); err != nil {
		return mapCatalogAppendError("publish generation", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return mapCatalogAppendError("commit generation", err)
	}
	return nil
}

func reconstructCatalogGeneration(identity CatalogIdentity, row db.GetPublishedCredentialHomeCatalogGenerationRow, entries []db.CredentialHomeCatalogEntry, lifecycle []db.CredentialHomeCatalogLifecycle) (CatalogGeneration, error) {
	if row.PreviousGeneration < 0 || row.Generation <= 0 || !row.StartedAt.Valid || !row.PublishedAt.Valid {
		return CatalogGeneration{}, fmt.Errorf("%w: malformed generation row", ErrInvalidGeneration)
	}
	result := CatalogGeneration{
		CatalogIdentity: identity, PreviousGeneration: uint64(row.PreviousGeneration), Generation: uint64(row.Generation),
		ScanKind: ScanKind(row.ScanKind), Watermark: Watermark(row.Watermark), CatalogDigest: row.CatalogDigest,
		StartedAt: row.StartedAt.Time, PublishedAt: row.PublishedAt.Time,
		Entries: make([]CatalogEntryRecord, 0, len(entries)), Lifecycle: make([]LifecycleRecord, 0, len(lifecycle)),
	}
	for _, item := range entries {
		if item.Generation != row.Generation || !item.FirstSeenAt.Valid || !item.LastSeenAt.Valid ||
			!item.LastFullScanAt.Valid || !item.RetentionDeadline.Valid || item.TtlNanoseconds <= 0 {
			return CatalogGeneration{}, fmt.Errorf("%w: malformed entry row", ErrInvalidGeneration)
		}
		result.Entries = append(result.Entries, CatalogEntryRecord{
			HomeRef: uuidString(item.HomeRef), NameRef: nullableText(item.NameRef), Provider: Provider(item.Provider),
			Approved: item.Approved, State: State(item.State), ReasonCode: QuarantineReason(nullableText(item.ReasonCode)),
			ActiveRefs: int(item.ActiveRefs), FirstSeenAt: item.FirstSeenAt.Time, LastSeenAt: item.LastSeenAt.Time,
			LastFullScanAt: item.LastFullScanAt.Time, HealthWatermark: nullableTime(item.HealthWatermark),
			MissingWatermark: nullableTime(item.MissingWatermark), TTL: time.Duration(item.TtlNanoseconds),
			RetentionDeadline: item.RetentionDeadline.Time,
		})
	}
	for _, item := range lifecycle {
		if item.Generation != row.Generation || !item.UpdatedAt.Valid || !item.RetentionDeadline.Valid {
			return CatalogGeneration{}, fmt.Errorf("%w: malformed lifecycle row", ErrInvalidGeneration)
		}
		result.Lifecycle = append(result.Lifecycle, LifecycleRecord{
			HomeRef: uuidString(item.HomeRef), NameRef: nullableText(item.NameRef), Provider: Provider(item.Provider),
			State: State(item.State), ReasonCode: QuarantineReason(nullableText(item.ReasonCode)), ActiveRefs: int(item.ActiveRefs),
			Generation: uint64(item.Generation), UpdatedAt: item.UpdatedAt.Time, RetentionDeadline: item.RetentionDeadline.Time,
		})
	}
	return result, nil
}

func parseCatalogIdentity(identity CatalogIdentity) (pgtype.UUID, pgtype.UUID, error) {
	if !identity.valid() {
		return pgtype.UUID{}, pgtype.UUID{}, ErrInvalidGeneration
	}
	catalogID, err := parseUUID(identity.CatalogID, "catalog_id")
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, err
	}
	workspaceID, err := parseUUID(identity.WorkspaceID, "workspace_id")
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, err
	}
	return catalogID, workspaceID, nil
}

func parseUUID(value, field string) (pgtype.UUID, error) {
	var parsed pgtype.UUID
	if err := parsed.Scan(value); err != nil || !parsed.Valid {
		return pgtype.UUID{}, fmt.Errorf("%w: invalid %s", ErrInvalidGeneration, field)
	}
	return parsed, nil
}

func generationNumbers(generation CatalogGeneration) (int64, int64, error) {
	if generation.PreviousGeneration > math.MaxInt64 || generation.Generation > math.MaxInt64 {
		return 0, 0, fmt.Errorf("%w: generation overflow", ErrInvalidGeneration)
	}
	return int64(generation.PreviousGeneration), int64(generation.Generation), nil
}

func timestamp(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: !value.IsZero()}
}

func optionalTimestamp(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return timestamp(*value)
}

func nullableText(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func nullableTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	copy := value.Time
	return &copy
}

func uuidString(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		value.Bytes[0:4], value.Bytes[4:6], value.Bytes[6:8], value.Bytes[8:10], value.Bytes[10:16])
}

func mapCatalogAppendError(operation string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: %s", ErrGenerationConflict, operation)
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505", "40001", "40P01", "55006":
			return fmt.Errorf("%w: %s", ErrGenerationConflict, operation)
		}
	}
	return fmt.Errorf("credential catalog: %s: %w", operation, err)
}
