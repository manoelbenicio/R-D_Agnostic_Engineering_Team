package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/multica-ai/multica/server/internal/middleware"
	"github.com/multica-ai/multica/server/internal/service/runtimeconfig"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

type runtimeManagerActorContextKey struct{}

// WithRuntimeManagerActor binds the authenticated human actor used by durable
// Runtime Manager writes. Shared authorization middleware is the only caller.
func WithRuntimeManagerActor(ctx context.Context, actorID string) context.Context {
	return context.WithValue(ctx, runtimeManagerActorContextKey{}, actorID)
}

func runtimeManagerActor(ctx context.Context) (pgtype.UUID, error) {
	if member, ok := middleware.MemberFromContext(ctx); ok {
		return member.UserID, nil
	}
	actorID, _ := ctx.Value(runtimeManagerActorContextKey{}).(string)
	if actorID == "" {
		return pgtype.UUID{}, errors.New("runtime manager: authenticated actor missing")
	}
	return util.ParseUUID(actorID)
}

// PostgresRuntimeManagerStore is the C2 durable implementation of both C3
// store contracts. Mutating transitions use one database transaction so the
// active pointer and activation audit row cannot diverge.
type PostgresRuntimeManagerStore struct {
	pool *pgxpool.Pool
}

func NewPostgresRuntimeManagerStore(pool *pgxpool.Pool) (*PostgresRuntimeManagerStore, error) {
	if pool == nil {
		return nil, errors.New("runtime manager: postgres pool is required")
	}
	return &PostgresRuntimeManagerStore{pool: pool}, nil
}

func (s *PostgresRuntimeManagerStore) GetBinding(ctx context.Context, workspaceID, bindingID string) (RuntimeBindingRecord, error) {
	workspaceUUID, bindingUUID, err := runtimeManagerScopeUUIDs(workspaceID, bindingID)
	if err != nil {
		return RuntimeBindingRecord{}, ErrRuntimeManagerNotFound
	}
	row, err := db.New(s.pool).GetRuntimeBindingForWorkspace(ctx, db.GetRuntimeBindingForWorkspaceParams{ID: bindingUUID, WorkspaceID: workspaceUUID})
	if err != nil {
		return RuntimeBindingRecord{}, runtimeManagerDBError(err, false)
	}
	var provider string
	if err := s.pool.QueryRow(ctx, `
		SELECT rs.provider
		FROM runtime_binding rb
		JOIN runtime_session rs ON rs.id = rb.session_id
		WHERE rb.id = $1 AND rb.workspace_id = $2`, bindingUUID, workspaceUUID).Scan(&provider); err != nil {
		return RuntimeBindingRecord{}, runtimeManagerDBError(err, false)
	}
	return bindingRecord(row, provider), nil
}

func (s *PostgresRuntimeManagerStore) GetBindingVersion(ctx context.Context, workspaceID, bindingID, versionID string) (RuntimeConfigurationVersionRecord, error) {
	binding, err := s.GetBinding(ctx, workspaceID, bindingID)
	if err != nil {
		return RuntimeConfigurationVersionRecord{}, err
	}
	versionUUID, err := util.ParseUUID(versionID)
	if err != nil {
		return RuntimeConfigurationVersionRecord{}, ErrRuntimeManagerNotFound
	}
	rows, err := db.New(s.pool).ListRuntimeConfigurationVersions(ctx, util.MustParseUUID(binding.ID))
	if err != nil {
		return RuntimeConfigurationVersionRecord{}, runtimeManagerDBError(err, false)
	}
	for _, row := range rows {
		if row.ID == versionUUID {
			return bindingVersionRecord(row, binding.ActiveConfigurationVersionID), nil
		}
	}
	return RuntimeConfigurationVersionRecord{}, ErrRuntimeManagerNotFound
}

