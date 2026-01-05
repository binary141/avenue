package shared

import (
	"context"
	"fmt"
	"net/mail"
	"os"
	"strconv"
)

type cookieStr string

const (
	SESSIONCOOKIENAME cookieStr = "session_id"
	USERCOOKIENAME    cookieStr = "user_id"
	USERCOOKIEVALUE   cookieStr = "test"
	ROOTFOLDERID                = "c32af1cc-aba9-4878-a305-5006dc7a5b76"
)

func GetEnvInt64(key string, defaultVal int64) int64 {
	envKey := os.Getenv(key)

	if envKey == "" {
		return defaultVal
	}

	castedKey, err := strconv.ParseInt(envKey, 10, 64)
	if err != nil {
		fmt.Printf("error parsing int: %s", err.Error())
		return defaultVal
	}
	return castedKey
}

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

func GetUserIDFromContext(ctx context.Context) (string, error) {
	val := ctx.Value(USERCOOKIENAME)

	if val == nil {
		return "", fmt.Errorf("unable to cast cookie val: '%v' to string", val)
	}

	return fmt.Sprint(val), nil
}
