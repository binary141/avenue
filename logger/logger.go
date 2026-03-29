package logger

import (
	"fmt"
	"log/slog"
	"os"
)

var log *slog.Logger

func init() {
	log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
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