func (s *PostgresRuntimeManagerStore) ListBindingVersions(ctx context.Context, workspaceID, bindingID string, limit int, cursor string) ([]RuntimeConfigurationVersionRecord, string, error) {
	binding, err := s.GetBinding(ctx, workspaceID, bindingID)
	if err != nil {
		return nil, "", err
	}
	rows, err := db.New(s.pool).ListRuntimeConfigurationVersions(ctx, util.MustParseUUID(binding.ID))
	if err != nil {
		return nil, "", runtimeManagerDBError(err, false)
	}
	start, err := runtimeManagerCursor(cursor, len(rows))
	if err != nil {
		return nil, "", err
	}
	end, next := runtimeManagerPageBounds(start, limit, len(rows))
	out := make([]RuntimeConfigurationVersionRecord, 0, end-start)
	for _, row := range rows[start:end] {
		out = append(out, bindingVersionRecord(row, binding.ActiveConfigurationVersionID))
	}
	return out, next, nil
}

func (s *PostgresRuntimeManagerStore) CreateBindingVersion(ctx context.Context, params CreateBindingVersionParams) (RuntimeConfigurationVersionRecord, error) {
	actor, err := runtimeManagerActor(ctx)
	if err != nil {
		return RuntimeConfigurationVersionRecord{}, err
	}
	binding, err := s.GetBinding(ctx, params.WorkspaceID, params.BindingID)
	if err != nil {
		return RuntimeConfigurationVersionRecord{}, err
	}
	workspaceUUID, bindingUUID, err := runtimeManagerScopeUUIDs(params.WorkspaceID, binding.ID)
	if err != nil {
		return RuntimeConfigurationVersionRecord{}, ErrRuntimeManagerNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return RuntimeConfigurationVersionRecord{}, err
	}
	defer tx.Rollback(ctx)
	q := db.New(tx)
	locked, err := q.LockRuntimeBinding(ctx, db.LockRuntimeBindingParams{ID: bindingUUID, WorkspaceID: workspaceUUID})
	if err != nil {
		return RuntimeConfigurationVersionRecord{}, runtimeManagerDBError(err, false)
	}
	if existing, replayErr := q.GetRuntimeConfigurationVersionByRequestID(ctx, db.GetRuntimeConfigurationVersionByRequestIDParams{BindingID: bindingUUID, RequestID: params.RequestID}); replayErr == nil {
		if existing.ConfigurationDigest != params.Digest || existing.ApplyClass != params.ApplyClass || existing.Reason != params.Reason {
			return RuntimeConfigurationVersionRecord{}, ErrRuntimeManagerConflict
		}
		return bindingVersionRecord(existing, uuidPointer(locked.ActiveConfigurationVersionID)), nil
	} else if !errors.Is(replayErr, pgx.ErrNoRows) {
		return RuntimeConfigurationVersionRecord{}, runtimeManagerDBError(replayErr, false)
	}
	rows, err := q.ListRuntimeConfigurationVersions(ctx, bindingUUID)
	if err != nil {
		return RuntimeConfigurationVersionRecord{}, runtimeManagerDBError(err, false)
	}
	row, err := q.CreateRuntimeConfigurationVersion(ctx, db.CreateRuntimeConfigurationVersionParams{
		BindingID: bindingUUID, VersionNumber: nextBindingVersion(rows),
		Configuration: params.Document, ConfigurationDigest: params.Digest,
		ApplyClass: params.ApplyClass, CreatedBy: actor, Reason: params.Reason, RequestID: params.RequestID,
	})
	if err != nil {
		return RuntimeConfigurationVersionRecord{}, runtimeManagerDBError(err, true)
	}
	if err := tx.Commit(ctx); err != nil {
		return RuntimeConfigurationVersionRecord{}, runtimeManagerDBError(err, true)
	}
	return bindingVersionRecord(row, uuidPointer(locked.ActiveConfigurationVersionID)), nil
}

