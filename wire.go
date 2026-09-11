package tensile

import (
	"context"
	"log/slog"
)

// Wire provides some base functionality to [Node].
type Wire interface {
	// Context returns a context that is valid for the lifetime of the [Node].
	// That means the Context is valid from the first call to Validation
	// and until Execute has finished.
	Context() context.Context

	// Logger returns a logger for the [Node].
	Logger() *slog.Logger
}

var _ Wire = (*DefaultWire)(nil)

// DefaultWire is a basic [Wire]. Zero value is usable.
type DefaultWire struct {
	Ctx context.Context //nolint:containedctx
	Log *slog.Logger
}

// Context returns the carried context, defaulting to [context.Background].
func (w *DefaultWire) Context() context.Context {
	if w.Ctx == nil {
		return context.Background()
	}
	return w.Ctx
}

// Logger returns the carried logger, defaulting to [slog.Default].
func (w *DefaultWire) Logger() *slog.Logger {
	if w.Log == nil {
		return slog.Default()
	}
	return w.Log
}
