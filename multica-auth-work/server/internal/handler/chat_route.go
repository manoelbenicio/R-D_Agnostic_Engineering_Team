package handler

import (
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// resolveChatTurnAgent decides which agent runs this chat turn.
//
// Default is the session's own agent — for a session created without an
// explicit agent_id that is the default squad's TL (see CreateChatSession).
// It is replaced only when the message explicitly addresses another agent with
// an `@agent` mention (the direct-to-agent escape hatch): the target must be a
// live, runtime-bound agent of this workspace that the sender may reach.
//
// The session row is left untouched, so the escape hatch is per-message: the
// next untargeted message routes back to the TL.
//
// An unusable mention (malformed id, unknown/foreign/archived agent, agent with
// no runtime, private agent the sender cannot access) falls back to the session
// agent instead of failing the send. That mirrors mention behavior elsewhere in
// the product — an unresolvable mention renders as text and enqueues nothing
// rather than rejecting the message — and keeps a turn from being lost to an
// addressing typo. Each fallback is logged so a user reporting "my @mention was
// ignored" is diagnosable.
func (h *Handler) resolveChatTurnAgent(r *http.Request, userID, workspaceID string, session db.ChatSession, content string) pgtype.UUID {
	raw, ok := util.FirstAgentMention(content)
	if !ok {
		return session.AgentID
	}
	target, err := util.ParseUUID(raw)
	if err != nil {
		slog.Warn("chat direct mention ignored: unparsable agent id",
			"chat_session_id", uuidToString(session.ID), "mention_agent_id", raw)
		return session.AgentID
	}
	if target == session.AgentID {
		// Addressing the agent that already owns the session — nothing to
		// re-route, and skipping the lookups keeps that case cheap.
		return session.AgentID
	}

	agent, err := h.Queries.GetAgentInWorkspace(r.Context(), db.GetAgentInWorkspaceParams{
		ID:          target,
		WorkspaceID: session.WorkspaceID,
	})
	if err != nil {
		slog.Warn("chat direct mention ignored: agent not in this workspace",
			"chat_session_id", uuidToString(session.ID), "mention_agent_id", raw)
		return session.AgentID
	}
	if agent.ArchivedAt.Valid {
		slog.Warn("chat direct mention ignored: agent archived",
			"chat_session_id", uuidToString(session.ID), "mention_agent_id", raw)
		return session.AgentID
	}
	if !agent.RuntimeID.Valid {
		// Defensive: agent.runtime_id is NOT NULL today, but if that ever
		// relaxes, EnqueueChatTaskForAgent would answer
		// ErrChatTaskAgentNoRuntime and take the whole send down with it —
		// while the session agent is still a working destination.
		slog.Warn("chat direct mention ignored: agent has no runtime",
			"chat_session_id", uuidToString(session.ID), "mention_agent_id", raw)
		return session.AgentID
	}
	actorType, actorID := h.resolveActor(r, userID, workspaceID)
	if !h.canAccessPrivateAgent(r.Context(), agent, actorType, actorID, workspaceID) {
		slog.Warn("chat direct mention ignored: no access to private agent",
			"chat_session_id", uuidToString(session.ID), "mention_agent_id", raw)
		return session.AgentID
	}

	slog.Info("chat turn routed by direct mention",
		"chat_session_id", uuidToString(session.ID),
		"session_agent_id", uuidToString(session.AgentID),
		"target_agent_id", raw,
	)
	return target
}
