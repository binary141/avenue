package db

import (
	"database/sql"
	"errors"
)

// Access levels, as returned by the folder_access/file_access SQL functions
// (see migration 0019). Higher levels include everything below them.
const (
	AccessNone   = 0
	AccessViewer = 1 // list, preview, download
	AccessEditor = 2 // + upload, create, rename, and trash/move things inside the share
	AccessOwner  = 3 // + restore, purge, manage sharing
)

// AccessRole maps an access level to the role name used in the API.
func AccessRole(level int) string {
	switch level {
	case AccessOwner:
		return "owner"
	case AccessEditor:
		return "editor"
	case AccessViewer:
		return "viewer"
	default:
		return ""
	}
}

// FolderAccess returns userID's access level on the (non-trashed) folder
// with the given uuid. A folder that doesn't exist reports AccessNone rather
// than an error, so callers can treat "missing" and "not yours" identically
// and avoid leaking which folder uuids exist.
func FolderAccess(folderUUID, userID string) (int, error) {
	var level int
	err := DB.QueryRow(`
		SELECT folder_access(id, $2::BIGINT) FROM folders WHERE uuid = $1 AND deleted_at IS NULL
	`, folderUUID, userID).Scan(&level)
	if errors.Is(err, sql.ErrNoRows) {
		return AccessNone, nil
	}
	return level, err
}

// FileAccess is FolderAccess for a (non-trashed) file.
func FileAccess(fileUUID, userID string) (int, error) {
	var level sql.NullInt64
	err := DB.QueryRow(`
		SELECT file_access(id, $2::BIGINT) FROM files WHERE uuid = $1 AND deleted_at IS NULL
	`, fileUUID, userID).Scan(&level)
	if errors.Is(err, sql.ErrNoRows) {
		return AccessNone, nil
	}
	return int(level.Int64), err
}

// FolderOwnerID returns the owner of the folder with the given uuid. Every
// folder in a tree shares its root's owner (CreateFolder inherits it from the
// parent), so this is also who owns everything nested underneath it.
func FolderOwnerID(folderUUID string) (int64, error) {
	var owner int64
	err := DB.QueryRow(`SELECT owner_id FROM folders WHERE uuid = $1`, folderUUID).Scan(&owner)
	return owner, err
}

// CanRestructureFolder reports whether userID may trash or move the folder:
// its owner always can, and an editor can for anything inside a folder shared
// with them — but not for the shared folder itself (see
// folder_can_restructure). A missing folder reports false.
func CanRestructureFolder(folderUUID, userID string) (bool, error) {
	var ok sql.NullBool
	err := DB.QueryRow(`
		SELECT folder_can_restructure(id, $2::BIGINT) FROM folders WHERE uuid = $1 AND deleted_at IS NULL
	`, folderUUID, userID).Scan(&ok)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return ok.Bool, err
}

// CanRestructureFile is CanRestructureFolder for a file.
func CanRestructureFile(fileUUID, userID string) (bool, error) {
	var ok sql.NullBool
	err := DB.QueryRow(`
		SELECT file_can_restructure(id, $2::BIGINT) FROM files WHERE uuid = $1 AND deleted_at IS NULL
	`, fileUUID, userID).Scan(&ok)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return ok.Bool, err
}
