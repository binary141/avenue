package persist

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID        uint   `gorm:"primarykey" json:"id"`
	SessionID string `gorm:"not null;uniqueIndex" json:"-"`
	ExpiresAt int64  `gorm:"not null" json:"expiresAt"`
	IsValid   bool   `gorm:"not null" json:"isValid"`
	UserId    uint   `gorm:"not null" json:"userID"`
}

// CreateFile creates a new file record in the database.
func (p *Persist) GetSessionByID(idStr string) (Session, error) {
	var s Session
	err := p.db.Where("session_id = ?", idStr).First(&s).Error
	if err != nil {
		return s, err
	}
	return s, nil
}

func (p *Persist) DeleteSession(idStr string) error {
	return p.db.Where("session_id = ?", idStr).Delete(&Session{}).Error
}

func (p *Persist) UpdateSession(session Session) (Session, error) {
	res := p.db.Model(&Session{}).Where("id = ?", session.ID).Updates(session)

	return session, res.Error
}

func (p *Persist) CreateSession(userID int) (Session, error) {
	s := Session{
		SessionID: uuid.NewString(),
		ExpiresAt: time.Now().Add(15 * time.Minute).Unix(), // todo make env?
		IsValid:   true,
		UserId:    uint(userID),
	}

	res := p.db.Create(&s)

	return s, res.Error
}

func (p *Persist) IsValidSession(sessionID string) (Session, bool) {
	s, err := p.GetSessionByID(sessionID)
	if err != nil {
		return Session{}, false
	}

	return s, s.IsValid && s.ExpiresAt >= time.Now().Unix()
}
