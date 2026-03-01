CREATE TABLE IF NOT EXISTS folders (
    folder_id TEXT PRIMARY KEY,
    name      TEXT NOT NULL,
    parent    TEXT NOT NULL DEFAULT '',
    owner_id  TEXT NOT NULL
)
