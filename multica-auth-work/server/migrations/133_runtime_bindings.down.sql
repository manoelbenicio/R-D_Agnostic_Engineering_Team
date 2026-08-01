DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM runtime_binding WHERE state IN ('active', 'draining', 'blocked')
    ) OR EXISTS (
        SELECT 1 FROM runtime_home_assignment WHERE state IN ('active', 'draining')
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55006',
            MESSAGE = 'cannot roll back runtime bindings while active references remain';
    END IF;
END
$$;

DROP TRIGGER IF EXISTS runtime_home_assignment_native_only ON runtime_home_assignment;
DROP FUNCTION IF EXISTS enforce_native_runtime_home_assignment();
DROP TABLE runtime_home_assignment;
DROP TABLE runtime_binding;
ALTER TABLE runtime_session_enrollment
    DROP CONSTRAINT runtime_session_enrollment_projection_key;
