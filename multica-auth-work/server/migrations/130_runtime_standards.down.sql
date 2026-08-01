-- Runtime Standard versions and activation audit are durable history.
-- Operational rollback must activate a prior immutable version; it must not
-- drop these tables or rewrite version/audit rows.
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 130 is non-destructive and cannot be rolled down',
        HINT = 'Activate a prior Runtime Standard version and deploy forward; preserve version and activation history.';
END;
$$;
