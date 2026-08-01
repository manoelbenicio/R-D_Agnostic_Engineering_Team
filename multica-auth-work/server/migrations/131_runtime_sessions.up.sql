-- SPE-6: owner-global, reusable, accountless Runtime Sessions and workspace
-- enrollment onto existing agents and runtime rows.
--
-- Sessions deliberately contain no provider account, credential-home, host
-- path, daemon filesystem identity, or credential material.

CREATE TABLE runtime_session (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    standard_id UUID NOT NULL,
    name TEXT NOT NULL CHECK (btrim(name) <> ''),
    provider TEXT NOT NULL CHECK (btrim(provider) <> ''),
    runtime_kind TEXT NOT NULL CHECK (btrim(runtime_kind) <> ''),
    created_by UUID NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT runtime_session_owner_name_key UNIQUE (owner_id, name),
    CONSTRAINT runtime_session_id_owner_key UNIQUE (id, owner_id),
    CONSTRAINT runtime_session_standard_owner_fkey
        FOREIGN KEY (standard_id, owner_id)
        REFERENCES runtime_standard(id, owner_id)
        ON DELETE RESTRICT
);

-- Existing agent.runtime_id has an FK, but historically it did not prove that
-- the agent and runtime belong to the same workspace. Enrollment depends on
-- that invariant, so fail with actionable IDs before strengthening it.
DO $$
DECLARE
    mismatches TEXT;
BEGIN
    SELECT string_agg(
               format('agent=%s agent_workspace=%s runtime=%s runtime_workspace=%s',
                      id, agent_workspace_id, runtime_id, runtime_workspace_id),
               '; ' ORDER BY id
           )
      INTO mismatches
      FROM (
          SELECT a.id,
                 a.workspace_id AS agent_workspace_id,
                 a.runtime_id,
                 ar.workspace_id AS runtime_workspace_id
          FROM agent a
          JOIN agent_runtime ar ON ar.id = a.runtime_id
          WHERE a.workspace_id <> ar.workspace_id
          ORDER BY a.id
          LIMIT 25
      ) invalid_agent_runtime;

    IF mismatches IS NOT NULL THEN
        RAISE EXCEPTION
            'SPE-6 preflight: agents reference runtimes in another workspace: %. Reassign each agent to a runtime in its own workspace before retrying migration 131.',
            mismatches;
    END IF;
END
$$;

ALTER TABLE agent_runtime
    ADD CONSTRAINT agent_runtime_id_workspace_key UNIQUE (id, workspace_id);

ALTER TABLE agent
    ADD CONSTRAINT agent_id_workspace_runtime_key
        UNIQUE (id, workspace_id, runtime_id),
    ADD CONSTRAINT agent_runtime_workspace_fkey
        FOREIGN KEY (runtime_id, workspace_id)
        REFERENCES agent_runtime(id, workspace_id)
        ON DELETE RESTRICT;

CREATE TABLE runtime_session_enrollment (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES runtime_session(id) ON DELETE RESTRICT,
    workspace_id UUID NOT NULL REFERENCES workspace(id) ON DELETE RESTRICT,
    runtime_id UUID NOT NULL,
    agent_id UUID NOT NULL,
    enrolled_by UUID NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    state TEXT NOT NULL DEFAULT 'active'
        CHECK (state IN ('active', 'draining', 'inactive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deactivated_at TIMESTAMPTZ,
    CONSTRAINT runtime_session_enrollment_workspace_session_key
        UNIQUE (workspace_id, session_id),
    CONSTRAINT runtime_session_enrollment_runtime_workspace_fkey
        FOREIGN KEY (runtime_id, workspace_id)
        REFERENCES agent_runtime(id, workspace_id)
        ON DELETE RESTRICT,
    CONSTRAINT runtime_session_enrollment_agent_projection_fkey
        FOREIGN KEY (agent_id, workspace_id, runtime_id)
        REFERENCES agent(id, workspace_id, runtime_id)
        ON DELETE RESTRICT,
    CONSTRAINT runtime_session_enrollment_deactivation_check
        CHECK (
            (state = 'inactive' AND deactivated_at IS NOT NULL)
            OR (state <> 'inactive' AND deactivated_at IS NULL)
        )
);

-- Draining rows retain exclusivity until they become inactive. This prevents a
-- session, agent, or runtime from being silently shared during teardown.
CREATE UNIQUE INDEX runtime_session_enrollment_active_agent_key
    ON runtime_session_enrollment(agent_id)
    WHERE state IN ('active', 'draining');
CREATE UNIQUE INDEX runtime_session_enrollment_active_runtime_key
    ON runtime_session_enrollment(runtime_id)
    WHERE state IN ('active', 'draining');

CREATE INDEX idx_runtime_session_owner
    ON runtime_session(owner_id, created_at, id);
CREATE INDEX idx_runtime_session_standard
    ON runtime_session(standard_id, created_at, id);
CREATE INDEX idx_runtime_session_enrollment_workspace
    ON runtime_session_enrollment(workspace_id, state, created_at, id);
