package tensile

import (
	"context"
	"log/slog"
)

// Cable provides some base functionality to [Node].
type Cable interface {
	// Context returns a context that is valid for the lifetime of the [Node].
	// That means the Context is valid from the first call to Validation
	// and until Execute has finished.
	Context() context.Context

	// Logger returns a logger for the [Node].
	Logger() *slog.Logger
}
