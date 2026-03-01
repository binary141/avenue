CREATE TABLE IF NOT EXISTS files (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    extension  TEXT NOT NULL,
    mime_type  TEXT NOT NULL,
    file_size  BIGINT NOT NULL DEFAULT 0,
    parent     TEXT NOT NULL DEFAULT '',
    created_by TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now()
)
