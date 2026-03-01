CREATE TABLE IF NOT EXISTS sessions (
    id         BIGSERIAL PRIMARY KEY,
    session_id TEXT NOT NULL UNIQUE,
    expires_at BIGINT NOT NULL,
    is_valid   BOOLEAN NOT NULL DEFAULT true,
    user_id    BIGINT NOT NULL
)
