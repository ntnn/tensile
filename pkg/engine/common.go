package engine

import (
	"log/slog"
	"sync"

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
