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

func (w *Wire) Context() context.Context {
	return w.Ctx
}

func (w *Wire) Logger() *slog.Logger {
	return w.Log
}
