BEGIN;

-- ============================================================
-- 1. folders
--    folder_id TEXT PRIMARY KEY → uuid TEXT UNIQUE
--    new id BIGSERIAL PRIMARY KEY
--    parent TEXT → parent_id BIGINT NULL (NULL = root)
--    owner_id TEXT → BIGINT
-- ============================================================

ALTER TABLE folders ADD COLUMN id BIGSERIAL;
ALTER TABLE folders DROP CONSTRAINT folders_pkey;
ALTER TABLE folders ADD PRIMARY KEY (id);

ALTER TABLE folders RENAME COLUMN folder_id TO uuid;
ALTER TABLE folders ADD CONSTRAINT folders_uuid_unique UNIQUE (uuid);

ALTER TABLE folders ADD COLUMN parent_id BIGINT;
UPDATE folders child
    SET parent_id = parent_row.id
    FROM folders parent_row
    WHERE parent_row.uuid = child.parent
      AND child.parent <> '';
ALTER TABLE folders DROP COLUMN parent;

ALTER TABLE folders ALTER COLUMN owner_id TYPE BIGINT USING owner_id::BIGINT;

-- ============================================================
-- 2. files
--    id TEXT PRIMARY KEY → uuid TEXT UNIQUE
--    new id BIGSERIAL PRIMARY KEY
--    parent TEXT → parent_id BIGINT NULL (NULL = root)
--    created_by TEXT → BIGINT
-- ============================================================

ALTER TABLE files RENAME COLUMN id TO uuid;
ALTER TABLE files DROP CONSTRAINT files_pkey;
ALTER TABLE files ADD COLUMN id BIGSERIAL;
ALTER TABLE files ADD PRIMARY KEY (id);
ALTER TABLE files ADD CONSTRAINT files_uuid_unique UNIQUE (uuid);

ALTER TABLE files ADD COLUMN parent_id BIGINT;
UPDATE files fi
    SET parent_id = fo.id
    FROM folders fo
    WHERE fo.uuid = fi.parent
      AND fi.parent <> '';
ALTER TABLE files DROP COLUMN parent;

ALTER TABLE files ALTER COLUMN created_by TYPE BIGINT USING created_by::BIGINT;

-- ============================================================
-- 3. sessions
--    session_id TEXT → uuid TEXT
-- ============================================================

ALTER TABLE sessions RENAME COLUMN session_id TO uuid;

-- ============================================================
-- 4. share_links
--    file_id TEXT (uuid) → BIGINT FK to files.id
--    created_by TEXT → BIGINT
-- ============================================================

ALTER TABLE share_links ADD COLUMN file_id_new BIGINT;
UPDATE share_links sl
    SET file_id_new = f.id
    FROM files f
    WHERE f.uuid = sl.file_id;
ALTER TABLE share_links DROP COLUMN file_id;
ALTER TABLE share_links RENAME COLUMN file_id_new TO file_id;
ALTER TABLE share_links ALTER COLUMN file_id SET NOT NULL;

ALTER TABLE share_links ALTER COLUMN created_by TYPE BIGINT USING created_by::BIGINT;

COMMIT;
