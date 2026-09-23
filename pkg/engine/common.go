package engine

import (
	"log/slog"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/storage"
)

// Options are common options for all execution engines.
type Options struct {
	// Logger is the logger to use for logging.
	Logger *slog.Logger

	// Noop prevents execution of any actions. Instead the engine will
	// perform a dry run.
	Noop bool

	// Backend stores reported node outputs.
	Backend storage.Backend[tensile.Identity]
}

// WithDefaults returns a Options with default values.
func (o Options) WithDefaults() Options {
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	if o.Backend == nil {
		o.Backend = storage.NewDefaultBackend[tensile.Identity]()
	}
	return o
}