func (s *PostgresRuntimeManagerStore) ActivateBindingVersion(ctx context.Context, params ActivateBindingVersionParams) (BindingActivationRecord, error) {
	actor, err := runtimeManagerActor(ctx)
	if err != nil {
		return BindingActivationRecord{}, err
	}
	workspaceUUID, bindingUUID, err := runtimeManagerScopeUUIDs(params.WorkspaceID, params.BindingID)
	if err != nil {
		return BindingActivationRecord{}, ErrRuntimeManagerNotFound
	}
	versionUUID, err := util.ParseUUID(params.VersionID)
	if err != nil {
		return BindingActivationRecord{}, ErrRuntimeManagerNotFound
	}
	expectedActive, err := optionalUUID(params.ExpectedActiveVersionID)
	if err != nil {
		return BindingActivationRecord{}, ErrRuntimeManagerConflict
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return BindingActivationRecord{}, err
	}
	defer tx.Rollback(ctx)
	q := db.New(tx)
	before, err := q.LockRuntimeBinding(ctx, db.LockRuntimeBindingParams{ID: bindingUUID, WorkspaceID: workspaceUUID})
	if err != nil {
		return BindingActivationRecord{}, runtimeManagerDBError(err, false)
	}
	if existing, replayErr := q.GetRuntimeConfigurationActivationByRequestID(ctx, db.GetRuntimeConfigurationActivationByRequestIDParams{BindingID: bindingUUID, RequestID: params.RequestID}); replayErr == nil {
		if existing.NewVersionID != versionUUID || existing.EffectiveConfigurationDigest != params.EffectiveDigest ||
			existing.CapabilityDigest != params.CapabilityDigest || existing.ApplyClass != params.ApplyClass || existing.Reason != params.Reason {
			return BindingActivationRecord{}, ErrRuntimeManagerConflict
		}
		return BindingActivationRecord{ActiveVersionID: util.UUIDToString(existing.NewVersionID), PreviousVersionID: uuidPointer(existing.PreviousVersionID), BindingGeneration: existing.BindingGeneration}, nil
	} else if !errors.Is(replayErr, pgx.ErrNoRows) {
		return BindingActivationRecord{}, runtimeManagerDBError(replayErr, false)
	}
	after, err := q.ActivateRuntimeConfigurationVersion(ctx, db.ActivateRuntimeConfigurationVersionParams{
		NewVersionID: versionUUID, EffectiveConfigurationDigest: pgtype.Text{String: params.EffectiveDigest, Valid: true},
		BindingID: bindingUUID, WorkspaceID: workspaceUUID,
		ExpectedBindingGeneration: params.ExpectedBindingGeneration,
		ExpectedActiveVersionID:   expectedActive,
	})
	if err != nil {
		return BindingActivationRecord{}, runtimeManagerDBError(err, true)
	}
	if _, err := q.RecordRuntimeConfigurationActivation(ctx, db.RecordRuntimeConfigurationActivationParams{
		BindingID: bindingUUID, PreviousVersionID: before.ActiveConfigurationVersionID,
		NewVersionID: versionUUID, BindingGeneration: after.Generation,
		EffectiveConfigurationDigest: params.EffectiveDigest, CapabilityDigest: params.CapabilityDigest,
		ApplyClass: params.ApplyClass, ActorID: actor, RequestID: params.RequestID,
		Reason: params.Reason,
	}); err != nil {
		return BindingActivationRecord{}, runtimeManagerDBError(err, true)
	}
	if err := tx.Commit(ctx); err != nil {
		return BindingActivationRecord{}, runtimeManagerDBError(err, true)
	}
	return BindingActivationRecord{ActiveVersionID: util.UUIDToString(after.ActiveConfigurationVersionID), PreviousVersionID: uuidPointer(before.ActiveConfigurationVersionID), BindingGeneration: after.Generation}, nil
}

func (s *PostgresRuntimeManagerStore) ListStandards(ctx context.Context, limit int, cursor string) ([]RuntimeStandardRecord, string, error) {
	actor, err := runtimeManagerActor(ctx)
	if err != nil {
		return nil, "", err
	}
	rows, err := db.New(s.pool).ListRuntimeStandardsByOwner(ctx, actor)
	if err != nil {
		return nil, "", runtimeManagerDBError(err, false)
	}
	start, err := runtimeManagerCursor(cursor, len(rows))
	if err != nil {
		return nil, "", err
	}
	end, next := runtimeManagerPageBounds(start, limit, len(rows))
	out := make([]RuntimeStandardRecord, 0, end-start)
	for _, row := range rows[start:end] {
		record, err := s.standardRecord(ctx, row)
		if err != nil {
			return nil, "", err
		}
		out = append(out, record)
	}
	return out, next, nil
}

