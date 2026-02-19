package log

import (
	"io"
	"log/slog"
	"os"
)

var logger *slog.Logger

func init() {
	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// Init sets up logging with the given level and optional file writer.
func Init(level string, fileWriter io.Writer) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: lvl}
	var w io.Writer = os.Stderr
	if fileWriter != nil {
		w = io.MultiWriter(os.Stderr, fileWriter)
	}
	logger = slog.New(slog.NewTextHandler(w, opts))
}

func Debug(msg string, args ...any) { logger.Debug(msg, args...) }
func Info(msg string, args ...any)  { logger.Info(msg, args...) }
func Warn(msg string, args ...any)  { logger.Warn(msg, args...) }
func Error(msg string, args ...any) { logger.Error(msg, args...) }
