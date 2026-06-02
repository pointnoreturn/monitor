package libsupport

import (
	"log/slog"
	"os"
	"strings"
)

var (
	appLog, libLog *slog.Logger
	envLogLevel    = os.Getenv("LOG_LEVEL")
)

func LoggersFromEnv() (*slog.Logger, *slog.Logger) {
	if appLog == nil {
		level := slog.LevelInfo

		if strings.EqualFold(envLogLevel, "debug") {
			level = slog.LevelDebug
		} else if strings.EqualFold(envLogLevel, "warn") {
			level = slog.LevelWarn
		} else if strings.EqualFold(envLogLevel, "error") {
			level = slog.LevelError
		}

		appLog = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: level,
				ReplaceAttr: func(
					groups []string,
					a slog.Attr,
				) slog.Attr {
					if a.Key == slog.TimeKey {
						return slog.Attr{}
					}
					return a
				},
			}),
		)
	}

	if libLog == nil {
		libLog = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelWarn,
				ReplaceAttr: func(
					groups []string,
					a slog.Attr,
				) slog.Attr {
					if a.Key == slog.TimeKey {
						return slog.Attr{}
					}
					return a
				},
			}),
		)
	}

	return appLog, libLog
}
