package engines

import "context"

// interfaces commonly used/implemented by engines

// Engine is the base interface that all engines must satisfy.
type Engine interface {
	// Config should return a pointer to its configuration, allowing
	// to modify the behaviour of the engine.
	// TODO This feels much more like a hack and is mostly required to
	// implement wrapped engines.
	Config() *Config

	Noop(ctx context.Context) error
	Run(ctx context.Context) error
}
