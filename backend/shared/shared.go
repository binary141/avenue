package shared

import (
	"context"
	"fmt"
	"net/mail"
	"os"
)

const (
	SESSIONCOOKIENAME = "session_id"
	USERCOOKIENAME    = "user_id"
	USERCOOKIEVALUE   = "test"
)

func GetEnv(key string, defaultVal string) string {
	envKey := os.Getenv(key)

	if envKey == "" {
		return defaultVal
	}

	return envKey
}

func IsValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func GetUserIdFromContext(ctx context.Context) (string, error) {
	val := ctx.Value(USERCOOKIENAME)

	if val == nil {
		return "", fmt.Errorf("Unable to cast cookie val: '%v' to string", val)
	}

	return fmt.Sprint(val), nil
}
