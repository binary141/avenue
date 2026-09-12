package db

import (
	"database/sql"
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

// ChunkRange is a byte range [Start, End) of a file that has already been
// received for an in-progress upload session.
type ChunkRange struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

// RecordUploadChunk records that the byte range [start, end) has been
// received for session id, and returns the total distinct bytes received
// across the whole session so far. It's idempotent — a chunk resent at the
// same offset (e.g. after a client-side retry timeout that actually
// succeeded) is a no-op rather than double-counted — which is what lets
// several chunks for the same session be in flight, and retried,
// concurrently instead of requiring strict in-order arrival.
func RecordUploadChunk(id string, userID int64, start, end int64) (int64, error) {
	tx, err := DB.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		INSERT INTO upload_session_chunks (session_id, start_offset, end_offset)
		SELECT $1, $2, $3 WHERE EXISTS (SELECT 1 FROM upload_sessions WHERE id=$1 AND user_id=$4)
		ON CONFLICT (session_id, start_offset) DO NOTHING
	`, id, start, end, userID)
	if err != nil {
		return 0, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		// Either this exact chunk was already recorded, or the session
		// doesn't exist / doesn't belong to this user — tell those apart so
		// the latter still surfaces as sql.ErrNoRows like the rest of this
		// package's lookups.
		var exists bool
		if err := tx.Get(&exists, `SELECT EXISTS(SELECT 1 FROM upload_sessions WHERE id=$1 AND user_id=$2)`, id, userID); err != nil {
			return 0, err
		}
		if !exists {
			return 0, sql.ErrNoRows
		}
	}

	var total int64
	if err := tx.Get(&total, `SELECT COALESCE(SUM(end_offset - start_offset), 0) FROM upload_session_chunks WHERE session_id=$1`, id); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(`UPDATE upload_sessions SET bytes_received=$2, updated_at=now() WHERE id=$1`, id, total); err != nil {
		return 0, err
	}

	return total, tx.Commit()
}

// ListUploadSessionChunks returns every byte range accepted so far for
// session id, ordered by offset, so a resuming client can skip chunks it
// already sent instead of just resuming from a contiguous prefix.
func ListUploadSessionChunks(id string, userID int64) ([]ChunkRange, error) {
	rows, err := DB.Query(`
		SELECT c.start_offset, c.end_offset
		FROM upload_session_chunks c
		JOIN upload_sessions s ON s.id = c.session_id
		WHERE c.session_id=$1 AND s.user_id=$2
		ORDER BY c.start_offset
	`, id, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ranges := []ChunkRange{}
	for rows.Next() {
		var r ChunkRange
		if err := rows.Scan(&r.Start, &r.End); err != nil {
			return nil, err
		}
		ranges = append(ranges, r)
	}
	return ranges, rows.Err()
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
