package main

import (
	"avenue/backend/handlers"
	"avenue/backend/persist"
)

func main() {
	dbHost, dbUser, dbPassword, dbName := "0.0.0.0", "user", "secret", "avenue"
	persist := persist.NewPersist(dbHost, dbUser, dbPassword, dbName)

	server := handlers.SetupServer(persist)

	server.SetupRoutes()

	// Start the server
	server.Run(":8080")
}
