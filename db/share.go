package db

import (
	"crypto/rand"
	"database/sql"
	"time"

	"avenue/backend/sdk"
)

const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i, v := range b {
		b[i] = charset[int(v)%len(charset)]
	}
	return string(b), nil
}

func CreateShareLink(fileID, createdBy string, expiresAt *time.Time, requireLogin bool) (sdk.ShareLink, error) {
	var link sdk.ShareLink

	token, err := generateToken()
	if err != nil {
		return link, err
	}

	err = DB.QueryRow(`
		WITH ins AS (
			INSERT INTO share_links (token, file_id, created_by, expires_at, require_login)
			VALUES ($1, (SELECT id FROM files WHERE uuid=$2), $3::BIGINT, $4, $5)
			RETURNING id, token, file_id, created_by, expires_at, created_at, require_login
		)
		SELECT ins.id, ins.token, f.uuid, ins.created_by, ins.expires_at, ins.created_at, ins.require_login
		FROM ins
		JOIN files f ON f.id = ins.file_id
	`, token, fileID, createdBy, expiresAt, requireLogin).Scan(
		&link.ID, &link.Token, &link.FileID, &link.CreatedBy, &link.ExpiresAt, &link.CreatedAt, &link.RequireLogin,
	)
	if err != nil {
		return link, err
	}

	return link, nil
}

func touchShareLink(token string) {
	_, _ = DB.Exec(`UPDATE share_links SET last_accessed = now() WHERE token = $1`, token)
}

func GetShareLink(token string) (sdk.ShareLink, error) {
	var link sdk.ShareLink

	err := DB.QueryRow(`
		SELECT sl.id, sl.token, f.uuid, sl.created_by, sl.expires_at, sl.created_at, sl.require_login, sl.last_accessed
		FROM share_links sl
		JOIN files f ON f.id = sl.file_id
		WHERE sl.token = $1
		  AND (sl.expires_at IS NULL OR sl.expires_at > now())
	`, token).Scan(
		&link.ID, &link.Token, &link.FileID, &link.CreatedBy, &link.ExpiresAt, &link.CreatedAt, &link.RequireLogin, &link.LastAccessed,
	)
	if err != nil {
		return link, sql.ErrNoRows
	}

	touchShareLink(token)

	return link, nil
}

func ListSharesByFile(fileID, createdBy string) ([]sdk.ShareLink, error) {
	var links []sdk.ShareLink

	rows, err := DB.Query(`
		SELECT sl.id, sl.token, f.uuid, sl.created_by, sl.expires_at, sl.created_at, sl.require_login, sl.last_accessed
		FROM share_links sl
		JOIN files f ON f.id = sl.file_id
		WHERE f.uuid = $1 AND sl.created_by = $2::BIGINT
		  AND (sl.expires_at IS NULL OR sl.expires_at > now())
		ORDER BY sl.created_at DESC
	`, fileID, createdBy)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var l sdk.ShareLink
		if err := rows.Scan(&l.ID, &l.Token, &l.FileID, &l.CreatedBy, &l.ExpiresAt, &l.CreatedAt, &l.RequireLogin, &l.LastAccessed); err != nil {
			return nil, err
		}
		links = append(links, l)
	}

	return links, rows.Err()
}

func ListSharesByUser(createdBy string) ([]sdk.ShareLinkWithFileName, error) {
	var links []sdk.ShareLinkWithFileName

	rows, err := DB.Query(`
		SELECT sl.id, sl.token, COALESCE(f.uuid, ''), COALESCE(f.name, ''), sl.created_by, sl.expires_at, sl.created_at, sl.require_login, sl.last_accessed
		FROM share_links sl
		LEFT JOIN files f ON f.id = sl.file_id
		WHERE sl.created_by = $1::BIGINT
		  AND (sl.expires_at IS NULL OR sl.expires_at > now())
		ORDER BY sl.created_at DESC
	`, createdBy)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var l sdk.ShareLinkWithFileName
		if err := rows.Scan(&l.ID, &l.Token, &l.FileID, &l.FileName, &l.CreatedBy, &l.ExpiresAt, &l.CreatedAt, &l.RequireLogin, &l.LastAccessed); err != nil {
			return nil, err
		}
		links = append(links, l)
	}

	return links, rows.Err()
}

func ListExpiredSharesByUser(createdBy string) ([]sdk.ShareLinkWithFileName, error) {
	var links []sdk.ShareLinkWithFileName

	rows, err := DB.Query(`
		SELECT sl.id, sl.token, COALESCE(f.uuid, ''), COALESCE(f.name, ''), sl.created_by, sl.expires_at, sl.created_at, sl.require_login, sl.last_accessed
		FROM share_links sl
		LEFT JOIN files f ON f.id = sl.file_id
		WHERE sl.created_by = $1::BIGINT
		  AND sl.expires_at IS NOT NULL AND sl.expires_at <= now()
		ORDER BY sl.expires_at DESC
	`, createdBy)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var l sdk.ShareLinkWithFileName
		if err := rows.Scan(&l.ID, &l.Token, &l.FileID, &l.FileName, &l.CreatedBy, &l.ExpiresAt, &l.CreatedAt, &l.RequireLogin, &l.LastAccessed); err != nil {
			return nil, err
		}
		links = append(links, l)
	}

	return links, rows.Err()
}

func DeleteShareLink(token, createdBy string) error {
	_, err := DB.Exec(`DELETE FROM share_links WHERE token = $1 AND created_by = $2::BIGINT`, token, createdBy)
	return err
}

func DeleteShareLinksByFileID(fileID int64) error {
	_, err := DB.Exec(`DELETE FROM share_links WHERE file_id = $1`, fileID)
	return err
}
