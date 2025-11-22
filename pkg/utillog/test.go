package utillog

import (
	"log/slog"
	"os"
	"sync"
)

var setDebugLog sync.Once

// SetDebugLog configures the default slog logger with debug level
// logging.
func SetDebugLog() {
	setDebugLog.Do(func() {
		handlerOpts := new(slog.HandlerOptions)
		handlerOpts.Level = slog.LevelDebug
		handler := slog.NewTextHandler(os.Stderr, handlerOpts)
		logger := slog.New(handler)
		slog.SetDefault(logger)
	})
}
