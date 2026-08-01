-- Runtime Sessions and enrollment rows are durable identities and authorization
-- history. Operational rollback drains/deactivates enrollment and deploys
-- forward; it never deletes session identity or existing agent/runtime rows.
DO $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'migration 131 is non-destructive and cannot be rolled down',
        HINT = 'Drain or deactivate Runtime Session enrollments and deploy forward; preserve session and enrollment history.';
END;
$$;