func (s *PostgresRuntimeManagerStore) CreateStandard(ctx context.Context, params CreateStandardParams) (RuntimeStandardRecord, error) {
	actor, err := runtimeManagerActor(ctx)
	if err != nil {
		return RuntimeStandardRecord{}, err
	}
	row, err := db.New(s.pool).CreateRuntimeStandard(ctx, db.CreateRuntimeStandardParams{OwnerID: actor, Name: params.Name, Description: util.PtrToText(params.Description), RequestID: params.RequestID})
	if err != nil {
		return RuntimeStandardRecord{}, runtimeManagerDBError(err, true)
	}
	if row.Name != params.Name || !optionalTextMatches(row.Description, params.Description) {
		return RuntimeStandardRecord{}, ErrRuntimeManagerConflict
	}
	return s.standardRecord(ctx, row)
}

func (s *PostgresRuntimeManagerStore) GetStandard(ctx context.Context, standardID string) (RuntimeStandardRecord, error) {
	actor, err := runtimeManagerActor(ctx)
	if err != nil {
		return RuntimeStandardRecord{}, err
	}
	id, err := util.ParseUUID(standardID)
	if err != nil {
		return RuntimeStandardRecord{}, ErrRuntimeManagerNotFound
	}
	row, err := db.New(s.pool).GetRuntimeStandardForOwner(ctx, db.GetRuntimeStandardForOwnerParams{ID: id, OwnerID: actor})
	if err != nil {
		return RuntimeStandardRecord{}, runtimeManagerDBError(err, false)
	}
	return s.standardRecord(ctx, row)
}

func (s *PostgresRuntimeManagerStore) ListStandardVersions(ctx context.Context, standardID string, limit int, cursor string) ([]RuntimeConfigurationVersionRecord, string, error) {
	standard, err := s.GetStandard(ctx, standardID)
	if err != nil {
		return nil, "", err
	}
	rows, err := db.New(s.pool).ListRuntimeStandardVersions(ctx, util.MustParseUUID(standard.ID))
	if err != nil {
		return nil, "", runtimeManagerDBError(err, false)
	}
	start, err := runtimeManagerCursor(cursor, len(rows))
	if err != nil {
		return nil, "", err
	}
	end, next := runtimeManagerPageBounds(start, limit, len(rows))
	out := make([]RuntimeConfigurationVersionRecord, 0, end-start)
	for _, row := range rows[start:end] {
		record, err := standardVersionRecord(row, standard.ActiveVersionID)
		if err != nil {
			return nil, "", err
		}
		out = append(out, record)
	}
	return out, next, nil
}

func (s *PostgresRuntimeManagerStore) GetStandardVersion(ctx context.Context, standardID, versionID string) (RuntimeConfigurationVersionRecord, error) {
	standard, err := s.GetStandard(ctx, standardID)
	if err != nil {
		return RuntimeConfigurationVersionRecord{}, err
	}
	versionUUID, err := util.ParseUUID(versionID)
	if err != nil {
		return RuntimeConfigurationVersionRecord{}, ErrRuntimeManagerNotFound
	}
	rows, err := db.New(s.pool).ListRuntimeStandardVersions(ctx, util.MustParseUUID(standard.ID))
	if err != nil {
		return RuntimeConfigurationVersionRecord{}, runtimeManagerDBError(err, false)
	}
	for _, row := range rows {
		if row.ID == versionUUID {
			return standardVersionRecord(row, standard.ActiveVersionID)
		}
	}
	return RuntimeConfigurationVersionRecord{}, ErrRuntimeManagerNotFound
}

