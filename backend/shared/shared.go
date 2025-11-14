package shared

import "os"

func GetEnv(key string, defaultVal string) string {
	envKey := os.Getenv(key)

	if envKey == "" {
		return defaultVal
	}

	return envKey
}
