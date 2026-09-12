package db

import (
	"time"

	"github.com/google/uuid"
)

// UploadSession tracks an in-progress chunked upload. Its id doubles as the
// eventual file's UUID once the upload completes, so the blob can be written
// straight to its final sharded path (shared.BlobPath(id)) without a rename.
type UploadSession struct {
	ID            string
	UserID        int64
	Name          string
	Extension     string
	Parent        string
	TotalSize     int64
	BytesReceived int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// CreateUploadSession starts a new chunked upload and returns the session
// (its ID is the future file UUID).
func CreateUploadSession(userID int64, name, extension, parent string, totalSize int64) (*UploadSession, error) {
	s := UploadSession{
		ID:        uuid.NewString(),
		UserID:    userID,
		Name:      name,
		Extension: extension,
		Parent:    parent,
		TotalSize: totalSize,
	}
	err := DB.QueryRow(`
		INSERT INTO upload_sessions (id, user_id, name, extension, parent, total_size)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at
	`, s.ID, userID, name, extension, parent, totalSize).Scan(&s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GetUploadSession fetches an upload session owned by userID.
func GetUploadSession(id string, userID int64) (*UploadSession, error) {
	var s UploadSession
	err := DB.QueryRow(`
		SELECT id, user_id, name, extension, parent, total_size, bytes_received, created_at, updated_at
		FROM upload_sessions WHERE id=$1 AND user_id=$2
	`, id, userID).Scan(&s.ID, &s.UserID, &s.Name, &s.Extension, &s.Parent, &s.TotalSize, &s.BytesReceived, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// AdvanceUploadSession records that a chunk landing at [prevBytesReceived,
// prevBytesReceived+chunkLen) was written, moving bytes_received forward by
// chunkLen. It's conditioned on the row still being at prevBytesReceived so
// two racing requests for the same session can't both apply their chunk.
func AdvanceUploadSession(id string, userID int64, prevBytesReceived, chunkLen int64) (int64, error) {
	var newBytesReceived int64
	err := DB.QueryRow(`
		UPDATE upload_sessions SET bytes_received = bytes_received + $3, updated_at = now()
		WHERE id=$1 AND user_id=$2 AND bytes_received=$4
		RETURNING bytes_received
	`, id, userID, chunkLen, prevBytesReceived).Scan(&newBytesReceived)
	if err != nil {
		return 0, err
	}
	return newBytesReceived, nil
}

// DeleteUploadSession removes an upload session row (its blob is the
// caller's responsibility).
func DeleteUploadSession(id string, userID int64) error {
	_, err := DB.Exec(`DELETE FROM upload_sessions WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}

// ListExpiredUploadSessions returns upload sessions that haven't received a
// chunk since before cutoff, for the sweeper to purge abandoned uploads.
func ListExpiredUploadSessions(cutoff time.Time) ([]UploadSession, error) {
	rows, err := DB.Query(`
		SELECT id, user_id, name, extension, parent, total_size, bytes_received, created_at, updated_at
		FROM upload_sessions WHERE updated_at < $1
	`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []UploadSession
	for rows.Next() {
		var s UploadSession
		if err := rows.Scan(&s.ID, &s.UserID, &s.Name, &s.Extension, &s.Parent, &s.TotalSize, &s.BytesReceived, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}
