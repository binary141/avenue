package shared

import (
	"context"
	"fmt"
	"net/mail"
	"os"
	"strconv"
	"time"
)

type cookieStr string

const (
	SESSIONCOOKIENAME  cookieStr = "session_id"
	USERCOOKIENAME     cookieStr = "user_id"
	USERCOOKIEVALUE    cookieStr = "test"
	ROOTFOLDERID                 = "c32af1cc-aba9-4878-a305-5006dc7a5b76"
	DEFAULTMAXFILESIZE int64     = 209715200

	DEFAULTPAGE      = 1
	DEFAULTPAGELIMIT = 50
	MAXPAGELIMIT     = 200
)

func GetEnvBool(key string, defaultVal bool) bool {
	envKey := os.Getenv(key)
	if envKey == "" {
		return defaultVal
	}
	val, err := strconv.ParseBool(envKey)
	if err != nil {
		return defaultVal
	}
	return val
}

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

// GetEnvDuration parses key as a Go duration string (e.g. "5m", "720h"),
// falling back to defaultVal (also parsed as a duration) if key is unset or
// invalid.
func GetEnvDuration(key string, defaultVal string) time.Duration {
	envKey := os.Getenv(key)
	if envKey == "" {
		envKey = defaultVal
	}

	dur, err := time.ParseDuration(envKey)
	if err != nil {
		fmt.Printf("error parsing duration: %s", err.Error())
		dur, _ = time.ParseDuration(defaultVal)
	}
	return dur
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

// ParsePagination validates page/limit query params, defaulting and
// clamping them to sane values, and returns the corresponding SQL
// LIMIT/OFFSET pair.
func ParsePagination(pageStr, limitStr string) (page, limit, offset int) {
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = DEFAULTPAGE
	}

	limit, err = strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = DEFAULTPAGELIMIT
	} else if limit > MAXPAGELIMIT {
		limit = MAXPAGELIMIT
	}

	offset = (page - 1) * limit

	return page, limit, offset
}

func GetSessionIDFromContext(ctx context.Context) (string, error) {
	val := ctx.Value(SESSIONCOOKIENAME)

	if val == nil {
		return "", fmt.Errorf("unable to cast cookie val: '%v' to string", val)
	}

	return fmt.Sprint(val), nil
}
