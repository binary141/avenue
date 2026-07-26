package db

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID        int64     `json:"id"`
	SessionID string    `json:"-"`
	ExpiresAt int64     `json:"expiresAt"`
	IsValid   bool      `json:"isValid"`
	UserId    int64     `json:"userID"`
	CreatedAt time.Time `json:"createdAt"`
	UserAgent string    `json:"userAgent"`
	IPAddress string    `json:"ipAddress"`
}

func CreateSession(userId int64, userAgent, ipAddress string) (Session, error) {
	s := Session{
		SessionID: uuid.NewString(),
		ExpiresAt: time.Now().Add(15 * time.Minute).Unix(),
		IsValid:   true,
		UserId:    userId,
		UserAgent: userAgent,
		IPAddress: ipAddress,
	}

	err := DB.QueryRow(`
		INSERT INTO sessions (uuid, expires_at, is_valid, user_id, user_agent, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`, s.SessionID, s.ExpiresAt, s.IsValid, s.UserId, s.UserAgent, s.IPAddress).Scan(&s.ID, &s.CreatedAt)
	return s, err
}

func getSessionByToken(token string) (Session, error) {
	var s Session
	err := DB.QueryRow(
		`SELECT id, uuid, expires_at, is_valid, user_id, created_at, COALESCE(user_agent,''), COALESCE(ip_address,'') FROM sessions WHERE uuid = $1`,
		token,
	).Scan(&s.ID, &s.SessionID, &s.ExpiresAt, &s.IsValid, &s.UserId, &s.CreatedAt, &s.UserAgent, &s.IPAddress)
	return s, err
}

// ListSessionsForUser returns the user's active (valid, unexpired) sessions,
// most recently created first.
func ListSessionsForUser(userID int64) ([]Session, error) {
	rows, err := DB.Query(`
		SELECT id, uuid, expires_at, is_valid, user_id, created_at, COALESCE(user_agent,''), COALESCE(ip_address,'')
		FROM sessions
		WHERE user_id = $1 AND is_valid = true AND expires_at >= $2
		ORDER BY created_at DESC
	`, userID, time.Now().Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.ID, &s.SessionID, &s.ExpiresAt, &s.IsValid, &s.UserId, &s.CreatedAt, &s.UserAgent, &s.IPAddress); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// DeleteSessionByIDForUser removes a single session by its row ID, scoped to
// userID so a caller can only revoke their own sessions.
func DeleteSessionByIDForUser(id, userID int64) error {
	_, err := DB.Exec(`DELETE FROM sessions WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}

func IsValidSession(token string) (Session, bool) {
	s, err := getSessionByToken(token)
	if err != nil {
		return Session{}, false
	}
	return s, s.IsValid && s.ExpiresAt >= time.Now().Unix()
}

func UpdateSession(session Session) (Session, error) {
	_, err := DB.Exec(
		`UPDATE sessions SET expires_at=$2 WHERE id=$1`,
		session.ID, session.ExpiresAt,
	)
	return session, err
}

func DeleteSession(token string) error {
	_, err := DB.Exec(`DELETE FROM sessions WHERE uuid=$1`, token)
	return err
}

// DeleteOtherSessions removes all sessions for the given user except the one
// identified by exceptToken. Used to revoke other sessions on password change
// while keeping the session that made the change logged in.
func DeleteOtherSessions(userID int64, exceptToken string) error {
	_, err := DB.Exec(`DELETE FROM sessions WHERE user_id=$1 AND uuid<>$2`, userID, exceptToken)
	return err
}

// DeleteSessionsForUser removes all sessions for the given user.
func DeleteSessionsForUser(userID int64) error {
	_, err := DB.Exec(`DELETE FROM sessions WHERE user_id=$1`, userID)
	return err
}

// DeleteExpiredSessions removes every session that is expired or has been
// marked invalid, and returns how many rows were removed.
func DeleteExpiredSessions() (int64, error) {
	res, err := DB.Exec(`DELETE FROM sessions WHERE expires_at < $1 OR is_valid = false`, time.Now().Unix())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
