CREATE TABLE share_folder_links (
    id            BIGSERIAL PRIMARY KEY,
    token         TEXT NOT NULL UNIQUE,
    folder_id     BIGINT NOT NULL REFERENCES folders(id) ON DELETE CASCADE,
    created_by    BIGINT NOT NULL,
    expires_at    TIMESTAMP,
    require_login BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMP DEFAULT now()
);