func (s *PostgresRuntimeManagerStore) CreateStandardVersion(ctx context.Context, params CreateStandardVersionParams) (RuntimeConfigurationVersionRecord, error) {
	actor, err := runtimeManagerActor(ctx)
	if err != nil {
		return RuntimeConfigurationVersionRecord{}, err
	}
	standardUUID, err := util.ParseUUID(params.StandardID)
	if err != nil {
		return RuntimeConfigurationVersionRecord{}, ErrRuntimeManagerNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return RuntimeConfigurationVersionRecord{}, err
	}
	defer tx.Rollback(ctx)
	q := db.New(tx)
	standard, err := q.LockRuntimeStandardForOwner(ctx, db.LockRuntimeStandardForOwnerParams{ID: standardUUID, OwnerID: actor})
	if err != nil {
		return RuntimeConfigurationVersionRecord{}, runtimeManagerDBError(err, false)
	}
	if existing, replayErr := q.GetRuntimeStandardVersionByRequestID(ctx, db.GetRuntimeStandardVersionByRequestIDParams{StandardID: standardUUID, RequestID: params.RequestID}); replayErr == nil {
		if existing.ConfigurationDigest != params.Digest || existing.ApplyClass != params.ApplyClass || existing.Reason != params.Reason {
			return RuntimeConfigurationVersionRecord{}, ErrRuntimeManagerConflict
		}
		return standardVersionRecord(existing, uuidPointer(standard.ActiveVersionID))
	} else if !errors.Is(replayErr, pgx.ErrNoRows) {
		return RuntimeConfigurationVersionRecord{}, runtimeManagerDBError(replayErr, false)
	}
	rows, err := q.ListRuntimeStandardVersions(ctx, standardUUID)
	if err != nil {
		return RuntimeConfigurationVersionRecord{}, runtimeManagerDBError(err, false)
	}
	row, err := q.CreateRuntimeStandardVersion(ctx, db.CreateRuntimeStandardVersionParams{
		StandardID: standardUUID, VersionNumber: nextStandardVersion(rows),
		Configuration: params.Document, ConfigurationDigest: params.Digest, ApplyClass: params.ApplyClass,
		CreatedBy: actor, Reason: params.Reason, RequestID: params.RequestID,
	})
	if err != nil {
		return RuntimeConfigurationVersionRecord{}, runtimeManagerDBError(err, true)
	}
	if err := tx.Commit(ctx); err != nil {
		return RuntimeConfigurationVersionRecord{}, runtimeManagerDBError(err, true)
	}
	return standardVersionRecord(row, uuidPointer(standard.ActiveVersionID))
}

func (s *PostgresRuntimeManagerStore) ActivateStandardVersion(ctx context.Context, params ActivateStandardVersionParams) (StandardActivationRecord, error) {
	actor, err := runtimeManagerActor(ctx)
	if err != nil {
		return StandardActivationRecord{}, err
	}
	standardUUID, err := util.ParseUUID(params.StandardID)
	if err != nil {
		return StandardActivationRecord{}, ErrRuntimeManagerNotFound
	}
	versionUUID, err := util.ParseUUID(params.VersionID)
	if err != nil {
		return StandardActivationRecord{}, ErrRuntimeManagerNotFound
	}
	expectedActive, err := optionalUUID(params.ExpectedActiveVersionID)
	if err != nil {
		return StandardActivationRecord{}, ErrRuntimeManagerConflict
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return StandardActivationRecord{}, err
	}
	defer tx.Rollback(ctx)
	q := db.New(tx)
	before, err := q.LockRuntimeStandardForOwner(ctx, db.LockRuntimeStandardForOwnerParams{ID: standardUUID, OwnerID: actor})
	if err != nil {
		return StandardActivationRecord{}, runtimeManagerDBError(err, false)
	}
	if existing, replayErr := q.GetRuntimeStandardActivationByRequestID(ctx, db.GetRuntimeStandardActivationByRequestIDParams{StandardID: standardUUID, RequestID: params.RequestID}); replayErr == nil {
		if existing.NewVersionID != versionUUID || existing.CapabilityDigest != params.CapabilityDigest || existing.Reason != params.Reason {
			return StandardActivationRecord{}, ErrRuntimeManagerConflict
		}
		return StandardActivationRecord{ActiveVersionID: util.UUIDToString(existing.NewVersionID), PreviousVersionID: uuidPointer(existing.PreviousVersionID)}, nil
	} else if !errors.Is(replayErr, pgx.ErrNoRows) {
		return StandardActivationRecord{}, runtimeManagerDBError(replayErr, false)
	}
	after, err := q.ActivateRuntimeStandardVersion(ctx, db.ActivateRuntimeStandardVersionParams{NewVersionID: versionUUID, StandardID: standardUUID, ExpectedActiveVersionID: expectedActive})
	if err != nil {
		return StandardActivationRecord{}, runtimeManagerDBError(err, true)
	}
	if _, err := q.RecordRuntimeStandardActivation(ctx, db.RecordRuntimeStandardActivationParams{
		StandardID: standardUUID, PreviousVersionID: before.ActiveVersionID,
		NewVersionID: versionUUID, ActorID: actor, RequestID: params.RequestID,
		Reason: params.Reason, CapabilityDigest: params.CapabilityDigest,
	}); err != nil {
		return StandardActivationRecord{}, runtimeManagerDBError(err, true)
	}
	if err := tx.Commit(ctx); err != nil {
		return StandardActivationRecord{}, runtimeManagerDBError(err, true)
	}
	return StandardActivationRecord{ActiveVersionID: util.UUIDToString(after.ActiveVersionID), PreviousVersionID: uuidPointer(before.ActiveVersionID)}, nil
}

