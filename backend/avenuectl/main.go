package main

import (
	"avenue/backend/handlers"
	"avenue/backend/persist"
	"avenue/backend/shared"
)

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

	// Start the server
	_ = server.Run(":8080")
}
