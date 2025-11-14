package persist

import "log"

type User struct {
	ID       int    `gorm:"primaryKey"`
	Username string `gorm:"not null"`
	Email    string `gorm:"not null"`
	Password string `gorm:"not null"`
	CanLogin bool   `gorm:"not null"`
}

// CreateFile creates a new file record in the database.
func (p *Persist) GetUserById(id int) (User, error) {
	var u User

	err := p.db.First(&u, id).Error
	if err != nil {
		return u, err
	}
	return u, nil
}

func (p *Persist) UpsertRootUser() error {
	user := User{
		ID:       1,
		Username: "root",
		Email:    "root@gmail.com",
		Password: "password",
		CanLogin: true,
	}

	res := p.db.Save(&user)
	return res.Error
}

func (p *Persist) GetUserByUsername(username string) (User, error) {
	var u User
	res := p.db.First(&u).Where("username = ?", username)

	if res.Error != nil {
		return u, res.Error
	}

	return u, nil
}

func (p *Persist) GetUserByEmail(email string) (User, error) {
	var u User
	res := p.db.First(&u).Where("email = ?", email)

	if res.Error != nil {
		return u, res.Error
	}

	return u, nil
}

func (p *Persist) CreateUser(user *User) error {
	return p.db.Create(user).Error
}

func (p *Persist) IsUniqueEmail(email string) bool {
	u, err := p.GetUserByEmail(email)
	if err != nil {
		log.Print(err)
		return false
	}

	// 0 would mean it is the default value, so nothing was found?
	if u.ID == 0 {
		return false
	}

	return true
}

func (p *Persist) IsUniqueUsername(username string) bool {
	u, err := p.GetUserByUsername(username)
	if err != nil {
		log.Print(err)
		return false
	}

	// 0 would mean it is the default value, so nothing was found?
	if u.ID == 0 {
		return false
	}

	return true
}
