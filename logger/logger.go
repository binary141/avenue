package logger

import (
	"fmt"
	"log/slog"
	"os"
)

var log *slog.Logger

func init() {
	level := slog.LevelDebug
	switch os.Getenv("LOG_LEVEL") {
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
}

func Debugf(msg string, args ...any) {
	log.Debug(fmt.Sprintf(msg, args...))
}

func Infof(msg string, args ...any) {
	log.Info(fmt.Sprintf(msg, args...))
}

func Warnf(msg string, args ...any) {
	log.Warn(fmt.Sprintf(msg, args...))
}

func Errorf(msg string, args ...any) {
	log.Error(fmt.Sprintf(msg, args...))
}
