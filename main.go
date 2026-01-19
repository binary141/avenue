package main

import (
	"avenue/backend/handlers"
	"avenue/backend/persist"
	"avenue/backend/shared"
	"embed"

	"github.com/gin-gonic/gin"
)

var frontendFS embed.FS

func main() {
	persist := persist.NewPersist(
		shared.GetEnv("DB_HOST", "localhost"),
		shared.GetEnv("DB_USER", "user"),
		shared.GetEnv("DB_PASSWORD", "secret"),
		shared.GetEnv("DB_DATABASE", "avenue"),
	)

	_ = persist.UpsertRootUser()

	server := handlers.SetupServer(persist)

	server.SetupRoutes()

	if shared.GetEnv("APP_ENV", "dev") == "production" {
		gin.SetMode(gin.ReleaseMode)
		server.ServeUI(frontendFS)
	}

	// Start the server
	_ = server.Run(":8080")
}
