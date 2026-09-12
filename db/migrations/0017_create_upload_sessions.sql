CREATE TABLE IF NOT EXISTS upload_sessions (
    id             UUID PRIMARY KEY,
    user_id        BIGINT NOT NULL REFERENCES users(id),
    name           TEXT NOT NULL,
    extension      TEXT NOT NULL,
    parent         TEXT NOT NULL DEFAULT '',
    total_size     BIGINT NOT NULL,
    bytes_received BIGINT NOT NULL DEFAULT 0,
    created_at     TIMESTAMP NOT NULL DEFAULT now(),
    updated_at     TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_upload_sessions_updated_at ON upload_sessions (updated_at);
