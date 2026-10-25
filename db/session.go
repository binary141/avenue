package db

import (
	"time"

	"github.com/google/uuid"
)

// Client types recorded on a session. They must match the seeded rows in
// session_types (migration 0020), which sessions.client_type references.
const (
	ClientTypeWeb    = "web"
	ClientTypeMobile = "mobile"
)

type Session struct {
	ID         int64     `json:"id"`
	SessionID  string    `json:"-"`
	ExpiresAt  int64     `json:"expiresAt"`
	IsValid    bool      `json:"isValid"`
	UserId     int64     `json:"userID"`
	CreatedAt  time.Time `json:"createdAt"`
	UserAgent  string    `json:"userAgent"`
	IPAddress  string    `json:"ipAddress"`
	ClientType string    `json:"clientType"`
}

// CreateSession starts a session of clientType that expires window from now.
func CreateSession(userId int64, userAgent, ipAddress, clientType string, window time.Duration) (Session, error) {
	s := Session{
		SessionID:  uuid.NewString(),
		ExpiresAt:  time.Now().Add(window).Unix(),
		IsValid:    true,
		UserId:     userId,
		UserAgent:  userAgent,
		IPAddress:  ipAddress,
		ClientType: clientType,
	}

	// The session_type_id subquery returns NULL for an unknown name, which the
	// NOT NULL constraint rejects, so a bad clientType can't be stored.
	err := DB.QueryRow(`
		INSERT INTO sessions (uuid, expires_at, is_valid, user_id, user_agent, ip_address, session_type_id)
		VALUES ($1, $2, $3, $4, $5, $6, (SELECT id FROM session_types WHERE name = $7))
		RETURNING id, created_at
	`, s.SessionID, s.ExpiresAt, s.IsValid, s.UserId, s.UserAgent, s.IPAddress, s.ClientType).Scan(&s.ID, &s.CreatedAt)
	return s, err
}

func getSessionByToken(token string) (Session, error) {
	var s Session
	err := DB.QueryRow(
		`SELECT s.id, s.uuid, s.expires_at, s.is_valid, s.user_id, s.created_at, COALESCE(s.user_agent,''), COALESCE(s.ip_address,''), st.name
		FROM sessions s
		JOIN session_types st ON st.id = s.session_type_id
		WHERE s.uuid = $1`,
		token,
	).Scan(&s.ID, &s.SessionID, &s.ExpiresAt, &s.IsValid, &s.UserId, &s.CreatedAt, &s.UserAgent, &s.IPAddress, &s.ClientType)
	return s, err
}

// ListSessionsForUser returns the user's active (valid, unexpired) sessions,
// most recently created first.
func ListSessionsForUser(userID int64) ([]Session, error) {
	rows, err := DB.Query(`
		SELECT s.id, s.uuid, s.expires_at, s.is_valid, s.user_id, s.created_at, COALESCE(s.user_agent,''), COALESCE(s.ip_address,''), st.name
		FROM sessions s
		JOIN session_types st ON st.id = s.session_type_id
		WHERE s.user_id = $1 AND s.is_valid = true AND s.expires_at >= $2
		ORDER BY s.created_at DESC
	`, userID, time.Now().Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.ID, &s.SessionID, &s.ExpiresAt, &s.IsValid, &s.UserId, &s.CreatedAt, &s.UserAgent, &s.IPAddress, &s.ClientType); err != nil {
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
