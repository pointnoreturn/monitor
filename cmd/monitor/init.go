package main

import (
	"log/slog"
	"os"
)

var log *slog.Logger

func init() {

	level := slog.LevelInfo

	if os.Getenv("LOG_LEVEL") == "debug" {
		level = slog.LevelDebug
	}

	log = slog.New(
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
