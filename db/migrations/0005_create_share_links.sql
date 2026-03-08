CREATE TABLE IF NOT EXISTS share_links (
    id         BIGSERIAL PRIMARY KEY,
    token      TEXT NOT NULL UNIQUE,
    file_id    TEXT NOT NULL,
    created_by TEXT NOT NULL,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT now()
);
