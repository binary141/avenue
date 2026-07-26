package db

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID        int64  `json:"id"`
	SessionID string `json:"-"`
	ExpiresAt int64  `json:"expiresAt"`
	IsValid   bool   `json:"isValid"`
	UserId    int64  `json:"userID"`
}

func CreateSession(userId int64) (Session, error) {
	s := Session{
		SessionID: uuid.NewString(),
		ExpiresAt: time.Now().Add(15 * time.Minute).Unix(),
		IsValid:   true,
		UserId:    userId,
	}

	err := DB.QueryRow(`
		INSERT INTO sessions (uuid, expires_at, is_valid, user_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, s.SessionID, s.ExpiresAt, s.IsValid, s.UserId).Scan(&s.ID)
	return s, err
}

func getSessionByToken(token string) (Session, error) {
	var s Session
	err := DB.QueryRow(
		`SELECT id, uuid, expires_at, is_valid, user_id FROM sessions WHERE uuid = $1`,
		token,
	).Scan(&s.ID, &s.SessionID, &s.ExpiresAt, &s.IsValid, &s.UserId)
	return s, err
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
