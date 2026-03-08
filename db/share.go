package db

import (
	"crypto/rand"
	"database/sql"
	"time"
)

const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

type ShareLink struct {
	ID        int64
	Token     string
	FileID    string
	CreatedBy string
	ExpiresAt *time.Time
	CreatedAt time.Time
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
