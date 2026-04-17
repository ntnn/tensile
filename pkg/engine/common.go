package engine

import (
	"log/slog"
	"sync"
)

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
	mu            sync.Mutex
	nodesExecuted int
}

// IncrementNodesExecuted safely increments the count of nodes executed.
func (s *Summary) IncrementNodesExecuted() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodesExecuted++
}

// NodesExecuted returns the number of nodes that were executed.
func (s *Summary) NodesExecuted() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.nodesExecuted
}
