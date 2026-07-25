package db

import (
	"database/sql"
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

func GetFolder(folderID, userID string) (*sdk.Folder, error) {
	var f sdk.Folder
	err := DB.QueryRow(
		`SELECT id, uuid, name, COALESCE(parent_id, 0), owner_id FROM folders WHERE uuid=$1 AND owner_id=$2::BIGINT AND deleted_at IS NULL`,
		folderID, userID,
	).Scan(&f.ID, &f.UUID, &f.Name, &f.ParentID, &f.OwnerID)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func UpdateFolder(f sdk.Folder) error {
	_, err := DB.Exec(
		`UPDATE folders SET name=$2 WHERE uuid=$1 AND owner_id=$3`,
		f.UUID, f.Name, f.OwnerID,
	)
	return err
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
// folderID, owned by ownerID, along with the path of the containing folder
// relative to folderID's own name. Used to build a folder download zip that
// preserves the folder's directory structure.
func ListFolderFilesForZip(folderID, ownerID string) ([]ZipFileEntry, error) {
	rows, err := DB.Query(`
		WITH RECURSIVE subtree AS (
			SELECT id, name::text AS dir_path
			FROM folders
			WHERE uuid = $1 AND owner_id = $2::BIGINT AND deleted_at IS NULL
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
	`, folderID, ownerID)
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
// and all files within them), and revokes any share links pointing into
// that subtree.
func TrashFolder(folderID, ownerID string) error {
	var rootID int64
	err := DB.QueryRow(
		`SELECT id FROM folders WHERE uuid=$1 AND owner_id=$2::BIGINT AND deleted_at IS NULL`,
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

// ListChildFolder lists the child folders of parentID, ordered by name.
// sortDir is interpolated directly into the query since placeholders can't
// parameterize ORDER BY; it's a typed sdk.SortDirection so callers can't pass
// arbitrary strings.
func ListChildFolder(parentID, ownerID string, sortDir sdk.SortDirection) ([]sdk.Folder, error) {
	var (
		rows *sql.Rows
		err  error
	)

	if parentID == "" || parentID == rootFolderID {
		rows, err = DB.Query(
			`SELECT id, uuid, name, COALESCE(parent_id, 0), owner_id FROM folders WHERE parent_id IS NULL AND owner_id=$1::BIGINT AND deleted_at IS NULL ORDER BY name `+string(sortDir),
			ownerID,
		)
	} else {
		rows, err = DB.Query(`
			SELECT id, uuid, name, COALESCE(parent_id, 0), owner_id FROM folders
			WHERE parent_id = (SELECT id FROM folders WHERE uuid = $1)
			  AND owner_id = $2::BIGINT AND deleted_at IS NULL ORDER BY name `+string(sortDir),
			parentID, ownerID,
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

func ListFolderParents(folderID, ownerID string) ([]sdk.Folder, error) {
	rows, err := DB.Query(`
		WITH RECURSIVE folder_breadcrumbs (id, uuid, name, parent_id, owner_id) AS (
			SELECT id, uuid, name, parent_id, owner_id
			FROM folders
			WHERE owner_id = $1::BIGINT AND uuid = $2

			UNION ALL

			SELECT p.id, p.uuid, p.name, p.parent_id, p.owner_id
			FROM folders p
			INNER JOIN folder_breadcrumbs c ON p.id = c.parent_id
		)
		SELECT id, uuid, name, COALESCE(parent_id, 0), owner_id FROM folder_breadcrumbs
	`, ownerID, folderID)
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
