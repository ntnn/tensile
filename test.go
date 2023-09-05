package tensile

import (
	"log/slog"
	"os"
	"sync"
)

var setDebugLog sync.Once

// SetDebugLog is called from tensiles packages during unit tests to
// enable debug logging
func SetDebugLog() {
	setDebugLog.Do(func() {
		// in tests set a custom handler on debug level as default
		handlerOpts := new(slog.HandlerOptions)
		handlerOpts.Level = slog.LevelDebug
		handler := slog.NewTextHandler(os.Stderr, handlerOpts)
		logger := slog.New(handler)
		slog.SetDefault(logger)
	})
}
