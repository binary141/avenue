package db

import (
	"database/sql"

	"avenue/backend/sdk"
)

// UpsertUserShare grants granteeID the given role ("viewer" or "editor") on
// a folder or file, or changes the role if they already have a grant on it.
// The caller is responsible for checking that sharedBy owns the resource.
func UpsertUserShare(resourceType, resourceUUID string, sharedBy, granteeID int64, role string) (int64, error) {
	var id int64
	var err error

	switch resourceType {
	case sdk.FolderItemTypeFolder:
		err = DB.QueryRow(`
			INSERT INTO user_shares (folder_id, shared_by, grantee_id, role)
			SELECT id, $2, $3, $4 FROM folders WHERE uuid = $1 AND deleted_at IS NULL
			ON CONFLICT (folder_id, grantee_id) WHERE folder_id IS NOT NULL DO UPDATE SET role = EXCLUDED.role
			RETURNING id
		`, resourceUUID, sharedBy, granteeID, role).Scan(&id)
	default:
		err = DB.QueryRow(`
			INSERT INTO user_shares (file_id, shared_by, grantee_id, role)
			SELECT id, $2, $3, $4 FROM files WHERE uuid = $1 AND deleted_at IS NULL
			ON CONFLICT (file_id, grantee_id) WHERE file_id IS NOT NULL DO UPDATE SET role = EXCLUDED.role
			RETURNING id
		`, resourceUUID, sharedBy, granteeID, role).Scan(&id)
	}
	return id, err
}

// ListUserSharesForFolder lists everyone a folder is directly shared with.
func ListUserSharesForFolder(folderUUID string) ([]sdk.UserShare, error) {
	return listUserShares(`
		SELECT s.id, 'folder', f.uuid, s.grantee_id, u.email, COALESCE(u.first_name, ''), COALESCE(u.last_name, ''), s.role, s.created_at
		FROM user_shares s
		JOIN folders f ON f.id = s.folder_id
		JOIN users u ON u.id = s.grantee_id
		WHERE f.uuid = $1
		ORDER BY u.email
	`, folderUUID)
}

// ListUserSharesForFile lists everyone a file is directly shared with.
func ListUserSharesForFile(fileUUID string) ([]sdk.UserShare, error) {
	return listUserShares(`
		SELECT s.id, 'file', f.uuid, s.grantee_id, u.email, COALESCE(u.first_name, ''), COALESCE(u.last_name, ''), s.role, s.created_at
		FROM user_shares s
		JOIN files f ON f.id = s.file_id
		JOIN users u ON u.id = s.grantee_id
		WHERE f.uuid = $1
		ORDER BY u.email
	`, fileUUID)
}

func listUserShares(query string, arg any) ([]sdk.UserShare, error) {
	rows, err := DB.Query(query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	shares := []sdk.UserShare{}
	for rows.Next() {
		var s sdk.UserShare
		if err := rows.Scan(&s.ID, &s.ResourceType, &s.ResourceID, &s.GranteeID, &s.GranteeEmail, &s.GranteeFirstName, &s.GranteeLastName, &s.Role, &s.CreatedAt); err != nil {
			return nil, err
		}
		shares = append(shares, s)
	}
	return shares, rows.Err()
}

// UpdateUserShareRole changes the role of grant shareID, provided userID
// owns the file or folder it was made on. Returns sql.ErrNoRows if there's
// no such grant or userID doesn't own its resource.
func UpdateUserShareRole(shareID int64, userID, role string) error {
	res, err := DB.Exec(`
		UPDATE user_shares SET role = $3
		WHERE id = $1
		  AND ((folder_id IS NOT NULL AND folder_access(folder_id, $2::BIGINT) = 3)
		    OR (file_id IS NOT NULL AND file_access(file_id, $2::BIGINT) = 3))
	`, shareID, userID, role)
	return requireRowsAffected(res, err)
}

// DeleteUserShare removes grant shareID. Either the resource's owner
// (revoking it) or the grantee themselves (leaving the share) may do so.
// Returns sql.ErrNoRows if there's no such grant or userID is neither.
func DeleteUserShare(shareID int64, userID string) error {
	res, err := DB.Exec(`
		DELETE FROM user_shares
		WHERE id = $1
		  AND (grantee_id = $2::BIGINT
		    OR (folder_id IS NOT NULL AND folder_access(folder_id, $2::BIGINT) = 3)
		    OR (file_id IS NOT NULL AND file_access(file_id, $2::BIGINT) = 3))
	`, shareID, userID)
	return requireRowsAffected(res, err)
}

func requireRowsAffected(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ListSharedWithUser returns the files and folders other users have shared
// directly with userID, newest first. Trashed items are left out.
func ListSharedWithUser(userID string) ([]sdk.SharedWithMeItem, error) {
	rows, err := DB.Query(`
		SELECT s.id, 'folder' AS item_type, f.uuid, f.name, '' AS extension, '' AS mime_type,
		       0::BIGINT AS file_size, s.role, s.shared_by, u.email, s.created_at
		FROM user_shares s
		JOIN folders f ON f.id = s.folder_id AND f.deleted_at IS NULL
		JOIN users u ON u.id = s.shared_by
		WHERE s.grantee_id = $1::BIGINT

		UNION ALL

		SELECT s.id, 'file' AS item_type, fi.uuid, fi.name, fi.extension, fi.mime_type,
		       fi.file_size, s.role, s.shared_by, u.email, s.created_at
		FROM user_shares s
		JOIN files fi ON fi.id = s.file_id AND fi.deleted_at IS NULL
		JOIN users u ON u.id = s.shared_by
		WHERE s.grantee_id = $1::BIGINT

		ORDER BY created_at DESC, id DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []sdk.SharedWithMeItem{}
	for rows.Next() {
		var it sdk.SharedWithMeItem
		if err := rows.Scan(&it.ShareID, &it.Type, &it.UUID, &it.Name, &it.Extension, &it.MimeType, &it.FileSize, &it.Role, &it.SharedByID, &it.SharedByEmail, &it.SharedAt); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}
