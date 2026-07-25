ALTER TABLE folders ADD COLUMN deleted_at TIMESTAMP;
ALTER TABLE files ADD COLUMN deleted_at TIMESTAMP;

CREATE INDEX idx_folders_deleted_at ON folders (deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX idx_files_deleted_at ON files (deleted_at) WHERE deleted_at IS NOT NULL;
