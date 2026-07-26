package db

import (
	"database/sql"
	"strings"
	"time"

	"avenue/backend/sdk"

	"github.com/google/uuid"
)

func CreateFile(file *sdk.File) (string, error) {
	if file.UUID == "" {
		file.UUID = uuid.NewString()
	}
	err := DB.QueryRow(`
		INSERT INTO files (uuid, name, extension, mime_type, file_size, parent_id, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5,
			CASE WHEN $6 = '' THEN NULL
			     ELSE (SELECT id FROM folders WHERE uuid = $6)
			END,
			$7, now())
		RETURNING id
	`, file.UUID, file.Name, file.Extension, file.MimeType, file.FileSize, file.Parent, file.CreatedBy).Scan(&file.ID)
	if err != nil {
		return "", err
	}

	if err := UpdateUsage(file.CreatedBy, file.FileSize); err != nil {
		return "", err
	}

	return file.UUID, nil
}

func GetFileByID(id, creatorID string) (*sdk.File, error) {
	var f sdk.File
	var checksum sql.NullString
	err := DB.QueryRow(`
		SELECT id, uuid, name, extension, mime_type, file_size, checksum, created_by, created_at
		FROM files WHERE uuid=$1 AND created_by=$2::BIGINT AND deleted_at IS NULL
	`, id, creatorID).Scan(&f.ID, &f.UUID, &f.Name, &f.Extension, &f.MimeType, &f.FileSize, &checksum, &f.CreatedBy, &f.CreatedAt)
	if err != nil {
		return nil, err
	}
	f.Checksum = checksum.String
	return &f, nil
}

// GetFileByIDForUser returns a file if the user is either its creator or owns
// the folder it lives in. Used where folder owners need to manage uploaded files.
//
// The parent folder's UUID is included (via a join) so callers that fetch a
// file and pass it straight to UpdateFile don't accidentally null out
// parent_id — UpdateFile treats an empty Parent as "move to root".
func GetFileByIDForUser(id, userID string) (*sdk.File, error) {
	var f sdk.File
	var parent, checksum sql.NullString
	err := DB.QueryRow(`
		SELECT f.id, f.uuid, f.name, f.extension, f.mime_type, f.file_size, f.checksum, f.created_by, f.created_at, p.uuid
		FROM files f
		LEFT JOIN folders p ON p.id = f.parent_id
		WHERE f.uuid=$1 AND f.deleted_at IS NULL
		  AND (f.created_by=$2::BIGINT
		       OR f.parent_id IN (SELECT id FROM folders WHERE owner_id=$2::BIGINT))
	`, id, userID).Scan(&f.ID, &f.UUID, &f.Name, &f.Extension, &f.MimeType, &f.FileSize, &checksum, &f.CreatedBy, &f.CreatedAt, &parent)
	if err != nil {
		return nil, err
	}
	f.Parent = parent.String
	f.Checksum = checksum.String
	return &f, nil
}

func GetFileByIDPublic(id string) (*sdk.File, error) {
	var f sdk.File
	var checksum sql.NullString
	err := DB.QueryRow(`
		SELECT id, uuid, name, extension, mime_type, file_size, checksum, created_by, created_at
		FROM files WHERE uuid=$1 AND deleted_at IS NULL
	`, id).Scan(&f.ID, &f.UUID, &f.Name, &f.Extension, &f.MimeType, &f.FileSize, &checksum, &f.CreatedBy, &f.CreatedAt)
	if err != nil {
		return nil, err
	}
	f.Checksum = checksum.String
	return &f, nil
}

