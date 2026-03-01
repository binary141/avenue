package db

import (
	"fmt"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Password  string    `json:"-"`
	CanLogin  bool      `json:"canLogin"`
	IsAdmin   bool      `json:"isAdmin"`
	Quota     int64     `json:"quota"`
	SpaceUsed int64     `json:"spaceUsed"`
	CreatedAt time.Time `json:"createdAt"`
}

func GetUserByIDStr(idStr string) (User, error) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return User{}, err
	}
	return GetUserByID(id)
}

func GetUserByID(id int64) (User, error) {
	var u User
	err := DB.QueryRow(`
		SELECT id, email, COALESCE(first_name,''), COALESCE(last_name,''), password, can_login, is_admin, quota, space_used, created_at
		FROM users WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.Password, &u.CanLogin, &u.IsAdmin, &u.Quota, &u.SpaceUsed, &u.CreatedAt)
	return u, err
}

func GetUserByEmail(email string) (User, error) {
	var u User
	err := DB.QueryRow(`
		SELECT id, email, COALESCE(first_name,''), COALESCE(last_name,''), password, can_login, is_admin, quota, space_used, created_at
		FROM users WHERE email = $1 AND deleted_at IS NULL
	`, email).Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.Password, &u.CanLogin, &u.IsAdmin, &u.Quota, &u.SpaceUsed, &u.CreatedAt)
	return u, err
}

func GetUsers() ([]User, error) {
	rows, err := DB.Query(`
		SELECT id, email, COALESCE(first_name,''), COALESCE(last_name,''), can_login, is_admin, quota, space_used, created_at
		FROM users WHERE deleted_at IS NULL ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.CanLogin, &u.IsAdmin, &u.Quota, &u.SpaceUsed, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func CreateUser(email, password, firstName, lastName string, isAdmin bool) (User, error) {
	var u User
	err := DB.QueryRow(`
		INSERT INTO users (email, password, first_name, last_name, can_login, is_admin, created_at, updated_at)
		VALUES ($1, $2, NULLIF($3,''), NULLIF($4,''), true, $5, now(), now())
		RETURNING id, email, COALESCE(first_name,''), COALESCE(last_name,''), password, can_login, is_admin, quota, space_used, created_at
	`, email, password, firstName, lastName, isAdmin).Scan(
		&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.Password, &u.CanLogin, &u.IsAdmin, &u.Quota, &u.SpaceUsed, &u.CreatedAt,
	)
	return u, err
}

func UpdateUser(user User) (User, error) {
	_, err := DB.Exec(`
		UPDATE users
		SET is_admin=$2, first_name=NULLIF($3,''), last_name=NULLIF($4,''),
		    email=$5, password=$6, can_login=$7, quota=$8, updated_at=now()
		WHERE id=$1 AND deleted_at IS NULL
	`, user.ID, user.IsAdmin, user.FirstName, user.LastName, user.Email, user.Password, user.CanLogin, user.Quota)
	return user, err
}

func HasOtherAdmins(user User) (bool, error) {
	var exists bool
	err := DB.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM users WHERE is_admin=true AND id <> $1 AND deleted_at IS NULL)`,
		user.ID,
	).Scan(&exists)
	return exists, err
}

func IsUniqueEmail(email string) bool {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM users WHERE email=$1 AND deleted_at IS NULL`, email).Scan(&count)
	if err != nil {
		return false
	}
	return count == 0
}

func GetUserUsage(id string) (int64, error) {
	uid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid user id: %w", err)
	}
	var spaceUsed int64
	err = DB.QueryRow(`SELECT space_used FROM users WHERE id=$1 AND deleted_at IS NULL`, uid).Scan(&spaceUsed)
	return spaceUsed, err
}

func UpdateUsage(userID string, delta int64) error {
	uid, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}
	_, err = DB.Exec(`
		UPDATE users SET space_used = GREATEST(0, space_used + $2), updated_at=now() WHERE id=$1
	`, uid, delta)
	return err
}

// UpsertRootUser creates the admin account from env vars if it doesn't already exist.
// If ROOT_USER_RESET=true is set, the root user's password is also updated on startup.
func UpsertRootUser() error {
	email := getenv("ROOT_USER_EMAIL", "root@gmail.com")
	password := getenv("ROOT_USER_PASSWORD", "password")

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM users WHERE email=$1`, email).Scan(&count); err != nil {
		return err
	}

	if count == 0 {
		_, err = DB.Exec(`
			INSERT INTO users (email, password, can_login, is_admin, created_at, updated_at)
			VALUES ($1, $2, true, true, now(), now())
		`, email, string(hashed))
		return err
	}

	if getenv("ROOT_USER_RESET", "") == "true" {
		_, err = DB.Exec(`
			UPDATE users SET password=$2, updated_at=now() WHERE email=$1
		`, email, string(hashed))
		return err
	}

	return nil
}

