package testutils

import (
	"os"

	"golang.org/x/exp/slog"
)

func init() {
	// in tests set a custom handler on debug level as default
	handlerOpts := new(slog.HandlerOptions)
	handlerOpts.Level = slog.LevelDebug
	handler := handlerOpts.NewTextHandler(os.Stderr)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}
