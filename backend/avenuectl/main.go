package main

import (
	"os"

	"avenue/backend/handlers"
	"avenue/backend/persist"
)

func main() {
	persist := persist.NewPersist(
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_USER", "user"),
		getEnv("DB_PASSWORD", "secret"),
		getEnv("DB_DATABASE", "avenue"),
	)

	server := handlers.SetupServer(persist)

	server.SetupRoutes()

	// Start the server
	_ = server.Run(":8080")
}

func getEnv(key string, defaultVal string) string {
	envKey := os.Getenv(key)

	if envKey == "" {
		return defaultVal
	}

	return envKey
}
