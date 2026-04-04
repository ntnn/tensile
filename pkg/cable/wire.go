package cable

import (
	"context"
	"log/slog"

	"github.com/ntnn/tensile"
)

var _ tensile.Cable = (*Wire)(nil)

// Wire is a basic implementation of [tensile.Cable].
type Wire struct {
	Ctx context.Context //nolint:containedctx
	Log *slog.Logger
}

// Context returns a context that is valid for the lifetime of a [tensile.Node].
func (w *Wire) Context() context.Context {
	return w.Ctx
}

// Logger returns an initialized logger that is configured for the [tensile.Node].
func (w *Wire) Logger() *slog.Logger {
	return w.Log
}
