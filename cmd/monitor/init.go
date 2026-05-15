package main

import (
	"log/slog"
	"os"

	"github.com/pointnoreturn/monitor/libmetric"
)

var (
	libLog *slog.Logger = slog.New(
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
	appLog             *slog.Logger
	victoriaMetricsUrl = os.Getenv("VICTORIA_METRICS")
	envLogLevel        = os.Getenv("LOG_LEVEL")
)

func init() {
	level := slog.LevelDebug

	if envLogLevel == "debug" {
		level = slog.LevelDebug
	}

	r := func(
		groups []string,
		a slog.Attr,
	) slog.Attr {
		if a.Key == slog.TimeKey {
			return slog.Attr{}
		}
		return a
	}

	appLog = slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:       level,
			ReplaceAttr: r,
		}),
	)

	if victoriaMetricsUrl == "" {
		panic("No VICTORIA_METRICS env set.")
	}
	libmetric.Init(victoriaMetricsUrl, appLog)
}