// StaticCapabilityCatalog is a concrete immutable catalog suitable for a
// checked, versioned authority loaded by production composition.
type StaticCapabilityCatalog struct {
	providers map[string]runtimeconfig.ProviderCapabilities
}

func NewStaticCapabilityCatalog(providers map[string]runtimeconfig.ProviderCapabilities) (*StaticCapabilityCatalog, error) {
	if len(providers) == 0 {
		return nil, errors.New("runtime manager: capability catalog is empty")
	}
	cloned := make(map[string]runtimeconfig.ProviderCapabilities, len(providers))
	for provider, capabilities := range providers {
		if provider == "" || string(capabilities.Provider) != provider {
			return nil, fmt.Errorf("runtime manager: capability provider mismatch for %q", provider)
		}
		if _, err := runtimeconfig.CapabilityDigest(capabilities); err != nil {
			return nil, fmt.Errorf("runtime manager: invalid capabilities for %q: %w", provider, err)
		}
		copy, err := cloneJSON(capabilities)
		if err != nil {
			return nil, err
		}
		cloned[provider] = copy
	}
	return &StaticCapabilityCatalog{providers: cloned}, nil
}

func (c *StaticCapabilityCatalog) Capabilities(_ context.Context, provider string) (runtimeconfig.ProviderCapabilities, error) {
	capabilities, ok := c.providers[provider]
	if !ok {
		return runtimeconfig.ProviderCapabilities{}, ErrRuntimeManagerNotFound
	}
	return cloneJSON(capabilities)
}

// StaticPlatformLayerSource is an immutable platform floor. A nil layer is a
// deliberate empty platform declaration, not an unavailable dependency.
type StaticPlatformLayerSource struct {
	layer *runtimeconfig.Config
}

func NewStaticPlatformLayerSource(layer *runtimeconfig.Config) (*StaticPlatformLayerSource, error) {
	if layer == nil {
		return &StaticPlatformLayerSource{}, nil
	}
	if layer.Version != runtimeconfig.VersionV1 {
		return nil, errors.New("runtime manager: platform layer version must be v1")
	}
	copy, err := cloneJSON(*layer)
	if err != nil {
		return nil, err
	}
	return &StaticPlatformLayerSource{layer: &copy}, nil
}

func (s *StaticPlatformLayerSource) PlatformLayer(context.Context) (*runtimeconfig.Config, error) {
	if s.layer == nil {
		return nil, nil
	}
	copy, err := cloneJSON(*s.layer)
	if err != nil {
		return nil, err
	}
	return &copy, nil
}

func cloneJSON[T any](value T) (T, error) {
	var out T
	raw, err := json.Marshal(value)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, err
	}
	return out, nil
}

func (s *PostgresRuntimeManagerStore) standardRecord(ctx context.Context, row db.RuntimeStandard) (RuntimeStandardRecord, error) {
	record := RuntimeStandardRecord{ID: util.UUIDToString(row.ID), Name: row.Name, Description: util.TextToPtr(row.Description), ActiveVersionID: uuidPointer(row.ActiveVersionID), UpdatedAt: row.UpdatedAt.Time}
	if row.ActiveVersionID.Valid {
		versions, err := db.New(s.pool).ListRuntimeStandardVersions(ctx, row.ID)
		if err != nil {
			return RuntimeStandardRecord{}, runtimeManagerDBError(err, false)
		}
		for _, version := range versions {
			if version.ID == row.ActiveVersionID {
				n := version.VersionNumber
				record.ActiveVersionNumber = &n
				break
			}
		}
	}
	return record, nil
}

