-- Empty-only rollback. ACCESS EXCLUSIVE locks close the check/drop race and
-- leave the Migration-137 authority byte-for-byte untouched.
LOCK TABLE
    runtime_credential_readiness_probe_request,
    runtime_credential_readiness_probe_result
IN ACCESS EXCLUSIVE MODE;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM runtime_credential_readiness_probe_result LIMIT 1
    ) OR EXISTS (
        SELECT 1 FROM runtime_credential_readiness_probe_request LIMIT 1
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55006',
            MESSAGE = 'cannot roll back credential-readiness probes after use';
    END IF;
END
$$;

DROP TRIGGER runtime_credential_readiness_probe_result_guard
    ON runtime_credential_readiness_probe_result;
DROP FUNCTION enforce_runtime_credential_readiness_probe_result();
DROP TRIGGER runtime_credential_readiness_probe_request_guard
    ON runtime_credential_readiness_probe_request;
DROP FUNCTION enforce_runtime_credential_readiness_probe_request();
DROP TABLE runtime_credential_readiness_probe_result;
DROP TABLE runtime_credential_readiness_probe_request;
