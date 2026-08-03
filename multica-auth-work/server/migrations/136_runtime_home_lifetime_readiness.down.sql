-- Safety and audit state must never be discarded after use. Rollback is
-- permitted only before any lifetime, event, or readiness row exists.
-- The lock order matches lifetime writers (lifetime before event) and closes
-- the check/drop window against every application writer.
LOCK TABLE
    runtime_home_lifetime,
    runtime_home_lifetime_event,
    runtime_credential_readiness_attestation
IN ACCESS EXCLUSIVE MODE;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM runtime_home_lifetime LIMIT 1)
       OR EXISTS (SELECT 1 FROM runtime_home_lifetime_event LIMIT 1)
       OR EXISTS (SELECT 1 FROM runtime_credential_readiness_attestation LIMIT 1) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55006',
            MESSAGE = 'cannot roll back runtime-home lifetime/readiness authority after use';
    END IF;
END
$$;

DROP TRIGGER runtime_credential_readiness_attestation_guard
    ON runtime_credential_readiness_attestation;
DROP FUNCTION enforce_runtime_credential_readiness_attestation();

DROP TRIGGER runtime_home_lifetime_event_guard
    ON runtime_home_lifetime_event;
DROP FUNCTION enforce_runtime_home_lifetime_event();

DROP TRIGGER runtime_home_lifetime_event_required
    ON runtime_home_lifetime;
DROP FUNCTION require_runtime_home_lifetime_event();

DROP TRIGGER runtime_home_lifetime_state_guard
    ON runtime_home_lifetime;
DROP FUNCTION enforce_runtime_home_lifetime_state();

DROP TABLE runtime_credential_readiness_attestation;
DROP TABLE runtime_home_lifetime_event;
DROP TABLE runtime_home_lifetime;

ALTER TABLE runtime_home_assignment
    DROP CONSTRAINT runtime_home_assignment_state_projection_key;
