package persist

import (
	"errors"
	"strconv"
	"time"

	"avenue/backend/shared"

	"gorm.io/gorm"
)

type User struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	Email     string         `gorm:"not null;uniqueIndex" json:"email"`
	FirstName string         `gorm:"nullable" json:"firstName"`
	LastName  string         `gorm:"nullable" json:"lastName"`
	Password  string         `gorm:"not null" json:"-"` // omit password from json output
	CanLogin  bool           `gorm:"not null" json:"canLogin"`
	IsAdmin   bool           `gorm:"not null;default:false" json:"isAdmin"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt"`
}

func (p *Persist) GetUserByIDStr(idStr string) (User, error) {
	var u User

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return u, err
	}

	return p.getUserByID(id)
}

func (p *Persist) GetUsers() ([]User, error) {
	var users []User

	result := p.db.Order("id asc").Find(&users)
	if result.Error != nil {
		return users, result.Error
	}

	return users, nil
}

func (p *Persist) getUserByID(id int) (User, error) {
	var u User

	err := p.db.First(&u, id).Error
	if err != nil {
		return u, err
	}
	return u, nil
}

func (p *Persist) HasOtherAdmins(user User) (bool, error) {
	res := p.db.Model(&User{}).
		Where("is_admin = true").
		Where("id <> ?", user.ID).
		First(&User{})

	if res.Error != nil {
		return false, res.Error
	}

	return true, nil
}

func (p *Persist) UpdateUser(user User) (User, error) {
	res := p.db.Model(&User{}).
		Where("id = ?", user.ID).
		Select("IsAdmin", "FirstName", "LastName", "Email", "Password", "CanLogin").
		Updates(user)

	return user, res.Error
}

func (p *Persist) UpsertRootUser() error {
	user := User{
		ID:        1,
		Email:     shared.GetEnv("ROOT_USER_EMAIL", "root@gmail.com"),
		Password:  shared.GetEnv("ROOT_USER_PASSWORD", "password"),
		CanLogin:  true,
		IsAdmin:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		DeletedAt: gorm.DeletedAt{},
	}

	res := p.db.Save(&user)
	return res.Error
}

func (p *Persist) GetUserByEmail(email string) (User, error) {
	var u User
	res := p.db.First(&u, "email = ?", email)

	if res.Error != nil {
		return u, res.Error
	}

	return u, nil
}

// TODO this should take in a users struct instead of all the fields to make it
func (p *Persist) CreateUser(email, password, firstName, lastName string, isAdmin bool) (User, error) {
	u := User{
		Email:     email,
		IsAdmin:   isAdmin,
		FirstName: firstName,
		LastName:  lastName,
		Password:  password,
		CanLogin:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	res := p.db.Create(&u)

	return u, res.Error
}

func (p *Persist) IsUniqueEmail(email string) bool {
	u, err := p.GetUserByEmail(email)
	if err != nil {
		return errors.Is(err, gorm.ErrRecordNotFound)
	}

	// 0 would mean it is the default value, so nothing was found?
	if u.ID == 0 {
		return true
	}

	return false
}