func bindingRecord(row db.RuntimeBinding, provider string) RuntimeBindingRecord {
	return RuntimeBindingRecord{ID: util.UUIDToString(row.ID), WorkspaceID: util.UUIDToString(row.WorkspaceID), Provider: provider, TransportBinding: row.TransportBinding, BindingGeneration: row.Generation, ActiveConfigurationVersionID: uuidPointer(row.ActiveConfigurationVersionID), State: row.State}
}

func bindingVersionRecord(row db.RuntimeConfigurationVersion, active *string) RuntimeConfigurationVersionRecord {
	state := "inactive"
	if active != nil && *active == util.UUIDToString(row.ID) {
		state = "active"
	}
	return RuntimeConfigurationVersionRecord{ID: util.UUIDToString(row.ID), VersionNumber: row.VersionNumber, Document: append([]byte(nil), row.Configuration...), Digest: row.ConfigurationDigest, ApplyClass: row.ApplyClass, Reason: row.Reason, State: state, CreatedAt: row.CreatedAt.Time}
}

func standardVersionRecord(row db.RuntimeStandardVersion, active *string) (RuntimeConfigurationVersionRecord, error) {
	state := "inactive"
	if active != nil && *active == util.UUIDToString(row.ID) {
		state = "active"
	}
	return RuntimeConfigurationVersionRecord{ID: util.UUIDToString(row.ID), VersionNumber: row.VersionNumber, Document: append([]byte(nil), row.Configuration...), Digest: row.ConfigurationDigest, ApplyClass: row.ApplyClass, Reason: row.Reason, State: state, CreatedAt: row.CreatedAt.Time}, nil
}

func nextBindingVersion(rows []db.RuntimeConfigurationVersion) int64 {
	var max int64
	for _, row := range rows {
		if row.VersionNumber > max {
			max = row.VersionNumber
		}
	}
	return max + 1
}
func nextStandardVersion(rows []db.RuntimeStandardVersion) int64 {
	var max int64
	for _, row := range rows {
		if row.VersionNumber > max {
			max = row.VersionNumber
		}
	}
	return max + 1
}

func runtimeManagerScopeUUIDs(workspaceID, resourceID string) (pgtype.UUID, pgtype.UUID, error) {
	workspaceUUID, err := util.ParseUUID(workspaceID)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, err
	}
	resourceUUID, err := util.ParseUUID(resourceID)
	return workspaceUUID, resourceUUID, err
}

func optionalUUID(value *string) (pgtype.UUID, error) {
	if value == nil {
		return pgtype.UUID{}, nil
	}
	uuid, err := util.ParseUUID(*value)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return uuid, nil
}

func optionalTextMatches(stored pgtype.Text, expected *string) bool {
	if expected == nil {
		return !stored.Valid
	}
	return stored.Valid && stored.String == *expected
}

func uuidPointer(value pgtype.UUID) *string {
	if !value.Valid {
		return nil
	}
	text := util.UUIDToString(value)
	return &text
}

func runtimeManagerCursor(cursor string, length int) (int, error) {
	if cursor == "" {
		return 0, nil
	}
	start, err := strconv.Atoi(cursor)
	if err != nil || start < 0 || start > length {
		return 0, ErrRuntimeManagerNotFound
	}
	return start, nil
}
func runtimeManagerPageBounds(start, limit, length int) (int, string) {
	end := start + limit
	if end >= length {
		return length, ""
	}
	return end, strconv.Itoa(end)
}

func runtimeManagerDBError(err error, conflict bool) error {
	if errors.Is(err, pgx.ErrNoRows) {
		if conflict {
			return ErrRuntimeManagerConflict
		}
		return ErrRuntimeManagerNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrRuntimeManagerConflict
	}
	return err
}

var _ RuntimeConfigurationStore = (*PostgresRuntimeManagerStore)(nil)
var _ RuntimeStandardStore = (*PostgresRuntimeManagerStore)(nil)
var _ CapabilityCatalog = (*StaticCapabilityCatalog)(nil)
var _ PlatformLayerSource = (*StaticPlatformLayerSource)(nil)
