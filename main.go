package main

import (
	"embed"

	"avenue/backend/db"
	"avenue/backend/email"
	"avenue/backend/handlers"
	"avenue/backend/logger"
	"avenue/backend/shared"
	"avenue/backend/sweeper"
)

var frontendFS embed.FS

func main() {
	if err := db.Connect(); err != nil {
		logger.Errorf("db connect: %v", err)
		return
	}

	if err := db.RunMigrations(); err != nil {
		logger.Errorf("db migrations: %v", err)
		return
	}

	if err := db.UpsertRootUser(); err != nil {
		logger.Warnf("upsert root user: %v", err)
	}

	sender, err := email.NewSESSender()
	if err != nil {
		logger.Warnf("email sender not configured: %v", err)
	} else {
		email.Default = sender
	}

	server := handlers.SetupServer()

	sweeper.Start(server.FS())
	sweeper.StartSessionSweeper()

	server.SetupRoutes()

	if shared.GetEnv("APP_ENV", "production") == "production" {
		server.ServeUI(frontendFS)
	}

	// Start the server
	_ = server.Run(":8080")
}
