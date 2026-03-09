package db

import (
	"database/sql"
	"time"
)

// ShareFolderLink is returned from management endpoints and stored internally.
// FolderIntID is the integer FK used for subtree checks; it is not sent to clients.
type ShareFolderLink struct {
	ID           int64      `json:"id"`
	Token        string     `json:"token"`
	FolderUUID   string     `json:"folder_uuid"`
	FolderName   string     `json:"folder_name"`
	FolderIntID  int64      `json:"-"`
	CreatedBy    int64      `json:"created_by"`
	ExpiresAt    *time.Time `json:"expires_at"`
	CreatedAt    time.Time  `json:"created_at"`
	RequireLogin bool       `json:"require_login"`
	AllowUpload  bool       `json:"allow_upload"`
	MaxFileSize  int64      `json:"max_file_size"`
	LastAccessed *time.Time `json:"last_accessed"`
}

func CreateShareFolderLink(folderUUID, createdBy string, expiresAt *time.Time, requireLogin, allowUpload bool, maxFileSize int64) (ShareFolderLink, error) {
	token, err := generateToken()
	if err != nil {
		return ShareFolderLink{}, err
	}

	var link ShareFolderLink
	err = DB.QueryRow(`
		WITH ins AS (
			INSERT INTO share_folder_links (token, folder_id, created_by, expires_at, require_login, allow_upload, max_file_size)
			VALUES ($1, (SELECT id FROM folders WHERE uuid=$2), $3::BIGINT, $4, $5, $6, $7)
			RETURNING id, token, folder_id, created_by, expires_at, created_at, require_login, allow_upload, max_file_size
		)
		SELECT ins.id, ins.token, ins.folder_id, fo.uuid, fo.name,
		       ins.created_by, ins.expires_at, ins.created_at, ins.require_login, ins.allow_upload, ins.max_file_size
		FROM ins
		JOIN folders fo ON fo.id = ins.folder_id
	`, token, folderUUID, createdBy, expiresAt, requireLogin, allowUpload, maxFileSize).Scan(
		&link.ID, &link.Token, &link.FolderIntID, &link.FolderUUID, &link.FolderName,
		&link.CreatedBy, &link.ExpiresAt, &link.CreatedAt, &link.RequireLogin, &link.AllowUpload, &link.MaxFileSize,
	)
	if err != nil {
		return ShareFolderLink{}, err
	}
	return link, nil
}

func touchShareFolderLink(token string) {
	_, _ = DB.Exec(`UPDATE share_folder_links SET last_accessed = now() WHERE token = $1`, token)
}

func GetShareFolderLink(token string) (ShareFolderLink, error) {
	var link ShareFolderLink
	err := DB.QueryRow(`
		SELECT sl.id, sl.token, sl.folder_id, fo.uuid, fo.name,
		       sl.created_by, sl.expires_at, sl.created_at, sl.require_login, sl.allow_upload, sl.max_file_size, sl.last_accessed
		FROM share_folder_links sl
		JOIN folders fo ON fo.id = sl.folder_id
		WHERE sl.token = $1
		  AND (sl.expires_at IS NULL OR sl.expires_at > now())
	`, token).Scan(
		&link.ID, &link.Token, &link.FolderIntID, &link.FolderUUID, &link.FolderName,
		&link.CreatedBy, &link.ExpiresAt, &link.CreatedAt, &link.RequireLogin, &link.AllowUpload, &link.MaxFileSize, &link.LastAccessed,
	)
	if err != nil {
		return ShareFolderLink{}, sql.ErrNoRows
	}
	touchShareFolderLink(token)
	return link, nil
}

