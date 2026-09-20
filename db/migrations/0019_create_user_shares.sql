-- User-to-user sharing: a grant gives one user access to a single file or a
-- folder (and, because folder grants inherit, everything under it).
--
-- Access levels used throughout: 0 none, 1 viewer, 2 editor, 3 owner.
CREATE TABLE user_shares (
    id         BIGSERIAL PRIMARY KEY,
    folder_id  BIGINT REFERENCES folders(id) ON DELETE CASCADE,
    file_id    BIGINT REFERENCES files(id) ON DELETE CASCADE,
    shared_by  BIGINT NOT NULL REFERENCES users(id),
    grantee_id BIGINT NOT NULL REFERENCES users(id),
    role       TEXT NOT NULL CHECK (role IN ('viewer', 'editor')),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    CHECK ((folder_id IS NULL) <> (file_id IS NULL)),
    CHECK (shared_by <> grantee_id)
);

CREATE UNIQUE INDEX idx_user_shares_folder_grantee ON user_shares (folder_id, grantee_id) WHERE folder_id IS NOT NULL;
CREATE UNIQUE INDEX idx_user_shares_file_grantee ON user_shares (file_id, grantee_id) WHERE file_id IS NOT NULL;
CREATE INDEX idx_user_shares_grantee ON user_shares (grantee_id);

-- folder_access returns uid's access level on folder fid: the highest of
-- being the owner of fid or any ancestor (3), or holding a grant on fid or any
-- ancestor (2 for editor, 1 for viewer). Returns 0 when there is no access
-- and for a folder that doesn't exist.
CREATE FUNCTION folder_access(fid BIGINT, uid BIGINT) RETURNS INT
LANGUAGE sql STABLE AS $$
    WITH RECURSIVE chain AS (
        SELECT id, parent_id, owner_id FROM folders WHERE id = fid
        UNION ALL
        SELECT p.id, p.parent_id, p.owner_id
        FROM folders p
        INNER JOIN chain c ON p.id = c.parent_id
    )
    SELECT COALESCE(MAX(
        CASE WHEN c.owner_id = uid THEN 3
             ELSE COALESCE((
                 SELECT CASE s.role WHEN 'editor' THEN 2 ELSE 1 END
                 FROM user_shares s
                 WHERE s.folder_id = c.id AND s.grantee_id = uid
             ), 0)
        END
    ), 0)
    FROM chain c
$$;

-- file_access returns uid's access level on file fid: the creator is the
-- owner (3), otherwise the higher of a direct grant on the file and whatever
-- access uid has on the folder it lives in (see folder_access). NULL when the
-- file doesn't exist, so "file_access(...) >= n" is false for it.
CREATE FUNCTION file_access(fid BIGINT, uid BIGINT) RETURNS INT
LANGUAGE sql STABLE AS $$
    SELECT GREATEST(
        CASE WHEN f.created_by = uid THEN 3 ELSE 0 END,
        COALESCE((
            SELECT CASE s.role WHEN 'editor' THEN 2 ELSE 1 END
            FROM user_shares s
            WHERE s.file_id = f.id AND s.grantee_id = uid
        ), 0),
        CASE WHEN f.parent_id IS NULL THEN 0 ELSE folder_access(f.parent_id, uid) END
    )
    FROM files f
    WHERE f.id = fid
$$;

-- Trashing or moving a folder/file is a change to whatever contains it, so an
-- editor may only do it to things inside a shared folder — never to the item
-- a grant was made on directly (that's the owner's to trash or move). Owners
-- can always restructure.
CREATE FUNCTION folder_can_restructure(fid BIGINT, uid BIGINT) RETURNS BOOLEAN
LANGUAGE sql STABLE AS $$
    SELECT folder_access(f.id, uid) = 3
        OR (f.parent_id IS NOT NULL AND folder_access(f.parent_id, uid) >= 2)
    FROM folders f
    WHERE f.id = fid
$$;

CREATE FUNCTION file_can_restructure(fid BIGINT, uid BIGINT) RETURNS BOOLEAN
LANGUAGE sql STABLE AS $$
    SELECT file_access(f.id, uid) = 3
        OR (f.parent_id IS NOT NULL AND folder_access(f.parent_id, uid) >= 2)
    FROM files f
    WHERE f.id = fid
$$;
