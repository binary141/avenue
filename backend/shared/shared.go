package shared

import (
	"context"
	"errors"
	"net/mail"
	"os"
)

const (
	USERCOOKIENAME  = "user-id"
	USERCOOKIEVALUE = "test"
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

	str, ok := val.(string)
	if !ok {
		return "", errors.New("Unable to cast cookie val to string")
	}

	return str, nil
}