func ListFiles(creatorID string) ([]sdk.File, error) {
	rows, err := DB.Query(`
		SELECT id, uuid, name, extension, mime_type, file_size, checksum, created_by, created_at
		FROM files WHERE created_by=$1::BIGINT AND deleted_at IS NULL
	`, creatorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []sdk.File
	for rows.Next() {
		var f sdk.File
		var checksum sql.NullString
		if err := rows.Scan(&f.ID, &f.UUID, &f.Name, &f.Extension, &f.MimeType, &f.FileSize, &checksum, &f.CreatedBy, &f.CreatedAt); err != nil {
			return nil, err
		}
		f.Checksum = checksum.String
		files = append(files, f)
	}
	return files, rows.Err()
}

// DeleteFile hard-deletes a file record. This is only meant for rolling back
// a file record created during an upload that failed partway through — user
// facing deletes go through TrashFile/TrashFileForUser instead.
func DeleteFile(id, creatorID string) error {
	_, err := DB.Exec(`DELETE FROM files WHERE uuid=$1 AND created_by=$2::BIGINT`, id, creatorID)
	return err
}

// DeleteFileForUser hard-deletes a file if the user is its creator or owns
// its parent folder. Only meant for upload rollback, see DeleteFile.
func DeleteFileForUser(id, userID string) error {
	_, err := DB.Exec(`
		DELETE FROM files WHERE uuid=$1
		  AND (created_by=$2::BIGINT
		       OR parent_id IN (SELECT id FROM folders WHERE owner_id=$2::BIGINT))
	`, id, userID)
	return err
}

// TrashFileForUser soft-deletes a file if the user is its creator or owns
// its parent folder, and revokes any share links pointing at it.
func TrashFileForUser(id, userID string) error {
	tx, err := DB.Beginx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.Exec(`
		UPDATE files SET deleted_at = now() WHERE uuid=$1 AND deleted_at IS NULL
		  AND (created_by=$2::BIGINT
		       OR parent_id IN (SELECT id FROM folders WHERE owner_id=$2::BIGINT))
	`, id, userID)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return sql.ErrNoRows
	}

	if _, err := tx.Exec(`DELETE FROM share_links WHERE file_id = (SELECT id FROM files WHERE uuid=$1)`, id); err != nil {
		return err
	}

	return tx.Commit()
}

