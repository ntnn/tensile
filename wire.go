package tensile

import (
	"context"
	"errors"
	"log/slog"

	"github.com/ntnn/tensile/pkg/storage"
)

// Wire provides some base functionality to [Node].
type Wire interface {
	// Context returns a context that is valid for the lifetime of the [Node].
	// That means the Context is valid from the first call to Validation
	// and until Execute has finished.
	Context() context.Context

	// Logger returns a logger for the [Node].
	Logger() *slog.Logger

	// Storage the [storage.Store], which provides output of other nodes.
	// The [storage.Store] only gives access to output of [Node] declared as dependencies.
	Storage() *storage.Store[Identity]

	// Topology returns a read-only view of the execution graph.
	// Errors when the wire carries no topology.
	Topology() (Topology, error)
}

// Topology is a read-only view of the built execution graph.
type Topology interface {
	// Claimer returns the node claiming identity.
	Claimer(identity Identity) (Identity, bool)
}

var _ Wire = (*DefaultWire)(nil)

// DefaultWire is a basic [Wire]. Zero value is usable.
type DefaultWire struct {
	Ctx   context.Context //nolint:containedctx
	Log   *slog.Logger
	Store *storage.Store[Identity]
	Topo  Topology
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

// Storage returns the carried store, defaulting to an empty store.
func (w *DefaultWire) Storage() *storage.Store[Identity] {
	if w.Store == nil {
		return &storage.Store[Identity]{}
	}
	return w.Store
}

// ErrNoTopology is returned when a [Wire] carries no [Topology].
var ErrNoTopology = errors.New("wire carries no topology")

// Topology returns the carried topology, [ErrNoTopology] when none is set.
func (w *DefaultWire) Topology() (Topology, error) {
	if w.Topo == nil {
		return nil, ErrNoTopology
	}
	return w.Topo, nil
}
