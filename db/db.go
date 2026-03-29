package db

import (
	"fmt"
	"os"

	"avenue/backend/logger"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var DB *sqlx.DB

func Connect() error {
	host := getenv("DB_HOST", "localhost")
	port := getenv("DB_PORT", "5432")
	user := getenv("DB_USER", "user")
	password := getenv("DB_PASSWORD", "secret")
	dbname := getenv("DB_DATABASE", "avenue")

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)

	var err error
	DB, err = sqlx.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("db: open: %w", err)
	}

	if err = DB.Ping(); err != nil {
		return fmt.Errorf("db: ping: %w", err)
	}

	logger.Infof("connected to postgres at %s:%s/%s", host, port, dbname)
	return nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
