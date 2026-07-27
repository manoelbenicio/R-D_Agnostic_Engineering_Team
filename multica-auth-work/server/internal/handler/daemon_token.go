package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/auth"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

type createDaemonTokenRequest struct {
	DaemonID  string `json:"daemon_id"`
	ExpiresAt string `json:"expires_at"`
}

// CreateDaemonToken mints an mdt_ token for an owner/admin-managed daemon.
// Only the hash is persisted; the raw token is returned once to the caller.
func (h *Handler) CreateDaemonToken(w http.ResponseWriter, r *http.Request) {
	var req createDaemonTokenRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	daemonID := strings.TrimSpace(req.DaemonID)
	workspaceID := strings.TrimSpace(chi.URLParam(r, "id"))
	if daemonID == "" || workspaceID == "" {
		writeError(w, http.StatusBadRequest, "daemon_id is required")
		return
	}
	wsID := parseUUID(workspaceID)
	if !wsID.Valid {
		writeError(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	expiresAt := time.Now().UTC().Add(24 * time.Hour)
	var err error
	if strings.TrimSpace(req.ExpiresAt) != "" {
		expiresAt, err = time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil || !expiresAt.After(time.Now().UTC()) {
			writeError(w, http.StatusBadRequest, "expires_at must be a future RFC3339 timestamp")
			return
		}
	}
	// Keep expired rows bounded as part of the issuance path. ORQ-43B owns
	// rotation/revocation; this is only the ORQ-43A expiry hygiene hook.
	if err := h.Queries.DeleteExpiredDaemonTokens(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to purge expired daemon tokens")
		return
	}
	raw, err := auth.GenerateDaemonToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate daemon token")
		return
	}
	row, err := h.Queries.CreateDaemonToken(r.Context(), db.CreateDaemonTokenParams{
		TokenHash: auth.HashToken(raw), WorkspaceID: wsID, DaemonID: daemonID,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to persist daemon token")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusCreated, map[string]any{"token": raw, "id": row.ID, "expires_at": row.ExpiresAt})
}

// RevokeDaemonToken revokes exactly one bound token and invalidates its cache
// entry. It intentionally cannot delete the sibling token in an overlap.
func (h *Handler) RevokeDaemonToken(w http.ResponseWriter, r *http.Request) {
	workspaceID := strings.TrimSpace(chi.URLParam(r, "id"))
	tokenID := strings.TrimSpace(chi.URLParam(r, "tokenId"))
	daemonID := strings.TrimSpace(r.URL.Query().Get("daemon_id"))
	if workspaceID == "" || tokenID == "" || daemonID == "" {
		writeError(w, http.StatusBadRequest, "workspace, token and daemon are required")
		return
	}
	wsID := parseUUID(workspaceID)
	id := parseUUID(tokenID)
	if !wsID.Valid || !id.Valid {
		writeError(w, http.StatusBadRequest, "invalid token binding")
		return
	}
	hash, err := h.Queries.RevokeDaemonToken(r.Context(), db.RevokeDaemonTokenParams{ID: id, WorkspaceID: wsID, DaemonID: daemonID})
	if err != nil {
		writeError(w, http.StatusNotFound, "daemon token not found")
		return
	}
	if h.DaemonTokenCache != nil {
		h.DaemonTokenCache.Invalidate(r.Context(), hash)
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusNoContent, nil)
}
