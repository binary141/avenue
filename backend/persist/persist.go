package persist

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Persist struct {
	db *gorm.DB
}

func NewPersist(host, user, password, dbname string) *Persist {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=5432 sslmode=disable", host, user, password, dbname)

	// Open the connection
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("failed to connect database: %v", err))
	}
	// DB Migrations
	err = db.AutoMigrate(&File{})
	if err != nil {
		panic(fmt.Sprintf("failed to migrate database for files: %v", err))
	}

	err = db.AutoMigrate(&User{})
	if err != nil {
		panic(fmt.Sprintf("failed to migrate database for users: %v", err))
	}
	return &Persist{db: db}
}