// RestoreFileForUser un-trashes a file if the user is its creator or owns
// its parent folder.
func RestoreFileForUser(id, userID string) error {
	res, err := DB.Exec(`
		UPDATE files SET deleted_at = NULL WHERE uuid=$1 AND deleted_at IS NOT NULL
		  AND (created_by=$2::BIGINT
		       OR parent_id IN (SELECT id FROM folders WHERE owner_id=$2::BIGINT))
	`, id, userID)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// PurgeFileForUser permanently deletes a trashed file if the user is its
// creator or owns its parent folder, and returns the deleted row so the
// caller can remove its blob from the filesystem and reconcile quota usage.
func PurgeFileForUser(id, userID string) (*sdk.File, error) {
	var f sdk.File
	err := DB.QueryRow(`
		DELETE FROM files WHERE uuid=$1 AND deleted_at IS NOT NULL
		  AND (created_by=$2::BIGINT
		       OR parent_id IN (SELECT id FROM folders WHERE owner_id=$2::BIGINT))
		RETURNING uuid, created_by, file_size
	`, id, userID).Scan(&f.UUID, &f.CreatedBy, &f.FileSize)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// ListTrashedFiles returns the files the user explicitly trashed — i.e.
// trashed files whose parent folder is not itself trashed. Files that are
// only in the trash because their parent folder was trashed are omitted;
// they come back into view when that folder is restored.
func ListTrashedFiles(userID string) ([]sdk.File, error) {
	rows, err := DB.Query(`
		SELECT f.id, f.uuid, f.name, f.extension, f.mime_type, f.file_size, f.checksum, f.created_by, f.created_at, f.deleted_at
		FROM files f
		LEFT JOIN folders p ON p.id = f.parent_id
		WHERE f.created_by = $1::BIGINT AND f.deleted_at IS NOT NULL
		  AND (p.id IS NULL OR p.deleted_at IS NULL)
		ORDER BY f.deleted_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []sdk.File
	for rows.Next() {
		var f sdk.File
		var checksum sql.NullString
		if err := rows.Scan(&f.ID, &f.UUID, &f.Name, &f.Extension, &f.MimeType, &f.FileSize, &checksum, &f.CreatedBy, &f.CreatedAt, &f.DeletedAt); err != nil {
			return nil, err
		}
		f.Checksum = checksum.String
		files = append(files, f)
	}
	return files, rows.Err()
}

// ListTrashedFilesPage is the paginated counterpart to ListTrashedFiles, used
// by the trash-listing endpoint. ListTrashedFiles itself stays unpaginated
// since EmptyTrash needs every trashed file to purge them all.
func ListTrashedFilesPage(userID string, limit, offset int) ([]sdk.File, error) {
	rows, err := DB.Query(`
		SELECT f.id, f.uuid, f.name, f.extension, f.mime_type, f.file_size, f.checksum, f.created_by, f.created_at, f.deleted_at
		FROM files f
		LEFT JOIN folders p ON p.id = f.parent_id
		WHERE f.created_by = $1::BIGINT AND f.deleted_at IS NOT NULL
		  AND (p.id IS NULL OR p.deleted_at IS NULL)
		ORDER BY f.deleted_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []sdk.File
	for rows.Next() {
		var f sdk.File
		var checksum sql.NullString
		if err := rows.Scan(&f.ID, &f.UUID, &f.Name, &f.Extension, &f.MimeType, &f.FileSize, &checksum, &f.CreatedBy, &f.CreatedAt, &f.DeletedAt); err != nil {
			return nil, err
		}
		f.Checksum = checksum.String
		files = append(files, f)
	}
	return files, rows.Err()
}

// CountTrashedFiles returns the total number of trashed files for userID,
// for computing pagination totals alongside ListTrashedFilesPage.
func CountTrashedFiles(userID string) (int, error) {
	var count int
	err := DB.QueryRow(`
		SELECT COUNT(*)
		FROM files f
		LEFT JOIN folders p ON p.id = f.parent_id
		WHERE f.created_by = $1::BIGINT AND f.deleted_at IS NOT NULL
		  AND (p.id IS NULL OR p.deleted_at IS NULL)
	`, userID).Scan(&count)
	return count, err
}

// ListExpiredTrashedFiles returns files that have been sitting in the trash
// longer than cutoff and are eligible for a system-wide sweep — i.e. trashed
// files whose parent folder is not itself trashed. Files only in the trash
// because their parent folder was trashed get purged when that folder is
// swept instead (see ListExpiredTrashedFolders).
func ListExpiredTrashedFiles(cutoff time.Time) ([]sdk.File, error) {
	rows, err := DB.Query(`
		SELECT f.uuid, f.created_by, f.file_size
		FROM files f
		LEFT JOIN folders p ON p.id = f.parent_id
		WHERE f.deleted_at IS NOT NULL AND f.deleted_at < $1
		  AND (p.id IS NULL OR p.deleted_at IS NULL)
	`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []sdk.File
	for rows.Next() {
		var f sdk.File
		if err := rows.Scan(&f.UUID, &f.CreatedBy, &f.FileSize); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

// ListChildFilePublic lists all files in a folder regardless of who created them.
// Used by public shared folder endpoints.
func ListChildFilePublic(parentID string) ([]sdk.File, error) {
	rows, err := DB.Query(`
		SELECT id, uuid, name, extension, mime_type, file_size, checksum, created_by, created_at
		FROM files
		WHERE parent_id = (SELECT id FROM folders WHERE uuid = $1) AND deleted_at IS NULL
	`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []sdk.File
	for rows.Next() {
		var f sdk.File
		var checksum sql.NullString
		if err := rows.Scan(&f.ID, &f.UUID, &f.Name, &f.Extension, &f.MimeType, &f.FileSize, &checksum, &f.CreatedBy, &f.CreatedAt); err != nil {
			return nil, err
		}
		f.Checksum = checksum.String
		files = append(files, f)
	}
	return files, rows.Err()
}

// ListChildFile lists the files in parentID. sortBy must already be validated
// by the caller to be one of "name", "file_size", or "created_at". Both
// sortBy and sortDir are interpolated directly into the query since
// placeholders can't parameterize ORDER BY; sortDir is a typed
// sdk.SortDirection so callers can't pass arbitrary strings.
func ListChildFile(parentID, ownerID, sortBy string, sortDir sdk.SortDirection, limit, offset int) ([]sdk.File, error) {
	var (
		rows *sql.Rows
		err  error
	)

	orderBy := " ORDER BY " + sortBy + " " + string(sortDir) + " LIMIT $2 OFFSET $3"

	if parentID == "" {
		// Root: only files the user themselves created with no parent
		rows, err = DB.Query(`
			SELECT id, uuid, name, extension, mime_type, file_size, checksum, created_by, created_at
			FROM files WHERE parent_id IS NULL AND created_by=$1::BIGINT AND deleted_at IS NULL
		`+orderBy, ownerID, limit, offset)
	} else {
		// Folder: all files inside a folder owned by this user, regardless of uploader
		orderBy = " ORDER BY " + sortBy + " " + string(sortDir) + " LIMIT $3 OFFSET $4"
		rows, err = DB.Query(`
			SELECT id, uuid, name, extension, mime_type, file_size, checksum, created_by, created_at
			FROM files
			WHERE parent_id = (SELECT id FROM folders WHERE uuid = $1 AND owner_id = $2::BIGINT) AND deleted_at IS NULL
		`+orderBy, parentID, ownerID, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []sdk.File
	for rows.Next() {
		var f sdk.File
		var checksum sql.NullString
		if err := rows.Scan(&f.ID, &f.UUID, &f.Name, &f.Extension, &f.MimeType, &f.FileSize, &checksum, &f.CreatedBy, &f.CreatedAt); err != nil {
			return nil, err
		}
		f.Checksum = checksum.String
		files = append(files, f)
	}
	return files, rows.Err()
}

// CountChildFiles returns the total number of (non-deleted) files in
// parentID, for computing pagination totals alongside ListChildFile.
func CountChildFiles(parentID, ownerID string) (int, error) {
	var count int
	var err error

	if parentID == "" {
		err = DB.QueryRow(`
			SELECT COUNT(*) FROM files WHERE parent_id IS NULL AND created_by=$1::BIGINT AND deleted_at IS NULL
		`, ownerID).Scan(&count)
	} else {
		err = DB.QueryRow(`
			SELECT COUNT(*) FROM files
			WHERE parent_id = (SELECT id FROM folders WHERE uuid = $1 AND owner_id = $2::BIGINT) AND deleted_at IS NULL
		`, parentID, ownerID).Scan(&count)
	}
	return count, err
}

// escapeLikePattern escapes LIKE wildcard characters so user input is
// matched literally except for the trailing '%' SearchChildFiles appends.
func escapeLikePattern(s string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(s)
}

// SearchChildFiles returns files directly inside parentID (or the root, if
// parentID is "") owned by ownerID whose name starts with namePrefix.
func SearchChildFiles(parentID, ownerID, namePrefix string) ([]sdk.File, error) {
	pattern := escapeLikePattern(namePrefix) + "%"

	var (
		rows *sql.Rows
		err  error
	)

	if parentID == "" {
		rows, err = DB.Query(`
			SELECT id, uuid, name, extension, mime_type, file_size, checksum, created_by, created_at
			FROM files
			WHERE parent_id IS NULL AND created_by=$1::BIGINT AND name LIKE $2 ESCAPE '\' AND deleted_at IS NULL
		`, ownerID, pattern)
	} else {
		rows, err = DB.Query(`
			SELECT id, uuid, name, extension, mime_type, file_size, checksum, created_by, created_at
			FROM files
			WHERE parent_id = (SELECT id FROM folders WHERE uuid = $1 AND owner_id = $3::BIGINT)
			  AND name LIKE $2 ESCAPE '\' AND deleted_at IS NULL
		`, parentID, pattern, ownerID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []sdk.File
	for rows.Next() {
		var f sdk.File
		var checksum sql.NullString
		if err := rows.Scan(&f.ID, &f.UUID, &f.Name, &f.Extension, &f.MimeType, &f.FileSize, &checksum, &f.CreatedBy, &f.CreatedAt); err != nil {
			return nil, err
		}
		f.Checksum = checksum.String
		files = append(files, f)
	}
	return files, rows.Err()
}

// UpdateFile updates a file's metadata if userID is its creator or owns its
// parent folder.
func UpdateFile(f sdk.File, userID string) error {
	res, err := DB.Exec(`
		UPDATE files SET name=$2, extension=$3, mime_type=$4, file_size=$5, checksum=$7,
			parent_id = CASE WHEN $6 = '' THEN NULL
			                 ELSE (SELECT id FROM folders WHERE uuid = $6)
			            END
		WHERE uuid=$1
		  AND (created_by=$8::BIGINT
		       OR parent_id IN (SELECT id FROM folders WHERE owner_id=$8::BIGINT))
	`, f.UUID, f.Name, f.Extension, f.MimeType, f.FileSize, f.Parent, sql.NullString{String: f.Checksum, Valid: f.Checksum != ""}, userID)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// FileBlobRef is the minimal info needed to locate and reshard a file's blob
// on disk. Used by blob-storage maintenance tasks (see cmd/reshard-blobs)
// rather than normal request handling.
type FileBlobRef struct {
	UUID      string
	CreatedBy int64
	FileSize  int64
	Checksum  string
}

// ListAllFileBlobRefs returns every file row, including trashed ones whose
// blob hasn't been purged yet, for blob-storage maintenance tasks.
func ListAllFileBlobRefs() ([]FileBlobRef, error) {
	rows, err := DB.Query(`SELECT uuid, created_by, file_size, checksum FROM files`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var refs []FileBlobRef
	for rows.Next() {
		var ref FileBlobRef
		var checksum sql.NullString
		if err := rows.Scan(&ref.UUID, &ref.CreatedBy, &ref.FileSize, &checksum); err != nil {
			return nil, err
		}
		ref.Checksum = checksum.String
		refs = append(refs, ref)
	}
	return refs, rows.Err()
}

// SetFileChecksum backfills the checksum for a single file by uuid. Used by
// blob-storage maintenance tasks.
func SetFileChecksum(uuid, checksum string) error {
	_, err := DB.Exec(`UPDATE files SET checksum=$2 WHERE uuid=$1`, uuid, checksum)
	return err
}
