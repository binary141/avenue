package persist

import (
	"errors"
	"log"
	"strconv"
	"time"

	"avenue/backend/shared"

	"gorm.io/gorm"
)

// TODO set up indexes on the username and email fields
type User struct {
	ID        uint   `gorm:"primarykey"`
	Username  string `gorm:"not null;uniqueIndex"`
	Email     string `gorm:"not null;uniqueIndex"`
	Password  string `gorm:"not null"`
	CanLogin  bool   `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// CreateFile creates a new file record in the database.
func (p *Persist) GetUserByIdStr(idStr string) (User, error) {
	var u User

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return u, err
	}

	return p.GetUserById(id)
}

func (p *Persist) GetUserById(id int) (User, error) {
	var u User

	err := p.db.First(&u, id).Error
	if err != nil {
		return u, err
	}
	return u, nil
}

func (p *Persist) UpdateUser(user User) (User, error) {
	res := p.db.Model(&User{}).Where("id = ?", user.ID).Updates(user)

	return user, res.Error
}

func (p *Persist) UpsertRootUser() error {
	user := User{
		ID:        1,
		Username:  shared.GetEnv("ROOT_USERNAME", "root"),
		Email:     shared.GetEnv("ROOT_USER_EMAIL", "root@gmail.com"),
		Password:  shared.GetEnv("ROOT_USER_PASSWORD", "password"),
		CanLogin:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		DeletedAt: gorm.DeletedAt{},
	}

	res := p.db.Save(&user)
	return res.Error
}

func (p *Persist) GetUserByUsername(username string) (User, error) {
	var u User
	res := p.db.First(&u, "username = ?", username)

	if res.Error != nil {
		return u, res.Error
	}

	return u, nil
}

func (p *Persist) GetUserByEmail(email string) (User, error) {
	var u User
	res := p.db.First(&u, "email = ?", email)

	if res.Error != nil {
		return u, res.Error
	}

	return u, nil
}

func (p *Persist) CreateUser(username, email, password string) error {
	u := User{
		Username:  username,
		Email:     email,
		Password:  password,
		CanLogin:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return p.db.Create(&u).Error
}

func (p *Persist) IsUniqueEmail(email string) bool {
	u, err := p.GetUserByEmail(email)
	if err != nil {
		return errors.Is(err, gorm.ErrRecordNotFound)
	}

	log.Printf("unique email user: %+v", u)
	// 0 would mean it is the default value, so nothing was found?
	if u.ID == 0 {
		return true
	}

	return false
}

func (p *Persist) IsUniqueUsername(username string) bool {
	u, err := p.GetUserByUsername(username)
	if err != nil {
		return errors.Is(err, gorm.ErrRecordNotFound)
	}

	// 0 would mean it is the default value, so nothing was found?
	if u.ID == 0 {
		return true
	}

	return false
}
