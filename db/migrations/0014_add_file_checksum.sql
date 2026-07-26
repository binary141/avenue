ALTER TABLE files ADD COLUMN checksum CHAR(64);

-- Supports future dedup lookups (find other files with the same content).
CREATE INDEX idx_files_checksum ON files (checksum) WHERE checksum IS NOT NULL;
