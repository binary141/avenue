CREATE TABLE IF NOT EXISTS users (
    id         BIGSERIAL PRIMARY KEY,
    email      TEXT NOT NULL UNIQUE,
    first_name TEXT,
    last_name  TEXT,
    password   TEXT NOT NULL,
    can_login  BOOLEAN NOT NULL DEFAULT true,
    is_admin   BOOLEAN NOT NULL DEFAULT false,
    quota      BIGINT NOT NULL DEFAULT 0,
    space_used BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    deleted_at TIMESTAMP
)
