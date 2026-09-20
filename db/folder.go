package db

import (
	"database/sql"
	"errors"
	"time"

	"avenue/backend/sdk"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

const rootFolderID = "c32af1cc-aba9-4878-a305-5006dc7a5b76"

func CreateFolder(f *sdk.Folder) (string, error) {
	if f.UUID == "" {
		f.UUID = uuid.NewString()
	}
	err := DB.QueryRow(`
		INSERT INTO folders (uuid, name, parent_id, owner_id)
		VALUES ($1, $2, NULLIF($3, 0), $4)
		RETURNING id, COALESCE(parent_id, 0)
	`, f.UUID, f.Name, f.ParentID, f.OwnerID).Scan(&f.ID, &f.ParentID)
	return f.UUID, err
}

// GetFolder returns a non-trashed folder that userID owns. Use
// GetFolderForUser where shared access is enough.
func GetFolder(folderID, userID string) (*sdk.Folder, error) {
	return GetFolderForUser(folderID, userID, AccessOwner)
}

// GetFolderForUser returns a non-trashed folder if userID has at least the
// given access level on it (AccessViewer, AccessEditor or AccessOwner) —
// through owning it, or a grant on it or a folder above it. Returns
// sql.ErrNoRows otherwise, whether or not the folder exists.
func GetFolderForUser(folderID, userID string, minAccess int) (*sdk.Folder, error) {
	var f sdk.Folder
	err := DB.QueryRow(
		`SELECT id, uuid, name, COALESCE(parent_id, 0), owner_id FROM folders
		 WHERE uuid=$1 AND deleted_at IS NULL AND folder_access(id, $2::BIGINT) >= $3`,
		folderID, userID, minAccess,
	).Scan(&f.ID, &f.UUID, &f.Name, &f.ParentID, &f.OwnerID)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// UpdateFolder renames a folder if userID has edit access to it. Returns
// sql.ErrNoRows if the folder doesn't exist or userID can't edit it.
func UpdateFolder(f sdk.Folder, userID string) error {
	res, err := DB.Exec(
		`UPDATE folders SET name=$2 WHERE uuid=$1 AND deleted_at IS NULL AND folder_access(id, $3::BIGINT) >= 2`,
		f.UUID, f.Name, userID,
	)
	return requireRowsAffected(res, err)
}

// folderSubtreeIDs returns the id of rootID plus the ids of every folder
// nested under it, at any depth.
func folderSubtreeIDs(rootID int64) ([]int64, error) {
	rows, err := DB.Query(`
		WITH RECURSIVE subtree AS (
			SELECT id FROM folders WHERE id = $1
			UNION ALL
			SELECT f.id FROM folders f
			INNER JOIN subtree s ON f.parent_id = s.id
		)
		SELECT id FROM subtree
	`, rootID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ZipFileEntry is a file within a folder download, along with its path
// relative to the root of the zip archive (rooted at the downloaded folder
// itself, so nested subfolders are preserved).
type ZipFileEntry struct {
	UUID      string
	Name      string
	CreatedBy int64
	CreatedAt time.Time
	DirPath   string
}

// ListFolderFilesForZip returns every file nested (at any depth) under
// folderID, if userID can see it, along with the path of the containing folder
// relative to folderID's own name. Used to build a folder download zip that
// preserves the folder's directory structure.
func ListFolderFilesForZip(folderID, userID string) ([]ZipFileEntry, error) {
	rows, err := DB.Query(`
		WITH RECURSIVE subtree AS (
			SELECT id, name::text AS dir_path
			FROM folders
			WHERE uuid = $1 AND folder_access(id, $2::BIGINT) >= 1 AND deleted_at IS NULL
			UNION ALL
			SELECT f.id, s.dir_path || '/' || f.name
			FROM folders f
			INNER JOIN subtree s ON f.parent_id = s.id
			WHERE f.deleted_at IS NULL
		)
		SELECT files.uuid, files.name, files.created_by, files.created_at, subtree.dir_path
		FROM subtree
		JOIN files ON files.parent_id = subtree.id
		WHERE files.deleted_at IS NULL
	`, folderID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []ZipFileEntry
	for rows.Next() {
		var e ZipFileEntry
		if err := rows.Scan(&e.UUID, &e.Name, &e.CreatedBy, &e.CreatedAt, &e.DirPath); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// TrashFolder soft-deletes folderID and its entire subtree (nested folders
// and all files within them) if userID owns it or has edit access to the
// folder it lives in (see folder_can_restructure), and revokes any share links pointing into
// that subtree.
func TrashFolder(folderID, userID string) error {
	var rootID int64
	err := DB.QueryRow(
		`SELECT id FROM folders WHERE uuid=$1 AND folder_can_restructure(id, $2::BIGINT) AND deleted_at IS NULL`,
		folderID, userID,
	).Scan(&rootID)
	if err != nil {
		return err
	}

	ids, err := folderSubtreeIDs(rootID)
	if err != nil {
		return err
	}

	tx, err := DB.Beginx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`UPDATE folders SET deleted_at = now() WHERE id = ANY($1) AND deleted_at IS NULL`, pq.Array(ids)); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE files SET deleted_at = now() WHERE parent_id = ANY($1) AND deleted_at IS NULL`, pq.Array(ids)); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM share_folder_links WHERE folder_id = ANY($1)`, pq.Array(ids)); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM share_links WHERE file_id IN (SELECT id FROM files WHERE parent_id = ANY($1))`, pq.Array(ids)); err != nil {
		return err
	}

	return tx.Commit()
}

// BulkTrash soft-deletes the given files and folders (recursively for each
// folder, same as TrashFolder) for userID in a single transaction. IDs that
// don't exist or that userID can't trash are silently skipped, matching the
// access semantics of TrashFileForUser/TrashFolder.
func BulkTrash(fileIDs, folderIDs []string, userID string) error {
	tx, err := DB.Beginx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var subtreeIDs []int64
	if len(folderIDs) > 0 {
		rows, err := tx.Query(`
			WITH RECURSIVE subtree AS (
				SELECT id FROM folders
				WHERE uuid = ANY($1::text[]) AND folder_can_restructure(id, $2::BIGINT) AND deleted_at IS NULL
				UNION ALL
				SELECT f.id FROM folders f
				INNER JOIN subtree s ON f.parent_id = s.id
				WHERE f.deleted_at IS NULL
			)
			SELECT id FROM subtree
		`, pq.Array(folderIDs), userID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			subtreeIDs = append(subtreeIDs, id)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
	}

	if len(subtreeIDs) > 0 {
		if _, err := tx.Exec(`UPDATE folders SET deleted_at = now() WHERE id = ANY($1) AND deleted_at IS NULL`, pq.Array(subtreeIDs)); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE files SET deleted_at = now() WHERE parent_id = ANY($1) AND deleted_at IS NULL`, pq.Array(subtreeIDs)); err != nil {
			return err
		}
		if _, err := tx.Exec(`DELETE FROM share_folder_links WHERE folder_id = ANY($1)`, pq.Array(subtreeIDs)); err != nil {
			return err
		}
		if _, err := tx.Exec(`DELETE FROM share_links WHERE file_id IN (SELECT id FROM files WHERE parent_id = ANY($1))`, pq.Array(subtreeIDs)); err != nil {
			return err
		}
	}

	if len(fileIDs) > 0 {
		if _, err := tx.Exec(`
			UPDATE files SET deleted_at = now() WHERE uuid = ANY($1::text[]) AND deleted_at IS NULL
			  AND file_can_restructure(id, $2::BIGINT)
		`, pq.Array(fileIDs), userID); err != nil {
			return err
		}
		if _, err := tx.Exec(`DELETE FROM share_links WHERE file_id IN (SELECT id FROM files WHERE uuid = ANY($1::text[]))`, pq.Array(fileIDs)); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// BulkRestore un-trashes the given files and folders (recursively for each
// folder, same as RestoreFolder) for userID in a single transaction. IDs
// that don't exist, aren't owned by userID, or aren't currently trashed are
// silently skipped, matching the ownership semantics of
// RestoreFileForUser/RestoreFolder.
func BulkRestore(fileIDs, folderIDs []string, userID string) error {
	tx, err := DB.Beginx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var subtreeIDs []int64
	if len(folderIDs) > 0 {
		rows, err := tx.Query(`
			WITH RECURSIVE roots AS (
				SELECT id FROM folders
				WHERE uuid = ANY($1::text[]) AND owner_id = $2::BIGINT AND deleted_at IS NOT NULL
			), subtree AS (
				SELECT id FROM roots
				UNION ALL
				SELECT f.id FROM folders f
				INNER JOIN subtree s ON f.parent_id = s.id
			)
			SELECT id FROM subtree
		`, pq.Array(folderIDs), userID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			subtreeIDs = append(subtreeIDs, id)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
	}

	if len(subtreeIDs) > 0 {
		if _, err := tx.Exec(`UPDATE folders SET deleted_at = NULL WHERE id = ANY($1)`, pq.Array(subtreeIDs)); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE files SET deleted_at = NULL WHERE parent_id = ANY($1)`, pq.Array(subtreeIDs)); err != nil {
			return err
		}
	}

	if len(fileIDs) > 0 {
		if _, err := tx.Exec(`
			UPDATE files SET deleted_at = NULL WHERE uuid = ANY($1::text[]) AND deleted_at IS NOT NULL
			  AND file_access(id, $2::BIGINT) = 3
		`, pq.Array(fileIDs), userID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ErrCrossOwnerMove is returned by BulkMove when an item would end up in a
// different owner's tree than it came from — e.g. an editor moving something
// out of a folder shared with them into their own drive. Quota and ownership
// are tracked per tree owner, so that isn't a plain move.
var ErrCrossOwnerMove = errors.New("cannot move items between different owners' drives")

// BulkMove moves the given files and folders to a new parent folder (or the
// root, if parent is "") for userID in a single transaction. userID needs edit
// access to the destination folder, and to be allowed to restructure each item
// (see folder_can_restructure/file_can_restructure); IDs that don't exist or
// that userID can't move are silently skipped, matching the access semantics
// of UpdateFile/UpdateFolder. Moving a folder into itself or one of its own
// descendants is rejected, since that would disconnect it from the tree, and
// so is moving anything across owners (ErrCrossOwnerMove).
func BulkMove(fileIDs, folderIDs []string, parent, userID string) error {
	tx, err := DB.Beginx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// destOwner is who owns the tree the items are landing in: the caller for
	// the root, otherwise the destination folder's owner.
	var parentID sql.NullInt64
	var destOwner int64
	if parent != "" {
		var id, owner int64
		if err := tx.QueryRow(
			`SELECT id, owner_id FROM folders WHERE uuid=$1 AND deleted_at IS NULL AND folder_access(id, $2::BIGINT) >= 2`,
			parent, userID,
		).Scan(&id, &owner); err != nil {
			return err
		}
		parentID = sql.NullInt64{Int64: id, Valid: true}
		destOwner = owner
	} else if err := tx.QueryRow(`SELECT $1::BIGINT`, userID).Scan(&destOwner); err != nil {
		return err
	}

	var crossOwner int
	if err := tx.QueryRow(`
		SELECT
			(SELECT COUNT(*) FROM folders
			 WHERE uuid = ANY($1::text[]) AND deleted_at IS NULL
			   AND folder_can_restructure(id, $3::BIGINT) AND owner_id <> $4)
			+
			(SELECT COUNT(*) FROM files
			 WHERE uuid = ANY($2::text[]) AND deleted_at IS NULL
			   AND file_can_restructure(id, $3::BIGINT)
			   AND COALESCE((SELECT p.owner_id FROM folders p WHERE p.id = files.parent_id), created_by) <> $4)
	`, pq.Array(folderIDs), pq.Array(fileIDs), userID, destOwner).Scan(&crossOwner); err != nil {
		return err
	}
	if crossOwner > 0 {
		return ErrCrossOwnerMove
	}

	if len(folderIDs) > 0 && parentID.Valid {
		var inSubtree bool
		if err := tx.QueryRow(`
			WITH RECURSIVE subtree AS (
				SELECT id FROM folders WHERE uuid = ANY($1::text[]) AND folder_can_restructure(id, $2::BIGINT)
				UNION ALL
				SELECT f.id FROM folders f INNER JOIN subtree s ON f.parent_id = s.id
			)
			SELECT EXISTS(SELECT 1 FROM subtree WHERE id = $3)
		`, pq.Array(folderIDs), userID, parentID.Int64).Scan(&inSubtree); err != nil {
			return err
		}
		if inSubtree {
			return errors.New("cannot move a folder into itself or one of its own subfolders")
		}
	}

	if len(folderIDs) > 0 {
		if _, err := tx.Exec(
			`UPDATE folders SET parent_id=$1 WHERE uuid = ANY($2::text[]) AND deleted_at IS NULL AND folder_can_restructure(id, $3::BIGINT)`,
			parentID, pq.Array(folderIDs), userID,
		); err != nil {
			return err
		}
	}

	if len(fileIDs) > 0 {
		if _, err := tx.Exec(`
			UPDATE files SET parent_id=$1 WHERE uuid = ANY($2::text[]) AND deleted_at IS NULL
			  AND file_can_restructure(id, $3::BIGINT)
		`, parentID, pq.Array(fileIDs), userID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// RestoreFolder un-trashes folderID and its entire trashed subtree.
//
// Note: if a file or subfolder inside folderID was independently trashed
// before folderID itself was trashed, restoring folderID will also restore
// it — a folder restore brings back everything currently inside it.
func RestoreFolder(folderID, ownerID string) error {
	var rootID int64
	err := DB.QueryRow(
		`SELECT id FROM folders WHERE uuid=$1 AND owner_id=$2::BIGINT AND deleted_at IS NOT NULL`,
		folderID, ownerID,
	).Scan(&rootID)
	if err != nil {
		return err
	}

	ids, err := folderSubtreeIDs(rootID)
	if err != nil {
		return err
	}

	tx, err := DB.Beginx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`UPDATE folders SET deleted_at = NULL WHERE id = ANY($1)`, pq.Array(ids)); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE files SET deleted_at = NULL WHERE parent_id = ANY($1)`, pq.Array(ids)); err != nil {
		return err
	}

	return tx.Commit()
}

// PurgeFolder permanently deletes a trashed folder and its entire subtree
// (folders and files) from the database. It returns the files that were
// deleted so the caller can remove their blobs from the filesystem and
// reconcile quota usage.
func PurgeFolder(folderID, ownerID string) ([]sdk.File, error) {
	var rootID int64
	err := DB.QueryRow(
		`SELECT id FROM folders WHERE uuid=$1 AND owner_id=$2::BIGINT AND deleted_at IS NOT NULL`,
		folderID, ownerID,
	).Scan(&rootID)
	if err != nil {
		return nil, err
	}

	ids, err := folderSubtreeIDs(rootID)
	if err != nil {
		return nil, err
	}

	tx, err := DB.Beginx()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.Query(`DELETE FROM files WHERE parent_id = ANY($1) RETURNING uuid, created_by, file_size`, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	var files []sdk.File
	for rows.Next() {
		var f sdk.File
		if err := rows.Scan(&f.UUID, &f.CreatedBy, &f.FileSize); err != nil {
			rows.Close()
			return nil, err
		}
		files = append(files, f)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	if _, err := tx.Exec(`DELETE FROM folders WHERE id = ANY($1)`, pq.Array(ids)); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return files, nil
}

// ListExpiredTrashedFolders returns folders that have been sitting in the
// trash longer than cutoff and are eligible for a system-wide sweep — i.e.
// trashed folders whose parent is not itself trashed. Nested folders/files
// are purged along with their expired ancestor by PurgeFolder, regardless of
// their own deleted_at.
func ListExpiredTrashedFolders(cutoff time.Time) ([]sdk.Folder, error) {
	rows, err := DB.Query(`
		SELECT f.uuid, f.owner_id
		FROM folders f
		LEFT JOIN folders p ON p.id = f.parent_id
		WHERE f.deleted_at IS NOT NULL AND f.deleted_at < $1
		  AND (p.id IS NULL OR p.deleted_at IS NULL)
	`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []sdk.Folder
	for rows.Next() {
		var f sdk.Folder
		if err := rows.Scan(&f.UUID, &f.OwnerID); err != nil {
			return nil, err
		}
		folders = append(folders, f)
	}
	return folders, rows.Err()
}

// ListTrashedFolders returns the folders the user explicitly trashed —
// i.e. trashed folders whose parent is not itself trashed. Folders that are
// only in the trash because an ancestor was trashed are omitted; they come
// back into view when that ancestor is restored.
func ListTrashedFolders(ownerID string) ([]sdk.Folder, error) {
	rows, err := DB.Query(`
		SELECT f.id, f.uuid, f.name, COALESCE(f.parent_id, 0), f.owner_id, f.deleted_at
		FROM folders f
		LEFT JOIN folders p ON p.id = f.parent_id
		WHERE f.owner_id = $1::BIGINT AND f.deleted_at IS NOT NULL
		  AND (p.id IS NULL OR p.deleted_at IS NULL)
		ORDER BY f.deleted_at DESC
	`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []sdk.Folder
	for rows.Next() {
		var f sdk.Folder
		if err := rows.Scan(&f.ID, &f.UUID, &f.Name, &f.ParentID, &f.OwnerID, &f.DeletedAt); err != nil {
			return nil, err
		}
		folders = append(folders, f)
	}
	return folders, rows.Err()
}

// CountTrashedFolders returns the total number of trashed folders for
// ownerID, for computing pagination totals alongside db.ListTrashedItems.
func CountTrashedFolders(ownerID string) (int, error) {
	var count int
	err := DB.QueryRow(`
		SELECT COUNT(*)
		FROM folders f
		LEFT JOIN folders p ON p.id = f.parent_id
		WHERE f.owner_id = $1::BIGINT AND f.deleted_at IS NOT NULL
		  AND (p.id IS NULL OR p.deleted_at IS NULL)
	`, ownerID).Scan(&count)
	return count, err
}

// ListChildFolder lists the child folders of parentID that userID can see,
// ordered by name.
// sortDir is interpolated directly into the query since placeholders can't
// parameterize ORDER BY; it's a typed sdk.SortDirection so callers can't pass
// arbitrary strings.
func ListChildFolder(parentID, userID string, sortDir sdk.SortDirection, limit, offset int) ([]sdk.Folder, error) {
	var (
		rows *sql.Rows
		err  error
	)

	if parentID == "" || parentID == rootFolderID {
		rows, err = DB.Query(
			`SELECT id, uuid, name, COALESCE(parent_id, 0), owner_id FROM folders WHERE parent_id IS NULL AND owner_id=$1::BIGINT AND deleted_at IS NULL ORDER BY name `+string(sortDir)+` LIMIT $2 OFFSET $3`,
			userID, limit, offset,
		)
	} else {
		rows, err = DB.Query(`
			SELECT id, uuid, name, COALESCE(parent_id, 0), owner_id FROM folders
			WHERE parent_id = (SELECT id FROM folders WHERE uuid = $1 AND folder_access(id, $2::BIGINT) >= 1)
				  AND deleted_at IS NULL ORDER BY name `+string(sortDir)+` LIMIT $3 OFFSET $4`,
			parentID, userID, limit, offset,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []sdk.Folder
	for rows.Next() {
		var f sdk.Folder
		if err := rows.Scan(&f.ID, &f.UUID, &f.Name, &f.ParentID, &f.OwnerID); err != nil {
			return nil, err
		}
		folders = append(folders, f)
	}
	return folders, rows.Err()
}

// CountChildFolders returns the total number of (non-deleted) child folders
// of parentID, for computing pagination totals alongside ListChildFolder.
func CountChildFolders(parentID, userID string) (int, error) {
	var count int
	var err error

	if parentID == "" || parentID == rootFolderID {
		err = DB.QueryRow(
			`SELECT COUNT(*) FROM folders WHERE parent_id IS NULL AND owner_id=$1::BIGINT AND deleted_at IS NULL`,
			userID,
		).Scan(&count)
	} else {
		err = DB.QueryRow(`
			SELECT COUNT(*) FROM folders
			WHERE parent_id = (SELECT id FROM folders WHERE uuid = $1 AND folder_access(id, $2::BIGINT) >= 1)
				  AND deleted_at IS NULL`,
			parentID, userID,
		).Scan(&count)
	}
	return count, err
}

// ListFolderParents returns folderID followed by its ancestors, leaf-first,
// stopping at the last folder userID can see. For a shared folder that means
// the trail ends at the shared folder itself instead of leaking the names of
// the owner's private folders above it.
func ListFolderParents(folderID, userID string) ([]sdk.Folder, error) {
	rows, err := DB.Query(`
		WITH RECURSIVE folder_breadcrumbs (id, uuid, name, parent_id, owner_id) AS (
			SELECT id, uuid, name, parent_id, owner_id
			FROM folders
			WHERE uuid = $2 AND folder_access(id, $1::BIGINT) >= 1

			UNION ALL

			SELECT p.id, p.uuid, p.name, p.parent_id, p.owner_id
			FROM folders p
			INNER JOIN folder_breadcrumbs c ON p.id = c.parent_id
			WHERE folder_access(p.id, $1::BIGINT) >= 1
		)
		SELECT id, uuid, name, COALESCE(parent_id, 0), owner_id FROM folder_breadcrumbs
	`, userID, folderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []sdk.Folder
	for rows.Next() {
		var f sdk.Folder
		if err := rows.Scan(&f.ID, &f.UUID, &f.Name, &f.ParentID, &f.OwnerID); err != nil {
			return nil, err
		}
		folders = append(folders, f)
	}
	return folders, rows.Err()
}
