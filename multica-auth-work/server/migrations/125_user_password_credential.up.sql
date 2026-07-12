CREATE TABLE user_password_credential (
    user_id       UUID PRIMARY KEY REFERENCES "user"(id) ON DELETE CASCADE,
    password_hash TEXT NOT NULL CHECK (password_hash LIKE '$2%'),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
