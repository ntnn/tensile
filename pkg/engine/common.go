package engine

import "log/slog"

// Options are common options for all execution engines.
type Options struct {
	// Logger is the logger to use for logging.
	Logger *slog.Logger

	// Noop prevents execution of any actions. Instead the engine will
	// perform a dry run.
	Noop bool
}

// WithDefaults returns a Options with default values.
func (o Options) WithDefaults() Options {
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	return o
}

// Summary is the summary of an execution.
type Summary struct {
	// NodesExecuted is the number of nodes that were executed.
	// In noop mode this indicates the number of nodes that would have
	// been executed if noop were disabled.
	NodesExecuted int
}
