package db

import (
	"crypto/rand"
	"database/sql"
	"time"
)

const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

type ShareLink struct {
	ID        int64      `json:"id"`
	Token     string     `json:"token"`
	FileID    string     `json:"file_id"`
	CreatedBy string     `json:"created_by"`
	ExpiresAt *time.Time `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
}

type ShareLinkWithFileName struct {
	ID        int64      `json:"id"`
	Token     string     `json:"token"`
	FileID    string     `json:"file_id"`
	FileName  string     `json:"file_name"`
	CreatedBy string     `json:"created_by"`
	ExpiresAt *time.Time `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
}

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

func CreateShareLink(fileID, createdBy string, expiresAt *time.Time) (ShareLink, error) {
	token, err := generateToken()
	if err != nil {
		return ShareLink{}, err
	}

	var link ShareLink
	err = DB.QueryRow(`
		INSERT INTO share_links (token, file_id, created_by, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, token, file_id, created_by, expires_at, created_at
	`, token, fileID, createdBy, expiresAt).Scan(
		&link.ID, &link.Token, &link.FileID, &link.CreatedBy, &link.ExpiresAt, &link.CreatedAt,
	)
	if err != nil {
		return ShareLink{}, err
	}
	return link, nil
}

func GetShareLink(token string) (ShareLink, error) {
	var link ShareLink
	err := DB.QueryRow(`
		SELECT id, token, file_id, created_by, expires_at, created_at
		FROM share_links
		WHERE token = $1
		  AND (expires_at IS NULL OR expires_at > now())
	`, token).Scan(
		&link.ID, &link.Token, &link.FileID, &link.CreatedBy, &link.ExpiresAt, &link.CreatedAt,
	)
	if err != nil {
		return ShareLink{}, sql.ErrNoRows
	}
	return link, nil
}

func ListSharesByFile(fileID, createdBy string) ([]ShareLink, error) {
	rows, err := DB.Query(`
		SELECT id, token, file_id, created_by, expires_at, created_at
		FROM share_links
		WHERE file_id = $1 AND created_by = $2
		  AND (expires_at IS NULL OR expires_at > now())
		ORDER BY created_at DESC
	`, fileID, createdBy)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []ShareLink
	for rows.Next() {
		var l ShareLink
		if err := rows.Scan(&l.ID, &l.Token, &l.FileID, &l.CreatedBy, &l.ExpiresAt, &l.CreatedAt); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, rows.Err()
}

func ListSharesByUser(createdBy string) ([]ShareLinkWithFileName, error) {
	rows, err := DB.Query(`
		SELECT sl.id, sl.token, sl.file_id, COALESCE(f.name, ''), sl.created_by, sl.expires_at, sl.created_at
		FROM share_links sl
		LEFT JOIN files f ON f.id = sl.file_id
		WHERE sl.created_by = $1
		  AND (sl.expires_at IS NULL OR sl.expires_at > now())
		ORDER BY sl.created_at DESC
	`, createdBy)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []ShareLinkWithFileName
	for rows.Next() {
		var l ShareLinkWithFileName
		if err := rows.Scan(&l.ID, &l.Token, &l.FileID, &l.FileName, &l.CreatedBy, &l.ExpiresAt, &l.CreatedAt); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, rows.Err()
}

func DeleteShareLink(token, createdBy string) error {
	_, err := DB.Exec(`DELETE FROM share_links WHERE token = $1 AND created_by = $2`, token, createdBy)
	return err
}
