package main

import (
	"embed"
	"log"

	"avenue/backend/db"
	"avenue/backend/handlers"
	"avenue/backend/shared"

	"github.com/gin-gonic/gin"
)

var frontendFS embed.FS

func main() {
	if err := db.Connect(); err != nil {
		log.Fatalf("db connect: %v", err)
	}

	if err := db.RunMigrations(); err != nil {
		log.Fatalf("db migrations: %v", err)
	}

	if err := db.UpsertRootUser(); err != nil {
		log.Printf("upsert root user: %v", err)
	}

	server := handlers.SetupServer()

	server.SetupRoutes()

	if shared.GetEnv("APP_ENV", "dev") == "production" {
		gin.SetMode(gin.ReleaseMode)
		server.ServeUI(frontendFS)
	}

	// Start the server
	_ = server.Run(":8080")
}