func ListShareFoldersByFolder(folderUUID, createdBy string) ([]ShareFolderLink, error) {
	rows, err := DB.Query(`
		SELECT sl.id, sl.token, sl.folder_id, fo.uuid, fo.name,
		       sl.created_by, sl.expires_at, sl.created_at, sl.require_login, sl.allow_upload, sl.max_file_size, sl.last_accessed
		FROM share_folder_links sl
		JOIN folders fo ON fo.id = sl.folder_id
		WHERE fo.uuid = $1 AND sl.created_by = $2::BIGINT
		  AND (sl.expires_at IS NULL OR sl.expires_at > now())
		ORDER BY sl.created_at DESC
	`, folderUUID, createdBy)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []ShareFolderLink
	for rows.Next() {
		var l ShareFolderLink
		if err := rows.Scan(&l.ID, &l.Token, &l.FolderIntID, &l.FolderUUID, &l.FolderName,
			&l.CreatedBy, &l.ExpiresAt, &l.CreatedAt, &l.RequireLogin, &l.AllowUpload, &l.MaxFileSize, &l.LastAccessed); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, rows.Err()
}

func ListShareFoldersByUser(createdBy string) ([]ShareFolderLink, error) {
	rows, err := DB.Query(`
		SELECT sl.id, sl.token, sl.folder_id, COALESCE(fo.uuid, ''), COALESCE(fo.name, ''),
		       sl.created_by, sl.expires_at, sl.created_at, sl.require_login, sl.allow_upload, sl.max_file_size, sl.last_accessed
		FROM share_folder_links sl
		LEFT JOIN folders fo ON fo.id = sl.folder_id
		WHERE sl.created_by = $1::BIGINT
		  AND (sl.expires_at IS NULL OR sl.expires_at > now())
		ORDER BY sl.created_at DESC
	`, createdBy)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []ShareFolderLink
	for rows.Next() {
		var l ShareFolderLink
		if err := rows.Scan(&l.ID, &l.Token, &l.FolderIntID, &l.FolderUUID, &l.FolderName,
			&l.CreatedBy, &l.ExpiresAt, &l.CreatedAt, &l.RequireLogin, &l.AllowUpload, &l.MaxFileSize, &l.LastAccessed); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, rows.Err()
}

func ListExpiredShareFoldersByUser(createdBy string) ([]ShareFolderLink, error) {
	rows, err := DB.Query(`
		SELECT sl.id, sl.token, sl.folder_id, COALESCE(fo.uuid, ''), COALESCE(fo.name, ''),
		       sl.created_by, sl.expires_at, sl.created_at, sl.require_login, sl.allow_upload, sl.max_file_size, sl.last_accessed
		FROM share_folder_links sl
		LEFT JOIN folders fo ON fo.id = sl.folder_id
		WHERE sl.created_by = $1::BIGINT
		  AND sl.expires_at IS NOT NULL AND sl.expires_at <= now()
		ORDER BY sl.expires_at DESC
	`, createdBy)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []ShareFolderLink
	for rows.Next() {
		var l ShareFolderLink
		if err := rows.Scan(&l.ID, &l.Token, &l.FolderIntID, &l.FolderUUID, &l.FolderName,
			&l.CreatedBy, &l.ExpiresAt, &l.CreatedAt, &l.RequireLogin, &l.AllowUpload, &l.MaxFileSize, &l.LastAccessed); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, rows.Err()
}

func DeleteShareFolderLink(token, createdBy string) error {
	_, err := DB.Exec(`DELETE FROM share_folder_links WHERE token = $1 AND created_by = $2::BIGINT`, token, createdBy)
	return err
}

// IsFolderInSubtree returns true if the folder identified by candidateUUID is
// the root folder (rootFolderID) or a descendant of it.
func IsFolderInSubtree(rootFolderID int64, candidateUUID string) (bool, error) {
	var exists bool
	err := DB.QueryRow(`
		WITH RECURSIVE subtree AS (
			SELECT id FROM folders WHERE id = $1
			UNION ALL
			SELECT f.id FROM folders f
			INNER JOIN subtree s ON f.parent_id = s.id
		)
		SELECT EXISTS(
			SELECT 1 FROM subtree WHERE id = (SELECT id FROM folders WHERE uuid = $2)
		)
	`, rootFolderID, candidateUUID).Scan(&exists)
	return exists, err
}

// IsFileInSubtree returns true if the file identified by fileUUID has a parent_id
// that is the root folder or any of its descendants.
func IsFileInSubtree(rootFolderID int64, fileUUID string) (bool, error) {
	var exists bool
	err := DB.QueryRow(`
		WITH RECURSIVE subtree AS (
			SELECT id FROM folders WHERE id = $1
			UNION ALL
			SELECT f.id FROM folders f
			INNER JOIN subtree s ON f.parent_id = s.id
		)
		SELECT EXISTS(
			SELECT 1 FROM files WHERE uuid = $2
			  AND parent_id IN (SELECT id FROM subtree)
		)
	`, rootFolderID, fileUUID).Scan(&exists)
	return exists, err
}
